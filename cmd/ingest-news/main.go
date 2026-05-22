package main

import (
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
	"sync"
	"time"

	"strings"

	"connectrpc.com/connect"
	"gopkg.in/yaml.v3"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
)

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

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
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

func parseRSS(data []byte) ([]Item, error) {
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("XML parse failed: %w", err)
	}
	return rss.Channel.Items, nil
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

func processRSSFeed(ctx context.Context, feed FeedConfig, result FeedResult, projectID string, dryRun bool, statePath string, ingestionClient v1connect.IngestionServiceClient, httpClient *http.Client, dedupState *DedupState) FeedResult {
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

		result.NewItems++

		if dryRun {
			dedupState.Processed[key] = now
			continue
		}

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
	flag.Parse()

	if *projectID == "" {
		log.Fatal("project ID is required (-project)")
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

	httpClient := newHTTPClient(30 * time.Second)

	var ingestionClient v1connect.IngestionServiceClient
	if !*dryRun {
		ingestionClient = v1connect.NewIngestionServiceClient(
			httpClient,
			*serverAddr,
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
			result = processRSSFeed(ctx, feed, result, *projectID, *dryRun, *statePath, ingestionClient, httpClient, dedupState)
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
