package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRateLimitedClient() *RateLimitedClient {
	c := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	c.client = &http.Client{}
	return c
}

func TestParseLinkHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header string
		want   map[string]string
	}{
		{"empty", "", map[string]string{}},
		{"single_next", `<https://api.github.com/repos/owner/repo/issues?page=2>; rel="next"`,
			map[string]string{"next": "https://api.github.com/repos/owner/repo/issues?page=2"}},
		{"next_and_last", `<https://api.example.com/page=2>; rel="next", <https://api.example.com/page=5>; rel="last"`,
			map[string]string{"next": "https://api.example.com/page=2", "last": "https://api.example.com/page=5"}},
		{"with_extra_spaces", ` <https://example.com/2> ; rel="next" `,
			map[string]string{"next": "https://example.com/2"}},
		{"malformed_no_url", `rel="next"`, map[string]string{}},
		{"malformed_no_rel", `<https://example.com>`, map[string]string{}},
		{"only_prev", `<https://example.com/1>; rel="prev"`,
			map[string]string{"prev": "https://example.com/1"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseLinkHeader(tt.header)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitHubIngester_DefaultHeaders(t *testing.T) {
	t.Parallel()
	t.Run("with_token", func(t *testing.T) {
		g := NewGitHubIngester("ghp_test_token")
		h := g.defaultHeaders()
		assert.Equal(t, "application/vnd.github.v3+json", h["Accept"])
		assert.Equal(t, "Bearer ghp_test_token", h["Authorization"])
	})
	t.Run("without_token", func(t *testing.T) {
		g := NewGitHubIngester("")
		h := g.defaultHeaders()
		assert.Equal(t, "application/vnd.github.v3+json", h["Accept"])
		_, hasAuth := h["Authorization"]
		assert.False(t, hasAuth, "Authorization header should be absent with no token")
	})
}

func TestAPIConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     APIConfig
		wantErr bool
	}{
		{"valid_offset", APIConfig{BaseURL: "https://api.example.com", PaginationType: "offset", Limit: 100}, false},
		{"valid_page", APIConfig{BaseURL: "https://api.example.com", PaginationType: "page", Limit: 50}, false},
		{"valid_cursor", APIConfig{BaseURL: "https://api.example.com", PaginationType: "cursor", Limit: 100}, false},
		{"valid_none", APIConfig{BaseURL: "https://api.example.com", PaginationType: "none"}, false},
		{"valid_empty_type", APIConfig{BaseURL: "https://api.example.com", PaginationType: ""}, false},
		{"empty_url", APIConfig{BaseURL: ""}, true},
		{"unknown_type", APIConfig{BaseURL: "https://api.example.com", PaginationType: "weird", Limit: 100}, true},
		{"offset_no_limit", APIConfig{BaseURL: "https://api.example.com", PaginationType: "offset", Limit: 0}, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveJSONPath(t *testing.T) {
	t.Parallel()
	data := []byte(`{"data": {"items": [1,2,3], "meta": {"total": 100}}}`)
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"empty_path", "", `{"data": {"items": [1,2,3], "meta": {"total": 100}}}`, false},
		{"top_level", "data", `{"items":[1,2,3],"meta":{"total":100}}`, false},
		{"nested", "data.items", `[1,2,3]`, false},
		{"meta_total", "data.meta.total", `100`, false},
		{"missing_field", "data.nope", "", true},
		{"non_object_traversal", "data.items.bad", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveJSONPath(data, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, string(got))
			}
		})
	}
}

func TestExtractItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		dataPath string
		wantLen  int
		wantErr  bool
	}{
		{"root_array", `[{"id": 1}, {"id": 2}]`, "", 2, false},
		{"nested_data", `{"data": [{"a": 1}, {"b": 2}]}`, "data", 2, false},
		{"missing_path", `{"other": [1,2,3]}`, "data", 0, true},
		{"empty_array", `[]`, "", 0, false},
		{"single_object_wrapped", `{"id": 1, "name": "test"}`, "", 1, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items, err := extractItems([]byte(tt.body), tt.dataPath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, items, tt.wantLen)
			}
		})
	}
}

func TestExtractCursorNext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{"top_level_next", `{"next": "https://api.example.com/page/2"}`, "https://api.example.com/page/2"},
		{"links_next", `{"links": {"next": "https://api.example.com/page/2"}}`, "https://api.example.com/page/2"},
		{"data_next", `{"data": {"next": "https://api.example.com/page/2"}}`, "https://api.example.com/page/2"},
		{"meta_next", `{"meta": {"next": "https://api.example.com/page/2"}}`, "https://api.example.com/page/2"},
		{"empty", `{}`, ""},
		{"empty_next_string", `{"next": ""}`, ""},
		{"invalid_json", `not json`, ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCursorNext([]byte(tt.body))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifySourceType_JSONAPI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{"root_array", `[{"id": 1}]`, "array-root"},
		{"nested_items", `{"items": [{"a": 1}]}`, "object-with-nested"},
		{"nested_data", `{"data": [{"b": 2}]}`, "object-with-nested"},
		{"nested_results", `{"results": [{"c": 3}]}`, "object-with-nested"},
		{"nested_records", `{"records": [{"d": 4}]}`, "object-with-nested"},
		{"nested_values", `{"values": [{"e": 5}]}`, "object-with-nested"},
		{"empty_object", `{}`, "unknown"},
		{"empty_body", ``, "unknown"},
		{"plain_object_no_array", `{"meta": {}, "data": "string"}`, "unknown"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifySourceType([]byte(tt.body), nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveTotal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		body      string
		totalPath string
		want      int
		wantOK    bool
	}{
		{"empty_path", `{"total": 100}`, "", 0, false},
		{"top_level", `{"total": 42}`, "total", 42, true},
		{"nested", `{"meta": {"total": 99}}`, "meta.total", 99, true},
		{"missing", `{"other": 10}`, "total", 0, false},
		{"zero_value", `{"total": 0}`, "total", 0, false},
		{"string_value", `{"total": "not_number"}`, "total", 0, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, ok := resolveTotal([]byte(tt.body), tt.totalPath)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestIsXMLContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mediaType string
		want      bool
	}{
		{"application_xml", "application/xml", true},
		{"text_xml", "text/xml", true},
		{"atom_xml", "application/atom+xml", true},
		{"rss_xml", "application/rss+xml", true},
		{"text_html", "text/html", false},
		{"application_json", "application/json", false},
		{"text_plain", "text/plain", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isXMLContentType(tt.mediaType))
		})
	}
}

func TestIsTextContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mediaType string
		want      bool
	}{
		{"text_html", "text/html", true},
		{"text_plain", "text/plain", true},
		{"application_json", "application/json", true},
		{"application_xml", "application/xml", false},
		{"image_png", "image/png", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isTextContentType(tt.mediaType))
		})
	}
}

func TestFollowRedirects(t *testing.T) {
	client := followRedirects(3)
	require.NotNil(t, client, "followRedirects should return a client")
	assert.NotNil(t, client.CheckRedirect, "CheckRedirect should be set")
	assert.Equal(t, 30*time.Second, client.Timeout, "timeout should be 30s")
}

func TestCheckSheetsHTTPError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantErr    bool
	}{
		{"ok", http.StatusOK, []byte(`{"values": []}`), false},
		{"forbidden", http.StatusForbidden, []byte(`{"error": "access denied"}`), true},
		{"not_found", http.StatusNotFound, []byte(`{"error": "not found"}`), true},
		{"too_many", http.StatusTooManyRequests, []byte(`{"error": "rate limit"}`), true},
		{"server_error", http.StatusInternalServerError, []byte(`{"error": "internal"}`), true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkSheetsHTTPError(tt.statusCode, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseSheetIDFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantID  string
		wantErr bool
	}{
		{"standard", "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz/edit", "1AbCdEfGhIjKlMnOpQrStUvWxYz", false},
		{"with_trailing_slash", "https://docs.google.com/spreadsheets/d/12345/", "12345", false},
		{"with_gid", "https://docs.google.com/spreadsheets/d/abc123/edit#gid=0", "abc123", false},
		{"not_sheets_url", "https://example.com/data", "", true},
		{"invalid_url", "://invalid", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ParseSheetIDFromURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestNewGitHubIngester(t *testing.T) {
	t.Parallel()
	g := NewGitHubIngester("test-token")
	require.NotNil(t, g)
	assert.NotNil(t, g.client)
	assert.Equal(t, "test-token", g.token)
}

func TestNewJSONAPIIngester(t *testing.T) {
	t.Parallel()
	j := NewJSONAPIIngester()
	require.NotNil(t, j)
	assert.NotNil(t, j.client)
}

func TestNewSheetsIngester(t *testing.T) {
	t.Parallel()
	s := NewSheetsIngester("test-api-key")
	require.NotNil(t, s)
	assert.NotNil(t, s.client)
	assert.Equal(t, "test-api-key", s.apiKey)
}

func TestRateLimitedClient_Do(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimitedClient_Get(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "ok"}`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	resp, err := client.Get(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimitedClient_Get_Error(t *testing.T) {
	t.Parallel()
	client := newTestRateLimitedClient()
	_, err := client.Get(t.Context(), "://invalid-url")
	assert.Error(t, err)
}

func TestFetchPages_SinglePage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	var pages [][]byte
	err := FetchPages(t.Context(), client, server.URL, nil,
		func(body []byte) string { return "" },
		func(body []byte) error {
			pages = append(pages, body)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.JSONEq(t, `[{"id": 1}]`, string(pages[0]))
}

func TestFetchPages_HTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	err := FetchPages(t.Context(), client, server.URL, nil, nil,
		func(body []byte) error { return nil },
	)
	assert.Error(t, err)
}

func TestFetchPages_ConsumeError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	err := FetchPages(t.Context(), client, server.URL, nil,
		func(body []byte) string { return "" },
		func(body []byte) error {
			return assert.AnError
		},
	)
	assert.Error(t, err)
}

func TestFetchPages_WithHeaders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	err := FetchPages(t.Context(), client, server.URL,
		map[string]string{"X-Custom": "value"},
		func(body []byte) string { return "" },
		func(body []byte) error { return nil },
	)
	require.NoError(t, err)
}

func TestFetchPages_InvalidURL(t *testing.T) {
	t.Parallel()
	client := newTestRateLimitedClient()
	err := FetchPages(t.Context(), client, "://invalid", nil, nil,
		func(body []byte) error { return nil },
	)
	assert.Error(t, err)
}

func TestNewWorkerPool_ZeroConfig(t *testing.T) {
	t.Parallel()
	pool := NewWorkerPool(ChunkConfig{Workers: 0})
	require.NotNil(t, pool)
	assert.Equal(t, DefaultChunkConfig.Workers, pool.config.Workers)
}

func TestJSONAPIIngester_Probe_ArrayRoot(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "name": "test"}]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	result, err := ingester.Probe(t.Context(), server.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "array-root", result.SourceType)
	assert.Equal(t, "", result.DataPath)
	assert.NotNil(t, result.SampleBody)
}

func TestJSONAPIIngester_Probe_ObjectWithNested(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": 1}], "total": 100}`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	result, err := ingester.Probe(t.Context(), server.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "object-with-nested", result.SourceType)
	assert.Equal(t, "data", result.DataPath)
}

func TestJSONAPIIngester_Probe_HTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	_, err := ingester.Probe(t.Context(), server.URL, nil)
	assert.Error(t, err)
}

func TestJSONAPIIngester_Probe_EmptyBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	result, err := ingester.Probe(t.Context(), server.URL, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestJSONAPIIngester_DetectConfig_ArrayRoot(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	cfg, err := ingester.DetectConfig(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, server.URL, cfg.BaseURL)
	assert.Equal(t, "", cfg.DataPath)
}

func TestJSONAPIIngester_DetectConfig_NestedData(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": 1}], "meta": {}}`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	cfg, err := ingester.DetectConfig(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, "data", cfg.DataPath)
}

func TestJSONAPIIngester_DetectConfig_OffsetPagination(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	cfg, err := ingester.DetectConfig(t.Context(), server.URL+"?offset=0&limit=50")
	require.NoError(t, err)
	assert.Equal(t, "offset", cfg.PaginationType)
}

func TestJSONAPIIngester_DetectConfig_PagePagination(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	cfg, err := ingester.DetectConfig(t.Context(), server.URL+"?page=1&per_page=100")
	require.NoError(t, err)
	assert.Equal(t, "page", cfg.PaginationType)
}

func TestJSONAPIIngester_DetectConfig_HTTPError_ReturnsConfig(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	cfg, err := ingester.DetectConfig(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, server.URL, cfg.BaseURL)
}

func TestGitHubIngester_FetchPaginated(t *testing.T) {
	t.Parallel()
	callCount := 0
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Header().Set("Link", `<`+serverURL+`?page=2>; rel="next"`)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{{"id": 1, "title": "issue 1"}})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{{"id": 2, "title": "issue 2"}})
		}
	}))
	defer server.Close()
	serverURL = server.URL

	g := NewGitHubIngester("")
	g.client = newTestRateLimitedClient()
	data, err := g.fetchPaginated(t.Context(), server.URL)
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
	assert.Equal(t, 2, callCount)
}

func TestGitHubIngester_FetchIssues(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{{"title": "Test Issue"}})
	}))
	defer server.Close()

	g := &GitHubIngester{client: newTestRateLimitedClient()}
	data, err := g.fetchPaginated(t.Context(), server.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestWorkerPool_Run_WithRetries(t *testing.T) {
	t.Parallel()
	attempts := make([]int, 3)
	pool := NewWorkerPool(ChunkConfig{
		Workers:    1,
		BatchSize:  10,
		MaxRetries: 2,
		RetryDelay: 1,
	})

	jobs := []ChunkJob{
		{Index: 0, Data: []byte("a")},
		{Index: 1, Data: []byte("b")},
		{Index: 2, Data: []byte("c")},
	}

	ctx := t.Context()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		attempts[job.Index]++
		if attempts[job.Index] <= 1 {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []int{2, 2, 2}, attempts)
}

func TestJSONAPIIngester_FetchAll_None(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	data, err := ingester.FetchAll(t.Context(), APIConfig{
		BaseURL:        server.URL,
		PaginationType: "none",
		Limit:          100,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestJSONAPIIngester_FetchAll_InvalidConfig(t *testing.T) {
	t.Parallel()
	ingester := NewJSONAPIIngester()
	_, err := ingester.FetchAll(t.Context(), APIConfig{BaseURL: ""})
	assert.Error(t, err)
}

func TestWorkerPool_Run_CtxCancellation(t *testing.T) {
	t.Parallel()
	pool := NewWorkerPool(ChunkConfig{
		Workers:    2,
		BatchSize:  10,
		MaxRetries: 0,
		RetryDelay: 1,
	})

	jobs := []ChunkJob{
		{Index: 0, Data: []byte("x")},
		{Index: 1, Data: []byte("y")},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_ = pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		return nil
	})
}

func TestJSONAPIIngester_FetchAll_Page(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch callCount {
		case 1:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 1}]`))
		case 2:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 2}]`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	data, err := ingester.FetchAll(t.Context(), APIConfig{
		BaseURL:        server.URL,
		PaginationType: "page",
		PageParam:      "page",
		LimitParam:     "per_page",
		Limit:          100,
		MaxPages:       2,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestJSONAPIIngester_FetchAll_Offset(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch callCount {
		case 1:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": "a"}]`))
		case 2:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": "b"}]`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	data, err := ingester.FetchAll(t.Context(), APIConfig{
		BaseURL:        server.URL,
		PaginationType: "offset",
		PageParam:      "offset",
		LimitParam:     "limit",
		Limit:          50,
		MaxPages:       2,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestSheetsIngester_Construction(t *testing.T) {
	t.Parallel()
	s := NewSheetsIngester("test-api-key")
	s.client = newTestRateLimitedClient()
	require.NotNil(t, s.client)
	assert.Equal(t, "test-api-key", s.apiKey)
}

func TestGitHubIngester_FetchAll_Real(t *testing.T) {
	t.Parallel()
	g := NewGitHubIngester("")
	g.client = newTestRateLimitedClient()
	_, err := g.FetchAll(t.Context(), "owner", "repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP")
}

func TestJSONAPIIngester_FetchAll_Cursor(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer server.Close()

	ingester := NewJSONAPIIngester()
	ingester.client = newTestRateLimitedClient()
	data, err := ingester.FetchAll(t.Context(), APIConfig{
		BaseURL:        server.URL,
		PaginationType: "cursor",
		Limit:          100,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 1)
}

func TestWorkerPool_Run_ZeroWorkersDefault(t *testing.T) {
	t.Parallel()
	pool := NewWorkerPool(ChunkConfig{Workers: 0, BatchSize: 10, MaxRetries: 0})
	jobs := []ChunkJob{{Index: 0, Data: []byte("x")}}
	ctx := t.Context()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestFetchPages_MultiPage(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	client := newTestRateLimitedClient()
	var pages [][]byte
	nextCalled := false
	err := FetchPages(t.Context(), client, server.URL, nil,
		func(body []byte) string {
			if !nextCalled {
				nextCalled = true
				return server.URL
			}
			return ""
		},
		func(body []byte) error {
			pages = append(pages, body)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Len(t, pages, 2)
}

func TestRateLimitedClient_Do_RateLimit(t *testing.T) {
	t.Parallel()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 0, Burst: 0})
	require.NotNil(t, client)
	assert.NotNil(t, client.limiter)
	// Zero RPS should still construct a valid client with zero limiter
	assert.NotNil(t, client.client)
}

func TestGitHubIngester_FetchAll_Concurrent_Cleanup(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var receivedURLs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedURLs = append(receivedURLs, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	g := &GitHubIngester{client: newTestRateLimitedClient()}
	data, err := g.fetchPaginated(t.Context(), server.URL+"/issues?state=all&per_page=100")
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Empty(t, results)
}
func TestSitemapCrawl_URLSet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/page1</loc></url></urlset>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, result.SitemapURL)
	assert.Len(t, result.URLs, 1)
}

func TestSitemapCrawl_SitemapIndex(t *testing.T) {
	t.Parallel()
	childSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/page1</loc></url></urlset>`))
	}))
	defer childSrv.Close()

	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>` + childSrv.URL + `</loc></sitemap></sitemapindex>`))
	}))
	defer indexSrv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), indexSrv.URL)
	require.NoError(t, err)
	assert.Len(t, result.URLs, 1)
}

func TestSitemapCrawl_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	_, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.Error(t, err)
}

func TestSitemapCrawl_InvalidXML(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not xml at all`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	_, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.Error(t, err)
}

func TestNewRateLimitedClient_ZeroBurst(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RequestsPerSecond: 5, Burst: 0}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
	assert.NotNil(t, c.limiter)
}

func TestDetectRootElement_Error(t *testing.T) {
	t.Parallel()
	_, err := detectRootElement([]byte(``))
	assert.Error(t, err)

	name, err := detectRootElement([]byte(`<root></root>`))
	require.NoError(t, err)
	assert.Equal(t, "root", name)
}

func TestFollowRedirects_MaxRedirects(t *testing.T) {
	client := followRedirects(1)
	assert.NotNil(t, client)
	assert.NotNil(t, client.CheckRedirect)
}

func TestGitHubIngester_FetchPRs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"number":1}]`))
	}))
	defer srv.Close()

	g := &GitHubIngester{client: newTestRateLimitedClient()}
	data, err := g.fetchPaginated(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGitHubIngester_FetchCommits(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"sha":"abc"}]`))
	}))
	defer srv.Close()

	g := &GitHubIngester{client: newTestRateLimitedClient()}
	data, err := g.fetchPaginated(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestFetchPages_ChainedPages(t *testing.T) {
	t.Parallel()
	pageNum := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageNum++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"p":` + string(rune('0'+pageNum)) + `}]`))
	}))
	defer srv.Close()

	client := newTestRateLimitedClient()
	var pages [][]byte
	nextCalls := 0
	err := FetchPages(t.Context(), client, srv.URL, nil,
		func(body []byte) string {
			nextCalls++
			if nextCalls < 3 {
				return srv.URL
			}
			return ""
		},
		func(body []byte) error {
			pages = append(pages, body)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Len(t, pages, 3)
}

func TestRateLimitedClient_Do_WaitError(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RequestsPerSecond: 0.01, Burst: 1}
	c := NewRateLimitedClient(cfg)
	c.client = &http.Client{}
	c.limiter.Wait(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := c.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
}

func TestExtractItems_WrapperObj(t *testing.T) {
	t.Parallel()
	items, err := extractItems([]byte(`{"data":[{"a":1}]}`), "data")
	require.NoError(t, err)
	assert.Len(t, items, 1)

	items2, err2 := extractItems([]byte(`{"d":[{"b":2}]}`), "d")
	require.NoError(t, err2)
	assert.Len(t, items2, 1)
}

func TestSheetsIngester_AllConstructors(t *testing.T) {
	t.Parallel()
	s := NewSheetsIngester("key")
	assert.NotNil(t, s)
	assert.NotNil(t, s.client)
	assert.Equal(t, "key", s.apiKey)

	cfg := SheetConfig{SpreadsheetID: "123", Range: "A:Z", SheetName: "Sheet1"}
	assert.Equal(t, "123", cfg.SpreadsheetID)
	assert.Equal(t, "A:Z", cfg.Range)
}

func TestResolveURL_Error(t *testing.T) {
	t.Parallel()
	_, err := resolveURL("://invalid", "/path")
	assert.Error(t, err)
	_, err = resolveURL("/path", "://invalid")
	assert.Error(t, err)
}

func TestFollowRedirects_Zero(t *testing.T) {
	client := followRedirects(0)
	assert.NotNil(t, client)
	assert.NotNil(t, client.CheckRedirect)
}

func TestNewRateLimitedClient_DefaultBurst(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: -1}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
	assert.NotNil(t, c.limiter)

	cfg2 := RateLimitConfig{RequestsPerSecond: 0.5, Burst: 0}
	c2 := NewRateLimitedClient(cfg2)
	require.NotNil(t, c2)
}

func TestSitemapCrawl_InvalidSitemapRoot(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><randomroot></randomroot>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	_, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.Error(t, err)
}

func TestSitemapCrawl_EmptyLoc(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc></loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestFetchXML_BadContentType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not xml`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	_, err := s.fetchXML(t.Context(), srv.URL)
	assert.Error(t, err)
}

func TestFetchXML_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<root/>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	body, err := s.fetchXML(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestFetchXML_NoContentType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<root/>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	body, err := s.fetchXML(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestSitemapCrawl_URLSetWithURLs(t *testing.T) {
	t.Parallel()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>` + srv.URL + `/page1</loc></url><url><loc>` + srv.URL + `/page2</loc></url></urlset>`))
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html>page</html>`))
		}
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), srv.URL+"/sitemap.xml")
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/sitemap.xml", result.SitemapURL)
}

func TestSitemapCrawl_BadChildURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>://bad-url</loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestSitemapCrawl_BadChildSitemap(t *testing.T) {
	t.Parallel()
	childSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer childSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>` + childSrv.URL + `</loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = newTestRateLimitedClient()
	result, err := s.CrawlSitemap(t.Context(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestDetectRootElement_EndElement(t *testing.T) {
	t.Parallel()
	_, err := detectRootElement([]byte(`</root>`))
	assert.Error(t, err)
}

func TestDetectRootElement_ProcInst(t *testing.T) {
	t.Parallel()
	name, err := detectRootElement([]byte(`<?xml version="1.0"?><test/>`))
	require.NoError(t, err)
	assert.Equal(t, "test", name)
}
