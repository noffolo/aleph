# Italian News Ingestion & Web Scraper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand Aleph's ingestion pipeline with 15+ Italian news RSS feeds, a dedicated web scraping tool for non-RSS sources (DAGOSPIA, Corriere, Sole 24 Ore, Il Post), and frontend management UI.

**Architecture:** New `ScrapeIngester` in `internal/ingestion/sources/scraper.go` that uses `github.com/PuerkitoBio/goquery` for HTML parsing with CSS selectors. Registered as `source_type: "scrape"` in the Engine dispatch. RSS feeds added via batch API calls. Frontend extended in `DataSourcesView` and `DataSourceForm` to support the new source type.

**Tech Stack:** Go 1.23+ (goquery for HTML parsing), TypeScript/React (Shadcn/Radix UI components), DuckDB (auto-inferred schemas), Connect RPC (gRPC API)

---

### Task 1: Web Scraper Backend — goquery-based HTML scraper

**Files:**
- Create: `/tmp/opencode/aleph/internal/ingestion/sources/scraper.go`
- Modify: `/tmp/opencode/aleph/internal/ingestion/probe.go:60-63` (add "scrape" to Validate)
- Modify: `/tmp/opencode/aleph/internal/ingestion/probe.go:184-239` (add scrape to classifySourceType)
- Modify: `/tmp/opencode/aleph/internal/ingestion/engine.go:80-98` (add scraperIngester to Engine struct)
- Modify: `/tmp/opencode/aleph/internal/ingestion/engine.go:170-195` (add "scrape" case to RunTask dispatch)
- Modify: `/tmp/opencode/aleph/go.mod` (add goquery dependency)

- [ ] **Step 1: Add goquery dependency**

```bash
cd /tmp/opencode/aleph && go get github.com/PuerkitoBio/goquery
```

Run: `go mod tidy`
Expected: dependency added to go.mod

- [ ] **Step 2: Create `scraper.go` — ScrapeIngester struct and core logic**

```go
// Package sources implements W3 ingestion methods including HTML scraping.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// ScrapeConfig defines how to extract structured data from an HTML page.
type ScrapeConfig struct {
	URL            string `json:"url"`
	ArticleSelector string `json:"article_selector"` // CSS selector for each article container
	TitleSelector  string `json:"title_selector"`    // CSS selector within article for title
	LinkSelector   string `json:"link_selector"`     // CSS selector for article link (usually "a")
	DateSelector   string `json:"date_selector"`     // optional CSS selector for date
	AuthorSelector string `json:"author_selector"`   // optional CSS selector for author
	ContentSelector string `json:"content_selector"` // optional CSS selector for article body (detail page scraping)
	MaxArticles    int    `json:"max_articles"`      // max articles to extract (default: 50)
}

// ScrapeResult is a single extracted article.
type ScrapeResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Date    string `json:"date,omitempty"`
	Author  string `json:"author,omitempty"`
	Content string `json:"content,omitempty"`
}

// ScrapeIngester fetches HTML pages and extracts structured article data via CSS selectors.
type ScrapeIngester struct {
	client *RateLimitedClient
}

// NewScrapeIngester creates a ScrapeIngester with default rate limiting (1 req/s).
func NewScrapeIngester() *ScrapeIngester {
	return &ScrapeIngester{
		client: NewRateLimitedClient(DefaultRate),
	}
}

// Scrape fetches a page, extracts articles using the provided config, and returns them as JSON.
func (s *ScrapeIngester) Scrape(ctx context.Context, config *ScrapeConfig) ([]ScrapeResult, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("scrape: URL must be non-empty")
	}
	if config.ArticleSelector == "" {
		return nil, fmt.Errorf("scrape: article_selector must be non-empty")
	}
	if config.TitleSelector == "" {
		return nil, fmt.Errorf("scrape: title_selector must be non-empty")
	}
	if config.MaxArticles <= 0 {
		config.MaxArticles = 50
	}

	body, err := s.fetchHTML(ctx, config.URL)
	if err != nil {
		return nil, fmt.Errorf("scrape: fetch %s: %w", config.URL, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("scrape: parse HTML: %w", err)
	}

	var results []ScrapeResult
	baseURL, _ := url.Parse(config.URL)

	doc.Find(config.ArticleSelector).Each(func(i int, sel *goquery.Selection) {
		if len(results) >= config.MaxArticles {
			return
		}

		title := strings.TrimSpace(sel.Find(config.TitleSelector).First().Text())
		if title == "" {
			return // skip empty articles
		}

		var link string
		if config.LinkSelector != "" {
			linkEl := sel.Find(config.LinkSelector).First()
			if href, exists := linkEl.Attr("href"); exists {
				link = resolveHref(baseURL, href)
			}
		}
		if link == "" {
			// try to find any link within the article
			sel.Find("a[href]").First().Each(func(_ int, a *goquery.Selection) {
				if href, exists := a.Attr("href"); exists && link == "" {
					link = resolveHref(baseURL, href)
				}
			})
		}

		date := ""
		if config.DateSelector != "" {
			date = strings.TrimSpace(sel.Find(config.DateSelector).First().Text())
		}

		author := ""
		if config.AuthorSelector != "" {
			author = strings.TrimSpace(sel.Find(config.AuthorSelector).First().Text())
		}

		results = append(results, ScrapeResult{
			Title:   title,
			Link:    link,
			Date:    date,
			Author:  author,
		})
	})

	// Optional: scrape detail pages for full content
	if config.ContentSelector != "" && len(results) > 0 {
		for i := range results {
			if results[i].Link == "" {
				continue
			}
			contentBody, err := s.fetchHTML(ctx, results[i].Link)
			if err != nil {
				continue
			}
			contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(contentBody)))
			if err != nil {
				continue
			}
			results[i].Content = strings.TrimSpace(contentDoc.Find(config.ContentSelector).First().Text())
		}
	}

	return results, nil
}

// fetchHTML downloads HTML content from a URL.
func (s *ScrapeIngester) fetchHTML(ctx context.Context, urlStr string) ([]byte, error) {
	client := ssrf.NewClient()
	client.Timeout = 30 * time.Second

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	req.Header.Set("User-Agent", "Aleph-Scraper/1.0 (news ingestion bot)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5MB max
}

// resolveHref resolves a potentially relative href against a base URL.
func resolveHref(base *url.URL, href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	resolved := base.ResolveReference(u)
	return resolved.String()
}

// ScrapeToJSON performs a scrape and returns the results as JSON bytes.
func (s *ScrapeIngester) ScrapeToJSON(ctx context.Context, configJSON string) ([]byte, error) {
	var config ScrapeConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("scrape: invalid config JSON: %w", err)
	}
	results, err := s.Scrape(ctx, &config)
	if err != nil {
		return nil, err
	}
	return json.Marshal(results)
}
```

Write to: `/tmp/opencode/aleph/internal/ingestion/sources/scraper.go`

- [ ] **Step 3: Register "scrape" source_type in probe.go Validate**

In `/tmp/opencode/aleph/internal/ingestion/probe.go`, line 60, add `"scrape"` to the valid types:

```go
case "rest", "rss", "github", "sitemap", "generic_json", "web", "scrape":
```

Edit via: `edit` tool on line 60-61.

- [ ] **Step 4: Add scrape to classifySourceType in probe.go**

In `/tmp/opencode/aleph/internal/ingestion/probe.go`, `classifySourceType` function, add after the `reHTML` block detection (around line 224-226). When HTML is detected and reSitemap/reRSS don't match, add a secondary check: if the config contains `article_selector`, classify as "scrape" instead of "web". However, since `classifySourceType` doesn't have access to config, we need to handle this in `engine.go` RunTask. Leave classifySourceType as-is — the source_type will come from the task config explicitly.

Skip this step — scraping is configured explicitly, not auto-detected.

- [ ] **Step 5: Add scraperIngester to Engine struct**

In `/tmp/opencode/aleph/internal/ingestion/engine.go`, lines 91-94, add:

```go
scraperIngester *sources.ScrapeIngester
```

- [ ] **Step 6: Add "scrape" case to RunTask dispatch in engine.go**

In `/tmp/opencode/aleph/internal/ingestion/engine.go`, after the "sheets" case (~line 192), add:

```go
case "scrape":
    taskErr = e.runScrapeSource(taskCtx, f, projectID, task)
```

- [ ] **Step 7: Implement runScrapeSource method in engine.go**

Add a new method `runScrapeSource` to Engine (near runSitemapSource, around line 1300). This reuses the existing save-to-JSON + create-table pattern:

```go
func (e *Engine) runScrapeSource(ctx context.Context, w io.Writer, projectID string, task *v1.IngestionTask) error {
	var config struct {
		URL             string `json:"url"`
		ArticleSelector string `json:"article_selector"`
		TitleSelector   string `json:"title_selector"`
		LinkSelector    string `json:"link_selector"`
		DateSelector    string `json:"date_selector"`
		AuthorSelector  string `json:"author_selector"`
		ContentSelector string `json:"content_selector"`
		MaxArticles     int    `json:"max_articles"`
		TableName       string `json:"tableName"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("runScrapeSource: invalid config_json: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("runScrapeSource: url is required in config_json")
	}

	if e.scraperIngester == nil {
		e.scraperIngester = sources.NewScrapeIngester()
	}

	fmt.Fprintf(w, "Scraping %s with selector %s...\n", config.URL, config.ArticleSelector)

	scrapeConfig := &sources.ScrapeConfig{
		URL:             config.URL,
		ArticleSelector: config.ArticleSelector,
		TitleSelector:   config.TitleSelector,
		LinkSelector:    config.LinkSelector,
		DateSelector:    config.DateSelector,
		AuthorSelector:  config.AuthorSelector,
		ContentSelector: config.ContentSelector,
		MaxArticles:     config.MaxArticles,
	}
	if scrapeConfig.MaxArticles <= 0 {
		scrapeConfig.MaxArticles = 50
	}

	results, err := e.scraperIngester.Scrape(ctx, scrapeConfig)
	if err != nil {
		return fmt.Errorf("runScrapeSource: scrape failed: %w", err)
	}

	fmt.Fprintf(w, "Extracted %d articles\n", len(results))

	// Save results as JSON to the project's raw directory
	projectDir := filepath.Join(e.projectsRoot, projectID, "raw")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("runScrapeSource: mkdir raw: %w", err)
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("runScrapeSource: marshal results: %w", err)
	}

	tableName, err := resolveTableName(task)
	if err != nil {
		return fmt.Errorf("runScrapeSource: resolve table name: %w", err)
	}

	rawPath := filepath.Join(projectDir, tableName+".json")
	if err := os.WriteFile(rawPath, jsonData, 0644); err != nil {
		return fmt.Errorf("runScrapeSource: write raw: %w", err)
	}

	// Create DuckDB table from JSON
	viewSQL := fmt.Sprintf(
		"CREATE OR REPLACE TABLE %s AS SELECT * FROM read_json_auto('%s')",
		safeident.QuoteIdentifier(tableName), rawPath,
	)
	fmt.Fprintf(w, "Creating table: %s\n", viewSQL)
	if err := e.db.Exec(ctx, viewSQL); err != nil {
		return fmt.Errorf("runScrapeSource: create table: %w", err)
	}

	fmt.Fprintf(w, "Scrape complete. Table %s created with %d rows.\n", tableName, len(results))
	return nil
}
```

Add these imports to engine.go if not already present:
- `"os"` (already there)
- `"path/filepath"` (already there)
- `"github.com/ff3300/aleph-v2/internal/safeident"` (already there)

- [ ] **Step 8: Verify compilation**

```bash
cd /tmp/opencode/aleph && go build ./...
```

Expected: exit 0, no errors.

- [ ] **Step 9: Commit**

```bash
cd /tmp/opencode/aleph && git add go.mod go.sum internal/ingestion/sources/scraper.go internal/ingestion/probe.go internal/ingestion/engine.go
git commit -m "feat: add HTML scraping source type (goquery-based) for DAGOSPIA and other non-RSS sources"
```

---

### Task 2: RSS Feed Batch Ingestion — Add 15+ Italian news feeds

**Files:**
- Create: `/tmp/opencode/aleph/scripts/ingest-italian-feeds.sh`

- [ ] **Step 1: Create the batch ingestion script**

```bash
#!/bin/bash
# ingest-italian-feeds.sh — Batch-import Italian news RSS feeds into Aleph
set -euo pipefail

ALEPH_HOST="${ALEPH_HOST:-localhost:9999}"
PROJECT_ID="${PROJECT_ID:-demo}"
TOKEN="${ALEPH_API_KEY:-}" # optional JWT for auth

# Generate JWT if token not provided
if [ -z "$TOKEN" ]; then
    echo "No ALEPH_API_KEY set — generating JWT for local dev..."
    TOKEN=$(python3 -c "
import json, time, hmac, hashlib, base64
h = {'alg':'HS256','typ':'JWT'}
p = {'iss':'aleph-v2','sub':'admin','aud':['aleph-v2-api'],'iat':int(time.time()),'exp':int(time.time())+3600,'jti':'feedbatch01','project_id':'$PROJECT_ID'}
def b64(d): return base64.urlsafe_b64encode(d).rstrip(b'=').decode()
hp = b64(json.dumps(h).encode()) + '.' + b64(json.dumps(p).encode())
sig = hmac.new(b'test-jwt-secret-for-local-dev', hp.encode(), hashlib.sha256).digest()
print(hp + '.' + b64(sig))
")
fi

# Connect RPC helper
call_aleph() {
    local service="$1"
    local body="$2"
    local envelope_length=$(echo -n "$body" | wc -c)
    local envelope=$(printf '\x00' && perl -e "print pack('N', $envelope_length)" && echo -n "$body")
    curl -s -X POST "http://${ALEPH_HOST}/${service}" \
        -H "Content-Type: application/connect+json" \
        -H "Origin: http://localhost:5173" \
        -H "Cookie: aleph_jwt=${TOKEN}" \
        --data-binary @- <<< "$envelope"
}

create_and_run() {
    local name="$1"
    local config_json="$2"
    local source_type="${3:-rss}"
    
    # Create task
    local create_body=$(cat <<EOF
{"projectId":"$PROJECT_ID","task":{"name":"$name","sourceType":"$source_type","configJson":"$config_json","schedule":"","status":"idle"}}
EOF
)
    echo "Creating task: $name ..."
    local response=$(call_aleph "aleph.v1.IngestionService/CreateTask" "$create_body")
    echo "  Create response: $response"
    
    # Extract task ID
    local task_id=$(echo "$response" | python3 -c "import sys,json,struct; d=sys.stdin.buffer.read(); d2=struct.unpack('>I',d[1:5])[0]; print(json.loads(d[5:5+d2]).get('task',{}).get('id',''))" 2>/dev/null || echo "")
    if [ -z "$task_id" ]; then
        echo "  WARNING: Could not extract task ID, skipping run"
        return
    fi
    
    # Run task
    local run_body="{\"projectId\":\"$PROJECT_ID\",\"taskId\":\"$task_id\"}"
    echo "  Running task: $task_id ..."
    call_aleph "aleph.v1.IngestionService/RunTask" "$run_body"
    echo ""
}

# === WIRE SERVICES (P0) ===
create_and_run "rss_ansa_politica"        '{"url":"https://www.ansa.it/sito/notizie/politica/politica_rss.xml"}'
create_and_run "rss_ansa_top"             '{"url":"https://www.ansa.it/sito/ansait_rss.xml"}'
create_and_run "rss_ansa_economia"        '{"url":"https://www.ansa.it/sito/notizie/economia/economia_rss.xml"}'
create_and_run "rss_ansa_mondo"           '{"url":"https://www.ansa.it/sito/notizie/mondo/mondo_rss.xml"}'

# === MAJOR NEWSPAPERS (P1) ===
create_and_run "rss_repubblica_politica"  '{"url":"https://www.repubblica.it/rss/politica/"}'
create_and_run "rss_repubblica_homepage"  '{"url":"https://www.repubblica.it/rss/homepage/"}'
create_and_run "rss_lastampa"             '{"url":"https://www.lastampa.it/rss"}'
create_and_run "rss_ilfatto"              '{"url":"https://www.ilfattoquotidiano.it/feed/"}'
create_and_run "rss_ilfatto_politica"     '{"url":"https://www.ilfattoquotidiano.it/categoria/politica/feed/"}'
create_and_run "rss_ilfatto_economia"     '{"url":"https://www.ilfattoquotidiano.it/categoria/economia/feed/"}'

# === NATIVE DIGITAL (P2) ===
create_and_run "rss_fanpage"              '{"url":"https://www.fanpage.it/feed/"}'
create_and_run "rss_fanpage_politica"     '{"url":"https://www.fanpage.it/politica/feed/"}'
create_and_run "rss_open_online"          '{"url":"https://www.open.online/feed/"}'
create_and_run "rss_linkiesta"            '{"url":"https://www.linkiesta.it/feed/"}'
create_and_run "rss_startmag"             '{"url":"https://www.startmag.it/feed/"}'
create_and_run "rss_formiche"             '{"url":"https://formiche.net/feed/"}'
create_and_run "rss_insideover"           '{"url":"https://www.insideover.com/feed/"}'

# === POLLING & ANALYSIS ===
create_and_run "rss_youtrend"             '{"url":"https://www.youtrend.it/feed/"}'
create_and_run "rss_termometro_politico"  '{"url":"https://www.termometropolitico.it/feed/"}'
create_and_run "rss_ispi"                 '{"url":"https://www.ispionline.it/feed/"}'

# === INSTITUTIONAL ===
create_and_run "rss_governo"              '{"url":"https://www.governo.it/it/feed"}'

echo "=== INGESTION COMPLETE ==="
echo "Run 'SELECT name FROM information_schema.tables WHERE table_schema='main'' in DuckDB to verify."
```

Write to: `/tmp/opencode/aleph/scripts/ingest-italian-feeds.sh`

- [ ] **Step 2: Make script executable and run**

```bash
chmod +x /tmp/opencode/aleph/scripts/ingest-italian-feeds.sh
cd /tmp/opencode/aleph && bash scripts/ingest-italian-feeds.sh
```

Expected: Each feed creates a task and starts ingestion. Wait for all to complete (~2-5 min).

- [ ] **Step 3: Verify DuckDB tables**

```bash
cd /tmp/opencode/aleph && duckdb data/aleph.duckdb -c "SELECT table_name FROM information_schema.tables WHERE table_schema='main' AND table_name LIKE 'rss_%' ORDER BY table_name;"
```

Expected: 20+ tables listed (rss_ansa_politica, rss_repubblica_politica, etc.)

- [ ] **Step 4: Commit**

```bash
cd /tmp/opencode/aleph && git add scripts/ingest-italian-feeds.sh
git commit -m "feat: batch ingestion script for 20+ Italian news RSS feeds"
```

---

### Task 3: DAGOSPIA Scraping Configuration

**Files:**
- Modify: `/tmp/opencode/aleph/scripts/ingest-italian-feeds.sh` (add DAGOSPIA scrape task)

- [ ] **Step 1: Add DAGOSPIA scrape to the ingestion script**

Append to `/tmp/opencode/aleph/scripts/ingest-italian-feeds.sh`:

```bash
# === SCRAPE SOURCES (no RSS available) ===
# DAGOSPIA — Italian political insider blog, no RSS feed
# Config: main page has article blocks with title + link, detail pages have full body
create_and_run "scrape_dagospia" \
    '{"url":"https://www.dagospia.com/","article_selector":"article, .article-item, .news-item, div[class*=\"article\"]","title_selector":"h2, h3, .title, a","link_selector":"a","date_selector":"time, .date, .data","max_articles":30}' \
    "scrape"
```

Note: The exact CSS selectors for DAGOSPIA will need verification. DAGOSPIA's HTML structure changes periodically. The config can be tuned by inspecting https://www.dagospia.com/ in browser DevTools. If the default selectors don't yield results, adjust `article_selector`, `title_selector`, and `date_selector` based on the actual DOM.

- [ ] **Step 2: Run the DAGOSPIA scrape task**

```bash
cd /tmp/opencode/aleph && bash scripts/ingest-italian-feeds.sh
# or run just the DAGOSPIA line directly via curl
```

- [ ] **Step 3: Commit**

```bash
cd /tmp/opencode/aleph && git add scripts/ingest-italian-feeds.sh
git commit -m "feat: add DAGOSPIA scraping config to ingestion script"
```

---

### Task 4: Ontology Updates — Define objects for new tables

**Files:**
- Modify: `/tmp/opencode/aleph/data/projects/demo/ontologies/core.aleph`

- [ ] **Step 1: Read existing ontology**

Read `/tmp/opencode/aleph/data/projects/demo/ontologies/core.aleph` to understand the current DSL structure.

- [ ] **Step 2: Add newspaper article objects**

Append to the ontology file after existing object definitions:

```
// =============================================================================
// Italian News Sources (added 2026-05-22)
// =============================================================================

object ArticleANSA {
    from dataset rss_ansa_politica
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
    property source type text from source
}

object ArticleRepubblica {
    from dataset rss_repubblica_politica
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
    property category type text
}

object ArticleLaStampa {
    from dataset rss_lastampa
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleIlFatto {
    from dataset rss_ilfatto_politica
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
    property creator type text
    property category type text
}

object ArticleFanpage {
    from dataset rss_fanpage_politica
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleOpenOnline {
    from dataset rss_open_online
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleYouTrend {
    from dataset rss_youtrend
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleTermometroPolitico {
    from dataset rss_termometro_politico
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleISPI {
    from dataset rss_ispi
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

object ArticleGoverno {
    from dataset rss_governo
    id id
    property title type text
    property link type text
    property description type text
    property pubDate type datetime from pubdate
}

// === Scraped Sources ===

object ArticleDagospia {
    from dataset scrape_dagospia
    id id
    property title type text
    property link type text
    property date type text
    property author type text
    property content type text
}
```

Write to: end of `/tmp/opencode/aleph/data/projects/demo/ontologies/core.aleph`

- [ ] **Step 3: Verify ontology compiles**

Check that DuckDB can still resolve the ontology:

```bash
# This would be done via the EmergeOntology RPC or by checking query.go's resolveOntology logic.
# Alternative: run the Aleph binary and check the /healthz endpoint for ontology errors.
```

Expected: No syntax errors in DSL file.

- [ ] **Step 4: Commit**

```bash
cd /tmp/opencode/aleph && git add data/projects/demo/ontologies/core.aleph
git commit -m "feat: add ontology objects for 12+ Italian news sources (ANSA, Repubblica, La Stampa, Il Fatto, Fanpage, YouTrend, DAGOSPIA, etc.)"
```

---

### Task 5: Frontend — DataSourcesView: add "Scrape" source type support

**Files:**
- Modify: `/tmp/opencode/aleph/frontend/src/components/DataSourcesView.tsx`
- Modify: `/tmp/opencode/aleph/frontend/src/components/DataSourceForm.tsx`

- [ ] **Step 1: Read existing DataSourcesView and DataSourceForm**

```bash
cat /tmp/opencode/aleph/frontend/src/components/DataSourcesView.tsx
cat /tmp/opencode/aleph/frontend/src/components/DataSourceForm.tsx
```

- [ ] **Step 2: Add "scrape" to source type options in DataSourceForm**

Find the source_type dropdown/selector in `DataSourceForm.tsx`. Add `"scrape"` to the list of available types alongside "rss", "rest", "csv", "sitemap", etc. Add a new option:

```tsx
<SelectItem value="scrape">🌐 Web Scraper (HTML/CSS selectors)</SelectItem>
```

- [ ] **Step 3: Add conditional scrape config fields in DataSourceForm**

When `sourceType === "scrape"`, show additional fields after the URL field:

```tsx
{sourceType === "scrape" && (
  <>
    <FormField label="CSS Article Selector" name="article_selector" required>
      <Input placeholder="article, .article-item, div[class*=post]" />
      <FormDescription>
        CSS selector that matches each article container on the page (e.g., ".news-item", "article.post")
      </FormDescription>
    </FormField>
    <FormField label="CSS Title Selector" name="title_selector" required>
      <Input placeholder="h2, .title, .headline" />
      <FormDescription>
        CSS selector for article title within each article container
      </FormDescription>
    </FormField>
    <FormField label="CSS Link Selector" name="link_selector">
      <Input placeholder="a, .read-more" />
      <FormDescription>
        CSS selector for the article link (defaults to first &lt;a&gt; tag if empty)
      </FormDescription>
    </FormField>
    <FormField label="CSS Date Selector" name="date_selector">
      <Input placeholder="time, .date, .published" />
    </FormField>
    <FormField label="CSS Author Selector" name="author_selector">
      <Input placeholder=".author, .byline" />
    </FormField>
    <FormField label="Max Articles" name="max_articles" type="number">
      <Input type="number" min={1} max={200} defaultValue={50} />
      <FormDescription>
        Maximum number of articles to extract (default: 50)
      </FormDescription>
    </FormField>
  </>
)}
```

- [ ] **Step 4: Build config_json from scrape fields on submit**

In the form submit handler, when `sourceType === "scrape"`, construct `config_json` as a JSON string containing all the CSS selector fields:

```typescript
if (sourceType === "scrape") {
  configJson = JSON.stringify({
    url: values.url,
    article_selector: values.article_selector,
    title_selector: values.title_selector,
    link_selector: values.link_selector || "",
    date_selector: values.date_selector || "",
    author_selector: values.author_selector || "",
    content_selector: values.content_selector || "",
    max_articles: values.max_articles || 50,
    tableName: values.name, // optional: explicit table name
  });
}
```

- [ ] **Step 5: Add scrape source type badge/icon in DataSourcesView**

In `DataSourcesView.tsx`, where source types are displayed with badges/icons, add a case for `"scrape"`:

```tsx
{task.sourceType === "scrape" && (
  <Badge variant="outline" className="bg-purple-50 text-purple-700 border-purple-200">
    <Globe className="w-3 h-3 mr-1" /> Scraper
  </Badge>
)}
```

Import `Globe` from `lucide-react` if not already imported.

- [ ] **Step 6: Verify TypeScript compilation**

```bash
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit
```

Expected: no type errors.

- [ ] **Step 7: Commit**

```bash
cd /tmp/opencode/aleph && git add frontend/src/components/DataSourceForm.tsx frontend/src/components/DataSourcesView.tsx
git commit -m "feat: add web scraper source type support to DataSources UI (CSS selectors, max articles)"
```

---

### Task 6: Frontend — Source Health Dashboard Widget

**Files:**
- Modify: `/tmp/opencode/aleph/frontend/src/components/DataHealthView.tsx`

- [ ] **Step 1: Read existing DataHealthView**

```bash
cat /tmp/opencode/aleph/frontend/src/components/DataHealthView.tsx
```

- [ ] **Step 2: Add a "Last Updated" column and source status**

If the DataHealthView shows a table of sources, add a column showing when each source was last successfully scraped/ingested. For RSS sources, show feed freshness. For scrape sources, show article count.

This depends on the exact structure of DataHealthView. Minimal implementation:

```tsx
// In the data sources table, add a status indicator
const getSourceHealth = (task: IngestionTask) => {
  if (task.status === "completed") {
    return { label: "Healthy", color: "text-green-600" };
  }
  if (task.status === "failed") {
    return { label: "Failed", color: "text-red-600" };
  }
  if (task.status === "running") {
    return { label: "Running", color: "text-blue-600" };
  }
  return { label: "Idle", color: "text-gray-500" };
};
```

- [ ] **Step 3: Verify compilation**

```bash
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
cd /tmp/opencode/aleph && git add frontend/src/components/DataHealthView.tsx
git commit -m "feat: add source health indicators to DataHealthView"
```

---

### Task 7: Final Verification — Full integration test

- [ ] **Step 1: Recompile Aleph with all changes**

```bash
cd /tmp/opencode/aleph && CGO_ENABLED=1 go build -o aleph-server .
```

Expected: build succeeds.

- [ ] **Step 2: Restart Aleph**

```bash
kill $(lsof -ti:9999) 2>/dev/null
# Start Aleph with env vars (JWT_SECRET, KEY_ENCRYPTION_KEY, DUCKDB_PATH, POSTGRES_DSN, NLP_ADDR, PORT=9999, APP_ENV=development, GOSECRETS_ENV=ci, ALLOW_LOCALHOST_SSRF=true, LLM_TIMEOUT_SECONDS=120)
```

- [ ] **Step 3: Run the batch ingestion script**

```bash
cd /tmp/opencode/aleph && bash scripts/ingest-italian-feeds.sh
```

Expected: all tasks created and running. Check DuckDB after completion.

- [ ] **Step 4: Verify DuckDB tables have data**

```bash
duckdb /tmp/opencode/aleph/data/aleph.duckdb -c "
SELECT 'rss_ansa_politica' as tbl, count(*) as rows FROM rss_ansa_politica
UNION ALL SELECT 'rss_repubblica_politica', count(*) FROM rss_repubblica_politica
UNION ALL SELECT 'rss_ilfatto_politica', count(*) FROM rss_ilfatto_politica
UNION ALL SELECT 'scrape_dagospia', count(*) FROM scrape_dagospia
"
```

Expected: each table has >0 rows.

- [ ] **Step 5: Test chat with Aleph using new sources**

Send a chat request via Connect RPC to test that Aleph can reference the new data:

```python
# Python test — ask Aleph about Italian politics using the new sources
body = json.dumps({
    "agentId": "ollama-cloud-agent",
    "messages": [{"role": "user", "content": "Cosa dicono oggi ANSA e Il Fatto Quotidiano sulla politica italiana? Cerca nei dati di oggi."}],
    "stream": True,
    "projectId": "demo"
})
```

Expected: DeepSeek references articles from `rss_ansa_politica` and `rss_ilfatto_politica` tables via the `search_data` tool.

- [ ] **Step 6: Test DAGOSPIA scraping works**

```bash
# Manually run the DAGOSPIA scrape task via curl
# Verify scrape_dagospia table has articles
duckdb /tmp/opencode/aleph/data/aleph.duckdb -c "SELECT title, link, date FROM scrape_dagospia LIMIT 5;"
```

Expected: 5+ articles with titles and links. If empty, adjust CSS selectors.

- [ ] **Step 7: Commit**

```bash
cd /tmp/opencode/aleph && git add -A && git commit -m "test: verify Italian news ingestion pipeline — RSS feeds + DAGOSPIA scraper"
```
