package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDuckDBMigrations_FileNotWritable(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("INVALID SQL THAT WILL CAUSE SYNTAX ERROR;;;"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}

	err := RunDuckDBMigrations(":memory:", tmpDir)
	if err == nil {
		t.Error("expected error for invalid SQL in migration")
	}
}

func TestRunDuckDBMigrations_GapInVersions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "gap.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "000005_skip.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t5 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration 5: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations with gap: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "000003_mid.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t3 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration 3: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("second RunDuckDBMigrations: %v", err)
	}
}

func TestRunDuckDBMigrations_NonSQLFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonsql.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"),
		[]byte("# Migrations"), 0644); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "000002_not_a_migration.txt"),
		[]byte("not sql"), 0644); err != nil {
		t.Fatalf("failed to write txt: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations with non-sql files: %v", err)
	}
}

func TestRunDuckDBMigrations_MalformedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "malformed.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "notanumber_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations with malformed version: %v", err)
	}
}

func TestRunPostgresMigrations_InvalidDSN(t *testing.T) {
	err := RunPostgresMigrations("invalid://dsn", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid postgres migrations")
	}
}

func TestRunPostgresMigrations_EmptyDSN(t *testing.T) {
	err := RunPostgresMigrations("", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for empty postgres DSN with nonexistent path")
	}
}

func TestRunAllMigrations_PostgresOnly(t *testing.T) {
	err := RunAllMigrations("", "invalid://dsn")
	if err != nil {
		t.Errorf("RunAllMigrations should not fail: %v", err)
	}
}

func TestRunAllMigrations_BothInvalid(t *testing.T) {
	err := RunAllMigrations("invalid://dsn", "invalid://dsn")
	if err != nil {
		t.Errorf("RunAllMigrations should not fail on invalid DSNs: %v", err)
	}
}

func TestRunAllMigrations_PostgresInvalid(t *testing.T) {
	err := RunAllMigrations(":memory:", "invalid://dsn")
	if err != nil {
		t.Errorf("RunAllMigrations should not fail: %v", err)
	}
}
