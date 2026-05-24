# Election & Funding Ingestion Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the Eligendo API URL format (path-based, not query params), add election date support, and create CLI tools for batch election ingestion and party funding import.

**Architecture:** Fix `election.go` API calls to use path-based format (`getentiFI/DE/{date}/TE/{code}`) with hierarchical entity paths (`RE/{reg}/PR/{prov}/CM/{com}`). Add `election_date` to `ElectionConfig`. Create two standalone CLI tools in `cmd/` that use DuckDB directly for data storage. Funding CLI downloads the ondata CSV and imports via DuckDB `read_csv_auto`.

**Tech Stack:** Go, DuckDB (go-duckdb), `golang.org/x/text/encoding/charmap` (for funding CSV encoding if needed), net/http

---

### Task 1: Fix election.go — Path-based API URLs + Headers

**Files:**
- Modify: `internal/ingestion/sources/election.go`
- Modify: `internal/ingestion/sources/election_test.go`

**Details:**
The Eligendo API uses path-based URLs, not query params:
- `getentiFI/DE/{YYYYMMDD}/TE/{te}` (was `getentiFI?te=TE01&liv=regione`)
- `scrutiniFI/DE/{YYYYMMDD}/TE/{te}/RE/{reg}/PR/{prov}/CM/{com}` (was `scrutiniFI?te=TE01&cod=058091`)
- Entity codes are `RRPPPCCCC` — need to extract region (2 chars), province (3 chars), comune (4 chars)
- API required Origin/Referer headers

Changes needed:
1. Change `ElectionConfig` to include `ElectionDate string \`json:"election_date"\`` (YYYYMMDD format)
2. Add `Origin` and `Referer` headers to `doGet()`
3. Replace `GetEntities` URL builder from query to path-based
4. Replace scrutiniFI URL builder from query to path-based with RE/PR/CM extraction
5. Update test server to use path-based format
6. Update mock entities to have `RRPPPCCCC` format codes (e.g., `12058091` = Lazio 12 / Roma 058 / 0091)

The entity code structure is: `RRPPPCCCC` where:
- First 2 chars = region code (01-20)
- Next 3 chars = province ISTAT code
- Last 4 chars = comune ISTAT code

For scrutiniFI URL:
- RE = first 2 chars of entity code (e.g., "12")
- PR = next 3 chars (e.g., "058")
- CM = last 4 chars (e.g., "0091")

- [ ] **Step 1: Update ElectionConfig + test**

```go
// ElectionConfig holds the parameters used to identify a specific election to process.
type ElectionConfig struct {
	ElectionType string `json:"election_type"`
	Level        string `json:"level"`
	Year         int    `json:"year"`
	ElectionDate string `json:"election_date"` // YYYYMMDD format — required for path-based API
}
```

Update test in `election_test.go` to include `ElectionDate`:

```go
cfg := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022, ElectionDate: "20220925"}
```

- [ ] **Step 2: Update doGet with proper headers**

In `doGet()`, add Origin and Referer headers:

```go
req.Header.Set("Origin", "https://elezioni.interno.gov.it")
req.Header.Set("Referer", "https://elezioni.interno.gov.it/")
```

- [ ] **Step 3: Update GetEntities to use path-based URL**

Replace:
```go
url := fmt.Sprintf("%s/getentiFI?te=%s&liv=%s", f.baseURL, cfg.teCode(), cfg.Level)
```
With:
```go
url := fmt.Sprintf("%s/getentiFI/DE/%s/TE/%s", f.baseURL, cfg.ElectionDate, cfg.teCode())
```

The existing `Level` parameter is dropped — the API returns all levels in the entity tree.

- [ ] **Step 4: Update scrutiniFI to use path-based URL with hierarchical codes**

Extract RE/PR/CM from the entity code:

```go
func extractRegPrvCom(cod string) (reg, prv, com string, err error) {
	if len(cod) < 9 {
		return "", "", "", fmt.Errorf("invalid entity code: %s (need 9+ chars)", cod)
	}
	return cod[:2], cod[2:5], cod[5:9], nil
}
```

Replace scrutiniFI URL:
```go
reg, prv, com, err := extractRegPrvCom(ent.Cod)
if err != nil {
    slog.Error("invalid entity code", "cod", ent.Cod, "error", err)
    continue
}
url := fmt.Sprintf("%s/scrutiniFI/DE/%s/TE/%s/RE/%s/PR/%s/CM/%s", baseURL, cfg.ElectionDate, cfg.teCode(), reg, prv, com)
```

- [ ] **Step 5: Update test server to handle path-based format**

```go
func TestElectionFetcherGetEntities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "https://elezioni.interno.gov.it", r.Header.Get("Origin"))
		// Check path pattern
		assert.Contains(t, r.URL.Path, "/getentiFI/DE/20220925/TE/TE01")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"intestazione": map[string]string{"te": "TE01"},
			"enti": map[string]interface{}{
				"ente": []map[string]string{
					{"cod": "120580091", "desc": "ROMA"},
				},
			},
		})
	}))
	defer srv.Close()

	fetcher := NewElectionFetcher(srv.URL, 100.0)
	cfg := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022, ElectionDate: "20220925"}
	entities, err := fetcher.GetEntities(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, "120580091", entities[0].Cod)
	assert.Equal(t, "ROMA", entities[0].Desc)
}
```

- [ ] **Step 6: Update RunElection test server and test**

Update the RunElectionFullPipeline test to use path-based mock server and `ElectionDate` in config.

- [ ] **Step 7: Run tests, fix any issues, commit**

```bash
go test ./internal/ingestion/sources/ -run TestElection -v -count=1 2>&1
go test ./internal/ingestion/sources/ -run TestRunElection -v -count=1 2>&1
# All election tests must pass
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go
git commit -m "fix: update Eligendo API to use path-based URLs with proper headers"
```

---

### Task 2: Election date lookup

**Files:**
- Create: `internal/ingestion/sources/electiondates.go`

**Details:**
We need a mapping of election type + year → election date (YYYYMMDD) for batch ingestion. Hardcode known election dates from 2006-2026. The CLI tool will use this to find dates when iterating.

- [ ] **Step 1: Create electiondates.go**

```go
package sources

// ElectionDateMap maps (election_type, year) -> election_date (YYYYMMDD)
// Source: https://elezioni.interno.gov.it/ historical data
var ElectionDateMap = map[string]map[int]string{
	"politiche": {
		2006: "20060409",
		2008: "20080413",
		2013: "20130224",
		2018: "20180304",
		2022: "20220925",
	},
	"europee": {
		2009: "20090606",
		2014: "20140525",
		2019: "20190526",
		2024: "20240608",
	},
	"regionali": {
		2010: "20100328",
		2015: "20150531",
		2020: "20200920",
		2024: "20241028", // Emilia-Romagna, Umbria etc.
	},
	"comunali": {},  // varies by comune — too many to list
	"provinciali": {},
	"referendum": {
		2006: "20060625",
		2009: "20090621",
		2011: "20110612",
		2016: "20161204",
		2020: "20200920",
		2022: "20220612",
	},
}

// GetElectionDate returns the election date for a given type and year.
// Returns empty string if not found.
func GetElectionDate(electionType string, year int) string {
	if m, ok := ElectionDateMap[electionType]; ok {
		return m[year]
	}
	return ""
}
```

- [ ] **Step 2: Write and run tests**

```go
func TestElectionDateMap(t *testing.T) {
	assert.Equal(t, "20220925", GetElectionDate("politiche", 2022))
	assert.Equal(t, "", GetElectionDate("politiche", 2000))   // no data
	assert.Equal(t, "", GetElectionDate("comunali", 2023))    // empty map
	assert.Equal(t, "", GetElectionDate("fantasia", 2022))    // unknown type
}
```

```bash
go test ./internal/ingestion/sources/ -run TestElectionDate -v -count=1
git add internal/ingestion/sources/electiondates.go
git commit -m "feat: add election date lookup for batch ingestion"
```

---

### Task 3: Create cmd/ingest-elections — Batch Election Ingestion Tool

**Files:**
- Create: `cmd/ingest-elections/main.go`

**Details:**
Standalone CLI tool that iterates over election types/years, calls `RunElection` for each, saves results to a DuckDB database. Uses the fixed path-based API.

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

func main() {
	dbPath := flag.String("db", "elections.duckdb", "Path to DuckDB database")
	rawDir := flag.String("raw", "./raw", "Directory for raw API dumps")
	years := flag.String("years", "", "Comma-separated years (default: all available)")
	types := flag.String("types", "", "Election types (default: all)")
	baseURL := flag.String("base-url", "https://eleapi.interno.gov.it/siel/PX", "Eligendo API base URL")
	rate := flag.Float64("rate", 0.5, "Requests per second")
	flag.Parse()

	db, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=READ_WRITE", *dbPath))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	mapper := sources.NewPartyMapper()
	_ = mapper // Load from embedded aliases

	// Determine which elections to process
	allTypes := []string{"politiche", "europee", "regionali", "comunali", "referendum"}
	if *types != "" {
		allTypes = splitAndTrim(*types)
	}

	electionYears := map[string][]int{
		"politiche":    {2006, 2008, 2013, 2018, 2022},
		"europee":      {2009, 2014, 2019, 2024},
		"regionali":    {2010, 2015, 2020},
		"comunali":     {}, // skipped (no generic date)
		"referendum":   {2006, 2009, 2011, 2016, 2020, 2022},
	}

	for _, et := range allTypes {
		yearsForType := electionYears[et]
		if len(yearsForType) == 0 {
			slog.Info("skipping type (no generic date)", "type", et)
			continue
		}
		for _, year := range yearsForType {
			dateStr := sources.GetElectionDate(et, year)
			if dateStr == "" {
				slog.Info("skipping (no date)", "type", et, "year", year)
				continue
			}

			cfg := sources.ElectionConfig{
				ElectionType: et,
				Level:        "comune",
				Year:         year,
				ElectionDate: dateStr,
			}

			slog.Info("processing election", "type", et, "year", year, "date", dateStr)
			results, err := sources.RunElection(context.Background(), db, *baseURL, cfg, mapper, *rawDir)
			if err != nil {
				slog.Error("election failed", "type", et, "year", year, "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			slog.Info("election complete", "type", et, "year", year, "results", len(results))
		}
	}
}

func splitAndTrim(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /tmp/opencode/aleph && go build ./cmd/ingest-elections/ 2>&1
# Should compile without errors
```

- [ ] **Step 3: Commit**

```bash
git add cmd/ingest-elections/main.go
git commit -m "feat: add batch election ingestion CLI tool"
```

---

### Task 4: Create cmd/ingest-funding — Party Funding Import Tool

**Files:**
- Create: `cmd/ingest-funding/main.go`

**Details:**
Standalone CLI tool that downloads the ondata/liberiamoli-tutti political_finance.csv and imports it into DuckDB using the existing `ImportFundingCSV` function.

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

const defaultCSVURL = "https://raw.githubusercontent.com/ondata/liberiamoli-tutti/main/soldi_e_politica/dati/political_finance.csv"

func main() {
	dbPath := flag.String("db", "funding.duckdb", "Path to DuckDB database")
	rawDir := flag.String("raw", "./raw", "Directory for raw CSV storage")
	csvURL := flag.String("csv-url", defaultCSVURL, "URL of the party funding CSV")
	csvPath := flag.String("csv-path", "", "Local path to CSV (alternative to download)")
	flag.Parse()

	db, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=READ_WRITE", *dbPath))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var localPath string
	if *csvPath != "" {
		localPath = *csvPath
		slog.Info("using local CSV", "path", localPath)
	} else {
		localPath = filepath.Join(*rawDir, "party_funding", "political_finance.csv")
		os.MkdirAll(filepath.Dir(localPath), 0755)

		slog.Info("downloading CSV", "url", *csvURL)
		resp, err := http.Get(*csvURL)
		if err != nil {
			log.Fatalf("download CSV: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Fatalf("HTTP %d downloading CSV", resp.StatusCode)
		}

		f, err := os.Create(localPath)
		if err != nil {
			log.Fatalf("create file: %v", err)
		}

		written, err := io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			log.Fatalf("write CSV: %v", err)
		}
		slog.Info("downloaded CSV", "bytes", written)
	}

	wm := &simpleWatermark{}
	if err := sources.ImportFundingCSV(context.Background(), db, wm, localPath, *rawDir); err != nil {
		log.Fatalf("import CSV: %v", err)
	}
	slog.Info("import complete")
}

// simpleWatermark implements sources.WatermarkSetter (minimal, no DB persistence)
type simpleWatermark struct{}

func (s *simpleWatermark) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	slog.Info("watermark set", "source", sourceName, "last_run", lastRun)
	return nil
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /tmp/opencode/aleph && go build ./cmd/ingest-funding/ 2>&1
# Should compile without errors
```

- [ ] **Step 3: Add strings import if missing, fix any build issues, commit**

```bash
git add cmd/ingest-funding/main.go
git commit -m "feat: add party funding ingestion CLI tool"
```

---

### Task 5: Integrate — Wire funding source into engine registry

**Files:**
- Modify: `internal/ingestion/engine.go`
- Modify: `internal/app/app.go`

**Details:**
The election.go notes that GlobalRegistry registration must happen externally. Currently the engine validates registry sources but doesn't execute them. We need to add actual execution for "election" and "party_funding" source types, wiring them through the engine's existing task execution system.

This is lower priority since the CLI tools handle immediate ingestion. The registry integration ensures the regular ingestion pipeline works too.

- [ ] **Step 1: Add election execution in engine.go**

After the existing registry validation (line 193), instead of returning an error, add execution:

```go
case "election", "party_funding":
    // These source types use the engine's execution via cmd tools for now
    taskErr = fmt.Errorf("use cmd/ingest-elections or cmd/ingest-funding for %s", task.SourceType)
```

- [ ] **Step 2: Commit**

```bash
git add internal/ingestion/engine.go
git commit -m "chore: clarify execution path for election/funding source types"
```

---

## VERIFICA (QA Scenarios)

### Election Ingestion Tool
- **Tool:** `cd /tmp/opencode/aleph && go build ./cmd/ingest-elections/`
- **Steps:** (1) Build succeeds, (2) Run `go test ./internal/ingestion/sources/ -run TestElection -v` passes, (3) All election tests pass with path-based URL format
- **Expected result:** Binary compiles. API calls use `getentiFI/DE/{date}/TE/{type}` format with Origin/Referer headers.

### Party Funding Import Tool
- **Tool:** `cd /tmp/opencode/aleph && go build ./cmd/ingest-funding/`
- **Steps:** (1) Build succeeds, (2) Dry-run with `-csv-path` pointing to a local test CSV
- **Expected result:** Binary compiles. CSV downloaded and imported into DuckDB.
