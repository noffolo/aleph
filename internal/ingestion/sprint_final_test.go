package ingestion

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSprintGetOrCreateGithubIngester(t *testing.T) {
	eng, _ := setupEngineFull(t)
	assert.Nil(t, eng.githubIngester)
	g1 := eng.getOrCreateGitHubIngester(&v1.IngestionTask{ConfigJson: `{"token":"t"}`})
	assert.NotNil(t, g1)
	g2 := eng.getOrCreateGitHubIngester(&v1.IngestionTask{})
	assert.Same(t, g1, g2)
}

func TestSprintGetOrCreateSheetsIngester(t *testing.T) {
	eng, _ := setupEngineFull(t)
	assert.Nil(t, eng.sheetsIngester)
	s1 := eng.getOrCreateSheetsIngester(&v1.IngestionTask{ConfigJson: mj(map[string]string{"api_key": "k"})})
	assert.NotNil(t, s1)
}

func TestSprintGetOrCreateSitemapIngester(t *testing.T) {
	eng, _ := setupEngineFull(t)
	assert.Nil(t, eng.sitemapIngester)
	s1 := eng.getOrCreateSitemapIngester()
	assert.NotNil(t, s1)
}

func TestSprintGetOrCreateJsonapiIngester(t *testing.T) {
	eng, _ := setupEngineFull(t)
	assert.Nil(t, eng.jsonapiIngester)
	j1 := eng.getOrCreateJSONAPIIngester()
	assert.NotNil(t, j1)
}

func TestSprintGetOrCreateProbeRunner(t *testing.T) {
	eng, _ := setupEngineFull(t)
	assert.Nil(t, eng.probeRunner)
	p1 := eng.getOrCreateProbeRunner()
	assert.NotNil(t, p1)
}

func TestSprintRunEmailAllEmpty(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "eeall")
	task := &v1.IngestionTask{Id: "eeall", SourceType: "email", ConfigJson: `{"host":"","user":"","pass":""}`}
	err := eng.RunTask(t.Context(), "eeall", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email config requires")
}

func TestSprintRunEmailNoUser(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "enouser")
	task := &v1.IngestionTask{Id: "enouser", SourceType: "email", ConfigJson: `{"host":"h","pass":"p"}`}
	err := eng.RunTask(t.Context(), "enouser", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email config requires")
}

func TestSprintRunEmailIMAPConn(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "eimap")
	task := &v1.IngestionTask{Id: "eimap", SourceType: "email", ConfigJson: mj(map[string]string{"host": "127.0.0.1:19999", "user": "u", "pass": "p"})}
	err := eng.RunTask(t.Context(), "eimap", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IMAP fetch failed")
}

func TestSprintRunEmailBadTable(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "ebad")
	task := &v1.IngestionTask{Id: "bad;name", SourceType: "email", ConfigJson: mj(map[string]string{"host": "h", "user": "u", "pass": "p"})}
	err := eng.RunTask(t.Context(), "ebad", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid table name")
}

func TestSprintRunGitHubEmpty(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "ghe")
	task := &v1.IngestionTask{Id: "ghe", SourceType: "github", ConfigJson: `{"owner":"","repo":""}`}
	err := eng.RunTask(t.Context(), "ghe", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repo")
}

func TestSprintRunPostgresEmptyDSN(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "pge")
	task := &v1.IngestionTask{Id: "pge", SourceType: "postgres", ConfigJson: `{"dsn":""}`}
	err := eng.RunTask(t.Context(), "pge", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty DSN")
}

func TestSprintRunDynamicEmptyCode(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "dne")
	task := &v1.IngestionTask{Id: "dne", SourceType: "custom_code", ConfigJson: `{"code":""}`}
	err := eng.RunTask(t.Context(), "dne", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty code")
}

func TestSprintRunSheetsNoID(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "sne")
	task := &v1.IngestionTask{Id: "sne", SourceType: "sheets", ConfigJson: `{}`}
	err := eng.RunTask(t.Context(), "sne", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spreadsheet_id")
}

func TestSprintEnrichEmpty(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "ene")
	eng.db.Exec(t.Context(), `CREATE TABLE ene (id INT, x VARCHAR)`)
	eng.nlpHandler = &stubNLP{score: 0.7, label: "pos"}
	eng.enrichPredictiveMetadata(t.Context(), "ene", "ene")
}

func TestSprintEnrichNLPErr(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "enle")
	eng.nlpHandler = &stubNLP{score: 0, label: "", err: assert.AnError}
	eng.db.Exec(t.Context(), `CREATE TABLE enle (id INT, t VARCHAR)`)
	eng.db.Exec(t.Context(), `INSERT INTO enle VALUES (1, 'long enough text for nlp enrichment check okay')`)
	eng.enrichPredictiveMetadata(t.Context(), "enle", "enle")
}

func TestSprintEnrichNilNLP(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "enni")
	eng.nlpHandler = nil
	eng.db.Exec(t.Context(), `CREATE TABLE enni (id INT, t VARCHAR)`)
	eng.enrichPredictiveMetadata(t.Context(), "enni", "enni")
}

func TestSprintRunDynamicBuildFail(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "dbb")
	logPath := filepath.Join(root, "dbb", "logs", "dbb.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	code := "package main\nfunc main() { var x int = \"hello\" }\n"
	task := &v1.IngestionTask{Id: "dbb", SourceType: "custom_code", ConfigJson: mj(map[string]string{"code": code})}
	err := eng.runDynamic(t.Context(), f, "dbb", task)
	assert.Error(t, err)
}

func TestSprintRunPostgresInstall(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "pgi")
	logPath := filepath.Join(root, "pgi", "logs", "pgi.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, _ := os.Create(logPath)
	defer f.Close()
	task := &v1.IngestionTask{Id: "pgi", SourceType: "postgres", ConfigJson: mj(map[string]string{"dsn": "host=localhost port=5432 dbname=mydb user=test"})}
	err := eng.runPostgresLoad(t.Context(), f, "pgi", task)
	assert.Error(t, err)
}

func TestSprintRegisterViews(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "rv")
	ontPath := filepath.Join(root, "rv", "ontologies", "core.aleph")
	os.MkdirAll(filepath.Dir(ontPath), 0755)
	os.WriteFile(ontPath, []byte("object tv from dataset tv id id\n"), 0644)
	rawDir := filepath.Join(root, "rv", "raw")
	os.MkdirAll(rawDir, 0755)
	os.WriteFile(filepath.Join(rawDir, "tv.csv"), []byte("id,name\n1,a\n"), 0644)
	eng.registerViews(t.Context(), "rv")
}

func TestSprintRunPrecompiledJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"k":"v"}]`))
	}))
	t.Cleanup(srv.Close)
	eng, root := setupEngineFull(t)
	createDirs(t, root, "rpj")
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "rpj", SourceType: "rss", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(t.Context(), "rpj", task)
	require.NoError(t, err)
}

func TestSprintRunURLJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"a":1}]`))
	}))
	t.Cleanup(srv.Close)
	eng, root := setupEngineFull(t)
	createDirs(t, root, "ruj")
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "ruj", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL + "/data.json"})}
	err := eng.RunTask(t.Context(), "ruj", task)
	require.NoError(t, err)
}

func TestSprintRunURLCSV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("c1,c2\nv1,v2\n"))
	}))
	t.Cleanup(srv.Close)
	eng, root := setupEngineFull(t)
	createDirs(t, root, "ruc")
	eng.httpClient = srv.Client()
	task := &v1.IngestionTask{Id: "ruc", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(t.Context(), "ruc", task)
	require.NoError(t, err)
}

func TestSprintRunCSVFull(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "rcf")
	csvPath := filepath.Join(t.TempDir(), "d.csv")
	os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0644)
	task := &v1.IngestionTask{Id: "rcf", SourceType: "csv", ConfigJson: mj(map[string]string{"path": csvPath})}
	err := eng.RunTask(t.Context(), "rcf", task)
	require.NoError(t, err)
}

func TestSprintRunCopyFull(t *testing.T) {
	eng, root := setupEngineFull(t)
	srcID := "sc"
	destID := "dc"
	createDirs(t, root, srcID)
	createDirs(t, root, destID)
	os.WriteFile(filepath.Join(root, srcID, "raw", "f.csv"), []byte("x,y\n1,2\n"), 0644)
	task := &v1.IngestionTask{Id: "cpf", SourceType: "copy", ConfigJson: mj(map[string]string{"source": srcID})}
	err := eng.RunTask(t.Context(), destID, task)
	require.NoError(t, err)
}

func TestSprintRunPostgresQueryBlocked(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "pqb")
	task := &v1.IngestionTask{Id: "pqb", SourceType: "postgres", ConfigJson: mj(map[string]string{"dsn": "p", "query": "SELECT 1"})}
	err := eng.RunTask(t.Context(), "pqb", task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom queries not allowed")
}

func TestSprintRunSitemap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://x.com/p1</loc></url></urlset>`))
	}))
	t.Cleanup(srv.Close)
	eng, root := setupEngineFull(t)
	createDirs(t, root, "rsm")
	eng.httpClient = srv.Client()
	si := sources.NewSitemapIngester()
	si.UseNonSSRFClient()
	eng.sitemapIngester = si
	task := &v1.IngestionTask{Id: "rsm", SourceType: "sitemap", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(t.Context(), "rsm", task)
	require.NoError(t, err)
}

func TestSprintProbeExecute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1}]`))
	}))
	t.Cleanup(srv.Close)
	pr := NewProbeRunner(nil)
	pr.client = sources.NewTestRateLimitedClient()
	result := &SourceProbeResult{SrcType: "rest", URL: srv.URL, Pag: PaginationInfo{Type: "none", MaxLimit: -1}}
	err := pr.Execute(t.Context(), result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.DataSample)
}

func TestSprintRunTaskEnrichment(t *testing.T) {
	eng, root := setupEngineFull(t)
	createDirs(t, root, "rte")
	eng.db.Exec(t.Context(), `CREATE TABLE IF NOT EXISTS system_features (project_id VARCHAR, task_id VARCHAR, entity_id VARCHAR, feature_type VARCHAR, feature_value FLOAT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":7,"note":"long enough text for enrichment testing sent"}]`))
	}))
	t.Cleanup(srv.Close)
	eng.httpClient = srv.Client()
	eng.nlpHandler = &stubNLP{score: 0.9, label: "positive"}
	task := &v1.IngestionTask{Id: "rte", SourceType: "url", ConfigJson: mj(map[string]string{"url": srv.URL})}
	err := eng.RunTask(t.Context(), "rte", task)
	require.NoError(t, err)
}

func TestSprintValidateCode(t *testing.T) {
	assert.Error(t, validateCode("package main\nimport \"os\"\nfunc main(){}"))
	assert.Error(t, validateCode("package main\nimport \"net/http\"\nfunc main(){}"))
	assert.NoError(t, validateCode("package main\nimport \"fmt\"\nfunc main(){}"))
	assert.NoError(t, validateCode("package main\nfunc main(){}\n"))
}

func TestSprintResolveTableName(t *testing.T) {
	n1, _ := resolveTableName(&v1.IngestionTask{Id: "simple", ConfigJson: `{}`})
	assert.NotEmpty(t, n1)
	n2, _ := resolveTableName(&v1.IngestionTask{Id: "with-dash", Name: "d", ConfigJson: `{"tableName":"custom"}`})
	assert.NotEmpty(t, n2)
}

func TestSprintNextCursorURL(t *testing.T) {
	assert.Equal(t, "c1", nextCursorURL([]byte(`{"cursor":"c1"}`), "cursor"))
	assert.Equal(t, "n1", nextCursorURL([]byte(`{"next_cursor":"n1"}`), "cursor"))
	assert.Empty(t, nextCursorURL([]byte(`{}`), "cursor"))
}

func TestSprintNextPageOffset(t *testing.T) {
	u := nextPageURL([]byte(`{"meta":{"offset":50}}`), "https://api.x.com?offset=50", PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: 50})
	assert.Contains(t, u, "offset=100")
}
