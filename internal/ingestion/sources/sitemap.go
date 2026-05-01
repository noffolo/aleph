// Package sources implements W3 ingestion methods: rate-limited HTTP fetcher,
// chunker/worker pool, GitHub repos, sitemap XML, ProbeRunner, generic JSON/API, Google Sheets.
package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// =============================================================================
// W3-06: Sitemap XML Structs
// =============================================================================

// SitemapIndex represents a sitemap index file (<sitemapindex> root).
type SitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

// SitemapEntry is a single <sitemap> entry within a sitemap index.
type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// URLSet represents a sitemap URL set (<urlset> root).
type URLSet struct {
	XMLName xml.Name   `xml:"urlset"`
	URLs    []URLEntry `xml:"url"`
}

// URLEntry is a single <url> entry within a URL set.
type URLEntry struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

// =============================================================================
// W3-06: Crawl Result Types
// =============================================================================

// CrawlResult holds the outcome of crawling a single sitemap.
type CrawlResult struct {
	SitemapURL string
	URLs       []PageResult
}

// PageResult holds the outcome of fetching a single page.
type PageResult struct {
	URL     string
	Content []byte
	Size    int64
	Status  int
	Err     error
}

// =============================================================================
// W3-06: SitemapIngester
// =============================================================================

// SitemapIngester parses sitemap XML and fetches all listed URLs.
type SitemapIngester struct {
	client *RateLimitedClient
}

// NewSitemapIngester creates a SitemapIngester with default rate limiting.
func NewSitemapIngester() *SitemapIngester {
	return &SitemapIngester{
		client: NewRateLimitedClient(DefaultRate),
	}
}

// CrawlSitemap fetches a sitemap, parses it, and crawls all discovered URLs.
// It handles both sitemap index files (<sitemapindex>) and URL sets (<urlset>).
// Sitemap indices are followed one level deep.
func (s *SitemapIngester) CrawlSitemap(ctx context.Context, sitemapURL string) (*CrawlResult, error) {
	result := &CrawlResult{SitemapURL: sitemapURL}

	// Step 1: fetch the sitemap XML
	body, err := s.fetchXML(ctx, sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap %s: %w", sitemapURL, err)
	}

	// Step 2: detect root element to decide parser
	rootName, err := detectRootElement(body)
	if err != nil {
		return nil, fmt.Errorf("parse sitemap %s: %w", sitemapURL, err)
	}

	var urls []string

	switch strings.ToLower(rootName) {
	case "sitemapindex":
		// Step 3a: parse as sitemap index, recurse one level
		var idx SitemapIndex
		if err := xml.Unmarshal(body, &idx); err != nil {
			return nil, fmt.Errorf("unmarshal sitemap index %s: %w", sitemapURL, err)
		}
		for _, entry := range idx.Sitemaps {
			childURL := strings.TrimSpace(entry.Loc)
			if childURL == "" {
				continue
			}
			// Resolve relative URLs against the parent sitemap
			absURL, err := resolveURL(sitemapURL, childURL)
			if err != nil {
				continue
			}
			childResult, err := s.CrawlSitemap(ctx, absURL)
			if err != nil {
				// Log and continue; don't let one child failure kill the whole crawl
				continue
			}
			result.URLs = append(result.URLs, childResult.URLs...)
		}

	case "urlset":
		// Step 3b: parse as URL set
		var set URLSet
		if err := xml.Unmarshal(body, &set); err != nil {
			return nil, fmt.Errorf("unmarshal urlset %s: %w", sitemapURL, err)
		}
		for _, entry := range set.URLs {
			u := strings.TrimSpace(entry.Loc)
			if u != "" {
				urls = append(urls, u)
			}
		}

	default:
		return nil, fmt.Errorf("unrecognized sitemap root element: %s (expected sitemapindex or urlset)", rootName)
	}

	// Step 4: fetch all discovered URLs in parallel using WorkerPool
	if len(urls) > 0 {
		pageResults, err := s.fetchAllPages(ctx, urls)
		if err != nil {
			// Partial failures are recorded per-PageResult; only context errors propagate here
			return nil, err
		}
		result.URLs = append(result.URLs, pageResults...)
	}

	return result, nil
}

// =============================================================================
// Internal helpers
// =============================================================================

// fetchXML downloads raw XML content from a URL.
func (s *SitemapIngester) fetchXML(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/xml, text/xml, */*")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Check content-type for XML
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, _ := mime.ParseMediaType(contentType)
		if mediaType != "" && !isXMLContentType(mediaType) && !isTextContentType(mediaType) {
			return nil, fmt.Errorf("unexpected content-type %s (expected XML)", mediaType)
		}
	}

	return body, nil
}

// detectRootElement peeks at the root XML element name without full parsing.
func detectRootElement(data []byte) (string, error) {
	// Use a decoder to get the first start element
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			return tok.Name.Local, nil
		case xml.EndElement:
			// Empty document
			return "", fmt.Errorf("empty XML document")
		}
	}
}

// isXMLContentType returns true for XML-related media types.
func isXMLContentType(mediaType string) bool {
	return mediaType == "application/xml" ||
		mediaType == "text/xml" ||
		strings.HasSuffix(mediaType, "+xml")
}

// isTextContentType returns true for text-based media types.
func isTextContentType(mediaType string) bool {
	return mediaType == "text/html" ||
		mediaType == "text/plain" ||
		mediaType == "application/json"
}

// resolveURL resolves a (possibly relative) URL against a base URL.
func resolveURL(base, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.IsAbs() {
		return rawURL, nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	resolved := baseURL.ResolveReference(u)
	return resolved.String(), nil
}

// followRedirects returns an HTTP client that limits redirects to maxRedirects.
func followRedirects(maxRedirects int) *http.Client {
	client := ssrf.NewClient()
	client.Timeout = 30 * time.Second
	originalCheck := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("too many redirects")
		}
		if originalCheck != nil {
			return originalCheck(req, via)
		}
		return nil
	}
	return client
}

// maxPageSize is the maximum number of bytes to read per page fetch (1 MB).
const maxPageSize int64 = 1 << 20 // 1 MB

// fetchSinglePage downloads a single page, respecting content-type and size limits.
// Errors are returned as *PageResult values (not propagated) so the caller can
// record them per-URL without failing the entire crawl.
func fetchSinglePage(ctx context.Context, pageURL string, resultCh chan<- PageResult) {
	// Create a dedicated client with redirect limiting
	client := followRedirects(5)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		resultCh <- PageResult{URL: pageURL, Err: err}
		return
	}
	req.Header.Set("Accept", "text/html,text/plain,application/json,application/xml,*/*")
	req.Header.Set("User-Agent", "Aleph-Sitemap-Crawler/1.0")

	resp, err := client.Do(req)
	if err != nil {
		resultCh <- PageResult{URL: pageURL, Err: err}
		return
	}
	defer resp.Body.Close()

	// Check content-type — skip binary content
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil && !isAllowedContentType(mediaType) {
			resultCh <- PageResult{
				URL:    pageURL,
				Status: resp.StatusCode,
				Err:    fmt.Errorf("skipped binary content-type: %s", mediaType),
			}
			return
		}
	}

	// Read up to maxPageSize bytes
	limitReader := io.LimitReader(resp.Body, maxPageSize)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		resultCh <- PageResult{URL: pageURL, Status: resp.StatusCode, Err: fmt.Errorf("read body: %w", err)}
		return
	}

	resultCh <- PageResult{
		URL:     pageURL,
		Content: body,
		Size:    int64(len(body)),
		Status:  resp.StatusCode,
	}
}

// isAllowedContentType returns true for text-based content types we want to download.
func isAllowedContentType(mediaType string) bool {
	switch {
	case mediaType == "text/html",
		mediaType == "text/plain",
		mediaType == "application/json",
		mediaType == "application/xml",
		mediaType == "text/xml",
		strings.HasSuffix(mediaType, "+xml"):
		return true
	default:
		return false
	}
}

// fetchAllPages fetches all given URLs concurrently using a worker pool pattern.
func (s *SitemapIngester) fetchAllPages(ctx context.Context, urls []string) ([]PageResult, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	// Use the existing WorkerPool to parallelise page fetching.
	// We create a ChunkJob per URL and map results back via index.
	jobs := make([]ChunkJob, len(urls))
	for i, u := range urls {
		// We store the URL in Data as a workaround since ChunkJob has Index+Data fields.
		// This lets us use the existing WorkerPool without modifying it.
		jobs[i] = ChunkJob{Index: i, Data: []byte(u)}
	}

	results := make([]PageResult, len(urls))
	var mu sync.Mutex

	pool := NewWorkerPool(DefaultChunkConfig)
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		pageURL := string(job.Data)
		ch := make(chan PageResult, 1)

		// Create a per-page timeout context
		pageCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		go func() {
			fetchSinglePage(pageCtx, pageURL, ch)
		}()

		select {
		case pr := <-ch:
			mu.Lock()
			results[job.Index] = pr
			mu.Unlock()
			if pr.Err != nil {
				return nil // Don't propagate page errors — record and continue
			}
			return nil
		case <-pageCtx.Done():
			mu.Lock()
			results[job.Index] = PageResult{URL: pageURL, Err: pageCtx.Err()}
			mu.Unlock()
			return nil // Timeout is recorded per-page, not propagated
		}
	})

	// WorkerPool.Run returns the first error from any worker retry loop.
	// If the error is a context cancellation, propagate it.
	if err != nil && ctx.Err() != nil {
		return nil, err
	}

	// Filter out any results that weren't set (shouldn't happen, but be safe)
	finalResults := make([]PageResult, 0, len(results))
	for _, r := range results {
		if r.URL != "" || r.Err != nil {
			finalResults = append(finalResults, r)
		}
	}

	return finalResults, nil
}
