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

// ─── APIConfig.Validate ─────────────────────────────────────────────────────

func TestAPIConfig_Validate_HappyPath(t *testing.T) {
	cfg := APIConfig{BaseURL: "https://api.example.com", PaginationType: "offset", Limit: 100}
	assert.NoError(t, cfg.Validate())
}

func TestAPIConfig_Validate_EmptyBaseURL(t *testing.T) {
	cfg := APIConfig{BaseURL: ""}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BaseURL is required")
}

func TestAPIConfig_Validate_UnknownPaginationType(t *testing.T) {
	cfg := APIConfig{BaseURL: "https://api.example.com", PaginationType: "weird", Limit: 100}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown PaginationType")
}

func TestAPIConfig_Validate_NoLimitWithPagination(t *testing.T) {
	cfg := APIConfig{BaseURL: "https://api.example.com", PaginationType: "page", Limit: 0}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Limit must")
}

func TestAPIConfig_Validate_NoneTypeNoLimit(t *testing.T) {
	cfg := APIConfig{BaseURL: "https://api.example.com", PaginationType: "none", Limit: 0}
	assert.NoError(t, cfg.Validate())
}

func TestAPIConfig_Validate_EmptyPaginationType(t *testing.T) {
	cfg := APIConfig{BaseURL: "https://api.example.com", PaginationType: ""}
	assert.NoError(t, cfg.Validate())
}

// ─── NewJSONAPIIngester ──────────────────────────────────────────────────────

func TestNewJSONAPIIngester_HappyPath(t *testing.T) {
	j := NewJSONAPIIngester()
	require.NotNil(t, j)
	assert.NotNil(t, j.client)
}

func TestNewJSONAPIIngester_ReturnsInstance(t *testing.T) {
	j1 := NewJSONAPIIngester()
	j2 := NewJSONAPIIngester()
	assert.NotSame(t, j1, j2)
}

func TestNewJSONAPIIngester_ClientNotNil(t *testing.T) {
	j := NewJSONAPIIngester()
	assert.NotNil(t, j.client)
}

// ─── FetchAll ────────────────────────────────────────────────────────────────

func TestFetchAll_HappyPath_NonePagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "none",
		Limit:          100,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestFetchAll_InvalidConfig(t *testing.T) {
	j := NewJSONAPIIngester()
	_, err := j.FetchAll(context.Background(), APIConfig{BaseURL: ""})
	assert.Error(t, err)
}

func TestFetchAll_NestedData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"a": 1}, {"b": 2}], "total": 2}`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "none",
		DataPath:       "data",
		Limit:          100,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestFetchAll_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "none",
		Limit:          100,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(data))
}

func TestFetchAll_PagePagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
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

func TestFetchAll_OffsetPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
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

func TestFetchAll_CursorPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if callCount == 1 {
			w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
		} else {
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	data, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "cursor",
		Limit:          100,
	})
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestFetchAll_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	_, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "none",
		Limit:          100,
	})
	assert.Error(t, err)
}

func TestFetchAll_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	_, err := j.FetchAll(context.Background(), APIConfig{
		BaseURL:        srv.URL,
		PaginationType: "none",
		Limit:          100,
	})
	assert.Error(t, err)
}

// ─── Probe ───────────────────────────────────────────────────────────────────

func TestProbe_HappyPath_ArrayRoot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "name": "test"}]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	result, err := j.Probe(context.Background(), srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "array-root", result.SourceType)
	assert.Equal(t, "", result.DataPath)
	assert.NotNil(t, result.SampleBody)
}

func TestProbe_ObjectWithNested(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": 1}], "total": 100}`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	result, err := j.Probe(context.Background(), srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "object-with-nested", result.SourceType)
	assert.Equal(t, "data", result.DataPath)
}

func TestProbe_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	_, err := j.Probe(context.Background(), srv.URL, nil)
	assert.Error(t, err)
}

func TestProbe_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	result, err := j.Probe(context.Background(), srv.URL, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, srv.URL, result.PageURL)
}

func TestProbe_WithHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	result, err := j.Probe(context.Background(), srv.URL,
		map[string]string{"Authorization": "Bearer token123"},
	)
	require.NoError(t, err)
	assert.Equal(t, "array-root", result.SourceType)
}

func TestProbe_InvalidURL(t *testing.T) {
	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	_, err := j.Probe(context.Background(), "://invalid-url", nil)
	assert.Error(t, err)
}

// ─── DetectConfig ────────────────────────────────────────────────────────────

func TestDetectConfig_HappyPath_Defaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	cfg, err := j.DetectConfig(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, cfg.BaseURL)
	assert.Equal(t, "none", cfg.PaginationType)
	assert.Equal(t, 100, cfg.Limit)
}

func TestDetectConfig_OffsetPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	cfg, err := j.DetectConfig(context.Background(), srv.URL+"?offset=0&limit=50")
	require.NoError(t, err)
	assert.Equal(t, "offset", cfg.PaginationType)
	assert.Equal(t, "offset", cfg.PageParam)
	assert.Equal(t, "limit", cfg.LimitParam)
}

func TestDetectConfig_PagePagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	cfg, err := j.DetectConfig(context.Background(), srv.URL+"?page=1&per_page=100")
	require.NoError(t, err)
	assert.Equal(t, "page", cfg.PaginationType)
}

func TestDetectConfig_HTTPError_ReturnsConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	cfg, err := j.DetectConfig(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, cfg.BaseURL)
}

func TestDetectConfig_InvalidURL(t *testing.T) {
	j := NewJSONAPIIngester()
	j.client = NewTestRateLimitedClient()
	cfg, err := j.DetectConfig(context.Background(), "://invalid")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

// ─── resolveJSONPath ─────────────────────────────────────────────────────────

func TestResolveJSONPath_HappyPath(t *testing.T) {
	data := []byte(`{"data": {"items": [1,2,3], "meta": {"total": 100}}}`)
	raw, err := resolveJSONPath(data, "data.meta.total")
	require.NoError(t, err)
	assert.JSONEq(t, `100`, string(raw))
}

func TestResolveJSONPath_EmptyPath(t *testing.T) {
	data := []byte(`{"key": "value"}`)
	raw, err := resolveJSONPath(data, "")
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"value"}`, string(raw))
}

func TestResolveJSONPath_MissingField(t *testing.T) {
	data := []byte(`{"a": 1}`)
	_, err := resolveJSONPath(data, "a.nonexistent")
	assert.Error(t, err)
}

func TestResolveJSONPath_NonObjectTraversal(t *testing.T) {
	data := []byte(`{"data": {"items": [1,2,3]}}`)
	_, err := resolveJSONPath(data, "data.items.bad")
	assert.Error(t, err)
}

// ─── extractItems ────────────────────────────────────────────────────────────

func TestExtractItems_HappyPath_RootArray(t *testing.T) {
	items, err := extractItems([]byte(`[{"id": 1}, {"id": 2}]`), "")
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestExtractItems_NestedDataPath(t *testing.T) {
	items, err := extractItems([]byte(`{"data": [{"a": 1}, {"b": 2}]}`), "data")
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestExtractItems_MissingPath(t *testing.T) {
	_, err := extractItems([]byte(`{"other": [1,2,3]}`), "data")
	assert.Error(t, err)
}

func TestExtractItems_EmptyArray(t *testing.T) {
	items, err := extractItems([]byte(`[]`), "")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestExtractItems_SingleObject(t *testing.T) {
	items, err := extractItems([]byte(`{"id": 1, "name": "test"}`), "")
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

// ─── extractCursorNext ───────────────────────────────────────────────────────

func TestExtractCursorNext_HappyPath_TopLevel(t *testing.T) {
	next := extractCursorNext([]byte(`{"next": "https://api.example.com/page/2"}`))
	assert.Equal(t, "https://api.example.com/page/2", next)
}

func TestExtractCursorNext_LinksNext(t *testing.T) {
	next := extractCursorNext([]byte(`{"links": {"next": "https://api.example.com/page/2"}}`))
	assert.Equal(t, "https://api.example.com/page/2", next)
}

func TestExtractCursorNext_Empty(t *testing.T) {
	next := extractCursorNext([]byte(`{}`))
	assert.Empty(t, next)
}

func TestExtractCursorNext_InvalidJSON(t *testing.T) {
	next := extractCursorNext([]byte(`not json`))
	assert.Empty(t, next)
}

// ─── classifySourceType (jsonapi.go) ─────────────────────────────────────────

func TestClassifySourceType_ArrayRoot(t *testing.T) {
	got := classifySourceType([]byte(`[{"id": 1}]`), nil)
	assert.Equal(t, "array-root", got)
}

func TestClassifySourceType_ObjectWithNested(t *testing.T) {
	got := classifySourceType([]byte(`{"data": [{"a": 1}]}`), nil)
	assert.Equal(t, "object-with-nested", got)
}

func TestClassifySourceType_EmptyBody(t *testing.T) {
	got := classifySourceType([]byte(``), nil)
	assert.Equal(t, "unknown", got)
}

func TestClassifySourceType_PlainObject(t *testing.T) {
	got := classifySourceType([]byte(`{"meta": {}, "data": "string"}`), nil)
	assert.Equal(t, "unknown", got)
}

// ─── resolveTotal ────────────────────────────────────────────────────────────

func TestResolveTotal_HappyPath(t *testing.T) {
	n, ok := resolveTotal([]byte(`{"total": 42}`), "total")
	assert.True(t, ok)
	assert.Equal(t, 42, n)
}

func TestResolveTotal_EmptyPath(t *testing.T) {
	n, ok := resolveTotal([]byte(`{"total": 100}`), "")
	assert.False(t, ok)
	assert.Equal(t, 0, n)
}

func TestResolveTotal_Missing(t *testing.T) {
	_, ok := resolveTotal([]byte(`{"other": 10}`), "total")
	assert.False(t, ok)
}

func TestResolveTotal_ZeroValue(t *testing.T) {
	n, ok := resolveTotal([]byte(`{"total": 0}`), "total")
	assert.False(t, ok)
	assert.Equal(t, 0, n)
}

func TestResolveTotal_NestedPath(t *testing.T) {
	n, ok := resolveTotal([]byte(`{"meta": {"total": 99}}`), "meta.total")
	assert.True(t, ok)
	assert.Equal(t, 99, n)
}

func TestResolveTotal_StringValue(t *testing.T) {
	n, ok := resolveTotal([]byte(`{"total": "not_number"}`), "total")
	assert.False(t, ok)
	assert.Equal(t, 0, n)
}
