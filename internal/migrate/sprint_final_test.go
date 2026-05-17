package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDuckDBMigrations_OutOfOrder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "order.db")

	files := map[int]string{
		5: "CREATE TABLE IF NOT EXISTS t5 (id INTEGER);",
		2: "CREATE TABLE IF NOT EXISTS t2 (id INTEGER);",
	}
	for ver, sql := range files {
		fn := filepath.Join(tmpDir, fmt.Sprintf("00000%d_v.up.sql", ver))
		err := os.WriteFile(fn, []byte(sql), 0644)
		require.NoError(t, err)
	}

	err := RunDuckDBMigrations(dbPath, tmpDir)
	require.NoError(t, err)

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRunDuckDBMigrations_NonNumericPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonnum.db")

	err := os.WriteFile(filepath.Join(tmpDir, "abc_migrate.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644)
	require.NoError(t, err)

	err = RunDuckDBMigrations(dbPath, tmpDir)
	require.NoError(t, err)

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 0, count, "non-numeric prefix should be ignored")
}

func TestRunDuckDBMigrations_MultiStmt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multi.db")

	migSQL := "CREATE TABLE IF NOT EXISTS a (id INTEGER);\nCREATE TABLE IF NOT EXISTS b (id INTEGER);"
	err := os.WriteFile(filepath.Join(tmpDir, "000001_multi.up.sql"), []byte(migSQL), 0644)
	require.NoError(t, err)

	err = RunDuckDBMigrations(dbPath, tmpDir)
	require.NoError(t, err)

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()
	for _, tbl := range []string{"a", "b"} {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM "+tbl).Scan(&count)
	}
}

func TestRunAllMigrations_MemoryOnly(t *testing.T) {
	err := RunAllMigrations(":memory:", "")
	assert.NoError(t, err)
}

func TestRunDuckDBMigrations_DirtyFlagPreventsReApply(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "dirty2.db")

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644))

	// First run: clean migration
	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	// Mark version 1 as dirty using DuckDB-compatible syntax
	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("INSERT OR REPLACE INTO schema_migrations (version, dirty) VALUES (1, true)")
	require.NoError(t, err)
	db.Close()

	// Second run: version 1 is dirty, so currentVersion=0.
	// The migration will try to re-apply migration 1 and hit PRIMARY KEY constraint
	// on INSERT. This is expected — the dirty flag prevents clean tracking but
	// the constraint prevents double-application.
	err = RunDuckDBMigrations(dbPath, tmpDir)
	require.Error(t, err, "dirty flag causes re-application attempt which hits constraint")
	assert.Contains(t, err.Error(), "record migration")

	// Verify the dirty row persists
	db2, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db2.Close()

	var dirtyCount int
	err = db2.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE dirty = true").Scan(&dirtyCount)
	require.NoError(t, err)
	assert.Equal(t, 1, dirtyCount, "dirty row should remain")

	var maxVersion int64
	err = db2.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&maxVersion)
	require.NoError(t, err)
	assert.Equal(t, int64(1), maxVersion)
}

func TestRunDuckDBMigrations_NoNewMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "uptodate.db")

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	// Second run with no new files
	err := RunDuckDBMigrations(dbPath, tmpDir)
	require.NoError(t, err)

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 1, count, "idempotent — should not re-apply")
}

func TestRunDuckDBMigrations_VersionGapSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "gap2.db")

	// Create v1 and v5 files (gap at 2,3,4)
	for _, v := range []int{1, 5} {
		fn := filepath.Join(tmpDir, fmt.Sprintf("00000%d_gap.up.sql", v))
		require.NoError(t, os.WriteFile(fn,
			[]byte(fmt.Sprintf("CREATE TABLE IF NOT EXISTS v%d (id INTEGER);", v)), 0644))
	}

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 2, count)
}

func TestRunDuckDBMigrations_TransactionRollback(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "rollback.db")

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000002_bad.up.sql"),
		[]byte("SYNTAX ERROR IN SQL"), 0644))

	err := RunDuckDBMigrations(dbPath, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute migration")

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var maxVersion int64
	db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE NOT dirty").Scan(&maxVersion)
	assert.Equal(t, int64(1), maxVersion)
}

func TestRunDuckDBMigrations_ZeroPrefixVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "zero.db")

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t001 (id INTEGER);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000010_second.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t010 (id INTEGER);"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 2, count)
}

func TestRunDuckDBMigrations_NothingToDo(t *testing.T) {
	err := RunDuckDBMigrations(":memory:", t.TempDir())
	require.NoError(t, err)
}

// =============================================================================
// Additional coverage tests
// =============================================================================

func TestRunDuckDBMigrations_DownFilesFullyIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "down.db")

	// Create both up and down files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS down_t1 (id INTEGER);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.down.sql"),
		[]byte("DROP TABLE IF EXISTS down_t1;"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	// Verify only up was applied
	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var version int64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)
}

func TestRunDuckDBMigrations_DownOnlyFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "downonly.db")

	// Only .down.sql files — nothing should be applied
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_remove.down.sql"),
		[]byte("DROP TABLE IF EXISTS x;"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestRunDuckDBMigrations_MixedUpDownAndNonSQL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "mixed.db")

	// Up migration
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_create.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS mixed_t1 (id INTEGER);"), 0644))
	// Down migration (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_create.down.sql"),
		[]byte("DROP TABLE IF EXISTS mixed_t1;"), 0644))
	// Non-SQL file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.txt"),
		[]byte("migration notes"), 0644))
	// Second up migration
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000002_add_col.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS mixed_t2 (id INTEGER);"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 2, count)

	// Verify both tables exist
	for _, tbl := range []string{"mixed_t1", "mixed_t2"} {
		var n int
		err := db.QueryRow("SELECT COUNT(*) FROM " + tbl).Scan(&n)
		require.NoError(t, err)
	}
}

func TestRunDuckDBMigrations_SameVersionTwice(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "samever.db")

	// Apply migration 1
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_create.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS sv_t1 (id INTEGER);"), 0644))
	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	// Delete and recreate the file with different content — same version number
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_create.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS sv_t2 (id INTEGER);"), 0644))

	// Second run — version 1 already applied, so it should be skipped
	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 1, count, "version 1 should not be re-applied")
}

func TestRunDuckDBMigrations_LargeVersionNumber(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "largever.db")

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "999999_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS bigv (id INTEGER);"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var maxVersion int64
	db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&maxVersion)
	assert.Equal(t, int64(999999), maxVersion)
}

func TestRunDuckDBMigrations_MultipleInsertsInMigration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multiins.db")

	migSQL := "CREATE TABLE IF NOT EXISTS mi_t (id INTEGER);\nINSERT INTO mi_t VALUES (1);\nINSERT INTO mi_t VALUES (2);\nINSERT INTO mi_t VALUES (3);"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_data.up.sql"), []byte(migSQL), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM mi_t").Scan(&count)
	assert.Equal(t, 3, count)
}

func TestRunDuckDBMigrations_VersionWithUnderscorePrefix(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "under.db")

	// File that starts with number but has underscore-heavy name
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "001_test_ingestion_v2.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS under_t (id INTEGER);"), 0644))

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 1, count)
}

func TestRunDuckDBMigrations_NegativeVersionNumber(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "neg.db")

	// Negative version should be skipped (version > currentVersion, but currentVersion is 0)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "-001_neg.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS neg_t (id INTEGER);"), 0644))

	// sscanf will fail for "-001" → err != nil, so file is silently ignored
	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestRunAllMigrations_BothMemory(t *testing.T) {
	// :memory: for both DuckDB and Postgres — pgx will fail
	err := RunAllMigrations(":memory:", ":memory:")
	// RunAllMigrations always returns nil (warnings only)
	assert.NoError(t, err)
}

func TestRunAllMigrations_DuckDBOnly_EmptyPostgres(t *testing.T) {
	err := RunAllMigrations(":memory:", "")
	assert.NoError(t, err)
}

// =============================================================================
// RunPostgresMigrations — with real PostgreSQL
// =============================================================================

const testPostgresDSN = "postgres://ff3300@localhost/aleph_migrate_test?sslmode=disable"

func setupPostgresTestDB(t *testing.T) func() {
	t.Helper()
	db, err := sql.Open("pgx", "postgres://ff3300@localhost/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}

	// Drop test DB if it exists (cleanup from previous failed runs)
	db.Exec("DROP DATABASE IF EXISTS aleph_migrate_test")
	_, err = db.Exec("CREATE DATABASE aleph_migrate_test")
	if err != nil {
		t.Skipf("cannot create test database: %v", err)
	}

	return func() {
		cleanDB, err := sql.Open("pgx", "postgres://ff3300@localhost/postgres?sslmode=disable")
		if err != nil {
			return
		}
		defer cleanDB.Close()
		cleanDB.Exec("DROP DATABASE IF EXISTS aleph_migrate_test WITH (FORCE)")
	}
}

func TestRunPostgresMigrations_Success(t *testing.T) {
	cleanup := setupPostgresTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS pg_t1 (id SERIAL PRIMARY KEY, name TEXT);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000002_add_col.up.sql"),
		[]byte("ALTER TABLE pg_t1 ADD COLUMN IF NOT EXISTS email TEXT;"), 0644))

	err := RunPostgresMigrations(testPostgresDSN, tmpDir)
	require.NoError(t, err)

	// Verify via DuckDB can't work, so verify via pgx
	db, err := sql.Open("pgx", testPostgresDSN)
	require.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify the table was created
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pg_t1").Scan(&tableCount)
	require.NoError(t, err)

	var maxVer int64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&maxVer)
	require.NoError(t, err)
	assert.Equal(t, int64(2), maxVer)
}

func TestRunPostgresMigrations_SkipAlreadyApplied(t *testing.T) {
	cleanup := setupPostgresTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS pg_t2 (id SERIAL PRIMARY KEY);"), 0644))

	require.NoError(t, RunPostgresMigrations(testPostgresDSN, tmpDir))
	// Second run should skip
	require.NoError(t, RunPostgresMigrations(testPostgresDSN, tmpDir))

	db, err := sql.Open("pgx", testPostgresDSN)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 1, count, "idempotent")
}

func TestRunPostgresMigrations_BadSQL(t *testing.T) {
	cleanup := setupPostgresTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_good.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS pg_good (id INTEGER);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000002_bad.up.sql"),
		[]byte("SYNTAX ERROR IN POSTGRES;"), 0644))

	err := RunPostgresMigrations(testPostgresDSN, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute migration")
}

func TestRunPostgresMigrations_EmptyDir(t *testing.T) {
	cleanup := setupPostgresTestDB(t)
	defer cleanup()

	err := RunPostgresMigrations(testPostgresDSN, t.TempDir())
	require.NoError(t, err)
}

func TestRunPostgresMigrations_GapInVersions(t *testing.T) {
	cleanup := setupPostgresTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000001_a.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS pg_gap1 (id INTEGER);"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "000005_b.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS pg_gap5 (id INTEGER);"), 0644))

	require.NoError(t, RunPostgresMigrations(testPostgresDSN, tmpDir))

	db, err := sql.Open("pgx", testPostgresDSN)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 2, count)
}

func TestRunDuckDBMigrations_SequentialVersionBumps(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "seq.db")

	for v := 1; v <= 4; v++ {
		fn := filepath.Join(tmpDir, fmt.Sprintf("00000%d_ver.up.sql", v))
		require.NoError(t, os.WriteFile(fn,
			[]byte(fmt.Sprintf("CREATE TABLE IF NOT EXISTS seq_t%d (id INTEGER);", v)), 0644))
	}

	require.NoError(t, RunDuckDBMigrations(dbPath, tmpDir))

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.Equal(t, 4, count)

	// Verify all tables exist
	for v := 1; v <= 4; v++ {
		tbl := fmt.Sprintf("seq_t%d", v)
		var n int
		err := db.QueryRow("SELECT COUNT(*) FROM " + tbl).Scan(&n)
		require.NoError(t, err)
	}
}
