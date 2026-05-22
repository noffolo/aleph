package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helper: build a ProbeRunner with a custom HTTP server ───────────────────

func newProbeRunnerForTest(t *testing.T) *ProbeRunner {
	t.Helper()
	return NewProbeRunner(nil)
}

// ─── SourceProbeResult.SourceType ────────────────────────────────────────────

func TestSourceProbeResult_SourceType_HappyPath(t *testing.T) {
	r := &SourceProbeResult{SrcType: "rest"}
	assert.Equal(t, "rest", r.SourceType())
}

func TestSourceProbeResult_SourceType_EmptyString(t *testing.T) {
	r := &SourceProbeResult{SrcType: ""}
	assert.Equal(t, "", r.SourceType())
}

func TestSourceProbeResult_SourceType_GitHub(t *testing.T) {
	r := &SourceProbeResult{SrcType: "github"}
	assert.Equal(t, "github", r.SourceType())
}

// ─── SourceProbeResult.Pagination ────────────────────────────────────────────

func TestSourceProbeResult_Pagination_HappyPath(t *testing.T) {
	pi := PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: 50}
	r := &SourceProbeResult{Pag: pi}
	assert.Equal(t, pi, r.Pagination())
}

func TestSourceProbeResult_Pagination_NoneType(t *testing.T) {
	pi := PaginationInfo{Type: "none", MaxLimit: -1}
	r := &SourceProbeResult{Pag: pi}
	assert.Equal(t, pi, r.Pagination())
}

func TestSourceProbeResult_Pagination_ZeroValue(t *testing.T) {
	r := &SourceProbeResult{Pag: PaginationInfo{}}
	assert.Equal(t, PaginationInfo{}, r.Pagination())
}

// ─── SourceProbeResult.Columns ───────────────────────────────────────────────

func TestSourceProbeResult_Columns_HappyPath(t *testing.T) {
	cols := []ColumnInfo{{Name: "id", Type: "number"}}
	r := &SourceProbeResult{Cols: cols}
	assert.Equal(t, cols, r.Columns())
}

func TestSourceProbeResult_Columns_Nil(t *testing.T) {
	r := &SourceProbeResult{Cols: nil}
	assert.Nil(t, r.Columns())
}

func TestSourceProbeResult_Columns_EmptySlice(t *testing.T) {
	r := &SourceProbeResult{Cols: []ColumnInfo{}}
	assert.Empty(t, r.Columns())
}

// ─── SourceProbeResult.Validate ──────────────────────────────────────────────

func TestSourceProbeResult_Validate_HappyPath(t *testing.T) {
	r := &SourceProbeResult{
		URL:     "https://api.example.com",
		SrcType: "rest",
		Pag:     PaginationInfo{MaxLimit: 100},
	}
	assert.NoError(t, r.Validate())
}

func TestSourceProbeResult_Validate_EmptyURL(t *testing.T) {
	r := &SourceProbeResult{
		URL:     "",
		SrcType: "rest",
		Pag:     PaginationInfo{MaxLimit: 100},
	}
	err := r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL must be non-empty")
}

func TestSourceProbeResult_Validate_UnknownSourceType(t *testing.T) {
	r := &SourceProbeResult{
		URL:     "https://api.example.com",
		SrcType: "unknown_type_xyz",
		Pag:     PaginationInfo{MaxLimit: 100},
	}
	err := r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source type")
}

func TestSourceProbeResult_Validate_MaxLimitZero(t *testing.T) {
	r := &SourceProbeResult{
		URL:     "https://api.example.com",
		SrcType: "rest",
		Pag:     PaginationInfo{MaxLimit: 0},
	}
	err := r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MaxLimit")
}

func TestSourceProbeResult_Validate_MaxLimitNegativeOne(t *testing.T) {
	r := &SourceProbeResult{
		URL:     "https://api.example.com",
		SrcType: "rest",
		Pag:     PaginationInfo{MaxLimit: -1},
	}
	assert.NoError(t, r.Validate())
}

func TestSourceProbeResult_Validate_AllValidTypes(t *testing.T) {
	validTypes := []string{"rest", "rss", "github", "sitemap", "generic_json", "web"}
	for _, st := range validTypes {
		r := &SourceProbeResult{
			URL:     "https://api.example.com",
			SrcType: st,
			Pag:     PaginationInfo{MaxLimit: 100},
		}
		assert.NoErrorf(t, r.Validate(), "type %q should be valid", st)
	}
}

// ─── NewProbeRunner ──────────────────────────────────────────────────────────

func TestNewProbeRunner_HappyPath(t *testing.T) {
	pr := NewProbeRunner(nil)
	require.NotNil(t, pr)
	assert.NotNil(t, pr.client)
	assert.Nil(t, pr.llmClient)
}

func TestNewProbeRunner_WithLLMClient(t *testing.T) {
	llm := &stubLLMProber{}
	pr := NewProbeRunner(llm)
	require.NotNil(t, pr)
	assert.NotNil(t, pr.llmClient)
}

func TestNewProbeRunner_NilLLM(t *testing.T) {
	pr := NewProbeRunner(nil)
	require.NotNil(t, pr)
	assert.Nil(t, pr.llmClient)
}

type stubLLMProber struct {
	result *SourceProbeResult
	err    error
}

func (m *stubLLMProber) ProbeEndpoint(_ context.Context, _ string, _ []byte) (*SourceProbeResult, error) {
	return m.result, m.err
}

// ─── ProbeRunner.Probe ───────────────────────────────────────────────────────

func TestProbeRunner_Probe_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "name": "test"}]`))
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()

	result, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, result.URL)
	assert.NotEmpty(t, result.SrcType)
}

func TestProbeRunner_Probe_EmptyEndpoint(t *testing.T) {
	pr := newProbeRunnerForTest(t)
	_, err := pr.Probe(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint must be non-empty")
}

func TestProbeRunner_Probe_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()

	_, err := pr.Probe(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "probe:")
}

func TestProbeRunner_Probe_WithLLMClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	llm := &stubLLMProber{
		result: &SourceProbeResult{
			SrcType:       "rest",
			Cols:          []ColumnInfo{{Name: "id", Type: "number", Path: "$.id"}},
			TotalEstimate: 42,
		},
	}
	pr := NewProbeRunner(llm)
	pr.client = sources.NewTestRateLimitedClient()

	result, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "rest", result.SrcType)
	assert.Len(t, result.Cols, 1)
	assert.Equal(t, int64(42), result.TotalEstimate)
}

func TestProbeRunner_Probe_LLMFailsButProbeSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	llm := &stubLLMProber{err: assert.AnError}
	pr := NewProbeRunner(llm)
	pr.client = sources.NewTestRateLimitedClient()

	result, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, result.URL)
	assert.NotNil(t, result.DataSample)
}

func TestProbeRunner_Probe_InvalidURL(t *testing.T) {
	pr := newProbeRunnerForTest(t)
	_, err := pr.Probe(context.Background(), "://invalid-url")
	require.Error(t, err)
}

// ─── ProbeRunner.Execute ─────────────────────────────────────────────────────

func TestProbeRunner_Execute_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"a": 1}], "page": 1}`))
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()

	result := &SourceProbeResult{
		URL:     srv.URL,
		SrcType: "rest",
		Pag:     PaginationInfo{Type: "none", MaxLimit: -1},
		Cols:    []ColumnInfo{{Name: "a", Type: "number", Path: "$.a"}},
	}
	err := pr.Execute(context.Background(), result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.DataSample)
}

func TestProbeRunner_Execute_NilResult_New(t *testing.T) {
	pr := newProbeRunnerForTest(t)
	err := pr.Execute(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil probe result")
}

func TestProbeRunner_Execute_InvalidResult_New(t *testing.T) {
	pr := newProbeRunnerForTest(t)
	result := &SourceProbeResult{
		URL:     "",
		SrcType: "rest",
		Pag:     PaginationInfo{MaxLimit: 0},
	}
	err := pr.Execute(context.Background(), result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid probe result")
}

// ─── classifySourceType ──────────────────────────────────────────────────────

func TestClassifySourceType_GitHubAPI(t *testing.T) {
	got := classifySourceType("https://api.github.com/org/repo/repos/issues", "", []byte(`[]`))
	assert.Equal(t, "github", got)
}

func TestClassifySourceType_RSSXML(t *testing.T) {
	got := classifySourceType("https://example.com/feed", "application/rss+xml", []byte(`<rss version="2.0"><channel></channel></rss>`))
	assert.Equal(t, "rss", got)
}

func TestClassifySourceType_GenericJSON(t *testing.T) {
	got := classifySourceType("https://example.com/data", "", []byte(`plain text data`))
	assert.Equal(t, "generic_json", got)
}

func TestClassifySourceType_SitemapBody(t *testing.T) {
	got := classifySourceType("https://example.com/sitemap.xml", "text/html",
		[]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
	assert.Equal(t, "sitemap", got)
}

func TestClassifySourceType_JSONArray(t *testing.T) {
	got := classifySourceType("https://example.com/api/items", "", []byte(`[{"id":1}]`))
	assert.Equal(t, "rest", got)
}

func TestClassifySourceType_JSONObject(t *testing.T) {
	got := classifySourceType("https://example.com/api/item", "", []byte(`{"id":1}`))
	assert.Equal(t, "rest", got)
}

func TestClassifySourceType_WebHTML(t *testing.T) {
	got := classifySourceType("https://example.com", "", []byte(`<!DOCTYPE html><html><head></head><body></body></html>`))
	assert.Equal(t, "web", got)
}

func TestClassifySourceType_GenericXML(t *testing.T) {
	got := classifySourceType("https://example.com/config", "",
		[]byte(`<?xml version="1.0"?><config><key>val</key></config>`))
	assert.Equal(t, "generic_json", got)
}

func TestClassifySourceType_SitemapInBody(t *testing.T) {
	got := classifySourceType("https://example.com/", "",
		[]byte(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></sitemapindex>`))
	assert.Equal(t, "sitemap", got)
}

// ─── detectPagination ────────────────────────────────────────────────────────

func TestDetectPagination_LinkHeaderNext(t *testing.T) {
	resp := buildRespHeader("Link", `<https://api.example.com/page=2>; rel="next"`)
	pi := detectPagination("https://api.example.com", resp, []byte(`[]`))
	assert.Equal(t, "cursor", pi.Type)
	assert.Equal(t, "cursor", pi.PageParam)
}

func TestDetectPagination_URLQueryParams(t *testing.T) {
	resp := buildRespHeader()
	pi := detectPagination("https://api.example.com?page=1&per_page=50", resp, []byte(`[]`))
	assert.Equal(t, "offset", pi.Type)
}

func TestDetectPagination_NoPagination(t *testing.T) {
	resp := buildRespHeader()
	pi := detectPagination("https://api.example.com", resp, []byte(`[]`))
	assert.Equal(t, "none", pi.Type)
	assert.Equal(t, -1, pi.MaxLimit)
}

func TestDetectPagination_BodyOffset(t *testing.T) {
	resp := buildRespHeader()
	pi := detectPagination("https://api.example.com", resp,
		[]byte(`{"offset": 0, "limit": 50, "data": [{"a":1}]}`))
	assert.Equal(t, "offset", pi.Type)
}

func TestDetectPagination_BodyPage(t *testing.T) {
	resp := buildRespHeader()
	pi := detectPagination("https://api.example.com", resp,
		[]byte(`{"page": 1, "per_page": 100, "data": []}`))
	assert.Equal(t, "page", pi.Type)
}

func TestDetectPagination_MetaPage(t *testing.T) {
	resp := buildRespHeader()
	pi := detectPagination("https://api.example.com", resp,
		[]byte(`{"meta": {"page": 1, "total": 100}, "data": []}`))
	assert.Equal(t, "page", pi.Type)
}

func buildRespHeader(kv ...string) *http.Response {
	h := http.Header{}
	for i := 0; i+1 < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return &http.Response{Header: h}
}

// ─── detectColumns ───────────────────────────────────────────────────────────

func TestDetectColumns_JSONArray(t *testing.T) {
	cols := detectColumns([]byte(`[{"id": 1, "name": "test", "active": true}]`))
	require.Len(t, cols, 3)
	names := make(map[string]ColumnInfo)
	for _, c := range cols {
		names[c.Name] = c
	}
	idCol := names["id"]
	assert.Equal(t, "number", idCol.Type)
	assert.Equal(t, "$.id", idCol.Path)
	nameCol := names["name"]
	assert.Equal(t, "string", nameCol.Type)
	activeCol := names["active"]
	assert.Equal(t, "boolean", activeCol.Type)
}

func TestDetectColumns_NestedData(t *testing.T) {
	cols := detectColumns([]byte(`{"data": [{"x": 10, "y": "hello"}], "total": 2}`))
	require.Len(t, cols, 2)
}

func TestDetectColumns_EmptyBody(t *testing.T) {
	cols := detectColumns([]byte(``))
	assert.Nil(t, cols)
}

func TestDetectColumns_EmptyArray(t *testing.T) {
	cols := detectColumns([]byte(`[]`))
	assert.Nil(t, cols)
}

func TestDetectColumns_NonJSON(t *testing.T) {
	cols := detectColumns([]byte(`not json`))
	assert.Nil(t, cols)
}

func TestDetectColumns_SingleObject(t *testing.T) {
	cols := detectColumns([]byte(`{"name": "alice", "score": 95}`))
	assert.NotNil(t, cols)
}

// ─── columnsFromMap ──────────────────────────────────────────────────────────

func TestColumnsFromMap_HappyPath(t *testing.T) {
	m := map[string]any{"id": float64(1), "name": "test"}
	cols := columnsFromMap(m, "")
	require.Len(t, cols, 2)
	paths := make(map[string]bool)
	for _, c := range cols {
		paths[c.Path] = true
	}
	assert.True(t, paths["$.id"])
	assert.True(t, paths["$.name"])
}

func TestColumnsFromMap_WithPrefix_New(t *testing.T) {
	m := map[string]any{"value": float64(42)}
	cols := columnsFromMap(m, "data")
	assert.Equal(t, "$.data.value", cols[0].Path)
}

func TestColumnsFromMap_EmptyMap(t *testing.T) {
	cols := columnsFromMap(map[string]any{}, "")
	assert.Empty(t, cols)
}

func TestColumnsFromMap_NestedObject(t *testing.T) {
	m := map[string]any{"nested": map[string]any{"a": 1}}
	cols := columnsFromMap(m, "")
	assert.Equal(t, "object", cols[0].Type)
}

// ─── columnsFromMapSkip ──────────────────────────────────────────────────────

func TestColumnsFromMapSkip_HappyPath(t *testing.T) {
	m := map[string]any{"id": float64(1), "meta": map[string]any{}, "links": map[string]any{}}
	skip := map[string]bool{"meta": true, "links": true}
	cols := columnsFromMapSkip(m, "", skip)
	assert.Len(t, cols, 1)
	assert.Equal(t, "id", cols[0].Name)
}

func TestColumnsFromMapSkip_NoSkipKeys(t *testing.T) {
	m := map[string]any{"a": 1, "b": 2}
	cols := columnsFromMapSkip(m, "", map[string]bool{})
	assert.Len(t, cols, 2)
}

func TestColumnsFromMapSkip_AllSkipped(t *testing.T) {
	m := map[string]any{"meta": map[string]any{}}
	skip := map[string]bool{"meta": true}
	cols := columnsFromMapSkip(m, "", skip)
	assert.Empty(t, cols)
}

// ─── goValueToColumnType ─────────────────────────────────────────────────────

func TestGoValueToColumnType_Number(t *testing.T) {
	assert.Equal(t, "number", goValueToColumnType(float64(42)))
}

func TestGoValueToColumnType_String(t *testing.T) {
	assert.Equal(t, "string", goValueToColumnType("hello"))
}

func TestGoValueToColumnType_Boolean(t *testing.T) {
	assert.Equal(t, "boolean", goValueToColumnType(true))
}

func TestGoValueToColumnType_Object(t *testing.T) {
	assert.Equal(t, "object", goValueToColumnType(map[string]any{}))
}

func TestGoValueToColumnType_Array(t *testing.T) {
	assert.Equal(t, "array", goValueToColumnType([]any{}))
}

func TestGoValueToColumnType_Nil(t *testing.T) {
	assert.Equal(t, "string", goValueToColumnType(nil))
}

func TestGoValueToColumnType_Int(t *testing.T) {
	assert.Equal(t, "string", goValueToColumnType(42))
}

// ─── buildNextURLFn ──────────────────────────────────────────────────────────

func TestBuildNextURLFn_HappyPath_Page(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com?page=1&per_page=100",
		PaginationInfo{Type: "page", PageParam: "page", LimitParam: "per_page", MaxLimit: 100})
	require.NotNil(t, fn)
	next := fn([]byte(`{"page": 1}`))
	assert.Contains(t, next, "page=2")
}

func TestBuildNextURLFn_NoneTypeReturnsNil(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com", PaginationInfo{Type: "none", MaxLimit: -1})
	assert.Nil(t, fn)
}

func TestBuildNextURLFn_EmptyTypeReturnsNil(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com", PaginationInfo{Type: "", MaxLimit: -1})
	assert.Nil(t, fn)
}

func TestBuildNextURLFn_OffsetType(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com?offset=0&limit=50",
		PaginationInfo{Type: "offset", PageParam: "offset", LimitParam: "limit", MaxLimit: 50})
	require.NotNil(t, fn)
	next := fn([]byte(`{"page": 1}`))
	assert.Contains(t, next, "offset=51")
}

// ─── nextCursorURL ───────────────────────────────────────────────────────────

func TestNextCursorURL_HappyPath(t *testing.T) {
	next := nextCursorURL([]byte(`{"next_cursor": "cursor123"}`), "cursor")
	assert.Equal(t, "cursor123", next)
}

func TestNextCursorURL_MetaPagination(t *testing.T) {
	next := nextCursorURL([]byte(`{"meta": {"pagination": {"next": "https://api.example.com/next"}}}`), "cursor")
	assert.Equal(t, "https://api.example.com/next", next)
}

func TestNextCursorURL_NotFound(t *testing.T) {
	next := nextCursorURL([]byte(`{"data": [{"id": 1}]}`), "cursor")
	assert.Empty(t, next)
}

// ─── nextPageURL ─────────────────────────────────────────────────────────────

func TestNextPageURL_HappyPath(t *testing.T) {
	next := nextPageURL([]byte(`{"page": 1}`),
		"https://api.example.com?page=1&per_page=100",
		PaginationInfo{Type: "page", PageParam: "page", LimitParam: "per_page", MaxLimit: 100})
	assert.Contains(t, next, "page=2")
}

func TestNextPageURL_Offset(t *testing.T) {
	next := nextPageURL([]byte(`{"page": 1}`),
		"https://api.example.com?offset=0&limit=50",
		PaginationInfo{Type: "offset", PageParam: "offset", LimitParam: "limit", MaxLimit: 50})
	assert.Contains(t, next, "offset=51")
}

func TestNextPageURL_NoPageInBody(t *testing.T) {
	next := nextPageURL([]byte(`{"data": []}`),
		"https://api.example.com",
		PaginationInfo{Type: "page", PageParam: "page", MaxLimit: 100})
	assert.Empty(t, next)
}

func TestNextPageURL_InvalidJSON(t *testing.T) {
	next := nextPageURL([]byte(`not json`),
		"https://api.example.com",
		PaginationInfo{Type: "page", PageParam: "page", MaxLimit: 100})
	assert.Empty(t, next)
}

// ─── toFloat64 ───────────────────────────────────────────────────────────────

func TestToFloat64_Float64(t *testing.T) {
	f, ok := toFloat64(float64(3.14))
	assert.True(t, ok)
	assert.Equal(t, 3.14, f)
}

func TestToFloat64_Int(t *testing.T) {
	f, ok := toFloat64(42)
	assert.True(t, ok)
	assert.Equal(t, float64(42), f)
}

func TestToFloat64_Int64(t *testing.T) {
	f, ok := toFloat64(int64(99))
	assert.True(t, ok)
	assert.Equal(t, float64(99), f)
}

func TestToFloat64_JSONNumber(t *testing.T) {
	f, ok := toFloat64(json.Number("123.45"))
	assert.True(t, ok)
	assert.Equal(t, 123.45, f)
}

func TestToFloat64_String(t *testing.T) {
	_, ok := toFloat64("not a number")
	assert.False(t, ok)
}

func TestToFloat64_Nil(t *testing.T) {
	_, ok := toFloat64(nil)
	assert.False(t, ok)
}

// ─── Integration: ProbeRunner.Probe with classification ──────────────────────

func TestProbeRunner_Probe_DetectsRestType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "name": "hello"}]`))
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()
	result, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "rest", result.SrcType)
}

func TestProbeRunner_Probe_SitemapXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()
	result, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "sitemap", result.SrcType)
}

func TestProbeRunner_Execute_PagePagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if callCount == 1 {
			w.Write([]byte(`{"data": [{"x": 1}], "page": 1}`))
		} else if callCount == 2 {
			w.Write([]byte(`{"data": [{"x": 2}], "page": 2}`))
		} else {
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()

	result := &SourceProbeResult{
		URL:     srv.URL,
		SrcType: "rest",
		Pag:     PaginationInfo{Type: "page", PageParam: "page", LimitParam: "per_page", MaxLimit: 100},
		Cols:    []ColumnInfo{{Name: "x", Type: "number", Path: "$.x"}},
	}
	err := pr.Execute(context.Background(), result)
	require.NoError(t, err)
	assert.Greater(t, len(result.DataSample), 0)
}
