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
)

// ScrapeConfig defines how to extract structured data from an HTML page.
type ScrapeConfig struct {
	URL             string `json:"url"`
	ArticleSelector string `json:"article_selector"`
	TitleSelector   string `json:"title_selector"`
	LinkSelector    string `json:"link_selector"`
	DateSelector    string `json:"date_selector"`
	DateFormat      string `json:"date_format"`
	AuthorSelector  string `json:"author_selector"`
	ContentSelector string `json:"content_selector"`
	MaxArticles     int    `json:"max_articles"`
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
			return
		}

		var link string
		if config.LinkSelector != "" {
			linkEl := sel.Find(config.LinkSelector).First()
			if href, exists := linkEl.Attr("href"); exists {
				link = resolveHref(baseURL, href)
			}
		}
		if link == "" {
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
			Title:  title,
			Link:   link,
			Date:   date,
			Author: author,
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

// fetchHTML downloads HTML content from a URL, rate-limited via the ingester's client.
func (s *ScrapeIngester) fetchHTML(ctx context.Context, urlStr string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	req.Header.Set("User-Agent", "Aleph-Scraper/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
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
