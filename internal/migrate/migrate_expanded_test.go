package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func TestRunAll_SuccessPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migrations := map[string]string{
		"000001_create_users.up.sql": "CREATE TABLE IF NOT EXISTS users (id INTEGER, name VARCHAR);",
		"000002_add_email.up.sql":    "ALTER TABLE users ADD COLUMN email VARCHAR;",
		"000003_create_posts.up.sql": "CREATE TABLE IF NOT EXISTS posts (id INTEGER, user_id INTEGER);",
	}

	for name, sqlContent := range migrations {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(sqlContent), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE NOT dirty").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 migrations applied, got %d", count)
	}
}

func TestRunAll_MigrationError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write good migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "000002_bad.up.sql"),
		[]byte("SYNTAX ERROR NOT VALID SQL ###"), 0644); err != nil {
		t.Fatalf("failed to write bad migration: %v", err)
	}

	err := RunDuckDBMigrations(dbPath, tmpDir)
	if err == nil {
		t.Fatal("expected error for migration with SQL syntax error, got nil")
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	var maxVersion int64
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE NOT dirty").Scan(&maxVersion); err != nil {
		t.Fatalf("version query failed: %v", err)
	}
	if maxVersion != 1 {
		t.Errorf("expected max version 1, got %d", maxVersion)
	}
}

func TestMigrationVersion_Table(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"),
		[]byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER);"), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'schema_migrations' ORDER BY ordinal_position")
	if err != nil {
		t.Fatalf("info schema query failed: %v", err)
	}
	defer rows.Close()

	columns := map[string]string{}
	for rows.Next() {
		var colName, colType string
		if err := rows.Scan(&colName, &colType); err != nil {
			t.Fatalf("column scan failed: %v", err)
		}
		columns[colName] = colType
	}

	if v, ok := columns["version"]; !ok || v == "" {
		t.Error("schema_migrations table missing version column")
	}
	if v, ok := columns["dirty"]; !ok || v == "" {
		t.Error("schema_migrations table missing dirty column")
	}

	var version int64
	if err := db.QueryRow("SELECT version FROM schema_migrations WHERE NOT dirty").Scan(&version); err != nil {
		t.Fatalf("version query failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	var dirty bool
	if err := db.QueryRow("SELECT dirty FROM schema_migrations WHERE version = 1").Scan(&dirty); err != nil {
		t.Fatalf("dirty query failed: %v", err)
	}
	if dirty {
		t.Error("expected dirty flag to be false after successful migration")
	}
}

func TestMigration_Ordering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migrationFiles := []struct {
		version int
		sql     string
	}{
		{3, "INSERT INTO ordering_log VALUES (3, 'third');"},
		{1, "CREATE TABLE ordering_log (applied_at INTEGER, label VARCHAR);"},
		{2, "INSERT INTO ordering_log VALUES (2, 'second');"},
	}

	for _, m := range migrationFiles {
		fn := filepath.Join(tmpDir, fmt.Sprintf("00000%d_migration.up.sql", m.version))
		if err := os.WriteFile(fn, []byte(m.sql), 0644); err != nil {
			t.Fatalf("failed to write migration %d: %v", m.version, err)
		}
	}

	if err := RunDuckDBMigrations(dbPath, tmpDir); err != nil {
		t.Fatalf("RunDuckDBMigrations failed: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT applied_at, label FROM ordering_log ORDER BY applied_at")
	if err != nil {
		t.Fatalf("order query failed: %v", err)
	}
	defer rows.Close()

	expectedRows := []struct {
		appliedAt int
		label     string
	}{
		{2, "second"},
		{3, "third"},
	}

	i := 0
	for rows.Next() {
		var appliedAt int
		var label string
		if err := rows.Scan(&appliedAt, &label); err != nil {
			t.Fatalf("row scan failed: %v", err)
		}
		if i >= len(expectedRows) {
			t.Fatalf("unexpected extra row: %d, %s", appliedAt, label)
		}
		if appliedAt != expectedRows[i].appliedAt || label != expectedRows[i].label {
			t.Errorf("row %d: expected %d/%s, got %d/%s",
				i, expectedRows[i].appliedAt, expectedRows[i].label, appliedAt, label)
		}
		i++
	}
	if i != len(expectedRows) {
		t.Errorf("expected %d rows, got %d", len(expectedRows), i)
	}
}

func TestMigration_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migrations := []struct {
		version int
		sql     string
	}{
		{1, "CREATE TABLE IF NOT EXISTS pf_t1 (id INTEGER);"},
		{2, "CREATE TABLE IF NOT EXISTS pf_t2 (id INTEGER);"},
		{3, "SYNTAX ERROR ###"},
		{4, "CREATE TABLE IF NOT EXISTS pf_t4 (id INTEGER);"},
	}

	for _, m := range migrations {
		fn := filepath.Join(tmpDir, fmt.Sprintf("00000%d_pf.up.sql", m.version))
		if err := os.WriteFile(fn, []byte(m.sql), 0644); err != nil {
			t.Fatalf("failed to write migration %d: %v", m.version, err)
		}
	}

	err := RunDuckDBMigrations(dbPath, tmpDir)
	if err == nil {
		t.Fatal("expected error from failed migration, got nil")
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT version FROM schema_migrations WHERE NOT dirty ORDER BY version")
	if err != nil {
		t.Fatalf("version query failed: %v", err)
	}
	defer rows.Close()

	var appliedVersions []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		appliedVersions = append(appliedVersions, v)
	}

	if len(appliedVersions) != 2 {
		t.Errorf("expected 2 applied migrations, got %d (versions: %v)", len(appliedVersions), appliedVersions)
		return
	}
	if appliedVersions[0] != 1 {
		t.Errorf("expected version 1 applied, got %d", appliedVersions[0])
	}
	if appliedVersions[1] != 2 {
		t.Errorf("expected version 2 applied, got %d", appliedVersions[1])
	}

	for _, tbl := range []string{"pf_t1", "pf_t2"} {
		var c int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + tbl).Scan(&c); err != nil {
			t.Errorf("table %s should exist after partial failure: %v", tbl, err)
		}
	}
}

func TestMigration_DirNotFound(t *testing.T) {
	err := RunDuckDBMigrations(":memory:", "/nonexistent/migration/directory/12345")
	if err == nil {
		t.Fatal("expected error for nonexistent migration directory, got nil")
	}
}
