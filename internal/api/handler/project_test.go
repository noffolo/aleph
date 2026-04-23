package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/storage"
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

	_, err := h.db.Exec("CREATE TABLE sample_data (id INTEGER, name VARCHAR)")
	require.NoError(t, err)

	resp, err := h.EmergeOntology(context.Background(), connect.NewRequest(&v1.EmergeOntologyRequest{}))
	require.NoError(t, err)
	assert.Contains(t, resp.Msg.AlephDefinition, "sample_data")
	assert.Contains(t, resp.Msg.AlephDefinition, "property name type text from name")
}
