package repository

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataRepository_CreateAndValidateAPIKey(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE system_api_keys (id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT)`)
	require.NoError(t, err)

	repo := &MetadataRepository{db: db}

	err = repo.CreateAPIKey("key1", "proj1", "test-label", "hashed-secret")
	require.NoError(t, err)

	pid, err := repo.ValidateAPIKey("hashed-secret")
	assert.NoError(t, err)
	assert.Equal(t, "proj1", pid)

	_, err = repo.ValidateAPIKey("bad-secret")
	assert.Error(t, err)
}
