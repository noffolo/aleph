package ingestion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_RunPrecompiled_JSONSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"name":"test"},{"id":2,"name":"test2"}]`))
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "precompiled_json_test",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "httptest-proj", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT * FROM "precompiled_json_test"`)
	require.NoError(t, err)
	defer rows.Close()
	assert.True(t, rows.Next())
}

func TestEngine_RunPrecompiled_CSVSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("name,age\nAlice,30\nBob,25\n"))
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "precompiled_csv_test",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "httptest-proj", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT * FROM "precompiled_csv_test"`)
	require.NoError(t, err)
	defer rows.Close()
	assert.True(t, rows.Next())
}

func TestEngine_RunPrecompiled_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "precompiled_err_test",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "httptest-proj", task)
	assert.Error(t, err)
}

func TestEngine_RunPrecompiled_InvalidJSON(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "bad_config",
		ConfigJson: `{invalid`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunPrecompiled_NoURL(t *testing.T) {
	t.Parallel()
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	task := &v1.IngestionTask{
		Id:         "no_url_task",
		ConfigJson: `{}`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "httptest-proj", task)
	assert.Error(t, err)
}

func TestEngine_RunPrecompiled_WithToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"authed":true}]`))
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "auth_task_test",
		ConfigJson: `{"url": "` + srv.URL + `", "token": "test-token-123"}`,
	}
	err := eng.runPrecompiled(context.Background(), os.Stdout, "httptest-proj", task)
	require.NoError(t, err)
}

func TestEngine_RunURLFetch_JSONSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"key":"val"}]`))
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "url_fetch_json",
		SourceType: "url",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runURLFetch(context.Background(), os.Stdout, "httptest-proj", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT * FROM "url_fetch_json"`)
	require.NoError(t, err)
	defer rows.Close()
	assert.True(t, rows.Next())
}

func TestEngine_RunURLFetch_CSVSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("x,y\n1,2\n3,4\n"))
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "url_fetch_csv",
		SourceType: "url",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runURLFetch(context.Background(), os.Stdout, "httptest-proj", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT * FROM "url_fetch_csv"`)
	require.NoError(t, err)
	defer rows.Close()
	assert.True(t, rows.Next())
}

func TestEngine_RunURLFetch_EmptyURL(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "empty_url_task",
		SourceType: "url",
		ConfigJson: `{"url": ""}`,
	}
	err := eng.runURLFetch(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunURLFetch_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.httpClient = srv.Client()

	task := &v1.IngestionTask{
		Id:         "url_fetch_err",
		SourceType: "url",
		ConfigJson: `{"url": "` + srv.URL + `"}`,
	}
	err := eng.runURLFetch(context.Background(), os.Stdout, "httptest-proj", task)
	assert.Error(t, err)
}

func TestEngine_InsertJSONArray_Success(t *testing.T) {
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "httptest-proj")
	eng.projectsRoot = projectsRoot

	arr := []any{
		map[string]any{"name": "Alice", "age": float64(30)},
		map[string]any{"name": "Bob", "age": float64(25)},
	}
	err := eng.insertJSONArray(context.Background(), "insert_test_table", arr, os.Stdout)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT name FROM "insert_test_table" ORDER BY name`)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	assert.Equal(t, []string{"Alice", "Bob"}, names)
}

func TestEngine_InsertJSONArray_EmptyArray(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	err := eng.insertJSONArray(context.Background(), "empty_tbl", []any{}, os.Stdout)
	assert.NoError(t, err)
}

func TestEngine_InsertJSONArray_NotObject(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	arr := []any{"not_an_object"}
	err := eng.insertJSONArray(context.Background(), "bad_tbl", arr, os.Stdout)
	assert.Error(t, err)
}

func TestEngine_RunCopy_Success(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)

	root := t.TempDir()
	srcID := "src-proj-httptest"
	destID := "dest-proj-httptest"
	createTestProject(t, root, srcID)
	createTestProject(t, root, destID)

	srcRaw := filepath.Join(root, srcID, "raw")
	require.NoError(t, os.WriteFile(filepath.Join(srcRaw, "data.csv"), []byte("a,b\n1,2\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRaw, "info.json"), []byte(`[{"k":"v"}]`), 0644))

	eng.projectsRoot = root

	task := &v1.IngestionTask{
		Id:         "copy_test_task",
		SourceType: "copy",
		ConfigJson: `{"source": "` + srcID + `"}`,
	}
	err := eng.runCopy(context.Background(), os.Stdout, destID, task)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(root, destID, "raw", "data.csv"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, destID, "raw", "info.json"))
	assert.NoError(t, err)
}

func TestEngine_RunCopy_EmptySource(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "copy_empty_src",
		SourceType: "copy",
		ConfigJson: `{"source": ""}`,
	}
	err := eng.runCopy(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunCopy_BadJSON(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "copy_bad_json",
		SourceType: "copy",
		ConfigJson: `{bad`,
	}
	err := eng.runCopy(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_Client_Injected(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	eng, _ := setupTestEngine(t)
	injectedClient := srv.Client()
	eng.httpClient = injectedClient

	c := eng.client()
	assert.NotNil(t, c)
	// Verify the injected client works by making an actual request
	resp, err := c.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestEngine_Client_Default(t *testing.T) {
	t.Parallel()
	eng := NewEngine("/tmp/test", nil, nil, nil)
	c := eng.client()
	assert.NotNil(t, c)
}

func TestEngine_RunPostgresLoad_EmptyDSN(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "pg_empty",
		SourceType: "postgres",
		ConfigJson: `{"dsn": ""}`,
	}
	err := eng.runPostgresLoad(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunDynamic_EmptyCode(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "dyn_empty",
		SourceType: "custom_code",
		ConfigJson: `{"code": ""}`,
	}
	err := eng.runDynamic(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunDynamic_BlockedImport(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "dyn_blocked",
		SourceType: "custom_code",
		ConfigJson: `{"code": "package main; import \"os/exec\"; func main() {}"}`,
	}
	err := eng.runDynamic(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunEmailFetch_BadJSON(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "email_bad",
		SourceType: "email",
		ConfigJson: `{not json`,
	}
	err := eng.runEmailFetch(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunEmailFetch_MissingFields(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "email_missing",
		SourceType: "email",
		ConfigJson: `{"host": ""}`,
	}
	err := eng.runEmailFetch(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunGitHubSource_MissingFields(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "gh_missing",
		SourceType: "github",
		ConfigJson: `{"owner": ""}`,
	}
	err := eng.runGitHubSource(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunSitemapSource_EmptyURL(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "sitemap_empty",
		SourceType: "sitemap",
		ConfigJson: `{"url": ""}`,
	}
	err := eng.runSitemapSource(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunJSONAPISource_EmptyURL(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "jsonapi_empty",
		SourceType: "jsonapi",
		ConfigJson: `{"url": ""}`,
	}
	err := eng.runJSONAPISource(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RunSheetsSource_MissingID(t *testing.T) {
	t.Parallel()
	eng, _ := setupTestEngine(t)
	task := &v1.IngestionTask{
		Id:         "sheets_missing",
		SourceType: "sheets",
		ConfigJson: `{"spreadsheet_id": ""}`,
	}
	err := eng.runSheetsSource(context.Background(), os.Stdout, "proj", task)
	assert.Error(t, err)
}

func TestEngine_RegisterViews_NoOntology(t *testing.T) {
	t.Parallel()
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "noont-httptest")
	err := eng.registerViews(context.Background(), "noont-httptest")
	assert.NoError(t, err)
}
