package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgres_InvalidDSN(t *testing.T) {
	t.Parallel()
	pg, err := NewPostgres("invalid://bad-dsn")
	assert.Error(t, err)
	assert.Nil(t, pg)
}

func TestPostgres_Lifecycle(t *testing.T) {
	dsn := os.Getenv("ALE_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ALE_POSTGRES_DSN not set — skipping Postgres lifecycle test")
	}

	pg, err := NewPostgres(dsn)
	require.NoError(t, err)
	require.NotNil(t, pg)

	db := pg.DB()
	require.NotNil(t, db)

	err = db.Ping()
	assert.NoError(t, err)

	err = pg.Close()
	assert.NoError(t, err)
}

func TestPostgres_DB_ReturnsConn(t *testing.T) {
	dsn := os.Getenv("ALE_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ALE_POSTGRES_DSN not set")
	}

	pg, err := NewPostgres(dsn)
	require.NoError(t, err)
	defer pg.Close()

	db := pg.DB()
	assert.NotNil(t, db)
}

func TestPostgres_Close_Idempotent(t *testing.T) {
	dsn := os.Getenv("ALE_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ALE_POSTGRES_DSN not set")
	}

	pg, err := NewPostgres(dsn)
	require.NoError(t, err)

	err = pg.Close()
	assert.NoError(t, err)

	// Close again — should not panic
	err = pg.Close()
	assert.Error(t, err) // double close is an error with sql.DB
}
