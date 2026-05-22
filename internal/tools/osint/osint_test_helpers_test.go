package osint

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/require"
)

func newMetadataRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE system_tools (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			code TEXT,
			category TEXT,
			version TEXT,
			health_status TEXT,
			source_type TEXT
		)
	`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}
