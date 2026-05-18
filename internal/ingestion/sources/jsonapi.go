// Package sources implements W3 ingestion methods.
package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// =============================================================================
// JSON/API Ingester
// =============================================================================

// APIConfig defines how to fetch pages from a JSON REST API.
type APIConfig struct {
	BaseURL        string
	Headers        map[string]string
	PaginationType string // "offset", "cursor", "page", "none"
	PageParam      string // e.g. "page" (default: "page")
	LimitParam     string // e.g. "per_page" (default: "per_page")
	Limit          int    // page size
	DataPath       string // JSONPath to data array, e.g. "data.items", "results" ("" means root is the array)
	TotalPath      string // JSONPath to total count, e.g. "total", "meta.total"
	MaxPages       int    // max pages to fetch (0 = all)
}

// Validate checks the APIConfig for common mistakes.
func (c APIConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}
	switch c.PaginationType {
	case "offset", "cursor", "page", "none", "":
		// valid
	default:
		return fmt.Errorf("unknown PaginationType %q", c.PaginationType)
	}
	if c.PaginationType != "" && c.PaginationType != "none" && c.Limit <= 0 {
		return fmt.Errorf("Limit must > 0 for pagination type %q", c.PaginationType)
	}
	return nil
}

// JSONAPIIngester ingests any REST API with JSON responses,
// auto-detecting pagination and flattening results.
type JSONAPIIngester struct {
	client *RateLimitedClient
}

// NewJSONAPIIngester creates an ingester with default rate limits.
func NewJSONAPIIngester() *JSONAPIIngester {
	return &JSONAPIIngester{
		client: NewRateLimitedClient(DefaultRate),
	}
}

// FetchAll fetches all pages from a paginated JSON API and returns
// a single JSON array of all accumulated items.
func (j *JSONAPIIngester) FetchAll(ctx context.Context, cfg APIConfig) ([]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("jsonapi FetchAll: invalid config: %w", err)
	}

	pageParam := cfg.PageParam
	if pageParam == "" {
		pageParam = "page"
	}
	limitParam := cfg.LimitParam
	if limitParam == "" {
		limitParam = "per_page"
	}
	limit := cfg.Limit
	if limit <= 0 && cfg.PaginationType != "none" {
		limit = 100
	}
	if limit <= 0 {
		limit = 100
	}

	// Build initial URL with first-page params.
	initialURL := cfg.BaseURL
	switch cfg.PaginationType {
	case "offset":
		u, err := url.Parse(cfg.BaseURL)
		if err == nil {
			q := u.Query()
			q.Set(limitParam, strconv.Itoa(limit))
			q.Set(pageParam, "0")
			u.RawQuery = q.Encode()
			initialURL = u.String()
		}
	case "page":
		u, err := url.Parse(cfg.BaseURL)
		if err == nil {
			q := u.Query()
			q.Set(limitParam, strconv.Itoa(limit))
			q.Set(pageParam, "1")
			u.RawQuery = q.Encode()
			initialURL = u.String()
		}
	case "cursor", "none":
		// Use BaseURL as-is.
	}

	pageCount := 1
	maxPages := cfg.MaxPages
	if maxPages <= 0 {
		maxPages = math.MaxInt
	}

	var allItems []json.RawMessage

	err := FetchPages(ctx, j.client, initialURL, cfg.Headers,
		// nextURLFn – return URL for next page, or "" to stop.
		func(body []byte) string {
			if pageCount >= maxPages {
				return ""
			}

			switch cfg.PaginationType {
			case "none":
				return ""
			case "offset":
				offset := pageCount * limit
				pageCount++
				u, err := url.Parse(cfg.BaseURL)
				if err != nil {
					return ""
				}
				q := u.Query()
				q.Set(limitParam, strconv.Itoa(limit))
				q.Set(pageParam, strconv.Itoa(offset))
				u.RawQuery = q.Encode()
				return u.String()
			case "page":
				pageCount++
				u, err := url.Parse(cfg.BaseURL)
				if err != nil {
					return ""
				}
				q := u.Query()
				q.Set(limitParam, strconv.Itoa(limit))
				q.Set(pageParam, strconv.Itoa(pageCount))
				u.RawQuery = q.Encode()
				return u.String()
			case "cursor":
				pageCount++
				return extractCursorNext(body)
			}
			return ""
		},
		// consumeFn – extract items from body.
		func(body []byte) error {
			items, err := extractItems(body, cfg.DataPath)
			if err != nil {
				return fmt.Errorf("extract items at path %q: %w", cfg.DataPath, err)
			}
			allItems = append(allItems, items...)
			return nil
		},
	)
	if err != nil {
		// Wrap with accumulated count context.
		return nil, fmt.Errorf("jsonapi FetchAll after %d pages: %w", pageCount, err)
	}

	if allItems == nil {
		allItems = []json.RawMessage{} // ensure "[]" not "null"
	}
	return json.Marshal(allItems)
}

// =============================================================================
// Probe: fetch a single page + auto-detect structure
// =============================================================================

// SourceProbeResult is the result of probing a JSON API endpoint.
type SourceProbeResult struct {
	SourceType string // "array-root" or "object-with-nested"
	DataPath   string // detected path to data array ("" if root is array)
	SampleBody []byte // raw first-page body (truncated to 64 KiB)
	PageURL    string
}

// Probe fetches a single page and returns a probe result describing the
// response structure. It does NOT fetch multiple pages.
func (j *JSONAPIIngester) Probe(ctx context.Context, apiURL string, headers map[string]string) (*SourceProbeResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("jsonapi Probe create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jsonapi Probe GET %s: %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jsonapi Probe HTTP %d fetching %s: %s", resp.StatusCode, apiURL, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jsonapi Probe read body %s: %w", apiURL, err)
	}

	result := &SourceProbeResult{
		PageURL:    apiURL,
		SampleBody: body,
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return result, nil
	}

	// Detect root type.
	switch trimmed[0] {
	case '[':
		result.SourceType = "array-root"
		result.DataPath = ""
	default:
		result.SourceType = "object-with-nested"
		// Try common array paths.
		for _, p := range []string{"data", "items", "results", "records", "values"} {
			raw, err := resolveJSONPath(body, p)
			if err != nil {
				continue
			}
			var testArr []json.RawMessage
			if json.Unmarshal(raw, &testArr) == nil && len(testArr) > 0 {
				result.DataPath = p
				break
			}
		}
	}

	// Truncate sample to 64 KiB.
	if len(result.SampleBody) > 64*1024 {
		result.SampleBody = result.SampleBody[:64*1024]
	}

	return result, nil
}

// =============================================================================
// DetectConfig: auto-detect APIConfig from a sample URL
// =============================================================================

// DetectConfig probes a sample URL and returns a best-guess APIConfig.
func (j *JSONAPIIngester) DetectConfig(ctx context.Context, sampleURL string) (*APIConfig, error) {
	cfg := &APIConfig{
		BaseURL:        sampleURL,
		Headers:        map[string]string{},
		PaginationType: "none",
		PageParam:      "page",
		LimitParam:     "per_page",
		Limit:          100,
	}

	// Detect pagination type from URL query params.
	u, err := url.Parse(sampleURL)
	if err == nil {
		q := u.Query()
		switch {
		case q.Has("offset") || q.Has("limit"):
			cfg.PaginationType = "offset"
			cfg.PageParam = "offset"
			cfg.LimitParam = "limit"
			if v := q.Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
					cfg.Limit = n
				}
			}
		case q.Has("cursor") || q.Has("after"):
			cfg.PaginationType = "cursor"
		case q.Has("page") || q.Has("per_page"):
			cfg.PaginationType = "page"
			if _, ok := q["per_page"]; ok {
				cfg.LimitParam = "per_page"
			}
			if _, ok := q["page"]; ok {
				cfg.PageParam = "page"
			}
			// Try to read existing limit.
			for _, key := range []string{"per_page", "limit", "page_size", "size"} {
				if v := q.Get(key); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10000 {
						cfg.Limit = n
					}
					break
				}
			}
		}
	}

	// Fetch first page to detect structure.
	req, err := http.NewRequestWithContext(ctx, "GET", sampleURL, nil)
	if err != nil {
		return cfg, nil // return best-effort config without probe
	}
	req.Header.Set("Accept", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return cfg, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cfg, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return cfg, nil
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return cfg, nil
	}

	// Detect root type.
	if trimmed[0] == '[' {
		cfg.DataPath = "" // root is the array
	} else {
		for _, p := range []string{"data", "items", "results", "records", "values"} {
			raw, err := resolveJSONPath(body, p)
			if err != nil {
				continue
			}
			var testArr []json.RawMessage
			if json.Unmarshal(raw, &testArr) == nil && len(testArr) > 0 {
				cfg.DataPath = p
				break
			}
		}
	}

	return cfg, nil
}

// =============================================================================
// JSONPath helper
// =============================================================================

// resolveJSONPath walks dot-notation like "data.items" or "meta.total"
// through nested JSON objects and returns the raw JSON bytes at that path.
// Returns an error if any segment is missing or is not a JSON object.
func resolveJSONPath(data []byte, path string) ([]byte, error) {
	if path == "" {
		return data, nil
	}
	parts := strings.Split(path, ".")
	current := json.RawMessage(data)
	for _, part := range parts {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(current, &obj); err != nil {
			return nil, fmt.Errorf("resolveJSONPath: expected object at %q (not traversable): %w", part, err)
		}
		val, ok := obj[part]
		if !ok {
			return nil, fmt.Errorf("resolveJSONPath: field %q not found", part)
		}
		current = val
	}
	return []byte(current), nil
}

// =============================================================================
// Internal helpers
// =============================================================================

// extractItems pulls the JSON array from body at the given DataPath.
func extractItems(body []byte, dataPath string) ([]json.RawMessage, error) {
	var data []byte
	var err error
	if dataPath == "" {
		data = body
	} else {
		data, err = resolveJSONPath(body, dataPath)
		if err != nil {
			return nil, fmt.Errorf("resolve data path %q: %w", dataPath, err)
		}
	}

	// Handle both array-at-root and array-as-field-in-object.
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		// Maybe the path resolved to an object that wraps the array.
		// Try common wrappers: "items", "data", "results".
		for _, p := range []string{"items", "data", "results", "records"} {
			raw, err := resolveJSONPath(data, p)
			if err == nil {
				var testArr []json.RawMessage
				if json.Unmarshal(raw, &testArr) == nil {
					return testArr, nil
				}
			}
		}
		// Single page object – wrap it as one item.
		return []json.RawMessage{json.RawMessage(data)}, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("expected JSON array: %w", err)
	}
	return items, nil
}

// extractCursorNext reads the next-page URL from a cursor-paginated response.
// Looks for: "next" as top-level, "links.next", "data.next", "meta.next".
func extractCursorNext(body []byte) string {
	// Flatten lookups across common cursor paths.
	for _, p := range []string{"next", "links.next", "data.next", "meta.next", "cursor.next"} {
		raw, err := resolveJSONPath(body, p)
		if err != nil {
			continue
		}
		var val string
		if json.Unmarshal(raw, &val) == nil && val != "" {
			return val
		}
	}
	return ""
}

// classifySourceType determines the response structure from headers and body.
// Deprecated: use Probe instead.
func classifySourceType(body []byte, contentTypes []string) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "unknown"
	}
	if trimmed[0] == '[' {
		return "array-root"
	}
	// Check if it has a common data wrapping field.
	for _, p := range []string{"data", "items", "results", "records", "values"} {
		raw, err := resolveJSONPath(body, p)
		if err != nil {
			continue
		}
		var testArr []json.RawMessage
		if json.Unmarshal(raw, &testArr) == nil && len(testArr) > 0 {
			return "object-with-nested"
		}
	}
	return "unknown"
}

// resolveTotal reads the total count from a response using TotalPath.
func resolveTotal(body []byte, totalPath string) (int, bool) {
	if totalPath == "" {
		return 0, false
	}
	raw, err := resolveJSONPath(body, totalPath)
	if err != nil {
		return 0, false
	}
	var total json.Number
	if err := json.Unmarshal(raw, &total); err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(string(total))
	if err != nil {
		return 0, false
	}
	return n, n > 0
}

// ensure at compile time JSONAPIIngester exists as the primary type.
var _ = (*JSONAPIIngester)(nil)
