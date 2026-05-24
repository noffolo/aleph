# Date Range Filtering — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional `start_date` / `end_date` temporal filtering to all ingestion source types in Aleph.

**Architecture:** A shared `DateFilter` component (`internal/ingestion/sources/datefilter.go`) with `ParseDateRangeFromConfig()` and `IsInRange()`. Each source handler embeds DateRangeConfig in its config parsing, extracts dates from items, and filters before writing. Fields live in `config_json` only — no proto/DB schema changes.

**Tech Stack:** Go 1.26, testify for tests, DuckDB for storage.

**Design doc:** `docs/specs/2026-05-22-temporal-filters-design.md`

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/ingestion/sources/datefilter.go` | DateRangeConfig, ParseDateRangeFromConfig, IsInRange (NEW) |
| `internal/ingestion/sources/datefilter_test.go` | Tests for DateFilter component (NEW) |
| `internal/ingestion/sources/sitemap.go` | PageResult gets ParsedDate; CrawlSitemap filters by LastMod (MODIFY) |
| `internal/ingestion/sources/sitemap_test.go` | Tests for sitemap date filtering (MODIFY) |
| `internal/ingestion/engine.go` | runSitemapSource, runPrecompiled get date range config (MODIFY) |
| `internal/ingestion/sources/rss.go` or inline in engine.go | Date extraction + filtering for RSS items (NEW or MODIFY) |

---

### Task 1: DateFilter Component

**Files:**
- Create: `internal/ingestion/sources/datefilter.go`
- Create: `internal/ingestion/sources/datefilter_test.go`

- [ ] **Step 1: Write the failing tests in datefilter_test.go**

```go
package sources

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseDateRangeFromConfig_Empty(t *testing.T) {
    raw := json.RawMessage(`{}`)
    dr, err := ParseDateRangeFromConfig(raw)
    require.NoError(t, err)
    assert.Nil(t, dr.StartDate)
    assert.Nil(t, dr.EndDate)
}

func TestParseDateRangeFromConfig_StartOnly(t *testing.T) {
    raw := json.RawMessage(`{"start_date":"2025-05-22"}`)
    dr, err := ParseDateRangeFromConfig(raw)
    require.NoError(t, err)
    require.NotNil(t, dr.StartDate)
    assert.Equal(t, 2025, dr.StartDate.Year())
    assert.Equal(t, time.May, dr.StartDate.Month())
    assert.Equal(t, 22, dr.StartDate.Day())
    assert.Nil(t, dr.EndDate)
}

func TestParseDateRangeFromConfig_Both(t *testing.T) {
    raw := json.RawMessage(`{"start_date":"2025-01-01","end_date":"2026-05-22"}`)
    dr, err := ParseDateRangeFromConfig(raw)
    require.NoError(t, err)
    require.NotNil(t, dr.StartDate)
    require.NotNil(t, dr.EndDate)
    assert.True(t, dr.EndDate.After(*dr.StartDate))
}

func TestParseDateRangeFromConfig_RFC3339(t *testing.T) {
    raw := json.RawMessage(`{"start_date":"2025-05-22T15:04:05Z"}`)
    dr, err := ParseDateRangeFromConfig(raw)
    require.NoError(t, err)
    require.NotNil(t, dr.StartDate)
    assert.Equal(t, 15, dr.StartDate.Hour())
    assert.Equal(t, 4, dr.StartDate.Minute())
}

func TestParseDateRangeFromConfig_UnixTimestamp(t *testing.T) {
    raw := json.RawMessage(`{"start_date":1700000000}`)
    dr, err := ParseDateRangeFromConfig(raw)
    require.NoError(t, err)
    require.NotNil(t, dr.StartDate)
    assert.Equal(t, int64(1700000000), dr.StartDate.Unix())
}

func TestParseDateRangeFromConfig_InvalidFormat(t *testing.T) {
    raw := json.RawMessage(`{"start_date":"not-a-date"}`)
    _, err := ParseDateRangeFromConfig(raw)
    assert.Error(t, err)
}

func TestIsInRange_NoFilter(t *testing.T) {
    dr := DateRangeConfig{}
    now := time.Now()
    assert.True(t, dr.IsInRange(&now))
}

func TestIsInRange_NilDate(t *testing.T) {
    dr := DateRangeConfig{StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))}
    assert.True(t, dr.IsInRange(nil))
}

func TestIsInRange_BeforeStart(t *testing.T) {
    start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{StartDate: &start}
    early := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
    assert.False(t, dr.IsInRange(&early))
}

func TestIsInRange_AfterEnd(t *testing.T) {
    end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{EndDate: &end}
    late := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
    assert.False(t, dr.IsInRange(&late))
}

func TestIsInRange_WithinRange(t *testing.T) {
    start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{StartDate: &start, EndDate: &end}
    mid := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
    assert.True(t, dr.IsInRange(&mid))
}

func TestIsInRange_OnStartBoundary(t *testing.T) {
    start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{StartDate: &start}
    same := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    assert.True(t, dr.IsInRange(&same))
}

func TestIsInRange_OnEndBoundary(t *testing.T) {
    end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{EndDate: &end}
    same := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
    assert.True(t, dr.IsInRange(&same))
}

func ptr(t time.Time) *time.Time { return &t }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestParseDateRange|TestIsInRange" -v`
Expected: `FAIL` — `ParseDateRangeFromConfig` not defined, `DateRangeConfig` not defined.

- [ ] **Step 3: Write DateFilter implementation in datefilter.go**

```go
package sources

import (
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
    "time"
)

// DateRangeConfig holds optional temporal filter bounds for ingestion tasks.
// Values are extracted from the task's config_json.
type DateRangeConfig struct {
    StartDate *time.Time `json:"start_date,omitempty"`
    EndDate   *time.Time `json:"end_date,omitempty"`
}

// ParseDateRangeFromConfig extracts DateRangeConfig from a task's config_json.
// Recognised formats: YYYY-MM-DD, RFC3339, Unix timestamp (int seconds).
// Returns zero-value DateRangeConfig (both nil) if neither field is present.
func ParseDateRangeFromConfig(raw json.RawMessage) (DateRangeConfig, error) {
    var rawMap map[string]json.RawMessage
    if err := json.Unmarshal(raw, &rawMap); err != nil {
        return DateRangeConfig{}, nil // ignore — not a date-range issue
    }

    var dr DateRangeConfig

    if rawStart, ok := rawMap["start_date"]; ok && len(rawStart) > 0 {
        t, err := parseDateValue(rawStart)
        if err != nil {
            return DateRangeConfig{}, fmt.Errorf("start_date: %w", err)
        }
        dr.StartDate = &t
    }

    if rawEnd, ok := rawMap["end_date"]; ok && len(rawEnd) > 0 {
        t, err := parseDateValue(rawEnd)
        if err != nil {
            return DateRangeConfig{}, fmt.Errorf("end_date: %w", err)
        }
        dr.EndDate = &t
    }

    return dr, nil
}

func parseDateValue(raw json.RawMessage) (time.Time, error) {
    var s string
    if err := json.Unmarshal(raw, &s); err == nil {
        s = strings.TrimSpace(s)
        // Try ISO date (YYYY-MM-DD)
        if t, err := time.Parse("2006-01-02", s); err == nil {
            return t, nil
        }
        // Try RFC3339
        if t, err := time.Parse(time.RFC3339, s); err == nil {
            return t, nil
        }
        // Try RFC3339 without timezone (assume UTC)
        if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
            return t, nil
        }
        return time.Time{}, fmt.Errorf("unrecognised date format: %q", s)
    }

    // Try integer (Unix timestamp)
    var n int64
    if err := json.Unmarshal(raw, &n); err == nil {
        if n > 0 {
            return time.Unix(n, 0).UTC(), nil
        }
    }

    // Try float (Unix timestamp with fractional seconds — rare, be defensive)
    var f float64
    if err := json.Unmarshal(raw, &f); err == nil && f > 0 {
        sec := int64(f)
        return time.Unix(sec, 0).UTC(), nil
    }

    return time.Time{}, fmt.Errorf("cannot parse date value from JSON: %s", string(raw))
}

// IsInRange checks an extracted time against the filter bounds.
// If dr has no bounds set, returns true (no filter).
// If itemDate is nil (date not extractable), returns true (include anyway:
// "meglio dati in più che in meno").
func (dr DateRangeConfig) IsInRange(itemDate *time.Time) bool {
    if dr.StartDate == nil && dr.EndDate == nil {
        return true // no filter
    }
    if itemDate == nil {
        return true // date not extractable — include anyway
    }
    if dr.StartDate != nil && itemDate.Before(*dr.StartDate) {
        return false
    }
    if dr.EndDate != nil && itemDate.After(*dr.EndDate) {
        return false
    }
    return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestParseDateRange|TestIsInRange" -v`
Expected: `PASS` — all 12 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/sources/datefilter.go internal/ingestion/sources/datefilter_test.go
git commit -m "feat(ingestion): add DateFilter component with config parsing and IsInRange"
```

---

### Task 2: Add ParsedDate to PageResult + Filter in Sitemap

**Files:**
- Modify: `internal/ingestion/sources/sitemap.go`
- Modify: `internal/ingestion/sources/sitemap_test.go`
- Modify: `internal/ingestion/engine.go` (runSitemapSource)

- [ ] **Step 1: Write failing test for sitemap filtering**

Add to `sitemap_test.go`:

```go
func TestCrawlSitemap_FiltersByLastMod(t *testing.T) {
    sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/old</loc><lastmod>2024-01-15</lastmod></url>
  <url><loc>http://example.com/new</loc><lastmod>2025-06-15</lastmod></url>
  <url><loc>http://example.com/nodate</loc></url>
</urlset>`

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/xml")
        w.Write([]byte(sitemapXML))
    }))
    defer ts.Close()

    s := NewSitemapIngester()
    // Override client to use test server
    s.client = &http.Client{Timeout: 5 * time.Second}

    result, err := s.CrawlSitemap(context.Background(), ts.URL+"/sitemap.xml")
    require.NoError(t, err)
    require.Len(t, result.URLs, 0) // no page fetching, only XML parse — URL set has 3 entries
    // We'll test filtering separately via the new date-aware ParseSitemapURLs function
}
```

Better approach: test the new `FilterPageResults` function directly:

```go
func TestFilterPageResults_ByDateRange(t *testing.T) {
    now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
    old := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
    pages := []PageResult{
        {URL: "/old", Status: 200, Content: []byte("old"), Size: 3, ParsedDate: &old},
        {URL: "/new", Status: 200, Content: []byte("new"), Size: 3, ParsedDate: &now},
        {URL: "/nodate", Status: 200, Content: []byte("nodate"), Size: 6, ParsedDate: nil},
    }

    start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    dr := DateRangeConfig{StartDate: &start}

    filtered := FilterPageResults(pages, dr)
    require.Len(t, filtered, 2) // /new and /nodate
    assert.Equal(t, "/new", filtered[0].URL)
    assert.Equal(t, "/nodate", filtered[1].URL)
}

func TestFilterPageResults_NoFilter(t *testing.T) {
    pages := []PageResult{
        {URL: "/a", ParsedDate: ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))},
        {URL: "/b", ParsedDate: nil},
    }
    filtered := FilterPageResults(pages, DateRangeConfig{})
    assert.Len(t, filtered, 2)
}
```

Add `FilterPageResults` and update `fetchAllPages` / `CrawlSitemap` to parse dates from URLEntry.LastMod.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestFilterPageResults" -v`

- [ ] **Step 3: Modify sitemap.go to add ParsedDate + FilterPageResults**

1. Add `ParsedDate *time.Time` to `PageResult` struct
2. Add `FilterPageResults(pages []PageResult, dr DateRangeConfig) []PageResult`
3. In `CrawlSitemap`, after collecting URLs from XML, parse each `URLEntry.LastMod` into `ParsedDate` on the `PageResult`
4. In `runSitemapSource` in `engine.go`, extract `DateRangeConfig` and call `FilterPageResults` before writing rows

```go
// Add to PageResult:
type PageResult struct {
    URL        string
    Content    []byte
    Size       int64
    Status     int
    Err        error
    ParsedDate *time.Time `json:"parsed_date,omitempty"`
}

// FilterPageResults filters pages by date range, keeping pages where:
// - date is within range, OR
// - date is nil (not extractable — include anyway)
// - no date range is configured (keep all)
func FilterPageResults(pages []PageResult, dr DateRangeConfig) []PageResult {
    if dr.StartDate == nil && dr.EndDate == nil {
        return pages
    }
    filtered := make([]PageResult, 0, len(pages))
    for _, p := range pages {
        if dr.IsInRange(p.ParsedDate) {
            filtered = append(filtered, p)
        }
    }
    return filtered
}
```

In `CrawlSitemap`, after parsing the URLSet, parse LastMod dates:

```go
// In the urlset case, after building the urls slice, also record LastMod:
for _, entry := range urlSet.URLs {
    urlStr := strings.TrimSpace(entry.Loc)
    if urlStr == "" {
        continue
    }
    urls = append(urls, urlStr)
}

// ... later, after fetchAllPages returns results:
// Parse LastMod dates into PageResult
for i, p := range result.URLs {
    // Match back to URLEntry by URL
    for _, entry := range urlSet.URLs {
        if strings.TrimSpace(entry.Loc) == p.URL && entry.LastMod != "" {
            parsed, err := parseSitemapDate(entry.LastMod)
            if err == nil {
                result.URLs[i].ParsedDate = &parsed
            }
            break
        }
    }
}
```

Add `parseSitemapDate` helper:

```go
func parseSitemapDate(s string) (time.Time, error) {
    s = strings.TrimSpace(s)
    // W3C Datetime format variants (ISO 8601 subset)
    formats := []string{
        "2006-01-02T15:04:05Z07:00",
        "2006-01-02T15:04:05Z",
        "2006-01-02T15:04:05",
        "2006-01-02",
    }
    for _, f := range formats {
        if t, err := time.Parse(f, s); err == nil {
            return t.UTC(), nil
        }
    }
    return time.Time{}, fmt.Errorf("unrecognised sitemap date: %q", s)
}
```

- [ ] **Step 4: Update runSitemapSource in engine.go**

Around line 1517, add date range parsing and filtering:

```go
func (e *Engine) runSitemapSource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
    var config struct {
        URL string `json:"url"`
        DateRangeConfig
    }
    if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
        return fmt.Errorf("sitemap config JSON invalid: %w", err)
    }
    if config.URL == "" {
        return fmt.Errorf("sitemap config requires url")
    }

    // Parse date range from config
    dr, err := ParseDateRangeFromConfig([]byte(task.ConfigJson))
    if err != nil {
        return fmt.Errorf("sitemap date range config: %w", err)
    }

    // ... existing crawl logic ...

    crawlResult, err := ingester.CrawlSitemap(ctx, config.URL)
    // ... error handling ...

    // Apply date filter
    pages := ingester.FilterPageResults(crawlResult.URLs, dr)
    fmt.Fprintf(w, "Found %d pages, %d after date filter\n", len(crawlResult.URLs), len(pages))

    // Use filtered `pages` instead of `crawlResult.URLs` for the rest
    // ... write rows to DuckDB using pages ...
}
```

- [ ] **Step 5: Run all sitemap tests**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestSitemap|TestFilterPage|TestCrawl" -v`
Expected: `PASS` (both new and existing tests)

- [ ] **Step 6: Build to verify compilation**

Run: `cd /tmp/opencode/aleph && go build ./cmd/aleph-server/`
Expected: exit code 0, no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/ingestion/sources/sitemap.go internal/ingestion/sources/sitemap_test.go internal/ingestion/engine.go
git commit -m "feat(ingestion): add ParsedDate to PageResult, date filtering in sitemap handler"
```

---

### Task 3: Add Date Filtering to RSS (runPrecompiled)

**Files:**
- Modify: `internal/ingestion/engine.go` (runPrecompiled)

RSS/Atom feeds return items with dates via `runPrecompiled` which fetches the URL and stores the raw response as a JSON/CSV view. For date filtering we need to parse the response, extract item dates, and only submit tasks for items in range.

Note: This is more complex because `runPrecompiled` stores the raw response directly. For RSS with date filtering, we need to:
1. Parse the RSS XML in-memory to extract item dates
2. Only submit `url` tasks for items within the date range
3. Fall back to storing the raw feed if no date filter is configured

- [ ] **Step 1: Write test for RSS date extraction**

Create `internal/ingestion/sources/rss_test.go`:

```go
package sources

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExtractRSSItemDate(t *testing.T) {
    // RSS 2.0 pubDate
    date, err := ExtractRSSItemDate(map[string]string{"pubDate": "Mon, 15 Jan 2025 10:00:00 GMT"})
    require.NoError(t, err)
    assert.Equal(t, 2025, date.Year())

    // Atom updated
    date2, err := ExtractRSSItemDate(map[string]string{"updated": "2025-06-15T14:30:00Z"})
    require.NoError(t, err)
    assert.Equal(t, time.June, date2.Month())

    // No date
    _, err = ExtractRSSItemDate(map[string]string{"title": "no date"})
    assert.Error(t, err)
}
```

- [ ] **Step 2: Implement RSS date extraction**

Add to a new file `internal/ingestion/sources/rss.go` or inline in engine.go.

- [ ] **Step 3: Update runPrecompiled to filter RSS items by date**

The handler currently stores the raw response as a DuckDB view. For RSS feeds with date range configured, change the flow to:
1. Parse the RSS/Atom XML (check if it's RSS or Atom)
2. Extract publication dates from each item
3. Only forward items within range
4. Store filtered results as the feed view

- [ ] **Step 4: Run all tests**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/... -v`
Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/engine.go internal/ingestion/sources/rss.go internal/ingestion/sources/rss_test.go
git commit -m "feat(ingestion): add date range filtering for RSS/Atom feed ingestion"
```

---

### Task 4: Add Date Filtering to JSONAPI, Scrape, CSV, GitHub, Email

**Files:**
- Modify: `internal/ingestion/engine.go` (runJSONAPISource, runScrapeSource, runCSVLoad)
- Modify: `internal/ingestion/sources/sources.go` or inline in each handler

Each source handler follows the same pattern:
1. Parse `DateRangeConfig` from `config_json`
2. Extract date from each item (using source-specific config fields)
3. Filter items with `dr.IsInRange(itemDate)`
4. Write only filtered items

- [ ] **Step 1: For each source type, add `date_path`/`date_format`/`date_column` config fields and date extraction**

- [ ] **Step 2: Run all tests**

Run: `cd /tmp/opencode/aleph && go test ./internal/ingestion/... -v`
Expected: `PASS`

- [ ] **Step 3: Build**

Run: `cd /tmp/opencode/aleph && go build ./cmd/aleph-server/`

- [ ] **Step 4: Commit**

```bash
git add internal/ingestion/engine.go
git commit -m "feat(ingestion): add date range filtering for jsonapi, scrape, csv, github, email"
```
