package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/safeident"
	"github.com/ff3300/aleph-v2/internal/ssrf"
	"golang.org/x/time/rate"
)

const polymarketSourceType = "polymarket"

// PolymarketFetcher implements the ingestion.Fetcher interface for polymorphic dispatch.
type PolymarketFetcher struct{}

func (p *PolymarketFetcher) SourceType() string { return polymarketSourceType }
func (p *PolymarketFetcher) Validate() error     { return nil }

// Polymarket Gamma API: 300 requests per 10s on /markets endpoint.
const polymarketGammaRPS = 30.0
const polymarketCLOBRPS = 10.0

type polymarketMarket struct {
	ConditionID   string      `json:"condition_id"`
	Question      string      `json:"question"`
	Description   string      `json:"description"`
	EndDateISO    string      `json:"end_date_iso"`
	TokenIDs      []string    `json:"token_ids"`
	Active        bool        `json:"active"`
	Closed        bool        `json:"closed"`
	Outcomes      []string    `json:"outcomes"`
	OutcomePrices []string    `json:"outcome_prices"`
	Volume        float64     `json:"volume"`
	Tags          []pmTag     `json:"tags"`
}

type pmTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type polymarketPricePoint struct {
	T int64   `json:"t"` // Unix epoch seconds
	P float64 `json:"p"`
}

type polymarketPriceHistory struct {
	History []polymarketPricePoint `json:"history"`
}

type polymarketEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Active      bool   `json:"active"`
	Closed      bool   `json:"closed"`
	Category    string `json:"category"`
}

var italianKeywords = []string{
	"italy", "italia", "meloni", "salvini",
	"italian election", "italian government",
}

// isItalianMarket returns true when a market's question or tags match Italy-related keywords.
func isItalianMarket(m polymarketMarket) bool {
	lower := strings.ToLower(m.Question)
	for _, kw := range italianKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	for _, t := range m.Tags {
		for _, kw := range italianKeywords {
			if strings.Contains(strings.ToLower(t.Label), kw) {
				return true
			}
		}
	}
	return false
}

func ensurePolymarketTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS polymarket_markets (
			token_id VARCHAR PRIMARY KEY,
			condition_id VARCHAR,
			question TEXT,
			description TEXT,
			end_date TIMESTAMP,
			active BOOLEAN,
			closed BOOLEAN,
			volume DOUBLE,
			outcomes JSON,
			tags JSON,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS polymarket_prices (
			token_id VARCHAR,
			timestamp TIMESTAMP,
			price_yes DOUBLE,
			price_no DOUBLE,
			PRIMARY KEY (token_id, timestamp)
		)`,
		`CREATE TABLE IF NOT EXISTS polymarket_events (
			id VARCHAR PRIMARY KEY,
			title TEXT,
			description TEXT,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			active BOOLEAN,
			closed BOOLEAN,
			category VARCHAR,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("create polymarket table: %w", err)
		}
	}
	return nil
}

type polymarketClient struct {
	httpClient *http.Client
	gammaLim   *rate.Limiter
	clobLim    *rate.Limiter
}

func newPolymarketClient() *polymarketClient {
	return &polymarketClient{
		httpClient: ssrf.NewClient(),
		gammaLim:   rate.NewLimiter(rate.Limit(polymarketGammaRPS), 1),
		clobLim:    rate.NewLimiter(rate.Limit(polymarketCLOBRPS), 2),
	}
}

func (c *polymarketClient) get(ctx context.Context, lim *rate.Limiter, urlStr string) ([]byte, error) {
	if err := lim.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", urlStr, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		trunc := body
		if len(trunc) > 500 {
			trunc = trunc[:500]
		}
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, urlStr, string(trunc))
	}
	return body, nil
}

func (c *polymarketClient) gammaGet(ctx context.Context, urlStr string) ([]byte, error) {
	return c.get(ctx, c.gammaLim, urlStr)
}

func (c *polymarketClient) clobGet(ctx context.Context, urlStr string) ([]byte, error) {
	return c.get(ctx, c.clobLim, urlStr)
}

const (
	gammaBaseURL = "https://gamma-api.polymarket.com"
	clobBaseURL  = "https://clob.polymarket.com"
)

func searchMarkets(ctx context.Context, client *polymarketClient, query string) ([]polymarketMarket, error) {
	u := fmt.Sprintf("%s/public-search?%s", gammaBaseURL,
		url.Values{"query": {query}, "limit": {"50"}}.Encode())
	body, err := client.gammaGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("search markets %q: %w", query, err)
	}
	var markets []polymarketMarket
	if err := json.Unmarshal(body, &markets); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	return markets, nil
}

func fetchPrices(ctx context.Context, client *polymarketClient, tokenID string) ([]polymarketPricePoint, error) {
	u := fmt.Sprintf("%s/prices-history?%s", clobBaseURL,
		url.Values{"market": {tokenID}, "interval": {"1d"}, "fidelity": {"60"}}.Encode())
	body, err := client.clobGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("fetch prices for %s: %w", tokenID, err)
	}
	var ph polymarketPriceHistory
	if err := json.Unmarshal(body, &ph); err != nil {
		return nil, fmt.Errorf("decode prices: %w", err)
	}
	return ph.History, nil
}

func fetchEvents(ctx context.Context, client *polymarketClient) ([]polymarketEvent, error) {
	u := fmt.Sprintf("%s/events?%s", gammaBaseURL,
		url.Values{"tag_id": {"2"}, "active": {"true"}, "closed": {"false"}, "limit": {"100"}}.Encode())
	body, err := client.gammaGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("fetch events: %w", err)
	}
	var events []polymarketEvent
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return events, nil
}

func parsePolymarketTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %s", s)
}

// RunPolymarket executes the full Polymarket ingestion pipeline.
func RunPolymarket(ctx context.Context, db *sql.DB, rawDir string) error {
	slog.Info("starting Polymarket ingestion")
	if err := ensurePolymarketTables(db); err != nil {
		return fmt.Errorf("ensure tables: %w", err)
	}
	rawPath := filepath.Join(rawDir, polymarketSourceType)
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}
	client := newPolymarketClient()

	allMarkets := make(map[string]polymarketMarket)
	for _, kw := range italianKeywords {
		markets, err := searchMarkets(ctx, client, kw)
		if err != nil {
			slog.Warn("polymarket search failed", "query", kw, "error", err)
			continue
		}
		for _, m := range markets {
			for _, tid := range m.TokenIDs {
				if _, exists := allMarkets[tid]; !exists {
					allMarkets[tid] = m
				}
			}
		}
	}

	var filtered []polymarketMarket
	for _, m := range allMarkets {
		if isItalianMarket(m) {
			filtered = append(filtered, m)
		}
	}
	slog.Info("polymarket Italian markets found", "total", len(allMarkets), "filtered", len(filtered))

	marketsJSON, _ := json.MarshalIndent(filtered, "", "  ")
	if err := os.WriteFile(filepath.Join(rawPath, "markets.json"), marketsJSON, 0644); err != nil {
		slog.Warn("failed to save raw markets", "error", err)
	}

	insertMarket := func(db *sql.DB, m polymarketMarket) error {
		outcomesJSON, _ := json.Marshal(m.Outcomes)
		tagsJSON, _ := json.Marshal(m.Tags)
		var endDate sql.NullTime
		if t, err := parsePolymarketTime(m.EndDateISO); err == nil && !t.IsZero() {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
		for _, tid := range m.TokenIDs {
			if tid == "" {
				continue
			}
			if err := safeident.ValidateStrictIdentifier("polymarket_markets"); err != nil {
				return err
			}
			_, err := db.Exec(
				`INSERT OR REPLACE INTO polymarket_markets
				 (token_id, condition_id, question, description, end_date, active, closed, volume, outcomes, tags)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				tid, m.ConditionID, m.Question, m.Description,
				endDate, m.Active, m.Closed, m.Volume,
				string(outcomesJSON), string(tagsJSON),
			)
			if err != nil {
				return fmt.Errorf("insert market %s: %w", tid, err)
			}
		}
		return nil
	}

	for _, m := range filtered {
		if err := insertMarket(db, m); err != nil {
			slog.Warn("polymarket insert market failed", "question", m.Question, "error", err)
		}
	}

	for _, m := range filtered {
		for _, tid := range m.TokenIDs {
			if tid == "" {
				continue
			}
			prices, err := fetchPrices(ctx, client, tid)
			if err != nil {
				slog.Warn("polymarket price fetch failed", "token_id", tid, "error", err)
				continue
			}
			for _, pp := range prices {
				ts := time.Unix(pp.T, 0).UTC()
				priceYes := pp.P
				priceNo := 1.0 - pp.P
				_, err := db.Exec(
					`INSERT OR REPLACE INTO polymarket_prices (token_id, timestamp, price_yes, price_no)
					 VALUES (?, ?, ?, ?)`,
					tid, ts, priceYes, priceNo,
				)
				if err != nil {
					slog.Warn("polymarket insert price failed", "token_id", tid, "t", pp.T, "error", err)
				}
			}
		}
	}

	pricesJSON, _ := json.MarshalIndent(filtered, "", "  ")
	_ = os.WriteFile(filepath.Join(rawPath, "prices_raw.json"), pricesJSON, 0644)

	events, err := fetchEvents(ctx, client)
	if err != nil {
		slog.Warn("polymarket events fetch failed", "error", err)
	} else {
		eventsJSON, _ := json.MarshalIndent(events, "", "  ")
		_ = os.WriteFile(filepath.Join(rawPath, "events.json"), eventsJSON, 0644)
		for _, ev := range events {
			startDate := sql.NullTime{}
			if t, err := parsePolymarketTime(ev.StartDate); err == nil && !t.IsZero() {
				startDate = sql.NullTime{Time: t, Valid: true}
			}
			endDate := sql.NullTime{}
			if t, err := parsePolymarketTime(ev.EndDate); err == nil && !t.IsZero() {
				endDate = sql.NullTime{Time: t, Valid: true}
			}
			_, err := db.Exec(
				`INSERT OR REPLACE INTO polymarket_events
				 (id, title, description, start_date, end_date, active, closed, category)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				ev.ID, ev.Title, ev.Description,
				startDate, endDate, ev.Active, ev.Closed, ev.Category,
			)
			if err != nil {
				slog.Warn("polymarket insert event failed", "id", ev.ID, "error", err)
			}
		}
		slog.Info("polymarket events ingested", "count", len(events))
	}

	slog.Info("Polymarket ingestion complete", "markets", len(filtered))
	return nil
}
