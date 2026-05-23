package ingestion

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTransport struct {
	err error
}

func (s *stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, s.err
}

// ===== runPrecompiled non-200 =====

func TestRunPrecompiled_HTTP404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rp1")
	eng.httpClient = srv.Client()
	assert.Error(t, eng.runPrecompiled(context.Background(), os.Stdout, "rp1", &v1.IngestionTask{Id: "r1", SourceType: "rss", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runURLFetch non-200 =====

func TestRunURLFetch_HTTP500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rf1")
	eng.httpClient = srv.Client()
	assert.Error(t, eng.runURLFetch(context.Background(), os.Stdout, "rf1", &v1.IngestionTask{Id: "f1", SourceType: "url", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runURLFetch CSV fallback =====

func TestRunURLFetch_CSVFallback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rc1")
	eng.httpClient = srv.Client()
	require.NoError(t, eng.runURLFetch(context.Background(), os.Stdout, "rc1", &v1.IngestionTask{Id: "c1", SourceType: "url", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runCSVLoad Name field =====

func TestRunCSVLoad_NameField(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rn1")
	p := filepath.Join(t.TempDir(), "n.csv")
	require.NoError(t, os.WriteFile(p, []byte("a,b\n1,2\n"), 0644))
	task := &v1.IngestionTask{Id: "idf", Name: "nm_fld", SourceType: "csv", ConfigJson: `{"path":"` + p + `"}`}
	lp := filepath.Join(pr, "rn1", "logs", "idf.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	require.NoError(t, eng.runCSVLoad(context.Background(), f, "rn1", task))
}

// ===== registerViews =====

func TestRegisterViews_NoOnt(t *testing.T) {
	eng, root := setupEngineFull(t)
	proj := "rv1"
	createDirs(t, root, proj)
	assert.NoError(t, eng.registerViews(context.Background(), proj))
}

func TestRegisterViews_Valid(t *testing.T) {
	eng, root := setupEngineFull(t)
	proj := "rv2"
	createDirs(t, root, proj)
	os.WriteFile(filepath.Join(root, proj, "raw", "rv2.csv"), []byte("c1,c2\nv1,v2\n"), 0644)
	os.WriteFile(filepath.Join(root, proj, "ontologies", "core.aleph"), []byte("object rv2 from dataset rv2 id c1\n"), 0644)
	assert.NoError(t, eng.registerViews(context.Background(), proj))
}

func TestRegisterViews_ParseErr(t *testing.T) {
	eng, root := setupEngineFull(t)
	proj := "rv3"
	createDirs(t, root, proj)
	os.WriteFile(filepath.Join(root, proj, "ontologies", "core.aleph"), []byte("!!!@@@\n"), 0644)
	assert.Error(t, eng.registerViews(context.Background(), proj))
}

func TestRegisterViews_ViewFail(t *testing.T) {
	eng, root := setupEngineFull(t)
	proj := "rv4"
	createDirs(t, root, proj)
	os.WriteFile(filepath.Join(root, proj, "raw", "rv4.csv"), []byte("c1,c2\nv1,v2\n"), 0644)
	os.WriteFile(filepath.Join(root, proj, "ontologies", "core.aleph"), []byte("object rv4 from dataset rv4 id c1\n"), 0644)
	eng.db.Exec(context.Background(), `CREATE TABLE "rv4_rv4" (dummy INTEGER)`)
	assert.NoError(t, eng.registerViews(context.Background(), proj))
}

// ===== classifySourceType =====

func TestClassifySourceType_PlainHTML(t *testing.T) {
	assert.Equal(t, "web", classifySourceType("https://x.com", "text/plain", []byte("<html><body>H</body></html>")))
}
func TestClassifySourceType_PlainRand(t *testing.T) {
	assert.Equal(t, "generic_json", classifySourceType("https://x.com", "text/plain", []byte("random")))
}
func TestClassifySourceType_XMLSite(t *testing.T) {
	b := []byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://x.com</loc></url></urlset>`)
	assert.Equal(t, "sitemap", classifySourceType("https://x.com", "text/plain", b))
}
func TestClassifySourceType_XMLRSS(t *testing.T) {
	b := []byte(`<?xml version="1.0"?><rss version="2.0"><channel><item><title>T</title></item></channel></rss>`)
	assert.Equal(t, "rss", classifySourceType("https://x.com", "text/plain", b))
}
func TestClassifySourceType_GenXML(t *testing.T) {
	assert.Equal(t, "generic_json", classifySourceType("https://x.com", "text/plain", []byte(`<?xml version="1.0"?><root><d>v</d></root>`)))
}
func TestClassifySourceType_JSArr(t *testing.T) {
	assert.Equal(t, "rest", classifySourceType("https://x.com", "text/plain", []byte(`[{"a":1}]`)))
}
func TestClassifySourceType_JSObj(t *testing.T) {
	assert.Equal(t, "rest", classifySourceType("https://x.com", "text/plain", []byte(`{"k":"v"}`)))
}
func TestClassifySourceType_CtXMLSite(t *testing.T) {
	b := []byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://x.com</loc></url></urlset>`)
	assert.Equal(t, "sitemap", classifySourceType("https://x.com", "application/xml", b))
}
func TestClassifySourceType_CtXMLRSS(t *testing.T) {
	b := []byte(`<?xml version="1.0"?><rss version="2.0"><channel><item><title>X</title></item></channel></rss>`)
	assert.Equal(t, "rss", classifySourceType("https://x.com", "text/xml", b))
}

// ===== detectPagination =====

func TestDetectPagination_PerPgCov(t *testing.T) {
	r := &http.Response{Header: http.Header{}}
	assert.Equal(t, "per_page", detectPagination("https://x.com?page=1&per_page=50", r, nil).LimitParam)
}
func TestDetectPagination_CntCov(t *testing.T) {
	r := &http.Response{Header: http.Header{}}
	assert.Equal(t, "count", detectPagination("https://x.com?page=1&count=25", r, nil).LimitParam)
}
func TestDetectPagination_OffMaxCov(t *testing.T) {
	r := &http.Response{Header: http.Header{}}
	assert.Equal(t, 50, detectPagination("https://x.com?offset=0", r, nil).MaxLimit)
}
func TestDetectPagination_MetaPagCov(t *testing.T) {
	r := &http.Response{Header: http.Header{}}
	g := detectPagination("https://x.com", r, []byte(`{"meta":{"page":1}}`))
	assert.Equal(t, "page", g.Type)
	assert.Equal(t, "per_page", g.LimitParam)
}
func TestDetectPagination_JSOffCov(t *testing.T) {
	r := &http.Response{Header: http.Header{}}
	g := detectPagination("https://x.com", r, []byte(`{"offset":0}`))
	assert.Equal(t, "offset", g.Type)
}
func TestDetectPagination_LinkCov(t *testing.T) {
	r := &http.Response{Header: http.Header{"Link": []string{`<https://x.com?c=a>; rel="next"`}}}
	assert.Equal(t, "cursor", detectPagination("https://x.com", r, nil).Type)
}

// ===== buildNextURLFn =====

func TestBuildNextURLFn_PgCov(t *testing.T) {
	fn := buildNextURLFn("https://x.com?page=1", PaginationInfo{Type: "page", PageParam: "page", MaxLimit: 100})
	require.NotNil(t, fn)
	assert.Empty(t, fn([]byte(`{}`)))
	assert.Contains(t, fn([]byte(`{"page":5}`)), "page=6")
}
func TestBuildNextURLFn_OffCov(t *testing.T) {
	fn := buildNextURLFn("https://x.com?offset=0", PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: 100})
	require.NotNil(t, fn)
	assert.Empty(t, fn([]byte(`{}`)))
	assert.Contains(t, fn([]byte(`{"meta":{"offset":100}}`)), "offset=200")
}
func TestBuildNextURLFn_CurCov(t *testing.T) {
	fn := buildNextURLFn("https://x.com", PaginationInfo{Type: "cursor", PageParam: "cursor"})
	require.NotNil(t, fn)
	assert.Equal(t, "cur_n", fn([]byte(`{"next_cursor":"cur_n"}`)))
}
func TestBuildNextURLFn_NoneCov(t *testing.T) {
	assert.Nil(t, buildNextURLFn("https://x.com", PaginationInfo{Type: "none"}))
}
func TestBuildNextURLFn_EmptyCov(t *testing.T) {
	assert.Nil(t, buildNextURLFn("https://x.com", PaginationInfo{}))
}
func TestBuildNextURLFn_UnkCov(t *testing.T) {
	fn := buildNextURLFn("https://x.com", PaginationInfo{Type: "xyz"})
	require.NotNil(t, fn)
	assert.Empty(t, fn(nil))
}

// ===== nextPageURL =====

func TestNextPageURL_JSONNumCov(t *testing.T) {
	assert.Contains(t, nextPageURL([]byte(`{"page":3}`), "https://x.com?page=3", PaginationInfo{Type: "page", PageParam: "page", MaxLimit: 100}), "page=4")
}
func TestNextPageURL_MetaPgCov(t *testing.T) {
	assert.Contains(t, nextPageURL([]byte(`{"meta":{"page":7}}`), "https://x.com?page=7", PaginationInfo{Type: "page", PageParam: "page", MaxLimit: 100}), "page=8")
}
func TestNextPageURL_MetaOffCov(t *testing.T) {
	assert.Contains(t, nextPageURL([]byte(`{"meta":{"offset":50}}`), "https://x.com?offset=50", PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: 50}), "offset=100")
}
func TestNextPageURL_ZeroPgCov(t *testing.T) {
	assert.Empty(t, nextPageURL([]byte(`{"page":0}`), "https://x.com", PaginationInfo{Type: "page"}))
}
func TestNextPageURL_BadJCov(t *testing.T) {
	assert.Empty(t, nextPageURL([]byte(`{bad`), "https://x.com", PaginationInfo{Type: "page"}))
}
func TestNextPageURL_BadUCov(t *testing.T) {
	assert.Empty(t, nextPageURL([]byte(`{"page":3}`), "://bad", PaginationInfo{Type: "page"}))
}
func TestNextPageURL_NegLimCov(t *testing.T) {
	assert.Contains(t, nextPageURL([]byte(`{"meta":{"offset":75}}`), "https://x.com?offset=75", PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: -1}), "offset=125")
}

// ===== nextCursorURL =====

func TestNextCursorURL_MetaNestCov(t *testing.T) {
	assert.Equal(t, "abc", nextCursorURL([]byte(`{"meta":{"pagination":{"next":"abc"}}}`), "x"))
}
func TestNextCursorURL_MetaNCCov(t *testing.T) {
	assert.Equal(t, "xyz", nextCursorURL([]byte(`{"meta":{"next_cursor":"xyz"}}`), "x"))
}
func TestNextCursorURL_TLCursCov(t *testing.T) {
	assert.Equal(t, "abc", nextCursorURL([]byte(`{"cursor":"abc"}`), "x"))
}
func TestNextCursorURL_TLNextCov(t *testing.T) {
	assert.Equal(t, "https://p2", nextCursorURL([]byte(`{"next":"https://p2"}`), "x"))
}
func TestNextCursorURL_TokCov(t *testing.T) {
	assert.Empty(t, nextCursorURL([]byte(`{"paging":{"cursors":{"after":"x"}}}`), "x"))
}
func TestNextCursorURL_EmpValCov(t *testing.T) {
	assert.Empty(t, nextCursorURL([]byte(`{"next_cursor":""}`), "x"))
}
func TestNextCursorURL_InvJCov(t *testing.T) {
	assert.Empty(t, nextCursorURL([]byte(`{bad`), "x"))
}

// ===== toFloat64 =====

func TestToFloat64_JNumCov(t *testing.T) {
	v, ok := toFloat64(json.Number("42.5"))
	assert.True(t, ok)
	assert.Equal(t, 42.5, v)
}
func TestToFloat64_I64Cov(t *testing.T) {
	v, ok := toFloat64(int64(100))
	assert.True(t, ok)
	assert.Equal(t, float64(100), v)
}
func TestToFloat64_IntCov(t *testing.T) {
	v, ok := toFloat64(42)
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}
func TestToFloat64_StrCov(t *testing.T) {
	_, ok := toFloat64("nope")
	assert.False(t, ok)
}

// ===== runPostgresLoad =====

func TestRunPostgresLoad_ExtInst(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rp1")
	task := &v1.IngestionTask{Id: "p1", SourceType: "postgres", ConfigJson: `{"dsn":"host=127.0.0.1 port=17432 dbname=none"}`}
	lp := filepath.Join(pr, "rp1", "logs", "p1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runPostgresLoad(context.Background(), f, "rp1", task))
}
func TestRunPostgresLoad_NoDsnCov(t *testing.T) {
	eng := NewEngine("/tmp", nil, nil, nil)
	assert.Error(t, eng.runPostgresLoad(context.Background(), os.Stdout, "p", &v1.IngestionTask{Id: "pd", SourceType: "postgres", ConfigJson: `{"dsn":""}`}))
}

// ===== resolveTableName =====

func TestResolveTableName_NmFldCov(t *testing.T) {
	n, err := resolveTableName(&v1.IngestionTask{Id: "t", Name: "my-data", ConfigJson: `{}`})
	require.NoError(t, err)
	assert.Equal(t, "my_data", n)
}
func TestResolveTableName_OnlyIDCov(t *testing.T) {
	n, err := resolveTableName(&v1.IngestionTask{Id: "task_id", ConfigJson: `{}`})
	require.NoError(t, err)
	assert.Equal(t, "task_id", n)
}
func TestResolveTableName_UIDCov(t *testing.T) {
	n, err := resolveTableName(&v1.IngestionTask{Id: "550e8400-e29b-41d4-a716-446655440000", ConfigJson: `{}`})
	require.NoError(t, err)
	assert.Contains(t, n, "task_")
}
func TestResolveTableName_WrdIDCov(t *testing.T) {
	n, err := resolveTableName(&v1.IngestionTask{Id: "my data!!!", ConfigJson: `{}`})
	require.NoError(t, err)
	assert.Equal(t, "my_data___", n)
}
func TestResolveTableName_CfgTNCov(t *testing.T) {
	n, err := resolveTableName(&v1.IngestionTask{Id: "a", ConfigJson: `{"tableName":"cfg"}`})
	require.NoError(t, err)
	assert.Equal(t, "cfg", n)
}

// ===== extractArray =====

func TestExtractArray_NoArrCov(t *testing.T) {
	_, ok := extractArray(map[string]any{"k": "v"})
	assert.False(t, ok)
}
func TestExtractArray_HasArrCov(t *testing.T) {
	a, ok := extractArray(map[string]any{"r": []any{map[string]any{"a": 1}}})
	assert.True(t, ok)
	assert.Len(t, a, 1)
}

// ===== probe validate =====

func TestProbeResult_ValUnkCov(t *testing.T) {
	assert.Error(t, (&SourceProbeResult{URL: "https://x.com", SrcType: "xyz"}).Validate())
}
func TestProbeResult_ValNoURLCov(t *testing.T) {
	assert.Error(t, (&SourceProbeResult{SrcType: "rest"}).Validate())
}

// ===== RunTask sourceType branches =====

func TestRunTask_RestST(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rs1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`[{"id":1}]`))
	}))
	t.Cleanup(srv.Close)
	eng.httpClient = srv.Client()
	require.NoError(t, eng.RunTask(context.Background(), "rs1", &v1.IngestionTask{Id: "rs1", SourceType: "rest", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}
func TestRunTask_ShSTCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "sh1")
	assert.Error(t, eng.RunTask(context.Background(), "sh1", &v1.IngestionTask{Id: "sh1", SourceType: "sheets", ConfigJson: `{"spreadsheetId":"x","range":"S"}`}))
}
func TestRunTask_GhSTCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "gh1")
	assert.Error(t, eng.RunTask(context.Background(), "gh1", &v1.IngestionTask{Id: "gh1", SourceType: "github", ConfigJson: `{"repo":"o/r"}`}))
}
func TestRunTask_EmSTCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "em1")
	assert.Error(t, eng.RunTask(context.Background(), "em1", &v1.IngestionTask{Id: "em1", SourceType: "email", ConfigJson: `{"server":"imap.x.com","username":"u","password":"p"}`}))
}
func TestRunTask_CcSTCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "cc1")
	assert.Error(t, eng.RunTask(context.Background(), "cc1", &v1.IngestionTask{Id: "cc1", SourceType: "custom_code", ConfigJson: `{"code":"package main\nfunc main(){println(1)}"}`}))
}
func TestRunTask_CpSTCov(t *testing.T) {
	root := t.TempDir()
	eng, _ := setupTestEngine(t)
	eng.projectsRoot = root
	createTestProject(t, root, "cs")
	createTestProject(t, root, "cd")
	os.WriteFile(filepath.Join(root, "cs", "raw", "x.csv"), []byte("a,b\n1,2\n"), 0644)
	require.NoError(t, eng.RunTask(context.Background(), "cd", &v1.IngestionTask{Id: "cp1", SourceType: "copy", ConfigJson: `{"source":"cs"}`}))
}

// ===== insertJSONArray =====

func TestInsertJSONArray_NonObjCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	eng.projectsRoot = pr
	assert.Error(t, eng.insertJSONArray(context.Background(), "no1", []any{"x"}, os.Stdout))
}
func TestInsertJSONArray_NilValCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	eng.projectsRoot = pr
	require.NoError(t, eng.insertJSONArray(context.Background(), "nv1", []any{map[string]any{"c": nil}}, os.Stdout))
}
func TestInsertJSONArray_BtCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	eng.projectsRoot = pr
	a := make([]any, 501)
	for i := range a {
		a[i] = map[string]any{"id": float64(i)}
	}
	require.NoError(t, eng.insertJSONArray(context.Background(), "bt1", a, os.Stdout))
}
func TestInsertJSONArray_MixCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	eng.projectsRoot = pr
	require.NoError(t, eng.insertJSONArray(context.Background(), "mx1", []any{map[string]any{"a": 1}, "x", map[string]any{"a": 2}}, os.Stdout))
}

// ===== runCSVLoad =====

func TestRunCSVLoad_ParqNmCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "pn1")
	p := filepath.Join(t.TempDir(), "m.parquet")
	require.NoError(t, os.WriteFile(p, []byte("PAR1"), 0644))
	task := &v1.IngestionTask{Id: "pid", Name: "pname", SourceType: "csv", ConfigJson: `{"path":"` + p + `"}`}
	lp := filepath.Join(pr, "pn1", "logs", "pid.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runCSVLoad(context.Background(), f, "pn1", task))
}
func TestRunCSVLoad_NoPthCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "ep1")
	task := &v1.IngestionTask{Id: "ep1", SourceType: "csv", ConfigJson: `{"path":""}`}
	lp := filepath.Join(pr, "ep1", "logs", "ep1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runCSVLoad(context.Background(), f, "ep1", task))
}
func TestRunCSVLoad_FRErrCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "fr1")
	task := &v1.IngestionTask{Id: "fr1", SourceType: "csv", ConfigJson: `{"path":"/nonexistent/x.csv"}`}
	lp := filepath.Join(pr, "fr1", "logs", "fr1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runCSVLoad(context.Background(), f, "fr1", task))
}

// ===== invalid URL =====

func TestRunURLFetch_BadURLCov(t *testing.T) {
	err := NewEngine("/tmp", nil, nil, nil).runURLFetch(context.Background(), os.Stdout, "p", &v1.IngestionTask{Id: "bu1", SourceType: "url", ConfigJson: `{"url":":/x"}`})
	assert.Error(t, err)
}
func TestRunPrecompiled_BadURLCov(t *testing.T) {
	err := NewEngine("/tmp", nil, nil, nil).runPrecompiled(context.Background(), os.Stdout, "p", &v1.IngestionTask{Id: "bp1", SourceType: "rss", ConfigJson: `{"url":":/x"}`})
	assert.Error(t, err)
}

// ===== column detection =====

func TestDetectColumns_EmptyBCov(t *testing.T) { assert.Nil(t, detectColumns([]byte{})) }
func TestDetectColumns_WhiteCov(t *testing.T) { assert.Nil(t, detectColumns([]byte("  \n\t "))) }
func TestDetectColumns_JSArrCov(t *testing.T) {
	assert.Len(t, detectColumns([]byte(`[{"n":"A","a":30}]`)), 2)
}
func TestColumnsFromMap_BscCov(t *testing.T) {
	assert.Len(t, columnsFromMap(map[string]any{"n": "A", "a": float64(30)}, ""), 2)
}
func TestColumnsFromMapSkip_UndCov(t *testing.T) {
	sk := map[string]bool{"_i": true, "_p": true}
	assert.Len(t, columnsFromMapSkip(map[string]any{"_i": 1, "_p": 2, "v": 3}, "", sk), 1)
}
func TestGoValueToColumnType_ObjCov(t *testing.T) {
	assert.Equal(t, "object", goValueToColumnType(map[string]any{}))
}
func TestGoValueToColumnType_ArrCov(t *testing.T) {
	assert.Equal(t, "array", goValueToColumnType([]any{}))
}
func TestGoValueToColumnType_NilCov(t *testing.T) {
	assert.Equal(t, "string", goValueToColumnType(nil))
}

// ===== runURLFetch JSON detect =====

func TestRunURLFetch_JSONDet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte(`{"k":"v"}`))
	}))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "jd1")
	eng.httpClient = srv.Client()
	require.NoError(t, eng.runURLFetch(context.Background(), os.Stdout, "jd1", &v1.IngestionTask{Id: "jd1", SourceType: "url", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runCopy =====

func TestRunCopy_NoSrcCov(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "cns1")
	task := &v1.IngestionTask{Id: "cn1", SourceType: "copy", ConfigJson: `{"source":"nope"}`}
	lp := filepath.Join(root, "cns1", "logs", "cn1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runCopy(context.Background(), f, "cns1", task))
}
func TestRunCopy_RealCov(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "cs1")
	createDirs(t, root, "cd1")
	os.WriteFile(filepath.Join(root, "cs1", "raw", "t.csv"), []byte("a,b\n1,2\n"), 0644)
	task := &v1.IngestionTask{Id: "cr1", SourceType: "copy", ConfigJson: `{"source":"cs1"}`}
	lp := filepath.Join(root, "cd1", "logs", "cr1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	require.NoError(t, eng.runCopy(context.Background(), f, "cd1", task))
}
func TestRunCopy_JSONCov(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "cjs")
	createDirs(t, root, "cjd")
	os.WriteFile(filepath.Join(root, "cjs", "raw", "t.json"), []byte(`{"a":1}`), 0644)
	task := &v1.IngestionTask{Id: "cj1", SourceType: "copy", ConfigJson: `{"source":"cjs"}`}
	lp := filepath.Join(root, "cjd", "logs", "cj1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	require.NoError(t, eng.runCopy(context.Background(), f, "cjd", task))
}
func TestRunCopy_BadCfgCov(t *testing.T) {
	eng := NewEngine("/tmp", nil, nil, nil)
	assert.Error(t, eng.runCopy(context.Background(), os.Stdout, "p", &v1.IngestionTask{Id: "c", SourceType: "copy", ConfigJson: `{bad`}))
}

// ===== client fallback =====

func TestClient_FallbackCov(t *testing.T) {
	c := NewEngine("/tmp", nil, nil, nil).client()
	assert.NotNil(t, c)
	assert.Equal(t, safeHTTPClient, c)
}

// ===== sanitize extra =====

func TestSanitizeID_UpperCov(t *testing.T) { assert.NoError(t, sanitizeIdentifier("TBL")) }
func TestSanitizeID_UnderCov(t *testing.T) { assert.NoError(t, sanitizeIdentifier("_t")) }

// ===== runPrecompiled unknown CT =====

func TestRunPrecompiled_UnkCTCov(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		w.Write([]byte("binary"))
	}))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "uc1")
	eng.httpClient = srv.Client()
	require.NoError(t, eng.runPrecompiled(context.Background(), os.Stdout, "uc1", &v1.IngestionTask{Id: "uc1", SourceType: "rss", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runDynamic =====

func TestRunDynamic_InvGoCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "di1")
	task := &v1.IngestionTask{Id: "di1", SourceType: "custom_code", ConfigJson: `{"code":"not go"}`}
	lp := filepath.Join(pr, "di1", "logs", "di1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runDynamic(context.Background(), f, "di1", task))
}
func TestRunDynamic_NoneCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "dn1")
	task := &v1.IngestionTask{Id: "dn1", SourceType: "custom_code", ConfigJson: `{"code":""}`}
	lp := filepath.Join(pr, "dn1", "logs", "dn1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runDynamic(context.Background(), f, "dn1", task))
}

// ===== runJSONAPI / runSitemap / runEmail =====

func TestRunJSONAPI_NoAutoCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "jn1")
	assert.Error(t, eng.runJSONAPISource(context.Background(), os.Stdout, "jn1", &v1.IngestionTask{Id: "jn1", SourceType: "jsonapi", ConfigJson: `{"url":"https://x.com","autoDetect":false}`}))
}
func TestRunSitemap_NonSiteCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "sn1")
	assert.Error(t, eng.runSitemapSource(context.Background(), os.Stdout, "sn1", &v1.IngestionTask{Id: "sn1", SourceType: "sitemap", ConfigJson: `{"url":"https://x.com"}`}))
}
func TestRunEmailFetch_NoCredCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "mc1")
	task := &v1.IngestionTask{Id: "mc1", SourceType: "email", ConfigJson: `{"host":"","user":"","pass":""}`}
	lp := filepath.Join(pr, "mc1", "logs", "mc1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runEmailFetch(context.Background(), f, "mc1", task))
}
func TestRunEmailFetch_DefFldCov(t *testing.T) {
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "df1")
	task := &v1.IngestionTask{Id: "df1", SourceType: "email", ConfigJson: `{"host":"imap.t","user":"t","pass":"t"}`}
	lp := filepath.Join(pr, "df1", "logs", "df1.log")
	os.MkdirAll(filepath.Dir(lp), 0755)
	f, _ := os.Create(lp)
	defer f.Close()
	assert.Error(t, eng.runEmailFetch(context.Background(), f, "df1", task))
}

// ===== runURLFetch transport error =====

func TestRunURLFetch_TransErrCov(t *testing.T) {
	t.Parallel()
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "rt1")
	eng.httpClient = &http.Client{Transport: &stubTransport{err: fmt.Errorf("dns err")}}
	assert.Error(t, eng.runURLFetch(context.Background(), os.Stdout, "rt1", &v1.IngestionTask{Id: "t1", SourceType: "url", ConfigJson: `{"url":"http://x.com"}`}))
}

// ===== decodeMIMEHeader, getOrCreate* =====

func TestDecodeMIMEHeader_EmptyCov(t *testing.T) { assert.Empty(t, decodeMIMEHeader("")) }
func TestDecodeMIMEHeader_DecErrCov(t *testing.T) {
	assert.NotEmpty(t, decodeMIMEHeader("=?utf-8?b?!!!bad!!!?="))
}
func TestGetOrCreateGitHub_BadCfgCov(t *testing.T) {
	assert.NotNil(t, NewEngine("/tmp", nil, nil, nil).getOrCreateGitHubIngester(&v1.IngestionTask{Id: "gh", ConfigJson: `{bad`}))
}
func TestGetOrCreateSheets_BadCfgCov(t *testing.T) {
	assert.NotNil(t, NewEngine("/tmp", nil, nil, nil).getOrCreateSheetsIngester(&v1.IngestionTask{Id: "sh", ConfigJson: `{bad`}))
}

// ===== fetchIMAP, escapeIMAP =====

func TestFetchIMAP_SSRFBlockCov(t *testing.T) {
	_, err := fetchIMAP("localhost:9993", "u", "p", "INBOX", 1)
	assert.Error(t, err)
}
func TestEscapeIMAP_NoChange(t *testing.T) { assert.Equal(t, "INBOX", escapeIMAP("INBOX")) }
func TestEscapeIMAP_Quote(t *testing.T) { assert.Equal(t, `F\"older`, escapeIMAP(`F"older`)) }
func TestContainsColon_Yes(t *testing.T) { assert.True(t, containsColon("x:y")) }
func TestContainsColon_No(t *testing.T) { assert.False(t, containsColon("xy")) }

// ===== readIMAPResponse =====

func TestReadIMAP_ResponseOK(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("A01 OK done\r\n"))
	result, err := readIMAPResponse(r, "A01")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
func TestReadIMAP_ResponseBAD(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("A01 BAD error\r\n"))
	_, err := readIMAPResponse(r, "A01")
	assert.Error(t, err)
}

// ===== parseIMAPSearch =====

func TestParseIMAPSearch_Some(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3}, parseIMAPSearch("* SEARCH 1 2 3\r\nA01 OK\r\n"))
}
func TestParseIMAPSearch_None(t *testing.T) {
	assert.Empty(t, parseIMAPSearch("* SEARCH\r\nA01 OK\r\n"))
}

// ===== parseIMAPFetchMessages =====

func TestParseIMAPFetch_MultiCov(t *testing.T) {
	input := "* 1 FETCH (BODY[] {5})\nFrom: a@b.com\nSubject: S1\n\nbody1\n* 2 FETCH (BODY[] {5})\nFrom: c@d.com\nSubject: S2\n\nbody2\n"
	rows, err := parseIMAPFetchMessages(input)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

// ===== parseRFC822 =====

func TestParseRFC822_ValidCov(t *testing.T) {
	r, err := parseRFC822("From: x@x.com\r\nSubject: S\r\n\r\nbody")
	require.NoError(t, err)
	assert.Equal(t, "x@x.com", r.From)
}
func TestParseRFC822_BadCov(t *testing.T) {
	_, err := parseRFC822("not email")
	assert.Error(t, err)
}

// ===== extractTextPart =====

func TestExtractTextPart_Plain(t *testing.T) {
	r := strings.NewReader("--b\r\nContent-Type: text/plain\r\n\r\nhi\r\n--b--\r\n")
	assert.Contains(t, extractTextPart(r, "b"), "hi")
}

// ===== decodeBody =====

func TestDecodeBody_B64(t *testing.T) {
	assert.Contains(t, decodeBody([]byte("aGVsbG8="), "base64"), "hello")
}
func TestDecodeBody_QP(t *testing.T) {
	assert.Contains(t, decodeBody([]byte("hello=20world"), "quoted-printable"), "hello")
}
func TestDecodeBody_7bit(t *testing.T) {
	assert.Contains(t, decodeBody([]byte("plain"), "7bit"), "plain")
}

// ===== validateSQLName / stripAndValidateName =====

func TestValidateSQLName_Ok(t *testing.T) { assert.NoError(t, validateSQLName("tbl")) }
func TestValidateSQLName_Bad(t *testing.T) { assert.Error(t, validateSQLName("123tbl")) }
func TestStripAndValidateName_Ok(t *testing.T) {
	n, err := stripAndValidateName("my-data")
	require.NoError(t, err)
	assert.Equal(t, "my_data", n)
}
func TestStripAndValidateName_Bad(t *testing.T) {
	_, err := stripAndValidateName("123")
	assert.Error(t, err)
}

// ===== resolveTableName invalid Name =====

func TestResolveTableName_BadNmCov(t *testing.T) {
	_, err := resolveTableName(&v1.IngestionTask{Id: "t", Name: "123", ConfigJson: `{}`})
	assert.Error(t, err)
}

// ===== runURLFetch invalid task ID =====

func TestRunURLFetch_InvIDCov(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`hello`))
	}))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "ri1")
	eng.httpClient = srv.Client()
	assert.Error(t, eng.runURLFetch(context.Background(), os.Stdout, "ri1", &v1.IngestionTask{Id: "123", SourceType: "url", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}

// ===== runPrecompiled invalid task ID =====

func TestRunPrecompiled_InvIDCov(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	eng, pr := setupTestEngine(t)
	createTestProject(t, pr, "pi1")
	eng.httpClient = srv.Client()
	assert.Error(t, eng.runPrecompiled(context.Background(), os.Stdout, "pi1", &v1.IngestionTask{Id: "123", SourceType: "rss", ConfigJson: `{"url":"` + srv.URL + `"}`}))
}
