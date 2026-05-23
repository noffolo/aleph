package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"gopkg.in/yaml.v3"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
)

var rssDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700",
	"2006-01-02T15:04:05",
}

func parseItemDate(pubDate string) (time.Time, error) {
	for _, f := range rssDateFormats {
		if t, err := time.Parse(f, strings.TrimSpace(pubDate)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", pubDate)
}

func isItemInRange(pubDate string, startDate, endDate time.Time) bool {
	t, err := parseItemDate(pubDate)
	if err != nil || t.IsZero() {
		// Can't determine the date — include the item (meglio dati in più che in meno)
		return true
	}
	if !startDate.IsZero() && t.Before(startDate) {
		return false
	}
	if !endDate.IsZero() && t.After(endDate) {
		return false
	}
	return true
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	Title     string `xml:"title"`
	Link      AtomLink `xml:"link"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Summary   string `xml:"summary"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

type FeedConfig struct {
	Name     string                 `yaml:"name"`
	URL      string                 `yaml:"url"`
	Category string                 `yaml:"category"`
	Source   string                 `yaml:"source"` // "rss" (default) or "scrape"
	Config   map[string]interface{} `yaml:"config"` // scrape selectors etc.
}

// feedSourceOrDefault returns the feed's source type, defaulting to "rss".
func feedSourceOrDefault(f FeedConfig) string {
	if f.Source == "" {
		return "rss"
	}
	return f.Source
}

type Config struct {
	Feeds []FeedConfig `yaml:"feeds"`
}

type DedupState struct {
	Processed map[string]time.Time `json:"processed"`
}

func loadDedupState(path string) (*DedupState, error) {
	state := &DedupState{Processed: make(map[string]time.Time)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, state); err != nil {
		return &DedupState{Processed: make(map[string]time.Time)}, nil
	}
	return state, nil
}

func saveDedupState(path string, state *DedupState) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func dedupKey(title, link string) string {
	h := sha256.New()
	h.Write([]byte(link))
	return fmt.Sprintf("%x", h.Sum([]byte(title)))[:16]
}

type originTransport struct {
	http.RoundTripper
	jwt    string
	apiKey string
}

func (t *originTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Origin", "http://localhost:5173")
	if t.jwt != "" {
		req.AddCookie(&http.Cookie{Name: "aleph_jwt", Value: t.jwt})
	}
	if t.apiKey != "" {
		req.Header.Set("X-Aleph-Api-Key", t.apiKey)
	}
	return t.RoundTripper.RoundTrip(req)
}

func newHTTPClient(timeout time.Duration, jwt, apiKey string) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &originTransport{
			jwt:    jwt,
			apiKey: apiKey,
			RoundTripper: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
			},
		},
	}
}

func fetchRSS(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("request creation failed: %w", err)
		}
		req.Header.Set("User-Agent", "Aleph-Ingest/1.0")
		req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			if attempt == 0 {
				log.Printf("retrying %s after network error: %v", url, err)
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, fmt.Errorf("HTTP fetch failed after retry: %w", err)
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound || resp.StatusCode >= 500 {
			resp.Body.Close()
			cancel()
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("body read failed: %w", err)
		}
		return body, nil
	}

	return nil, lastErr
}

func sanitizeXML(data []byte) []byte {
	var buf []byte
	for _, b := range data {
		if b >= 0x20 || b == 0x09 || b == 0x0A || b == 0x0D {
			buf = append(buf, b)
		}
	}
	return buf
}

type xmlDecoderWithCharset struct {
	data []byte
}

func (d *xmlDecoderWithCharset) Read(p []byte) (n int, err error) {
	if len(d.data) == 0 {
		return 0, io.EOF
	}
	n = copy(p, d.data)
	d.data = d.data[n:]
	return n, nil
}

func parseRSS(data []byte) ([]Item, error) {
	data = sanitizeXML(data)

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err == nil {
		return rss.Channel.Items, nil
	}

	if bytes.Contains(data, []byte("ISO-8859-1")) || bytes.Contains(data, []byte("iso-8859-1")) {
		decoder := &xmlDecoderWithCharset{data: data}
		dec := xml.NewDecoder(decoder)
		dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
			if charset == "ISO-8859-1" || charset == "iso-8859-1" {
				raw, err := io.ReadAll(input)
				if err != nil {
					return nil, err
				}
				utf8 := make([]byte, 0, len(raw)*2)
				for _, b := range raw {
					if b < 128 {
						utf8 = append(utf8, b)
					} else {
						utf8 = append(utf8, 0xC0|byte(b>>6), 0x80|byte(b&0x3F))
					}
				}
				return bytes.NewReader(utf8), nil
			}
			return nil, fmt.Errorf("unsupported charset: %s", charset)
		}
		if err := dec.Decode(&rss); err == nil {
			return rss.Channel.Items, nil
		}
	}

	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		items := make([]Item, len(atom.Entries))
		for i, e := range atom.Entries {
			pubDate := e.Published
			if pubDate == "" {
				pubDate = e.Updated
			}
			items[i] = Item{
				Title:       e.Title,
				Link:        e.Link.Href,
				Description: e.Summary,
				PubDate:     pubDate,
			}
		}
		return items, nil
	}

	return nil, fmt.Errorf("XML parse failed: not RSS 2.0, ISO-8859-1 RSS, or Atom")
}

type FeedResult struct {
	Name         string
	URL          string
	Category     string
	TotalItems   int
	NewItems     int
	SkippedItems int
	FailedItems  int
	Error        string
}

func processRSSFeed(ctx context.Context, feed FeedConfig, result FeedResult, projectID string, dryRun bool, statePath string, ingestionClient v1connect.IngestionServiceClient, httpClient *http.Client, dedupState *DedupState, startDate, endDate time.Time) FeedResult {
	body, err := fetchRSS(ctx, httpClient, feed.URL)
	if err != nil {
		result.Error = err.Error()
		log.Printf("  SKIP: %v", err)
		return result
	}

	items, err := parseRSS(body)
	if err != nil {
		result.Error = err.Error()
		log.Printf("  SKIP: %v", err)
		return result
	}

	result.TotalItems = len(items)
	log.Printf("  Parsed %d items", len(items))

	now := time.Now()

	for _, item := range items {
		if item.Link == "" {
			result.SkippedItems++
			continue
		}

		key := dedupKey(item.Title, item.Link)

		if _, exists := dedupState.Processed[key]; exists {
			result.SkippedItems++
			continue
		}

		// Date range filtering
		if !isItemInRange(item.PubDate, startDate, endDate) {
			result.SkippedItems++
			continue
		}

		result.NewItems++

		if dryRun {
			dedupState.Processed[key] = now
			continue
		}

		time.Sleep(1200 * time.Millisecond)

		configJSON, err := json.Marshal(map[string]string{"url": item.Link})
		if err != nil {
			result.FailedItems++
			continue
		}

		task := &v1.IngestionTask{
			Name:       fmt.Sprintf("news-%s-%s", feed.Name, item.Title),
			SourceType: "url",
			ConfigJson: string(configJSON),
		}

		createResp, err := ingestionClient.CreateTask(ctx, connect.NewRequest(&v1.CreateTaskRequest{
			ProjectId: projectID,
			Task:      task,
		}))
		if err != nil {
			result.FailedItems++
			log.Printf("    create task failed for %s: %v", item.Link, err)
			continue
		}

		taskID := createResp.Msg.Task.Id

		_, err = ingestionClient.RunTask(ctx, connect.NewRequest(&v1.RunTaskRequest{
			ProjectId: projectID,
			TaskId:    taskID,
		}))
		if err != nil {
			result.FailedItems++
			log.Printf("    run task failed for %s: %v", item.Link, err)
			continue
		}

		dedupState.Processed[key] = now
	}

	if err := saveDedupState(statePath, dedupState); err != nil {
		log.Printf("  WARN: failed to save dedup state: %v", err)
	}

	return result
}

func processScrapeFeed(ctx context.Context, feed FeedConfig, result FeedResult, projectID string, dryRun bool, ingestionClient v1connect.IngestionServiceClient) FeedResult {
	if feed.Config == nil {
		result.Error = "scrape feed missing config"
		log.Printf("  SKIP: %v", result.Error)
		return result
	}

	scrapeCfg := map[string]interface{}{
		"url":          feed.URL,
		"max_articles": 30,
	}

	for _, k := range []string{"article_selector", "title_selector", "link_selector", "date_selector", "author_selector", "content_selector"} {
		if v, ok := feed.Config[k]; ok {
			scrapeCfg[k] = v
		}
	}

	if v, ok := feed.Config["max_articles"]; ok {
		scrapeCfg["max_articles"] = v
	}

	configJSON, err := json.Marshal(scrapeCfg)
	if err != nil {
		result.Error = fmt.Sprintf("marshal scrape config: %v", err)
		log.Printf("  SKIP: %v", result.Error)
		return result
	}

	result.TotalItems = 1
	log.Printf("  Scrape source: %s", feed.URL)

	if dryRun {
		result.NewItems = 1
		return result
	}

	taskName := fmt.Sprintf("news-scrape-%s", feed.Name)
	taskName = strings.Map(func(r rune) rune {
		if r == ' ' {
			return '-'
		}
		return r
	}, taskName)

	task := &v1.IngestionTask{
		Name:       taskName,
		SourceType: "scrape",
		ConfigJson: string(configJSON),
	}

	createResp, err := ingestionClient.CreateTask(ctx, connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: projectID,
		Task:      task,
	}))
	if err != nil {
		result.Error = fmt.Sprintf("create task: %v", err)
		result.FailedItems++
		log.Printf("    create task failed: %v", err)
		return result
	}

	taskID := createResp.Msg.Task.Id

	_, err = ingestionClient.RunTask(ctx, connect.NewRequest(&v1.RunTaskRequest{
		ProjectId: projectID,
		TaskId:    taskID,
	}))
	if err != nil {
		result.Error = fmt.Sprintf("run task: %v", err)
		result.FailedItems++
		log.Printf("    run task failed: %v", err)
		return result
	}

	result.NewItems = 1
	return result
}

func main() {
	configPath := flag.String("config", "configs/news-feeds.yaml", "path to YAML config file")
	projectID := flag.String("project", "", "Aleph project ID (required)")
	serverAddr := flag.String("server", "http://localhost:8080", "Aleph server URL")
	statePath := flag.String("state", "configs/news-dedup-state.json", "path to dedup state file")
	dryRun := flag.Bool("dry-run", false, "fetch and parse feeds but do not ingest")
	sourceType := flag.String("source", "", "filter feeds by source type (rss or scrape)")
	feedsFilter := flag.String("feeds", "", "comma-separated feed names to process (empty = all)")
	jwtToken := flag.String("jwt", "", "Aleph JWT for authentication (set aleph_jwt cookie)")
	apiKey := flag.String("api-key", "", "Aleph API key for authentication (X-Aleph-Api-Key header)")
	startDateStr := flag.String("start-date", "", "filter items after this date (RFC3339 or YYYY-MM-DD)")
	endDateStr := flag.String("end-date", "", "filter items before this date (RFC3339 or YYYY-MM-DD)")
	flag.Parse()

	if *projectID == "" {
		log.Fatal("project ID is required (-project)")
	}

	var startDate, endDate time.Time
	if *startDateStr != "" {
		var err error
		startDate, err = time.Parse(time.RFC3339, *startDateStr)
		if err != nil {
			startDate, err = time.Parse("2006-01-02", *startDateStr)
			if err != nil {
				log.Fatalf("invalid -start-date %q: expected RFC3339 or YYYY-MM-DD", *startDateStr)
			}
		}
	}
	if *endDateStr != "" {
		var err error
		endDate, err = time.Parse(time.RFC3339, *endDateStr)
		if err != nil {
			endDate, err = time.Parse("2006-01-02", *endDateStr)
			if err != nil {
				log.Fatalf("invalid -end-date %q: expected RFC3339 or YYYY-MM-DD", *endDateStr)
			}
		}
		// Set to end of day for date-only inputs
		if endDate.Hour() == 0 && endDate.Minute() == 0 && endDate.Second() == 0 {
			endDate = endDate.Add(24*time.Hour - time.Second)
		}
	}

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", *configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("failed to parse config YAML: %v", err)
	}

	// Filter by source type if specified
	if *sourceType != "" {
		var filtered []FeedConfig
		for _, f := range config.Feeds {
			if feedSourceOrDefault(f) == *sourceType {
				filtered = append(filtered, f)
			}
		}
		config.Feeds = filtered
		if len(config.Feeds) == 0 {
			log.Fatalf("no feeds match source type %q", *sourceType)
		}
	}

	// Filter by feed names if specified
	if *feedsFilter != "" {
		wanted := make(map[string]bool)
		for _, name := range strings.Split(*feedsFilter, ",") {
			wanted[strings.TrimSpace(name)] = true
		}
		var filtered []FeedConfig
		for _, f := range config.Feeds {
			if wanted[f.Name] {
				filtered = append(filtered, f)
			}
		}
		config.Feeds = filtered
		if len(config.Feeds) == 0 {
			log.Fatalf("no feeds match names %q", *feedsFilter)
		}
	}

	dedupState, err := loadDedupState(*statePath)
	if err != nil {
		log.Fatalf("failed to load dedup state: %v", err)
	}

	httpClient := newHTTPClient(300*time.Second, *jwtToken, *apiKey)

	var ingestionClient v1connect.IngestionServiceClient
	if !*dryRun {
		customHTTPClient := newHTTPClient(120*time.Second, *jwtToken, *apiKey)
		ingestionClient = v1connect.NewIngestionServiceClient(
			customHTTPClient,
			*serverAddr,
			connect.WithGRPCWeb(),
		)
	}

	var results []FeedResult
	var mu sync.Mutex

	log.Printf("Processing %d feeds for project %s", len(config.Feeds), *projectID)
	log.Printf("Server: %s  Dry-run: %v", *serverAddr, *dryRun)
	log.Println(strings.Repeat("-", 60))

	for i, feed := range config.Feeds {
		log.Printf("[%d/%d] %s", i+1, len(config.Feeds), feed.Name)

		result := FeedResult{
			Name:     feed.Name,
			URL:      feed.URL,
			Category: feed.Category,
		}

		ctx := context.Background()

		source := feedSourceOrDefault(feed)

		if source == "scrape" {
			result = processScrapeFeed(ctx, feed, result, *projectID, *dryRun, ingestionClient)
		} else {
			result = processRSSFeed(ctx, feed, result, *projectID, *dryRun, *statePath, ingestionClient, httpClient, dedupState, startDate, endDate)
		}

		mu.Lock()
		results = append(results, result)
		mu.Unlock()

		log.Printf("  New: %d  Skipped: %d  Failed: %d",
			result.NewItems, result.SkippedItems, result.FailedItems)
	}

	log.Println(strings.Repeat("-", 60))
	log.Println("INGESTION SUMMARY")
	log.Println(strings.Repeat("-", 60))

	totalNew := 0
	totalSkipped := 0
	totalFailed := 0
	totalError := 0

	for _, r := range results {
		status := "OK"
		if r.Error != "" {
			status = fmt.Sprintf("ERROR: %s", r.Error)
			totalError++
		}
		log.Printf("  %-30s [%s]  parsed:%3d  new:%3d  skip:%3d  fail:%2d  %s",
			r.Name, r.Category, r.TotalItems, r.NewItems, r.SkippedItems, r.FailedItems, status)
		totalNew += r.NewItems
		totalSkipped += r.SkippedItems
		totalFailed += r.FailedItems
	}

	log.Println(strings.Repeat("-", 60))
	log.Printf("TOTAL: %d feeds  %d new  %d skipped  %d failed  %d errors",
		len(results), totalNew, totalSkipped, totalFailed, totalError)
	log.Printf("Dedup state saved to: %s", *statePath)
}
