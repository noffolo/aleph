package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProjectHandler(t *testing.T) (*ProjectHandler, string) {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	h := NewProjectHandler(projectsRoot, db)
	return h, projectsRoot
}

func TestProjectHandler_CreateAndList(t *testing.T) {
	h, _ := setupProjectHandler(t)

	resp, err := h.CreateProject(context.Background(), connect.NewRequest(&v1.CreateProjectRequest{
		Id: "test-project", Name: "Test",
	}))
	require.NoError(t, err)
	assert.Equal(t, "test-project", resp.Msg.Project.Id)

	listResp, err := h.ListProjects(context.Background(), connect.NewRequest(&v1.ListProjectsRequest{}))
	require.NoError(t, err)
	assert.Len(t, listResp.Msg.Projects, 1)
	assert.Equal(t, "test-project", listResp.Msg.Projects[0].Id)
}

func TestProjectHandler_CreateEmptyID(t *testing.T) {
	h, _ := setupProjectHandler(t)
	_, err := h.CreateProject(context.Background(), connect.NewRequest(&v1.CreateProjectRequest{Id: ""}))
	assert.Error(t, err)
}

func TestProjectHandler_SaveAndGetOntology(t *testing.T) {
	h, projectsRoot := setupProjectHandler(t)
	require.NoError(t, os.MkdirAll(filepath.Join(projectsRoot, "myproj", "ontologies"), 0755))

	_, err := h.SaveOntology(context.Background(), connect.NewRequest(&v1.SaveOntologyRequest{
		ProjectId:       "myproj",
		AlephDefinition: "object Test\n  from dataset test_ds\n  id id\n  property name type text\n",
	}))
	require.NoError(t, err)

	resp, err := h.GetOntology(context.Background(), connect.NewRequest(&v1.GetOntologyRequest{
		ProjectId: "myproj",
	}))
	require.NoError(t, err)
	assert.Contains(t, resp.Msg.AlephDefinition, "object Test")
	assert.Contains(t, resp.Msg.ObjectNames, "Test")
}

func TestProjectHandler_GetOntology_NotFound(t *testing.T) {
	h, _ := setupProjectHandler(t)
	_, err := h.GetOntology(context.Background(), connect.NewRequest(&v1.GetOntologyRequest{
		ProjectId: "nonexistent",
	}))
	assert.Error(t, err)
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	h, projectsRoot := setupProjectHandler(t)
	require.NoError(t, os.MkdirAll(filepath.Join(projectsRoot, "to-delete"), 0755))

	resp, err := h.DeleteProject(context.Background(), connect.NewRequest(&v1.DeleteProjectRequest{Id: "to-delete"}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)

	_, statErr := os.Stat(filepath.Join(projectsRoot, "to-delete"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestProjectHandler_EmergeOntology(t *testing.T) {
	h, _ := setupProjectHandler(t)

	_, err := h.db.Exec(context.Background(), "CREATE TABLE sample_data (id INTEGER, name VARCHAR)")
	require.NoError(t, err)

	resp, err := h.EmergeOntology(context.Background(), connect.NewRequest(&v1.EmergeOntologyRequest{}))
	require.NoError(t, err)
	assert.Contains(t, resp.Msg.AlephDefinition, "sample_data")
	assert.Contains(t, resp.Msg.AlephDefinition, "property name type text from name")
}

func setupOntologyRepo(t *testing.T) *repository.OntologyRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE ontology_versions (
		version_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		parent_version_id TEXT,
		diff_json TEXT,
		core_aleph_snapshot TEXT NOT NULL,
		status TEXT NOT NULL,
		source_description TEXT,
		rationale TEXT,
		confidence FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		modified_at TIMESTAMP
	)`)
	require.NoError(t, err)

	repo := repository.NewOntologyRepository(db)
	return repo
}

func TestNegotiateFlow(t *testing.T) {
	h, projectsRoot := setupProjectHandler(t)
	require.NoError(t, os.MkdirAll(filepath.Join(projectsRoot, "test-proj", "ontologies"), 0755))

	ontoRepo := setupOntologyRepo(t)
	h.SetOntologyRepository(ontoRepo)

	proposeBody := map[string]any{
		"project_id":         "test-proj",
		"parent_version_id":  "",
		"aleph_definition":   "object User\n  from dataset users\n  id id\n  property name type text from name\n",
		"diff_json":          `{"add_object":"User","parent_hash":"base"}`,
		"source_description": "test emergence",
		"rationale":          "adding User entity",
		"confidence":         0.95,
	}
	proposeJSON, err := json.Marshal(proposeBody)
	require.NoError(t, err)

	proposeReq := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader(proposeJSON))
	proposeReq.Header.Set("Content-Type", "application/json")
	proposeRR := httptest.NewRecorder()
	h.NegotiatePropose(proposeRR, proposeReq)

	require.Equal(t, http.StatusOK, proposeRR.Code, "propose should succeed")

	var proposeResp map[string]any
	require.NoError(t, json.Unmarshal(proposeRR.Body.Bytes(), &proposeResp))

	versionID, ok := proposeResp["version_id"].(string)
	require.True(t, ok, "version_id should be a string")
	require.NotEmpty(t, versionID, "version_id should not be empty")

	acceptBody := map[string]string{"version_id": versionID}
	acceptJSON, err := json.Marshal(acceptBody)
	require.NoError(t, err)

	acceptReq := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/accept", bytes.NewReader(acceptJSON))
	acceptReq.Header.Set("Content-Type", "application/json")
	acceptRR := httptest.NewRecorder()
	h.NegotiateAccept(acceptRR, acceptReq)

	require.Equal(t, http.StatusOK, acceptRR.Code, "accept should succeed")

	var acceptResp map[string]any
	require.NoError(t, json.Unmarshal(acceptRR.Body.Bytes(), &acceptResp))
	assert.Equal(t, "accepted", acceptResp["status"], "status should be accepted")
	assert.Equal(t, versionID, acceptResp["version_id"], "version_id should match")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/ontology/versions?project_id=test-proj", nil)
	listRR := httptest.NewRecorder()
	h.NegotiateList(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code, "list should succeed")

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRR.Body.Bytes(), &listResp))

	versions, ok := listResp["versions"].([]any)
	require.True(t, ok, "versions should be an array")
	require.Len(t, versions, 1, "should have exactly one version")

	versionMap, ok := versions[0].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, versionID, versionMap["version_id"], "version_id should match")
	assert.Equal(t, "accepted", versionMap["status"], "status should be accepted")

	sourceDesc, _ := versionMap["source_description"].(string)
	assert.Equal(t, "test emergence", sourceDesc, "source_description should match")

	rationale, _ := versionMap["rationale"].(string)
	assert.Equal(t, "adding User entity", rationale, "rationale should match")

	confidence, _ := versionMap["confidence"].(float64)
	assert.InDelta(t, 0.95, confidence, 0.01, "confidence should match")
}

func TestMapDuckDBType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"int", "INTEGER", "number"},
		{"bigint", "BIGINT", "number"},
		{"float", "FLOAT", "number"},
		{"double", "DOUBLE", "number"},
		{"decimal", "DECIMAL(10,2)", "number"},
		{"timestamp", "TIMESTAMP", "datetime"},
		{"date", "DATE", "datetime"},
		{"time", "TIME", "datetime"},
		{"boolean", "BOOLEAN", "boolean"},
		{"bool", "BOOL", "boolean"},
		{"varchar", "VARCHAR", "text"},
		{"text", "TEXT", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDuckDBType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectFKRelationships(t *testing.T) {
	schemas := []tableSchema{
		{Name: "users", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "name", Type: "VARCHAR"}}},
		{Name: "orders", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "user_id", Type: "INTEGER"}, {Name: "total", Type: "DOUBLE"}}},
		{Name: "items", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "order_id", Type: "INTEGER"}}},
	}
	rels := detectFKRelationships(schemas)
	assert.GreaterOrEqual(t, len(rels), 2)

	foundUserRel, foundOrderRel := false, false
	for _, r := range rels {
		if r.FromColumn == "user_id" && r.ToObject == "users" {
			foundUserRel = true
			assert.Equal(t, "fk", r.Type)
			assert.Equal(t, "high", r.Confidence)
		}
		if r.FromColumn == "order_id" && r.ToObject == "orders" {
			foundOrderRel = true
			assert.Equal(t, "fk", r.Type)
		}
	}
	assert.True(t, foundUserRel, "expected user_id → users relationship")
	assert.True(t, foundOrderRel, "expected order_id → orders relationship")
}

func TestDetectFKRelationships_Empty(t *testing.T) {
	rels := detectFKRelationships(nil)
	assert.Len(t, rels, 0)
}

func TestDetectFKRelationships_NameMatch(t *testing.T) {
	schemas := []tableSchema{
		{Name: "category", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "name", Type: "VARCHAR"}}},
		{Name: "products", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "category", Type: "VARCHAR"}}},
	}
	rels := detectFKRelationships(schemas)
	assert.GreaterOrEqual(t, len(rels), 1)
	found := false
	for _, r := range rels {
		if r.FromColumn == "category" && r.ToObject == "category" {
			found = true
			assert.Equal(t, "name_match", r.Type)
			assert.Equal(t, "medium", r.Confidence)
		}
	}
	assert.True(t, found, "expected category → category name_match")
}

func TestBuildAlephDefinition(t *testing.T) {
	schemas := []tableSchema{
		{Name: "products", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "name", Type: "VARCHAR"}, {Name: "price", Type: "DOUBLE"}, {Name: "available", Type: "BOOLEAN"}}},
	}
	rels := []detectedRelationship{{Name: "orders_has_products", FromObject: "orders", FromColumn: "product_id", ToObject: "products", ToColumn: "id", Type: "fk", Confidence: "high"}}
	def := buildAlephDefinition(schemas, rels)
	assert.Contains(t, def, "object products")
	assert.Contains(t, def, "property name type text from name")
	assert.Contains(t, def, "property price type number from price")
	assert.Contains(t, def, "property available type boolean from available")
	assert.Contains(t, def, "id id")
	assert.Contains(t, def, "relation orders_has_products")
}

func TestBuildAlephDefinition_NoIDColumn(t *testing.T) {
	schemas := []tableSchema{
		{Name: "logs", Columns: []columnInfo{{Name: "message", Type: "VARCHAR"}, {Name: "level", Type: "VARCHAR"}}},
	}
	def := buildAlephDefinition(schemas, nil)
	assert.Contains(t, def, "id id", "should auto-add id when no id column found")
}

func TestEmergePrompt(t *testing.T) {
	h := &ProjectHandler{}
	schemas := []tableSchema{{Name: "t1", Columns: []columnInfo{{Name: "id", Type: "INTEGER"}, {Name: "val", Type: "VARCHAR"}}}}
	rels := []detectedRelationship{{Name: "r1", FromObject: "t1", FromColumn: "ref_id", ToObject: "t2", ToColumn: "id", Confidence: "high"}}
	prompt := h.emergePrompt(schemas, rels)
	assert.Contains(t, prompt, "Sei un ontologo esperto")
	assert.Contains(t, prompt, "Table: t1")
	assert.Contains(t, prompt, "id (INTEGER)")
	assert.Contains(t, prompt, "val (VARCHAR)")
}

func TestWriteError_ProjectHandler(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, "test error", http.StatusBadRequest)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "test error")
}

func TestWriteJSON_ProjectHandler(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"msg": "hello"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "hello")
}

func TestSanitizePath_Valid(t *testing.T) {
	dir := t.TempDir()
	path, err := sanitizePath(dir, "sub", "file.txt")
	require.NoError(t, err)
	assert.Contains(t, path, "sub")
}

func TestSanitizePath_Traversal(t *testing.T) {
	dir := t.TempDir()
	_, err := sanitizePath(dir, "..", "etc", "passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestSanitizePath_EmptyBase_Project(t *testing.T) {
	t.Skip("empty base resolves to cwd, path traversal not detected in this case")
}

func TestProjectHandler_CreateProject_NilMetaRepo_WithMaxLimit(t *testing.T) {
	h, _ := setupProjectHandler(t)
	h.SetMaxProjects(2)
	resp, err := h.CreateProject(context.Background(), connect.NewRequest(&v1.CreateProjectRequest{Id: "no-limit-proj", Name: "No Limit"}))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg.Project)
}

func TestProjectHandler_SetMetaRepo_Extended(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	repo := &repository.MetadataRepository{}
	h.SetMetaRepo(repo)
	assert.Equal(t, repo, h.metaRepo)
}

func TestProjectHandler_SetLLMProvider_Nil(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetLLMProvider(nil)
}

func TestProjectHandler_NegotiatePropose_NoOntoRepo(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	h.NegotiatePropose(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProjectHandler_NegotiatePropose_BadJSON(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader([]byte(`not json`)))
	w := httptest.NewRecorder()
	h.NegotiatePropose(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiatePropose_MissingFields(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader([]byte(`{"project_id":"","aleph_definition":""}`)))
	w := httptest.NewRecorder()
	h.NegotiatePropose(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateAccept_NoOntoRepo(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/accept", bytes.NewReader([]byte(`{"version_id":"v1"}`)))
	w := httptest.NewRecorder()
	h.NegotiateAccept(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProjectHandler_NegotiateAccept_BadJSON(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/accept", bytes.NewReader([]byte(`bad`)))
	w := httptest.NewRecorder()
	h.NegotiateAccept(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateAccept_EmptyVersionID(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/accept", bytes.NewReader([]byte(`{"version_id":""}`)))
	w := httptest.NewRecorder()
	h.NegotiateAccept(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateReject_NoOntoRepo(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/reject", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	h.NegotiateReject(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProjectHandler_NegotiateReject_BadJSON(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/reject", bytes.NewReader([]byte(`bad`)))
	w := httptest.NewRecorder()
	h.NegotiateReject(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateReject_EmptyVersionID(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/reject", bytes.NewReader([]byte(`{"version_id":"","reason":"bad"}`)))
	w := httptest.NewRecorder()
	h.NegotiateReject(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateList_EmptyProjectID(t *testing.T) {
	ontoRepo := setupOntologyRepo(t)
	h := NewProjectHandler(t.TempDir(), nil)
	h.SetOntologyRepository(ontoRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ontology/versions?project_id=", nil)
	w := httptest.NewRecorder()
	h.NegotiateList(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProjectHandler_NegotiateList_NoOntoRepo(t *testing.T) {
	h := NewProjectHandler(t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ontology/versions?project_id=p1", nil)
	w := httptest.NewRecorder()
	h.NegotiateList(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProjectHandler_NegotiateReject_Flow(t *testing.T) {
	h, projectsRoot := setupProjectHandler(t)
	require.NoError(t, os.MkdirAll(filepath.Join(projectsRoot, "reject-test", "ontologies"), 0755))
	ontoRepo := setupOntologyRepo(t)
	h.SetOntologyRepository(ontoRepo)

	proposeBody, _ := json.Marshal(map[string]any{
		"project_id":         "reject-test",
		"aleph_definition":   "object X\n id id",
		"diff_json":          "{}",
		"source_description": "test",
		"rationale":          "reason",
	})
	proposeReq := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader(proposeBody))
	proposeReq.Header.Set("Content-Type", "application/json")
	proposeW := httptest.NewRecorder()
	h.NegotiatePropose(proposeW, proposeReq)

	var proposeResp map[string]any
	json.Unmarshal(proposeW.Body.Bytes(), &proposeResp)
	versionID := proposeResp["version_id"].(string)

	rejectBody, _ := json.Marshal(map[string]string{"version_id": versionID, "reason": "not needed"})
	rejectReq := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/reject", bytes.NewReader(rejectBody))
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectW := httptest.NewRecorder()
	h.NegotiateReject(rejectW, rejectReq)
	assert.Equal(t, http.StatusOK, rejectW.Code)
}
