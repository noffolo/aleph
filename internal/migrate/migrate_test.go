package migrate

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func TestRunDuckDBMigrations(t *testing.T) {
	err := RunDuckDBMigrations(":memory:", "../../migrations/duckdb")
	if err != nil {
		t.Errorf("RunDuckDBMigrations failed: %v", err)
	}

	err = RunDuckDBMigrations(":memory:", "../../migrations/duckdb")
	if err != nil {
		t.Errorf("Second RunDuckDBMigrations failed: %v", err)
	}
}

func TestRunDuckDBMigrations_InvalidPath(t *testing.T) {
	err := RunDuckDBMigrations(":memory:", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent migration path, got nil")
	}
}

func TestRunDuckDBMigrations_TempDir(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS test_table (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count); err != nil {
		t.Errorf("verification query failed: %v", err)
	}

	var version int64
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Errorf("schema_migrations query failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}
}

func TestRunDuckDBMigrations_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"000001_init.up.sql":      "CREATE TABLE IF NOT EXISTS t1 (id INTEGER);",
		"000002_add_col.up.sql":   "ALTER TABLE t1 ADD COLUMN name VARCHAR;",
		"000003_add_index.up.sql": "CREATE INDEX IF NOT EXISTS idx_t1_id ON t1(id);",
	}
	for filename, sql := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(sql), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("schema_migrations count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 migrations applied, got %d", count)
	}
}

func TestRunDuckDBMigrations_SkipAlreadyApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "000002_new.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t2 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write second migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 migrations, got %d", count)
	}
}

func TestRunDuckDBMigrations_BadSQL_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write good migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "000002_bad.up.sql"),
		[]byte("THIS IS NOT VALID SQL;"), 0644); err != nil {
		t.Fatalf("failed to write bad migration: %v", err)
	}

	err := RunDuckDBMigrations(dbPath, tmpDir)
	if err == nil {
		t.Error("expected error for bad SQL, got nil")
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var maxVersion int64
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&maxVersion); err != nil {
		t.Fatalf("max version query failed: %v", err)
	}
	if maxVersion != 1 {
		t.Errorf("expected max version 1 (bad migration rolled back), got %d", maxVersion)
	}
}

func TestRunDuckDBMigrations_IdempotentOnClean(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration (idempotent), got %d", count)
	}
}

func TestRunDuckDBMigrations_EmptyMigrationDir(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations with empty dir failed: %v", err)
	}
}

func TestRunDuckDBMigrations_DownFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write up migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.down.sql"),
		[]byte("DROP TABLE IF EXISTS t1;"), 0644); err != nil {
		t.Fatalf("failed to write down migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var version int64
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatalf("version query failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected up migration applied (down files ignored), got version %d", version)
	}
}

func TestRunDuckDBMigrations_OnlyPendingApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	for i, sql := range []string{
		"CREATE TABLE IF NOT EXISTS t1 (id INTEGER);",
		"CREATE TABLE IF NOT EXISTS t2 (id INTEGER);",
		"CREATE TABLE IF NOT EXISTS t3 (id INTEGER);",
		"CREATE TABLE IF NOT EXISTS t4 (id INTEGER);",
		"CREATE TABLE IF NOT EXISTS t5 (id INTEGER);",
	} {
		filename := filepath.Join(tmpDir, "00000"+string(rune('1'+i))+"_v"+string(rune('1'+i))+".up.sql")
		if err := os.WriteFile(filename, []byte(sql), 0644); err != nil {
			t.Fatalf("failed to write migration: %v", err)
		}
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 migrations applied, got %d", count)
	}
}

func TestRunAllMigrations_EmptyDSNs(t *testing.T) {
	err := RunAllMigrations("", "")
	if err != nil {
		t.Errorf("RunAllMigrations should return nil even on failure: %v", err)
	}
}

func TestRunAllMigrations_DuckDBOnly(t *testing.T) {
	err := RunAllMigrations(":memory:", "")
	if err != nil {
		t.Errorf("RunAllMigrations should return nil: %v", err)
	}
}
