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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
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

	proposeBody := map[string]interface{}{
		"project_id":          "test-proj",
		"parent_version_id":   "",
		"aleph_definition":    "object User\n  from dataset users\n  id id\n  property name type text from name\n",
		"diff_json":           `{"add_object":"User","parent_hash":"base"}`,
		"source_description":  "test emergence",
		"rationale":           "adding User entity",
		"confidence":          0.95,
	}
	proposeJSON, err := json.Marshal(proposeBody)
	require.NoError(t, err)

	proposeReq := httptest.NewRequest(http.MethodPost, "/api/v1/ontology/propose", bytes.NewReader(proposeJSON))
	proposeReq.Header.Set("Content-Type", "application/json")
	proposeRR := httptest.NewRecorder()
	h.NegotiatePropose(proposeRR, proposeReq)

	require.Equal(t, http.StatusOK, proposeRR.Code, "propose should succeed")

	var proposeResp map[string]interface{}
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

	var acceptResp map[string]interface{}
	require.NoError(t, json.Unmarshal(acceptRR.Body.Bytes(), &acceptResp))
	assert.Equal(t, "accepted", acceptResp["status"], "status should be accepted")
	assert.Equal(t, versionID, acceptResp["version_id"], "version_id should match")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/ontology/versions?project_id=test-proj", nil)
	listRR := httptest.NewRecorder()
	h.NegotiateList(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code, "list should succeed")

	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(listRR.Body.Bytes(), &listResp))

	versions, ok := listResp["versions"].([]interface{})
	require.True(t, ok, "versions should be an array")
	require.Len(t, versions, 1, "should have exactly one version")

	versionMap, ok := versions[0].(map[string]interface{})
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
