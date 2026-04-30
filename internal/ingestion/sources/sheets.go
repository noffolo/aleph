package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SheetsIngester reads Google Sheets via the Sheets API v4 using raw HTTP.
type SheetsIngester struct {
	client *RateLimitedClient
	apiKey string
}

// NewSheetsIngester creates a SheetsIngester with Google API rate limits.
func NewSheetsIngester(apiKey string) *SheetsIngester {
	return &SheetsIngester{
		client: NewRateLimitedClient(GoogleRate),
		apiKey: apiKey,
	}
}

// SheetConfig describes which sheet and range to fetch.
type SheetConfig struct {
	SpreadsheetID string // from URL: /spreadsheets/d/{ID}/
	Range         string // e.g. "Sheet1!A:Z" (empty = entire first sheet)
	SheetName     string // sheet name for metadata
}

const sheetsAPIHost = "https://sheets.googleapis.com/v4/spreadsheets"

// FetchSheet fetches a sheet range and returns a JSON array of objects.
// The first row is treated as headers; subsequent rows become objects keyed by
// those headers. Empty cells are omitted. All values are returned as strings.
func (s *SheetsIngester) FetchSheet(ctx context.Context, config SheetConfig) ([]byte, error) {
	rng := config.Range
	if rng == "" {
		rng = "A:Z"
	}

	u := fmt.Sprintf("%s/%s/values/%s?key=%s",
		sheetsAPIHost,
		url.PathEscape(config.SpreadsheetID),
		url.PathEscape(rng),
		url.QueryEscape(s.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("sheets: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: fetch %s: %w", config.SpreadsheetID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sheets: read body: %w", err)
	}

	if err := checkSheetsHTTPError(resp.StatusCode, body); err != nil {
		return nil, fmt.Errorf("sheets: %w", err)
	}

	var valueRange struct {
		Values [][]string `json:"values"`
	}
	if err := json.Unmarshal(body, &valueRange); err != nil {
		return nil, fmt.Errorf("sheets: parse response: %w", err)
	}

	if len(valueRange.Values) == 0 {
		return json.Marshal([]map[string]string{})
	}

	headers := make([]string, len(valueRange.Values[0]))
	for i, h := range valueRange.Values[0] {
		headers[i] = strings.TrimSpace(h)
	}

	rows := make([]map[string]string, 0, len(valueRange.Values)-1)
	for _, row := range valueRange.Values[1:] {
		obj := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) && row[i] != "" {
				obj[h] = row[i]
			}
		}
		if len(obj) > 0 {
			rows = append(rows, obj)
		}
	}

	return json.Marshal(rows)
}

type sheetsMetadata struct {
	Sheets []struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
	} `json:"sheets"`
}

// FetchAllSheets fetches every sheet in the spreadsheet and returns a map of
// sheet name to JSON bytes. Sheet fetches are parallelized via WorkerPool.
func (s *SheetsIngester) FetchAllSheets(ctx context.Context, spreadsheetID string) (map[string][]byte, error) {
	u := fmt.Sprintf("%s/%s?key=%s",
		sheetsAPIHost,
		url.PathEscape(spreadsheetID),
		url.QueryEscape(s.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("sheets: create metadata request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: fetch metadata %s: %w", spreadsheetID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sheets: read metadata body: %w", err)
	}

	if err := checkSheetsHTTPError(resp.StatusCode, body); err != nil {
		return nil, fmt.Errorf("sheets: metadata: %w", err)
	}

	var meta sheetsMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("sheets: parse metadata: %w", err)
	}

	if len(meta.Sheets) == 0 {
		return nil, fmt.Errorf("sheets: no sheets found in spreadsheet %s", spreadsheetID)
	}

	type sheetResult struct {
		name string
		data []byte
		err  error
	}

	results := make([]sheetResult, len(meta.Sheets))

	pool := NewWorkerPool(DefaultChunkConfig)
	jobs := make([]ChunkJob, 0, len(meta.Sheets))
	for i, sht := range meta.Sheets {
		jobs = append(jobs, ChunkJob{
			Index: i,
			Data:  []byte(sht.Properties.Title),
		})
	}

	if err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		name := string(job.Data)
		data, err := s.FetchSheet(ctx, SheetConfig{
			SpreadsheetID: spreadsheetID,
			Range:         "",
			SheetName:     name,
		})
		results[job.Index] = sheetResult{name: name, data: data, err: err}
		return err
	}); err != nil {
		return nil, err
	}

	out := make(map[string][]byte, len(results))
	for _, r := range results {
		out[r.name] = r.data
	}
	return out, nil
}

// ParseSheetIDFromURL extracts the spreadsheet ID from a Google Sheets URL.
// Expected format: https://docs.google.com/spreadsheets/d/{SPREADSHEET_ID}/...
func ParseSheetIDFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("sheets: parse URL: %w", err)
	}

	p := parsed.Path
	for strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}

	parts := strings.Split(p, "/")
	for i, part := range parts {
		if part == "d" && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1], nil
		}
	}

	return "", fmt.Errorf("sheets: could not extract spreadsheet ID from URL %s", rawURL)
}

// DetectConfig fetches spreadsheet metadata and returns a SheetConfig that
// covers the entire first sheet.
func (s *SheetsIngester) DetectConfig(ctx context.Context, spreadsheetID string) (*SheetConfig, error) {
	u := fmt.Sprintf("%s/%s?key=%s",
		sheetsAPIHost,
		url.PathEscape(spreadsheetID),
		url.QueryEscape(s.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("sheets: create detect request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: fetch detect %s: %w", spreadsheetID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sheets: read detect body: %w", err)
	}

	if err := checkSheetsHTTPError(resp.StatusCode, body); err != nil {
		return nil, fmt.Errorf("sheets: detect: %w", err)
	}

	var sMeta struct {
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal(body, &sMeta); err != nil {
		return nil, fmt.Errorf("sheets: parse detect: %w", err)
	}

	if len(sMeta.Sheets) == 0 {
		return nil, fmt.Errorf("sheets: no sheets in %s", spreadsheetID)
	}

	return &SheetConfig{
		SpreadsheetID: spreadsheetID,
		Range:         sMeta.Sheets[0].Properties.Title + "!A:Z",
		SheetName:     sMeta.Sheets[0].Properties.Title,
	}, nil
}

func checkSheetsHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("HTTP 403 (quota exceeded or access denied): %s",
			string(body[:min(len(body), 300)]))
	case http.StatusNotFound:
		return fmt.Errorf("HTTP 404 (spreadsheet not found): %s",
			string(body[:min(len(body), 300)]))
	case http.StatusTooManyRequests:
		return fmt.Errorf("HTTP 429 (rate limit exceeded): %s",
			string(body[:min(len(body), 300)]))
	default:
		if statusCode >= 400 {
			return fmt.Errorf("HTTP %d: %s", statusCode,
				string(body[:min(len(body), 500)]))
		}
		return nil
	}
}
