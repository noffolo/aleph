package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubNLP struct {
	score float32
	label string
	err   error
}

func (s *stubNLP) AnalyzeSentiment(ctx context.Context, text string) (float32, string, error) {
	return s.score, s.label, s.err
}

func setupEngineFull(t *testing.T) (*Engine, string) {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	tmpDir := t.TempDir()
	projRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projRoot, 0755))
	return NewEngine(projRoot, nil, db, nil), projRoot
}

func createDirs(t *testing.T, root, projID string) {
	t.Helper()
	for _, d := range []string{"raw", "ontologies", "agents", "skills", "logs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, projID, d), 0755))
	}
}

func mj(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestRunTask_Success(t *testing.T) {
	t.Parallel()
	eng, r := setupEngineFull(t)
	createDirs(t, r, "succ")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"name":"alice"}]`))
	}))
	defer srv.Close()
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "succ", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(context.Background(), "succ", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT id, name FROM "succ" ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var count int
	for rows.Next() {
		var id int
		var name string
		require.NoError(t, rows.Scan(&id, &name))
		assert.Equal(t, 1, id)
		assert.Equal(t, "alice", name)
		count++
	}
	assert.Equal(t, 1, count)
}

func TestRunTask_JSONAPI(t *testing.T) {
	t.Parallel()
	eng, r := setupEngineFull(t)
	createDirs(t, r, "japi")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"title":"hello"}]`))
	}))
	defer srv.Close()
	eng.httpClient = srv.Client()
	eng.jsonapiIngester = sources.NewJSONAPIIngester()
	eng.jsonapiIngester.UseNonSSRFClient()
	task := &v1.IngestionTask{Id: "jtask", SourceType: "jsonapi", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(context.Background(), "japi", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT id, title FROM "jtask" ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var count int
	for rows.Next() {
		var id int
		var title string
		require.NoError(t, rows.Scan(&id, &title))
		assert.Equal(t, 1, id)
		assert.Equal(t, "hello", title)
		count++
	}
	assert.Equal(t, 1, count)
}

func TestRunTask_GitHub(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)

	task := &v1.IngestionTask{Id: "gtask", SourceType: "github", ConfigJson: mj(map[string]string{"owner": "t", "repo": "t", "token": "tok"})}

	g1 := eng.getOrCreateGitHubIngester(task)
	require.NotNil(t, g1)

	g2 := eng.getOrCreateGitHubIngester(task)
	assert.Same(t, g1, g2, "getOrCreateGitHubIngester must return the same instance")

	g1.UseNonSSRFClient()
	require.NotNil(t, g1)
}

func TestRunTask_Sitemap(t *testing.T) {
	t.Parallel()
	eng, r := setupEngineFull(t)
	createDirs(t, r, "sm")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://e.com/</loc></url></urlset>`))
	}))
	defer srv.Close()
	eng.httpClient = srv.Client()
	eng.sitemapIngester = sources.NewSitemapIngester()
	eng.sitemapIngester.UseNonSSRFClient()
	task := &v1.IngestionTask{Id: "stask", SourceType: "sitemap", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(context.Background(), "sm", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT "url" FROM "stask" ORDER BY "url"`)
	require.NoError(t, err)
	defer rows.Close()
	var urls []string
	for rows.Next() {
		var url string
		require.NoError(t, rows.Scan(&url))
		urls = append(urls, url)
	}
	assert.Equal(t, []string{"https://e.com/"}, urls)
}

func TestRunTask_Sheets(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "sh")
	task := &v1.IngestionTask{Id: "htask", SourceType: "sheets", ConfigJson: mj(map[string]string{"spreadsheet_id": "x", "api_key": "k"})}
	err := eng.RunTask(context.Background(), "sh", task)
	assert.Error(t, err)
}

func TestProbeRunner_Probes(t *testing.T) {
	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()

	t.Run("json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id":1}]`))
		}))
		defer srv.Close()
		r, err := pr.Probe(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "rest", r.SourceType())
	})
	t.Run("rss", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<rss version="2.0"><channel></channel></rss>`))
		}))
		defer srv.Close()
		r, err := pr.Probe(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "rss", r.SourceType())
	})
	t.Run("html", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body>h</body></html>`))
		}))
		defer srv.Close()
		r, err := pr.Probe(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "web", r.SourceType())
	})
	_, err := pr.Probe(context.Background(), "")
	assert.Error(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err = pr.Probe(context.Background(), srv.URL)
	assert.Error(t, err)
}

type stubLLM struct {
	result *SourceProbeResult
	err    error
}

func (s *stubLLM) ProbeEndpoint(ctx context.Context, ep string, data []byte) (*SourceProbeResult, error) {
	return s.result, s.err
}

func TestProbeRunner_LLM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1}]`))
	}))
	defer srv.Close()

	llm := &stubLLM{result: &SourceProbeResult{SrcType: "rest", Cols: []ColumnInfo{{Name: "x", Type: "string", Path: "$.x"}}, TotalEstimate: 500}}
	pr := NewProbeRunner(llm)
	pr.client = sources.NewTestRateLimitedClient()
	r, err := pr.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "x", r.Cols[0].Name)

	pr2 := NewProbeRunner(&stubLLM{err: assert.AnError})
	pr2.client = sources.NewTestRateLimitedClient()
	r, err = pr2.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "rest", r.SourceType())

	pr3 := NewProbeRunner(&stubLLM{result: nil})
	pr3.client = sources.NewTestRateLimitedClient()
	r, err = pr3.Probe(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "rest", r.SourceType())
}

func TestProbeRunner_ExecuteFull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1}]`))
	}))
	defer srv.Close()
	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()
	result := &SourceProbeResult{SrcType: "rest", URL: srv.URL, Pag: PaginationInfo{Type: "none", MaxLimit: -1}}
	err := pr.Execute(context.Background(), result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.DataSample)

	pr2 := NewProbeRunner(nil)
	err = pr2.Execute(context.Background(), &SourceProbeResult{SrcType: "bad", URL: "http://x.com", Pag: PaginationInfo{MaxLimit: 0}})
	assert.Error(t, err)
}

func TestNLPAnalyzer(t *testing.T) {
	s := &stubNLP{score: 0.75, label: "pos"}
	sc, lb, err := s.AnalyzeSentiment(context.Background(), "text")
	assert.NoError(t, err)
	assert.Equal(t, float32(0.75), sc)
	assert.Equal(t, "pos", lb)

	s2 := &stubNLP{err: assert.AnError}
	_, _, err = s2.AnalyzeSentiment(context.Background(), "text")
	assert.Error(t, err)
}

func TestSourceProbeResult(t *testing.T) {
	s := &SourceProbeResult{SrcType: "rest", Pag: PaginationInfo{Type: "none", MaxLimit: -1}, Cols: []ColumnInfo{{Name: "id", Type: "number", Path: "$.id"}}}
	assert.Equal(t, "rest", s.SourceType())
	assert.Equal(t, "none", s.Pagination().Type)
	assert.Equal(t, "id", s.Columns()[0].Name)
}

func TestEnrichPredictiveMetadata(t *testing.T) {
	t.Run("no_ontology", func(t *testing.T) {
		eng, _ := setupEngineFull(t)
		eng.db.Exec(context.Background(), `CREATE TABLE "rich" (id INTEGER, text_col VARCHAR)`)
		eng.db.Exec(context.Background(), `INSERT INTO "rich" VALUES (1, 'long text for NLP analysis here')`)
		eng.nlpHandler = &stubNLP{score: 0.5, label: "neutral"}
		eng.enrichPredictiveMetadata(context.Background(), "p", "rich")
	})
	t.Run("nlp_error", func(t *testing.T) {
		eng, _ := setupEngineFull(t)
		eng.db.Exec(context.Background(), `CREATE TABLE "err_tbl" (id INTEGER, content VARCHAR)`)
		eng.db.Exec(context.Background(), `INSERT INTO "err_tbl" VALUES (1, 'long enough text here')`)
		eng.nlpHandler = &stubNLP{err: assert.AnError}
		eng.enrichPredictiveMetadata(context.Background(), "p", "err_tbl")
	})
}

func TestDetectPagination(t *testing.T) {
	tests := []struct {
		name, endpoint, linkHdr, body string
		want                          string
	}{
		{"cursor_link", "https://x.com", `<https://x.com?p=2>; rel="next"`, `[]`, "cursor"},
		{"page_param", "https://x.com?page=1", "", `[]`, "offset"},
		{"offset_param", "https://x.com?offset=0", "", `[]`, "offset"},
		{"body_page", "https://x.com", "", `{"page":1}`, "page"},
		{"meta_page", "https://x.com", "", `{"meta":{"page":1}}`, "page"},
		{"none", "https://x.com", "", `[]`, "none"},
		{"empty", "https://x.com", "", ``, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.linkHdr != "" {
				resp = &http.Response{Header: http.Header{"Link": []string{tt.linkHdr}}}
			} else {
				resp = &http.Response{Header: http.Header{}}
			}
			pag := detectPagination(tt.endpoint, resp, []byte(tt.body))
			assert.Equal(t, tt.want, pag.Type)
		})
	}
}

func TestRunPrecompiled_BadURL(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "bad")
	task := &v1.IngestionTask{Id: "badu", SourceType: "rss", ConfigJson: mj(map[string]string{"url": "://bad"})}
	logPath := filepath.Join(r, "bad", "logs", "badu.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runPrecompiled(context.Background(), f, "bad", task)
	assert.Error(t, err)
}

func TestRunURLFetch_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "ub")
	task := &v1.IngestionTask{Id: "ubcf", SourceType: "url", ConfigJson: `not-json`}
	eng.httpClient = &http.Client{}
	logPath := filepath.Join(r, "ub", "logs", "ubcf.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runURLFetch(context.Background(), f, "ub", task)
	assert.Error(t, err)
}

func TestRunPrecompiled_RepoOnly(t *testing.T) {
	task := &v1.IngestionTask{Id: "ro", SourceType: "rss", ConfigJson: mj(map[string]string{"repo": "o/r"})}
	logPath := filepath.Join(t.TempDir(), "ro.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	eng := NewEngine("/tmp", nil, nil, nil)
	err := eng.runPrecompiled(context.Background(), f, "p", task)
	assert.Error(t, err)
}

func TestRunDynamic_Blocked(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "db")
	task := &v1.IngestionTask{Id: "dbl", SourceType: "custom_code", ConfigJson: mj(map[string]string{"code": `package main; import "os"; func main() {}`})}
	logPath := filepath.Join(r, "db", "logs", "dbl.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runDynamic(context.Background(), f, "db", task)
	assert.Error(t, err)
}

func TestRunEmailFetch_Missing(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "em")
	task := &v1.IngestionTask{Id: "emnp", SourceType: "email", ConfigJson: `{"host":"h","user":"u"}`}
	logPath := filepath.Join(r, "em", "logs", "emnp.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runEmailFetch(context.Background(), f, "em", task)
	assert.Error(t, err)
}

func TestInsertJSONArray_ErrPaths(t *testing.T) {
	eng, _ := setupEngineFull(t)

	t.Run("empty", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "e.log")
		f, _ := os.Create(logPath)
		defer f.Close()
		assert.NoError(t, eng.insertJSONArray(context.Background(), "t", []any{}, f))
	})
	t.Run("bad_table", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "bt.log")
		f, _ := os.Create(logPath)
		defer f.Close()
		assert.Error(t, eng.insertJSONArray(context.Background(), "bad;name", []any{map[string]any{"k": "v"}}, f))
	})
	t.Run("bad_col", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "bc.log")
		f, _ := os.Create(logPath)
		defer f.Close()
		assert.Error(t, eng.insertJSONArray(context.Background(), "t", []any{map[string]any{"bad;col": "v"}}, f))
	})
	t.Run("not_obj", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "no.log")
		f, _ := os.Create(logPath)
		defer f.Close()
		assert.Error(t, eng.insertJSONArray(context.Background(), "t", []any{"not_obj"}, f))
	})
}

func TestRunPrecompiled_RepoWithURL(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rpu")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "rpu", SourceType: "rss", ConfigJson: mj(map[string]string{"repo": "o/r", "url": srv.URL})}
	logPath := filepath.Join(r, "rpu", "logs", "rpu.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runPrecompiled(context.Background(), f, "rpu", task)
	require.NoError(t, err)
}

func TestRunCopy_Direct(t *testing.T) {
	eng, root := setupEngineFull(t)
	src, dst := "src_copy", "dst_copy"
	createDirs(t, root, src)
	createDirs(t, root, dst)
	os.WriteFile(filepath.Join(root, src, "raw", "data.csv"), []byte("a,b\n1,2"), 0644)
	task := &v1.IngestionTask{Id: "cpdir", SourceType: "copy", ConfigJson: mj(map[string]string{"source": src})}
	logPath := filepath.Join(root, dst, "logs", "cpdir.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runCopy(context.Background(), f, dst, task)
	require.NoError(t, err)
}

func TestRunTask_Concurrent2(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "cc")
	t1 := &v1.IngestionTask{Id: "ca", SourceType: "csv", ConfigJson: `{"path":"/nope/a.csv"}`}
	t2 := &v1.IngestionTask{Id: "cb", SourceType: "csv", ConfigJson: `{"path":"/nope/b.csv"}`}
	ch := make(chan error, 2)
	go func() { ch <- eng.RunTask(context.Background(), "cc", t1) }()
	go func() { ch <- eng.RunTask(context.Background(), "cc", t2) }()
	for i := 0; i < 2; i++ {
		assert.Error(t, <-ch)
	}
}
func TestRunPrecompiled_RepoFallback(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rpf")
	task := &v1.IngestionTask{Id: "rpf", SourceType: "rss", ConfigJson: mj(map[string]string{"repo": "o/r"})}
	logPath := filepath.Join(r, "rpf", "logs", "rpf.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runPrecompiled(context.Background(), f, "rpf", task)
	assert.Error(t, err)
}

func TestRunURLFetch_CSVSuffix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("a,b\n1,2\n3,4"))
	}))
	defer srv.Close()
	eng, r := setupEngineFull(t)
	createDirs(t, r, "csvs")
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "csvs", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL + "/data.csv"})}
	logPath := filepath.Join(r, "csvs", "logs", "csvs.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runURLFetch(context.Background(), f, "csvs", task)
	require.NoError(t, err)
}

func TestRunURLFetch_DetectCSV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("a,b,c\n1,2,3"))
	}))
	defer srv.Close()
	eng, r := setupEngineFull(t)
	createDirs(t, r, "dcsv")
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "dcsv", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL})}
	logPath := filepath.Join(r, "dcsv", "logs", "dcsv.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runURLFetch(context.Background(), f, "dcsv", task)
	require.NoError(t, err)
}

func TestRunCSVLoad_ParquetSuffix(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "parq")
	csvPath := filepath.Join(t.TempDir(), "data.parquet")
	os.WriteFile(csvPath, []byte("PAR1"), 0644)
	task := &v1.IngestionTask{Id: "parq", SourceType: "csv", ConfigJson: mj(map[string]string{"path": csvPath})}
	logPath := filepath.Join(r, "parq", "logs", "parq.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runCSVLoad(context.Background(), f, "parq", task)
	assert.Error(t, err)
}

func TestResolveTableName_ConfigTableName(t *testing.T) {
	task := &v1.IngestionTask{Id: "x", Name: "ignored", ConfigJson: `{"tableName":"cfg_tbl"}`}
	name, err := resolveTableName(task)
	require.NoError(t, err)
	assert.Equal(t, "cfg_tbl", name)
}

func TestResolveTableName_TaskIDOnly(t *testing.T) {
	task := &v1.IngestionTask{Id: "myid", Name: "", ConfigJson: `{}`}
	name, err := resolveTableName(task)
	require.NoError(t, err)
	assert.Equal(t, "myid", name)
}

func TestRunDynamic_EmptyCode(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "dec")
	task := &v1.IngestionTask{Id: "dec", SourceType: "custom_code", ConfigJson: `{"code":""}`}
	logPath := filepath.Join(r, "dec", "logs", "dec.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runDynamic(context.Background(), f, "dec", task)
	assert.Error(t, err)
}

func TestRunDynamic_InvalidJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "dij")
	task := &v1.IngestionTask{Id: "dij", SourceType: "custom_code", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "dij", "logs", "dij.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runDynamic(context.Background(), f, "dij", task)
	assert.Error(t, err)
}

func TestRunEmailFetch_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "ebj")
	task := &v1.IngestionTask{Id: "ebj", SourceType: "email", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "ebj", "logs", "ebj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runEmailFetch(context.Background(), f, "ebj", task)
	assert.Error(t, err)
}

func TestRunEmailFetch_DefaultFolder(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "edf")
	task := &v1.IngestionTask{Id: "edf", SourceType: "email", ConfigJson: mj(map[string]string{"host": "h", "user": "u", "pass": "p"})}
	logPath := filepath.Join(r, "edf", "logs", "edf.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runEmailFetch(context.Background(), f, "edf", task)
	assert.Error(t, err)
}

func TestRunGitHubSource_MissingOwner(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "gmo")
	task := &v1.IngestionTask{Id: "gmo", SourceType: "github", ConfigJson: `{"repo":"r"}`}
	logPath := filepath.Join(r, "gmo", "logs", "gmo.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runGitHubSource(context.Background(), f, "gmo", task)
	assert.Error(t, err)
}

func TestRunSitemapSource_MissingURL(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "smu")
	task := &v1.IngestionTask{Id: "smu", SourceType: "sitemap", ConfigJson: `{"url":""}`}
	logPath := filepath.Join(r, "smu", "logs", "smu.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runSitemapSource(context.Background(), f, "smu", task)
	assert.Error(t, err)
}

func TestRunJSONAPISource_MissingURL(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "jmu")
	task := &v1.IngestionTask{Id: "jmu", SourceType: "jsonapi", ConfigJson: `{"url":""}`}
	logPath := filepath.Join(r, "jmu", "logs", "jmu.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runJSONAPISource(context.Background(), f, "jmu", task)
	assert.Error(t, err)
}

func TestRunSheetsSource_MissingID(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "smi")
	task := &v1.IngestionTask{Id: "smi", SourceType: "sheets", ConfigJson: `{"api_key":"k"}`}
	logPath := filepath.Join(r, "smi", "logs", "smi.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runSheetsSource(context.Background(), f, "smi", task)
	assert.Error(t, err)
}

func TestRunPostgresLoad_EmptyDSN(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "ped")
	task := &v1.IngestionTask{Id: "ped", SourceType: "postgres", ConfigJson: `{"dsn":""}`}
	logPath := filepath.Join(r, "ped", "logs", "ped.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runPostgresLoad(context.Background(), f, "ped", task)
	assert.Error(t, err)
}

func TestRunCopy_EmptySource(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "ces")
	task := &v1.IngestionTask{Id: "ces", SourceType: "copy", ConfigJson: `{"source":""}`}
	logPath := filepath.Join(r, "ces", "logs", "ces.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runCopy(context.Background(), f, "ces", task)
	assert.Error(t, err)
}

func TestBuildNextURLFn(t *testing.T) {
	fn := buildNextURLFn("https://x.com", PaginationInfo{Type: "cursor"})
	assert.NotNil(t, fn)
	assert.NotEmpty(t, fn([]byte(`{"next_cursor":"nxt"}`)))

	fn = buildNextURLFn("https://x.com", PaginationInfo{Type: "unknown"})
	assert.NotNil(t, fn)
	assert.Empty(t, fn([]byte(`{}`)))
}

func TestNextPageURL_2(t *testing.T) {
	got := nextPageURL([]byte(`{"page":2}`), "https://x.com", PaginationInfo{Type: "page", PageParam: "page"})
	assert.Contains(t, got, "page=3")
}

func TestClassifySourceType_More(t *testing.T) {
	assert.Equal(t, "sitemap", classifySourceType("https://x.com", "text/plain", []byte(`<urlset></urlset>`)))
	assert.Equal(t, "generic_json", classifySourceType("https://x.com", "text/plain", []byte(`<?xml version="1"?><r></r>`)))
	assert.Equal(t, "web", classifySourceType("https://x.com", "text/plain", []byte(`<html></html>`)))
	assert.Equal(t, "github", classifySourceType("https://github.com/o/r/repos/1", "text/plain", []byte(`{}`)))
}

func TestRunTask_AllSourceTypes(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "ast")
	eng.metaRepo = nil
	for _, src := range []string{"rss", "rest", "url", "csv", "postgres", "copy", "email", "custom_code", "github", "sitemap", "jsonapi", "sheets", "unknown"} {
		task := &v1.IngestionTask{Id: "ast_" + src, SourceType: src, ConfigJson: `{}`}
		eng.RunTask(context.Background(), "ast", task)
	}
}

func TestRunDynamic_BadCode(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rbc")
	task := &v1.IngestionTask{Id: "rbc", SourceType: "custom_code", ConfigJson: mj(map[string]string{"code": "syntax error {"})}
	logPath := filepath.Join(r, "rbc", "logs", "rbc.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runDynamic(context.Background(), f, "rbc", task)
	assert.Error(t, err)
}

func TestRunEmailFetch_NoHost(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "enh")
	task := &v1.IngestionTask{Id: "enh", SourceType: "email", ConfigJson: mj(map[string]string{"host": "", "user": "u", "pass": "p"})}
	logPath := filepath.Join(r, "enh", "logs", "enh.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runEmailFetch(context.Background(), f, "enh", task)
	assert.Error(t, err)
}

func TestRunEmailFetch_NoUser(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "enu")
	task := &v1.IngestionTask{Id: "enu", SourceType: "email", ConfigJson: mj(map[string]string{"host": "h", "pass": "p"})}
	logPath := filepath.Join(r, "enu", "logs", "enu.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runEmailFetch(context.Background(), f, "enu", task)
	assert.Error(t, err)
}

func TestRunGitHubSource_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "gbj")
	task := &v1.IngestionTask{Id: "gbj", SourceType: "github", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "gbj", "logs", "gbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runGitHubSource(context.Background(), f, "gbj", task)
	assert.Error(t, err)
}

func TestRunGitHubSource_NoRepo(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "gnr")
	task := &v1.IngestionTask{Id: "gnr", SourceType: "github", ConfigJson: mj(map[string]string{"owner": "o"})}
	logPath := filepath.Join(r, "gnr", "logs", "gnr.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runGitHubSource(context.Background(), f, "gnr", task)
	assert.Error(t, err)
}

func TestRunSitemapSource_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "sbj")
	task := &v1.IngestionTask{Id: "sbj", SourceType: "sitemap", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "sbj", "logs", "sbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runSitemapSource(context.Background(), f, "sbj", task)
	assert.Error(t, err)
}

func TestRunJSONAPISource_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "jbj")
	task := &v1.IngestionTask{Id: "jbj", SourceType: "jsonapi", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "jbj", "logs", "jbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runJSONAPISource(context.Background(), f, "jbj", task)
	assert.Error(t, err)
}

func TestRunSheetsSource_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "shbj")
	task := &v1.IngestionTask{Id: "shbj", SourceType: "sheets", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "shbj", "logs", "shbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runSheetsSource(context.Background(), f, "shbj", task)
	assert.Error(t, err)
}

func TestRunPostgresLoad_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "pbj")
	task := &v1.IngestionTask{Id: "pbj", SourceType: "postgres", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "pbj", "logs", "pbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runPostgresLoad(context.Background(), f, "pbj", task)
	assert.Error(t, err)
}

func TestRunCopy_NotFound(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "cnf")
	task := &v1.IngestionTask{Id: "cnf", SourceType: "copy", ConfigJson: mj(map[string]string{"source": "nonexistent"})}
	logPath := filepath.Join(r, "cnf", "logs", "cnf.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runCopy(context.Background(), f, "cnf", task)
	assert.Error(t, err)
}

func TestRunCopy_BadJSON(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "cbj")
	task := &v1.IngestionTask{Id: "cbj", SourceType: "copy", ConfigJson: `not-json`}
	logPath := filepath.Join(r, "cbj", "logs", "cbj.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	err := eng.runCopy(context.Background(), f, "cbj", task)
	assert.Error(t, err)
}

func TestRunTask_UnknownSource(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rus")
	task := &v1.IngestionTask{Id: "rus", SourceType: "bogus", ConfigJson: `{}`}
	err := eng.RunTask(context.Background(), "rus", task)
	assert.Error(t, err)
}

func TestRunTask_CSVInvalidID(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "cvi")
	task := &v1.IngestionTask{Id: "bad;id", SourceType: "csv", ConfigJson: `{"path":"/nope.csv"}`}
	err := eng.RunTask(context.Background(), "cvi", task)
	assert.Error(t, err)
}

func TestRegisterViews_WithOntology(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rvo")
	ontPath := filepath.Join(r, "rvo", "ontologies", "core.aleph")
	os.MkdirAll(filepath.Dir(ontPath), 0755)
	os.WriteFile(ontPath, []byte("object testobj from dataset testobj id id"), 0644)
	eng.db.Exec(context.Background(), `CREATE TABLE "testobj" (id INTEGER)`)
	err := eng.registerViews(context.Background(), "rvo")
	assert.NoError(t, err)
}

func TestEnrichPredictiveMetadata_WithOntology(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "evo")
	ontPath := filepath.Join(r, "evo", "ontologies", "core.aleph")
	os.MkdirAll(filepath.Dir(ontPath), 0755)
	os.WriteFile(ontPath, []byte("object rich from dataset rich id id"), 0644)
	eng.db.Exec(context.Background(), `CREATE TABLE "rich" (id INTEGER, text_col VARCHAR)`)
	eng.db.Exec(context.Background(), `INSERT INTO "rich" VALUES (1, 'this is long enough text for NLP analysis yes')`)
	eng.nlpHandler = &stubNLP{score: 0.5, label: "neutral"}
	eng.enrichPredictiveMetadata(context.Background(), "evo", "rich")
}

func TestRunTask_WithEnrichment(t *testing.T) {
	eng, r := setupEngineFull(t)
	createDirs(t, r, "rwe")
	ontPath := filepath.Join(r, "rwe", "ontologies", "core.aleph")
	os.MkdirAll(filepath.Dir(ontPath), 0755)
	os.WriteFile(ontPath, []byte("object fulltask from dataset fulltask id id"), 0644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1}]`))
	}))
	defer srv.Close()
	eng.httpClient = srv.Client()
	eng.nlpHandler = &stubNLP{score: 0.3, label: "neutral"}

	task := &v1.IngestionTask{Id: "fulltask", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(context.Background(), "rwe", task)
	require.NoError(t, err)
}
