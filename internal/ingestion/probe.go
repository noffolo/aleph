package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

// ProbeResult describes the result of probing a data source endpoint.
type ProbeResult interface {
	SourceType() string
	Pagination() PaginationInfo
	Columns() []ColumnInfo
	Validate() error
}

// PaginationInfo describes how to paginate through a data source.
type PaginationInfo struct {
	Type       string // "offset", "cursor", "page", "none"
	PageParam  string // e.g. "page", "offset", "cursor"
	LimitParam string // e.g. "limit", "per_page", "count"
	MaxLimit   int    // max page size; -1 for unlimited
}

// ColumnInfo describes a single column in the probed data.
type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"` // "string", "number", "boolean", "datetime", "object", "array"
	Path string `json:"path"` // JSONPath to value
}

// SourceProbeResult is the concrete ProbeResult implementation.
type SourceProbeResult struct {
	SrcType       string         `json:"src_type"`
	URL           string         `json:"url"`
	Pag           PaginationInfo `json:"pag"`
	Cols          []ColumnInfo   `json:"cols"`
	DataSample    []byte         `json:"data_sample"`
	TotalEstimate int64          `json:"total_estimate"`
}

func (s *SourceProbeResult) SourceType() string         { return s.SrcType }
func (s *SourceProbeResult) Pagination() PaginationInfo { return s.Pag }
func (s *SourceProbeResult) Columns() []ColumnInfo      { return s.Cols }

// Validate checks the probe result for correctness.
func (s *SourceProbeResult) Validate() error {
	if s.URL == "" {
		return fmt.Errorf("probe result: URL must be non-empty")
	}
	switch s.SrcType {
	case "rest", "rss", "github", "sitemap", "generic_json", "web":
		// valid
	default:
		return fmt.Errorf("probe result: unknown source type %q", s.SrcType)
	}
	if s.Pag.MaxLimit == 0 {
		return fmt.Errorf("probe result: Pagination.MaxLimit must be > 0 or -1 (unlimited), got 0")
	}
	return nil
}

// LLMProber uses an LLM to probe a data source endpoint and return structured
// metadata. If nil, ProbeRunner deduces metadata from response structure alone.
type LLMProber interface {
	ProbeEndpoint(ctx context.Context, endpoint string, sampleData []byte) (*SourceProbeResult, error)
}

// ProbeRunner probes data source endpoints to classify their type, detect
// pagination, infer column metadata, and support extraction execution.
type ProbeRunner struct {
	client    *sources.RateLimitedClient
	llmClient LLMProber
}

// NewProbeRunner creates a ProbeRunner with the given LLM client.
// If llmClient is nil, LLM probing is skipped.
func NewProbeRunner(llmClient LLMProber) *ProbeRunner {
	return &ProbeRunner{
		client:    sources.NewRateLimitedClient(sources.DefaultRate),
		llmClient: llmClient,
	}
}

// Probe probes an endpoint and returns classified source metadata.
func (p *ProbeRunner) Probe(ctx context.Context, endpoint string) (*SourceProbeResult, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("probe: endpoint must be non-empty")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("probe: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json, application/xml, text/html, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("probe: fetch %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("probe: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("probe: HTTP %d from %s", resp.StatusCode, endpoint)
	}

	ct := resp.Header.Get("Content-Type")
	sourceType := classifySourceType(endpoint, ct, body)
	pag := detectPagination(endpoint, resp, body)
	cols := detectColumns(body)

	result := &SourceProbeResult{
		SrcType:       sourceType,
		URL:           endpoint,
		Pag:           pag,
		Cols:          cols,
		DataSample:    body,
		TotalEstimate: -1,
	}

	if p.llmClient != nil {
		llmResult, err := p.llmClient.ProbeEndpoint(ctx, endpoint, body)
		if err == nil && llmResult != nil {
			if llmResult.SrcType != "" {
				result.SrcType = llmResult.SrcType
			}
			if len(llmResult.Cols) > 0 {
				result.Cols = llmResult.Cols
			}
			if llmResult.TotalEstimate > 0 {
				result.TotalEstimate = llmResult.TotalEstimate
			}
		}
	}

	return result, nil
}

// Execute performs full extraction based on a probe result using FetchPages.
func (p *ProbeRunner) Execute(ctx context.Context, result *SourceProbeResult) error {
	if result == nil {
		return fmt.Errorf("execute: nil probe result")
	}
	if err := result.Validate(); err != nil {
		return fmt.Errorf("execute: invalid probe result: %w", err)
	}

	nextURLFn := buildNextURLFn(result.URL, result.Pag)

	var allData bytes.Buffer
	consumeFn := func(body []byte) error {
		allData.Write(body)
		return nil
	}

	if err := sources.FetchPages(ctx, p.client, result.URL, nil, nextURLFn, consumeFn); err != nil {
		return fmt.Errorf("execute: fetch pages: %w", err)
	}

	result.DataSample = allData.Bytes()
	return nil
}

var (
	reSitemap = regexp.MustCompile(`(?i)<(urlset|sitemapindex)`)
	reRSS     = regexp.MustCompile(`(?i)<(rss|feed)\b`)
	reHTML    = regexp.MustCompile(`(?i)<!DOCTYPE\s+html|<html\b`)
	reGitHub  = regexp.MustCompile(`github\.com/.*/repos/`)
)

func classifySourceType(endpoint string, contentType string, body []byte) string {
	// 1. Check URL pattern for GitHub API
	if reGitHub.MatchString(endpoint) {
		return "github"
	}

	// 2. Check Content-Type for XML-based types
	ct := strings.ToLower(contentType)

	// 3. Check body for sitemap XML
	if strings.Contains(ct, "xml") || strings.Contains(ct, "text/xml") || strings.Contains(ct, "application/xml") {
		if reSitemap.Match(body) {
			return "sitemap"
		}
		if reRSS.Match(body) {
			return "rss"
		}
	}

	// 4. Check body regardless of Content-Type
	trimmed := bytes.TrimSpace(body)

	// Check for sitemap XML in body (even if Content-Type is wrong)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		if reSitemap.Match(trimmed) {
			return "sitemap"
		}
		if reRSS.Match(trimmed) {
			return "rss"
		}
		if reHTML.Match(trimmed) {
			return "web"
		}
		// Generic XML
		if bytes.Contains(trimmed, []byte("<?xml")) {
			return "generic_json" // treat as generic data
		}
	}

	// 5. Check body for HTML
	if reHTML.Match(body) {
		return "web"
	}

	// 6. JSON detection
	if len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
		if trimmed[0] == '[' {
			return "rest"
		}
		// Check if it's a JSON object (not an array wrapper)
		return "rest"
	}

	// 7. Default
	return "generic_json"
}

// =============================================================================
// Pagination Detection
// =============================================================================

var knownPageParams = []string{"page", "offset", "cursor", "start", "skip"}

func detectPagination(endpoint string, resp *http.Response, body []byte) PaginationInfo {
	// 1. Check Link header for cursor/rel=next pagination
	if link := resp.Header.Get("Link"); link != "" {
		if strings.Contains(link, `rel="next"`) {
			return PaginationInfo{
				Type:       "cursor",
				PageParam:  "cursor",
				LimitParam: "limit",
				MaxLimit:   100,
			}
		}
	}

	// 2. Check URL query params for known pagination parameters
	parsedURL, err := url.Parse(endpoint)
	if err == nil {
		query := parsedURL.Query()
		for _, param := range knownPageParams {
			if query.Has(param) {
				limit := "limit"
				if query.Has("per_page") {
					limit = "per_page"
				} else if query.Has("count") {
					limit = "count"
				}
				maxLimit := 100
				if param == "offset" {
					maxLimit = 50
				}
				return PaginationInfo{
					Type:       "offset",
					PageParam:  param,
					LimitParam: limit,
					MaxLimit:   maxLimit,
				}
			}
		}
	}

	// 3. Check for page/offset pagination in JSON body for REST endpoints
	if len(body) > 0 && body[0] == '{' {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil {
			// Check for common pagination fields in the JSON response
			if _, hasPage := parsed["page"]; hasPage {
				return PaginationInfo{
					Type:       "page",
					PageParam:  "page",
					LimitParam: "per_page",
					MaxLimit:   100,
				}
			}
			if _, hasOffset := parsed["offset"]; hasOffset {
				return PaginationInfo{
					Type:       "offset",
					PageParam:  "offset",
					LimitParam: "limit",
					MaxLimit:   50,
				}
			}
			// Check if the response has a nested data object with pagination
			if meta, ok := parsed["meta"]; ok {
				if metaMap, ok := meta.(map[string]any); ok {
					if _, hasPage := metaMap["page"]; hasPage {
						return PaginationInfo{
							Type:       "page",
							PageParam:  "page",
							LimitParam: "per_page",
							MaxLimit:   100,
						}
					}
				}
			}
		}
	}

	// 4. Default: no pagination detected
	return PaginationInfo{
		Type:       "none",
		PageParam:  "",
		LimitParam: "",
		MaxLimit:   -1,
	}
}

// =============================================================================
// Column Detection
// =============================================================================

func detectColumns(body []byte) []ColumnInfo {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}

	// Try to extract the first object from a JSON array response
	var data any
	if err := json.Unmarshal(trimmed, &data); err != nil {
		return nil
	}

	// Extract first object from array
	switch v := data.(type) {
	case []any:
		if len(v) > 0 {
			if obj, ok := v[0].(map[string]any); ok {
				return columnsFromMap(obj, "")
			}
		}
	case map[string]any:
		// Check for common data wrapper patterns
		for _, key := range []string{"data", "results", "items", "records", "entries"} {
			if arr, ok := v[key].([]any); ok && len(arr) > 0 {
				if obj, ok := arr[0].(map[string]any); ok {
					return columnsFromMap(obj, key)
				}
			}
		}
		// If no wrapper found, treat the object itself as the data
		// (skip meta, pagination etc)
		skipKeys := map[string]bool{"meta": true, "pagination": true, "links": true}
		objCols := columnsFromMapSkip(v, "", skipKeys)
		if len(objCols) > 0 {
			return objCols
		}
	}

	return nil
}

func columnsFromMap(m map[string]any, prefix string) []ColumnInfo {
	var cols []ColumnInfo
	for k, v := range m {
		path := prefix + "." + k
		if prefix == "" {
			path = k
		}
		colType := goValueToColumnType(v)
		cols = append(cols, ColumnInfo{
			Name: k,
			Type: colType,
			Path: "$." + path,
		})
	}
	return cols
}

func columnsFromMapSkip(m map[string]any, prefix string, skip map[string]bool) []ColumnInfo {
	var cols []ColumnInfo
	for k, v := range m {
		if skip[k] {
			continue
		}
		path := prefix + "." + k
		if prefix == "" {
			path = k
		}
		colType := goValueToColumnType(v)
		cols = append(cols, ColumnInfo{
			Name: k,
			Type: colType,
			Path: "$." + path,
		})
	}
	return cols
}

func goValueToColumnType(v any) string {
	if v == nil {
		return "string"
	}
	switch v.(type) {
	case float64:
		// JSON numbers always decode as float64
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "string"
	}
}

// =============================================================================
// NextURL Builder
// =============================================================================

func buildNextURLFn(initialURL string, pag PaginationInfo) func(body []byte) string {
	if pag.Type == "none" || pag.Type == "" {
		return nil
	}

	return func(body []byte) string {
		switch pag.Type {
		case "cursor":
			return nextCursorURL(body, pag.PageParam)
		case "page", "offset":
			return nextPageURL(body, initialURL, pag)
		default:
			return ""
		}
	}
}

func nextCursorURL(body []byte, cursorParam string) string {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	// Try common cursor locations
	for _, key := range []string{"next_cursor", "cursor", "next", "paging", "next_page_token"} {
		if cursor, ok := parsed[key]; ok {
			if cursorStr, ok := cursor.(string); ok && cursorStr != "" {
				// Reconstruct URL — append/replace cursor param
				return cursorStr
			}
		}
	}

	// Check nested meta.pagination
	if meta, ok := parsed["meta"]; ok {
		if metaMap, ok := meta.(map[string]any); ok {
			if cursor, ok := metaMap["next_cursor"]; ok {
				if cursorStr, ok := cursor.(string); ok && cursorStr != "" {
					return cursorStr
				}
			}
			if pag, ok := metaMap["pagination"]; ok {
				if pagMap, ok := pag.(map[string]any); ok {
					if next, ok := pagMap["next"]; ok {
						if nextStr, ok := next.(string); ok && nextStr != "" {
							return nextStr
						}
					}
				}
			}
		}
	}

	return ""
}

func nextPageURL(body []byte, initialURL string, pag PaginationInfo) string {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	// Determine current page/offset value
	var currentPage float64
	// Check top-level and nested meta
	if page, ok := parsed["page"]; ok {
		currentPage, _ = toFloat64(page)
	} else if meta, ok := parsed["meta"]; ok {
		if metaMap, ok := meta.(map[string]any); ok {
			if page, ok := metaMap["page"]; ok {
				currentPage, _ = toFloat64(page)
			} else if offset, ok := metaMap["offset"]; ok {
				currentPage, _ = toFloat64(offset)
			}
		}
	}

	if currentPage == 0 {
		return ""
	}

	// Build next URL with incremented param
	u, err := url.Parse(initialURL)
	if err != nil {
		return ""
	}
	q := u.Query()

	var nextVal string
	switch pag.Type {
	case "page":
		nextVal = fmt.Sprintf("%.0f", currentPage+1)
	case "offset":
		limit := pag.MaxLimit
		if limit <= 0 {
			limit = 50
		}
		nextVal = fmt.Sprintf("%.0f", currentPage+float64(limit))
	}

	q.Set(pag.PageParam, nextVal)
	u.RawQuery = q.Encode()
	return u.String()
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
