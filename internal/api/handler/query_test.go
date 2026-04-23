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

func setupQueryHandler(t *testing.T) (*QueryHandler, string) {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	h := NewQueryHandler(db, projectsRoot, nil, nil, nil)
	return h, projectsRoot
}

func createProjectWithOntology(t *testing.T, projectsRoot, projectID, ontology string) {
	t.Helper()
	ontDir := filepath.Join(projectsRoot, projectID, "ontologies")
	require.NoError(t, os.MkdirAll(ontDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ontDir, "core.aleph"), []byte(ontology), 0644))
}

func TestQueryHandler_ExecuteQuery_FallbackToTable(t *testing.T) {
	h, projectsRoot := setupQueryHandler(t)

	_, err := h.db.Exec("CREATE TABLE items (name VARCHAR, qty INTEGER)")
	require.NoError(t, err)
	_, err = h.db.Exec("INSERT INTO items VALUES ('apple', 10), ('banana', 5)")
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

	_, err := h.db.Exec("CREATE TABLE limited (val INTEGER)")
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		h.db.Exec("INSERT INTO limited VALUES (?)", i)
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

	_, err := h.db.Exec("CREATE TABLE orders (id INTEGER, amount DOUBLE, status VARCHAR)")
	require.NoError(t, err)
	_, err = h.db.Exec("INSERT INTO orders VALUES (1, 99.5, 'open'), (2, 150.0, 'closed')")
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

	_, err := h.db.Exec("CREATE TABLE global_test (k VARCHAR)")
	require.NoError(t, err)
	h.db.Exec("INSERT INTO global_test VALUES ('val')")

	createProjectWithOntology(t, projectsRoot, "gq-proj", "// empty\n")

	resp, err := h.GlobalQuery(context.Background(), connect.NewRequest(&v1.GlobalQueryRequest{
		ObjectType: "global_test",
		ProjectId:  "gq-proj",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 1)
}
