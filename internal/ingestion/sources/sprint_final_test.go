package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// rewriteTransport — rewrites all HTTP requests to target test server
// =============================================================================

type rewriteTransport struct {
	target string // e.g. "http://127.0.0.1:PORT"
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	u.Scheme = "http"
	u.Host = rt.target[len("http://"):]
	req.URL = &u
	req.Host = rt.target[len("http://"):]
	req.RequestURI = "" // Required when host changes
	return http.DefaultTransport.RoundTrip(req)
}

func newSheetsIngesterForTest(t *testing.T, srv *httptest.Server, apiKey string) *SheetsIngester {
	t.Helper()
	si := NewSheetsIngester(apiKey)
	si.UseNonSSRFClient()
	si.client.client.Transport = &rewriteTransport{target: srv.URL}
	return si
}

// =============================================================================
// SheetsIngester.FetchSheet — fully exercised via transport rewriting
// =============================================================================

func TestFetchSheet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"values": [][]string{
				{"name", "score"},
				{"Alice", "95"},
				{"Bob", "87"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	data, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.NoError(t, err)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal(data, &rows))
	assert.Len(t, rows, 2)
	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "95", rows[0]["score"])
}

func TestFetchSheet_HTTP403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"quota exceeded"}`))
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sheets:")
	assert.Contains(t, err.Error(), "403")
}

func TestFetchSheet_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not found`))
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sheets:")
}

func TestFetchSheet_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json at all`))
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestFetchSheet_EmptyValues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{"values": [][]string{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	data, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))
}

func TestFetchSheet_CustomRange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"values": [][]string{{"a"}, {"1"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	data, err := si.FetchSheet(context.Background(), SheetConfig{
		SpreadsheetID: "fake-id",
		Range:         "Custom!A:C",
	})
	require.NoError(t, err)
	require.NotNil(t, data)
}

func TestFetchSheet_EmptyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"values": [][]string{{"x"}, {"0"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "")
	data, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.NoError(t, err)
	require.NotNil(t, data)
}

func TestFetchSheet_SparseRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Header row: 4 cols. Data rows have missing/empty cells.
		resp := map[string]any{
			"values": [][]string{
				{"A", "B", "C", "D"},
				{"1", "", "3", ""},
				{"", "2", "", ""},
				{"x", "y", "z", "w"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	data, err := si.FetchSheet(context.Background(), SheetConfig{SpreadsheetID: "fake-id"})
	require.NoError(t, err)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal(data, &rows))
	assert.Len(t, rows, 3) // all 3 data rows have at least one non-empty cell
}

// =============================================================================
// SheetsIngester.FetchAllSheets — fully exercised
// =============================================================================

func TestFetchAllSheets_SingleSheet(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// metadata
			json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"title": "Sheet1"}},
				},
			})
		} else {
			// sheet values
			json.NewEncoder(w).Encode(map[string]any{
				"values": [][]string{{"h"}, {"v"}},
			})
		}
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	result, err := si.FetchAllSheets(context.Background(), "fake-id")
	require.NoError(t, err)
	assert.Contains(t, result, "Sheet1")
	assert.NotNil(t, result["Sheet1"])
}

func TestFetchAllSheets_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchAllSheets(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata")
}

func TestFetchAllSheets_BadMetadataJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{broken`))
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchAllSheets(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse metadata")
}

func TestFetchAllSheets_NoSheets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sheets": []any{},
		})
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.FetchAllSheets(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sheets found")
}

func TestFetchAllSheets_TwoSheets(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// metadata with 2 sheets
			json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"title": "Data"}},
					{"properties": map[string]any{"title": "Summary"}},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]any{
				"values": [][]string{{"h"}, {"v"}},
			})
		}
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	result, err := si.FetchAllSheets(context.Background(), "fake-id")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "Data")
	assert.Contains(t, result, "Summary")
}

// =============================================================================
// SheetsIngester.DetectConfig — fully exercised
// =============================================================================

func TestDetectConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sheets": []map[string]any{
				{"properties": map[string]any{"title": "MainData"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	cfg, err := si.DetectConfig(context.Background(), "fake-id")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "fake-id", cfg.SpreadsheetID)
	assert.Equal(t, "MainData!A:Z", cfg.Range)
	assert.Equal(t, "MainData", cfg.SheetName)
}

func TestDetectConfig_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.DetectConfig(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detect:")
}

func TestDetectConfig_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.DetectConfig(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse detect")
}

func TestDetectConfig_NoSheets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sheets": []any{},
		})
	}))
	t.Cleanup(srv.Close)

	si := newSheetsIngesterForTest(t, srv, "test-key")
	_, err := si.DetectConfig(context.Background(), "fake-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sheets in")
}

// =============================================================================
// ParseSheetIDFromURL — additional edge cases
// =============================================================================

func TestParseSheetIDFromURL_DoubleSlash(t *testing.T) {
	// URL with double slash after d/
	_, err := ParseSheetIDFromURL("https://docs.google.com/spreadsheets/d//edit")
	require.Error(t, err)
}

func TestParseSheetIDFromURL_CopySuffix(t *testing.T) {
	id, err := ParseSheetIDFromURL("https://docs.google.com/spreadsheets/d/abc123def/copy")
	require.NoError(t, err)
	assert.Equal(t, "abc123def", id)
}

func TestParseSheetIDFromURL_NoHTTPS(t *testing.T) {
	id, err := ParseSheetIDFromURL("http://docs.google.com/spreadsheets/d/xyz789")
	require.NoError(t, err)
	assert.Equal(t, "xyz789", id)
}

// =============================================================================
// checkSheetsHTTPError — edge cases
// =============================================================================

func TestCheckSheetsHTTPError_TooManyRequests(t *testing.T) {
	err := checkSheetsHTTPError(http.StatusTooManyRequests, []byte(`{"error":"rate limit"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestCheckSheetsHTTPError_Generic5xx(t *testing.T) {
	err := checkSheetsHTTPError(http.StatusBadGateway, []byte(`gateway error`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestCheckSheetsHTTPError_NonErrorStatus(t *testing.T) {
	assert.NoError(t, checkSheetsHTTPError(http.StatusOK, nil))
	assert.NoError(t, checkSheetsHTTPError(http.StatusCreated, nil))
	assert.NoError(t, checkSheetsHTTPError(http.StatusNoContent, nil))
}

// =============================================================================
// RateLimitedClient.Get — error path
// =============================================================================

func TestRateLimitedClient_Get_BadURL(t *testing.T) {
	rc := NewTestRateLimitedClient()
	_, err := rc.Get(context.Background(), "://bad-url")
	require.Error(t, err)
}

// =============================================================================
// RateLimitedClient.Do — cancelled context error path
// =============================================================================

func TestRateLimitedClient_Do_CancelledContext(t *testing.T) {
	rc := NewTestRateLimitedClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:9999/nonexistent", nil)
	require.NoError(t, err)

	_, err = rc.Do(req)
	require.Error(t, err)
}
