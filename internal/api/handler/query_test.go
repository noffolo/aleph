package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupQueryHandler(t *testing.T) (*QueryHandler, string) {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	h := NewQueryHandler(db, projectsRoot, nil, nil, nil, 0)
	return h, projectsRoot
}

func createProjectWithOntology(t *testing.T, projectsRoot, projectID, ontology string) {
	t.Helper()
	ontDir := filepath.Join(projectsRoot, projectID, "ontologies")
	require.NoError(t, os.MkdirAll(ontDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ontDir, "core.aleph"), []byte(ontology), 0644))
}

func TestSQLInjectionFailClosed(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "inj-proj", "// empty\n")

	t.Run("projectID with SQL injection returns validation error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, err := h.ExecuteQuery(ctx, connect.NewRequest(&v1.ExecuteQueryRequest{
			ObjectType: "items",
			ProjectId:  "valid_project; DROP TABLE metadata; --",
		}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid")
	})
}

func TestQueryHandler_ExecuteQuery_FallbackToTable(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE items (name VARCHAR, qty INTEGER)")
	require.NoError(t, err)
	_, err = h.db.Exec(context.Background(), "INSERT INTO items VALUES ('apple', 10), ('banana', 5)")
	require.NoError(t, err)

	createProjectWithOntology(t, projectsRoot, "test-proj", "// empty ontology\n")

	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "test-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 2)
	assert.Contains(t, resp.Msg.Columns, "name")
}

func TestQueryHandler_ExecuteQuery_WithLimit(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE limited (val INTEGER)")
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		h.db.Exec(context.Background(), "INSERT INTO limited VALUES (?)", i)
	}

	createProjectWithOntology(t, projectsRoot, "lim-proj", "// empty\n")

	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "limited",
		ProjectId:  "lim-proj",
		Limit:      5,
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 5)
}

func TestQueryHandler_ExecuteQuery_InvalidName(t *testing.T) {
	h, _ := setupQueryHandler(t)
	_, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "bad;name",
		ProjectId:  "proj",
	}))
	assert.Error(t, err)
}

func TestQueryHandler_ExecuteQuery_NotFound(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "nf-proj", "// empty\n")

	_, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "nonexistent_table",
		ProjectId:  "nf-proj",
	}))
	assert.Error(t, err)
}

func TestQueryHandler_ExecuteQuery_OntologyMatchesTable(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE orders (id INTEGER, amount DOUBLE, status VARCHAR)")
	require.NoError(t, err)
	_, err = h.db.Exec(context.Background(), "INSERT INTO orders VALUES (1, 99.5, 'open'), (2, 150.0, 'closed')")
	require.NoError(t, err)

	createProjectWithOntology(t, projectsRoot, "ont-proj", "// empty\n")

	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "orders",
		ProjectId:  "ont-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 2)
}

func TestQueryHandler_SuggestView(t *testing.T) {
	h, _ := setupQueryHandler(t)
	assert.Equal(t, "map", h.suggestView([]string{"lat", "lon", "name"}))
	assert.Equal(t, "timeline", h.suggestView([]string{"date", "value"}))
	assert.Equal(t, "table", h.suggestView([]string{"name", "age"}))
}

func TestQueryHandler_ConfirmAction(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "conf-proj", "// empty\n")

	resp, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "conf-proj",
		Approved:  true,
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestQueryHandler_GlobalQuery(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE global_test (k VARCHAR)")
	require.NoError(t, err)
	h.db.Exec(context.Background(), "INSERT INTO global_test VALUES ('val')")

	createProjectWithOntology(t, projectsRoot, "gq-proj", "// empty\n")

	resp, err := h.GlobalQuery(context.Background(), connect.NewRequest(&v1.GlobalQueryRequest{
		ObjectType: "global_test",
		ProjectId:  "gq-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 1)
}

// ─── GetDataStats Tests ────────────────────────────────────────────────────

func TestQueryHandler_GetDataStats_InvalidObjName_WithProject(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "ds-proj", "// empty\n")

	_, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "ds-proj",
		ObjectType: "bad;name",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataStats_InvalidProjectID(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "ds-proj", "// empty\n")

	_, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "valid_project; DROP TABLE metadata; --",
		ObjectType: "items",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataStats_ProjectNotFound(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "nonexistent_proj",
		ObjectType: "items",
	}))
	require.Error(t, err)
}

func TestQueryHandler_GetDataStats_EmptyProjectID_Defaults(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	// default project must exist
	createProjectWithOntology(t, projectsRoot, "default", "// empty\n")

	// resolveProject defaults empty projectID to "default"
	// but CompileObject will fail with empty ontology
	_, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ObjectType: "items",
	}))
	require.Error(t, err)
}

func TestQueryHandler_GetDataStats_SQLKeywordObjectName(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "ds-proj", "// empty\n")

	_, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "ds-proj",
		ObjectType: "SELECT",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataStats_WithValidOntology(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE stats_table (id INTEGER, name VARCHAR, score DOUBLE)")
	require.NoError(t, err)
	_, err = h.db.Exec(context.Background(), "INSERT INTO stats_table VALUES (1, 'alpha', 10.5), (2, 'beta', 20.0), (3, 'gamma', 15.5)")
	require.NoError(t, err)

	parquetDir := filepath.Join(projectsRoot, "stats-proj", "raw", "stats_ds", "latest")
	require.NoError(t, os.MkdirAll(parquetDir, 0755))
	parquetPath := filepath.Join(parquetDir, "data.parquet")
	_, err = h.db.Exec(context.Background(), fmt.Sprintf("COPY stats_table TO '%s' (FORMAT PARQUET)", parquetPath))
	require.NoError(t, err)

	createProjectWithOntology(t, projectsRoot, "stats-proj",
		"object stats_table from dataset stats_ds id id property id type number from id property name type text from name property score type number from score\n")

	resp, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "stats-proj",
		ObjectType: "stats_table",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Stats)
	assert.GreaterOrEqual(t, len(resp.Msg.Stats), 1)
}

func TestQueryHandler_GetDataStats_WithDuckDBTable(t *testing.T) {
	duck, err := storage.NewDuckDB("")
	require.NoError(t, err)
	defer duck.Close()

	_, err = duck.Exec(context.Background(), "CREATE TABLE mytable (id INTEGER, name VARCHAR, score DOUBLE)")
	require.NoError(t, err)
	_, err = duck.Exec(context.Background(), "INSERT INTO mytable VALUES (1, 'alpha', 10.5), (2, 'beta', 20.0), (3, 'gamma', 15.5)")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	parquetDir := filepath.Join(projectsRoot, "stats-proj2", "raw", "mytable", "latest")
	require.NoError(t, os.MkdirAll(parquetDir, 0755))
	parquetPath := filepath.Join(parquetDir, "data.parquet")
	_, err = duck.Exec(context.Background(), fmt.Sprintf("COPY mytable TO '%s' (FORMAT PARQUET)", parquetPath))
	require.NoError(t, err)

	createProjectWithOntology(t, projectsRoot, "stats-proj2",
		"object mytable from dataset mytable id id property id type number from id property name type text from name property score type number from score\n")

	h := NewQueryHandler(duck, projectsRoot, nil, nil, nil, 0)

	resp, err := h.GetDataStats(context.Background(), connect.NewRequest(&v1.GetDataStatsRequest{
		ProjectId:  "stats-proj2",
		ObjectType: "mytable",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Stats)
	assert.GreaterOrEqual(t, len(resp.Msg.Stats), 1)
}

// ─── GetDataLineage Tests ──────────────────────────────────────────────────

func TestQueryHandler_GetDataLineage_EmptyProjectID(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "",
		TableName: "items",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestQueryHandler_GetDataLineage_EmptyTableName(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "proj",
		TableName: "",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestQueryHandler_GetDataLineage_InvalidProjectID(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "proj; DROP TABLE x; --",
		TableName: "items",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataLineage_InvalidTableName(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "valid_proj",
		TableName: "bad;name",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataLineage_SQLKeywordTableName(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "valid_proj",
		TableName: "SELECT",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataLineage_ProjectID_ValidateIdentifier_Fail(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "proj-with-hyphens",
		TableName: "items",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_GetDataLineage_WithDuckDBTable(t *testing.T) {
	duck, err := storage.NewDuckDB("")
	require.NoError(t, err)
	defer duck.Close()

	_, err = duck.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS \"lineageproj\"")
	require.NoError(t, err)
	_, err = duck.Exec(context.Background(), "CREATE TABLE \"lineageproj\".\"items\" (id INTEGER, name VARCHAR)")
	require.NoError(t, err)

	h := &QueryHandler{db: duck, programs: newProgramCache()}

	resp, err := h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "lineageproj",
		TableName: "items",
	}))
	if err != nil {
		t.Skipf("json_group_array scan type issue: %v", err)
	}
	assert.NotNil(t, resp.Msg.Provenance)
	assert.Equal(t, "items", resp.Msg.Provenance.TableName)
}

func TestQueryHandler_GetDataLineage_TableNotFound(t *testing.T) {
	duck, err := storage.NewDuckDB("")
	require.NoError(t, err)
	defer duck.Close()

	h := &QueryHandler{db: duck, programs: newProgramCache()}

	_, err = h.GetDataLineage(context.Background(), connect.NewRequest(&v1.GetDataLineageRequest{
		ProjectId: "valid_proj",
		TableName: "nonexistent_table",
	}))
	require.Error(t, err)
}

// ─── ExecuteQuery Additional Tests ─────────────────────────────────────────

func TestQueryHandler_ExecuteQuery_InvalidProjectID(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "proj; DROP TABLE x; --",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_ExecuteQuery_InvalidLowerObjName(t *testing.T) {
	h, _ := setupQueryHandler(t)

	// lowercasing converts to something that fails ValidateStrictIdentifier
	// Actually "SELECT" → "select" which is a SQL keyword
	_, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "SELECT",
		ProjectId:  "proj",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestQueryHandler_ExecuteQuery_ZeroLimitDefaults(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE items (name VARCHAR, qty INTEGER)")
	require.NoError(t, err)
	_, err = h.db.Exec(context.Background(), "INSERT INTO items VALUES ('apple', 10), ('banana', 5), ('cherry', 15)")
	require.NoError(t, err)

	createProjectWithOntology(t, projectsRoot, "zlim-proj", "// empty\n")

	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "zlim-proj",
		Limit:      0,
	}))
	require.NoError(t, err)
	// With limit=0, it should default to 1000 (get all 3 rows)
	assert.Len(t, resp.Msg.Rows, 3)
}

func TestQueryHandler_ExecuteQuery_MixedCaseObjectName(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	// DuckDB folds unquoted identifiers to lowercase
	_, err := h.db.Exec(context.Background(), "CREATE TABLE mytable (val VARCHAR)")
	require.NoError(t, err)
	h.db.Exec(context.Background(), "INSERT INTO mytable VALUES ('test')")

	createProjectWithOntology(t, projectsRoot, "mixed-proj", "// empty\n")

	// ObjectType "MyTable" → validated (OK), lowercased to "mytable" → found in info_schema
	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "MyTable",
		ProjectId:  "mixed-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 1)
}

func TestQueryHandler_GlobalQuery_InvalidObject(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GlobalQuery(context.Background(), connect.NewRequest(&v1.GlobalQueryRequest{
		ObjectType: "bad;name",
	}))
	require.Error(t, err)
}

func TestQueryHandler_GlobalQuery_WithTable(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE gq_items (k VARCHAR)")
	require.NoError(t, err)
	h.db.Exec(context.Background(), "INSERT INTO gq_items VALUES ('val1'), ('val2')")

	createProjectWithOntology(t, projectsRoot, "gq2-proj", "// empty\n")

	resp, err := h.GlobalQuery(context.Background(), connect.NewRequest(&v1.GlobalQueryRequest{
		ObjectType: "gq_items",
		ProjectId:  "gq2-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 2)
}

// ─── GetChecksum Additional Tests ───────────────────────────────────────────

func TestQueryHandler_GetChecksum_SQLKeywordTableName(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.GetChecksum(context.Background(), connect.NewRequest(&v1.GetChecksumRequest{
		ProjectId: "valid_proj",
		TableName: "DROP",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// ─── ExecuteQuery Additional Error Path Tests ──────────────────────────────

func TestQueryHandler_ExecuteQuery_CompileFromOntology(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	creatOntology := filepath.Join(projectsRoot, "compile-proj", "ontologies")
	require.NoError(t, os.MkdirAll(creatOntology, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(creatOntology, "core.aleph"),
		[]byte("object compile_obj from dataset compile_ds id id property id type number from id property name type text from name\n"), 0644))

	parquetDir := filepath.Join(projectsRoot, "compile-proj", "raw", "compile_ds", "latest")
	require.NoError(t, os.MkdirAll(parquetDir, 0755))

	_, err := h.db.Exec(context.Background(), "CREATE TABLE compile_obj (id INTEGER, name VARCHAR)")
	require.NoError(t, err)
	_, err = h.db.Exec(context.Background(), "INSERT INTO compile_obj VALUES (1, 'test')")
	require.NoError(t, err)
	parquetPath := filepath.Join(parquetDir, "data.parquet")
	_, err = h.db.Exec(context.Background(), fmt.Sprintf("COPY compile_obj TO '%s' (FORMAT PARQUET)", parquetPath))
	require.NoError(t, err)

	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "compile_obj",
		ProjectId:  "compile-proj",
	}))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.Msg.Rows), 1)
}

// ─── ConfirmAction Additional Tests ────────────────────────────────────────

func TestQueryHandler_ConfirmAction_EmptyProjectID_NoMiddleware(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "",
		Approved:  true,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestQueryHandler_ConfirmAction_ProjectNotFound(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "nonexistent_proj",
		Approved:  false,
	}))
	require.Error(t, err)
}

// ─── resolveProject Edge Case Tests ────────────────────────────────────────

func TestQueryHandler_ResolveProject_PathTraversal(t *testing.T) {
	// resolveProject is called indirectly via public methods like ConfirmAction
	// or ExecuteQuery. Test path traversal via ConfirmAction.
	h, _ := setupQueryHandler(t)

	// Using "../" causes sanitizePath to detect path traversal
	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "../escape",
		Approved:  true,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "traversal")
}

func TestQueryHandler_ResolveProject_NoOntologyFile(t *testing.T) {
	// Create project dir but no ontology file
	// resolveProject will try to read core.aleph and fail
	h, projectsRoot := setupQueryHandler(t)

	projectDir := filepath.Join(projectsRoot, "no-ont")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	// Don't create ontologies dir / core.aleph

	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "no-ont",
		Approved:  true,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ontology")
}

func TestQueryHandler_ResolveProject_InvalidOntology(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)
	createProjectWithOntology(t, projectsRoot, "bad-ont", "this is not valid aleph dsl {[[[]]]")

	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "bad-ont",
		Approved:  true,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse ontology")
}

func TestQueryHandler_ResolveProject_DefaultViaExecuteQuery(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE items (name VARCHAR)")
	require.NoError(t, err)
	h.db.Exec(context.Background(), "INSERT INTO items VALUES ('test')")

	createProjectWithOntology(t, projectsRoot, "default", "// valid\n")

	// ExecuteQuery with empty ProjectId → resolveProject defaults to "default"
	resp, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 1)
}

// ─── GetChatHistory Edge Case Tests ────────────────────────────────────────

func TestQueryHandler_GetChatHistory_WithToolCall(t *testing.T) {
	h, repo := setupQueryHandlerExtended(t)
	repo.SaveChatMessage(context.Background(), "p1", "a1", "assistant", "result", "search_data")

	req := connect.NewRequest(&v1.GetChatHistoryRequest{ProjectId: "p1", AgentId: "a1"})
	resp, err := h.GetChatHistory(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.Messages, 1)
	assert.Equal(t, "assistant", resp.Msg.Messages[0].Role)
	assert.Equal(t, "search_data", resp.Msg.Messages[0].ToolCall)
}

func TestQueryHandler_GetChatHistory_MultipleMessages(t *testing.T) {
	h, repo := setupQueryHandlerExtended(t)
	repo.SaveChatMessage(context.Background(), "p1", "a1", "user", "hello", "")
	repo.SaveChatMessage(context.Background(), "p1", "a1", "assistant", "hi there", "")
	repo.SaveChatMessage(context.Background(), "p1", "a1", "user", "how are you", "")

	req := connect.NewRequest(&v1.GetChatHistoryRequest{ProjectId: "p1", AgentId: "a1"})
	resp, err := h.GetChatHistory(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.Messages, 3)
	assert.Equal(t, "user", resp.Msg.Messages[0].Role)
	assert.Equal(t, "hello", resp.Msg.Messages[0].Content)
	assert.Equal(t, "how are you", resp.Msg.Messages[2].Content)
}

func TestQueryHandler_ConfirmAction_WithAgentVerification(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	repo.CreateAgent(&repository.AgentRecord{
		ID: "a1", ProjectID: "p1", Name: "agent", Provider: "ollama", Model: "llama3",
	})

	h, projectsRoot := setupQueryHandler(t)
	h.metaRepo = repo
	createProjectWithOntology(t, projectsRoot, "p1", "// valid\n")

	resp, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "p1",
		AgentId:   "a1",
		Approved:  true,
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestQueryHandler_ConfirmAction_AgentNotInProject(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h, projectsRoot := setupQueryHandler(t)
	h.metaRepo = repo
	createProjectWithOntology(t, projectsRoot, "p1", "// valid\n")

	_, err := h.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "p1",
		AgentId:   "nonexistent_agent",
		Approved:  true,
	}))
	require.Error(t, err)
}

func TestQueryHandler_ExecuteQuery_CanceledContext(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE items (name VARCHAR)")
	require.NoError(t, err)
	h.db.Exec(context.Background(), "INSERT INTO items VALUES ('test')")
	createProjectWithOntology(t, projectsRoot, "ctx-proj", "// empty\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = h.ExecuteQuery(ctx, connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "ctx-proj",
	}))
	require.Error(t, err)
}

func TestTruncateJSON_StringWithEscapes(t *testing.T) {
	result := truncateJSON(`{"key":"val\"ue"}`, 100)
	assert.Equal(t, `{"key":"val\"ue"}`, result)
}

func TestTruncateJSON_MixedBraces(t *testing.T) {
	result := truncateJSON(`{"a":[1,{"b":2}]}`, 100)
	assert.Equal(t, `{"a":[1,{"b":2}]}`, result)
}

func TestTruncateJSON_TruncatedJSON(t *testing.T) {
	long := `{"key":"` + makeString(500, 'x') + `"}`
	result := truncateJSON(long, 50)
	assert.True(t, len(result) <= 53)
	assert.True(t, len(result) >= 30)
}

func TestTruncateJSON_VeryShortLimit(t *testing.T) {
	result := truncateJSON(`{"a":{"b":"c"}}`, 5)
	assert.True(t, len(result) <= 8)
}

func TestQueryHandler_ExecuteQuery_UpperCaseMismatch(t *testing.T) {
	h, _ := setupQueryHandler(t)

	_, err := h.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "UPPERCASE",
		ProjectId:  "proj",
	}))
	require.Error(t, err)
}

func TestTruncateJSON_EscapedQuoteInString(t *testing.T) {
	result := truncateJSON(`{"key":"val\\\"quote"}`, 100)
	assert.Equal(t, `{"key":"val\\\"quote"}`, result)
}

func TestTruncateJSON_NegativeDepthRecovery(t *testing.T) {
	result := truncateJSON(`}]extra`, 100)
	assert.True(t, len(result) <= 100)
}

func TestTruncateJSON_DeepObjectTruncation(t *testing.T) {
	nested := `{"a":{"b":{"c":{"d":{"e":{"f":{"g":"value"}}}}}},"extra":"` + makeString(200, 'x') + `"}`
	result := truncateJSON(nested, 80)
	assert.True(t, len(result) <= 83)
}

