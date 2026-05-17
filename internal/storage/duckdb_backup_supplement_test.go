package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- VerifyBackup ---

func TestVerifyBackup_ValidFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "verify_test.duckdb")

	db, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(context.Background(), "CREATE TABLE verify_items (id INTEGER, name VARCHAR)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO verify_items VALUES (1, 'alpha')")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	if err := db.VerifyBackup(dbPath); err != nil {
		t.Errorf("VerifyBackup on valid db file: unexpected error: %v", err)
	}
}

func TestVerifyBackup_NonexistentFile(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	want := "/nonexistent/path/that/does/not/exist.duckdb"
	if err := db.VerifyBackup(want); err == nil {
		t.Error("VerifyBackup on nonexistent file: expected error, got nil")
	}
}

func TestVerifyBackup_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyPath := filepath.Join(dir, "empty.duckdb")
	if err := os.WriteFile(emptyPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.VerifyBackup(emptyPath); err == nil {
		t.Error("VerifyBackup on empty file: expected error, got nil")
	}
}

// --- ExportDatabase ---

func TestExportDatabase_InMemoryFails(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.ExportDatabase(context.Background(), t.TempDir()); err == nil {
		t.Error("ExportDatabase on in-memory DB: expected error, got nil")
	}
}

func TestExportDatabase_FileBacked(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "export_test.duckdb")
	exportDir := filepath.Join(dir, "export_out")

	db, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(context.Background(), "CREATE TABLE export_data (id INTEGER, value VARCHAR)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO export_data VALUES (1, 'hello'), (2, 'world')")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.ExportDatabase(context.Background(), exportDir); err != nil {
		t.Fatalf("ExportDatabase failed: %v", err)
	}

	// Verify export files exist
	for _, name := range []string{"schema.sql", "load.sql"} {
		p := filepath.Join(exportDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("ExportDatabase: expected %s to exist", name)
		}
	}
}

func TestVerifyExportBackup_Valid(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "veb_test.duckdb")
	exportDir := filepath.Join(dir, "veb_out")

	db, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(context.Background(), "CREATE TABLE veb (x INTEGER)")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.ExportDatabase(context.Background(), exportDir); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	if err := db.VerifyExportBackup(exportDir); err != nil {
		t.Errorf("VerifyExportBackup: unexpected error: %v", err)
	}
}

func TestVerifyExportBackup_MissingSchemaSQL(t *testing.T) {
	dir := t.TempDir()
	exportDir := filepath.Join(dir, "missing_schema")
	os.MkdirAll(exportDir, 0o755)
	// Write load.sql but not schema.sql
	os.WriteFile(filepath.Join(exportDir, "load.sql"), []byte("COPY t FROM 'a.parquet';"), 0o644)

	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.VerifyExportBackup(exportDir); err == nil {
		t.Error("VerifyExportBackup missing schema.sql: expected error, got nil")
	}
}

func TestVerifyExportBackup_MissingParquet(t *testing.T) {
	dir := t.TempDir()
	exportDir := filepath.Join(dir, "missing_pqt")
	os.MkdirAll(exportDir, 0o755)
	os.WriteFile(filepath.Join(exportDir, "schema.sql"), []byte("CREATE TABLE t (x INTEGER);"), 0o644)
	// load.sql references a parquet that doesn't exist
	os.WriteFile(filepath.Join(exportDir, "load.sql"), []byte("COPY \"main\".\"t\" FROM 'nonexistent.parquet';"), 0o644)

	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.VerifyExportBackup(exportDir); err == nil {
		t.Error("VerifyExportBackup with missing parquet: expected error, got nil")
	}
}

// --- fsyncFile ---

func TestFSyncFile_Nonexistent(t *testing.T) {
	// fsyncFile skips missing files silently
	if err := fsyncFile("/nonexistent/path/file.parquet"); err != nil {
		t.Errorf("fsyncFile on nonexistent file: expected nil, got %v", err)
	}
}

func TestFSyncFile_Valid(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fsync_test.db")
	if err := os.WriteFile(fpath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsyncFile(fpath); err != nil {
		t.Errorf("fsyncFile on valid file: %v", err)
	}
}

// --- copyFile ---

func TestCopyFile_NonexistentSource(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.copyFile("/nonexistent/src.duckdb", t.TempDir()+"/dst.duckdb"); err == nil {
		t.Error("copyFile with nonexistent source: expected error, got nil")
	}
}

func TestCopyFile_Valid(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	dstPath := filepath.Join(dir, "dst.db")

	if err := os.WriteFile(srcPath, []byte("hello copy"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(data) != "hello copy" {
		t.Errorf("copyFile: expected 'hello copy', got %q", string(data))
	}
}

// --- checkReferencedParquetFiles ---

func TestCheckReferencedParquetFiles_AllPresent(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "present.parquet")
	os.WriteFile(fname, []byte("parquet data"), 0o644)

	missing := checkReferencedParquetFiles(dir, "COPY \"t\" FROM 'present.parquet';")
	if len(missing) != 0 {
		t.Errorf("checkReferencedParquetFiles: expected 0 missing, got %v", missing)
	}
}

func TestCheckReferencedParquetFiles_Missing(t *testing.T) {
	dir := t.TempDir()
	missing := checkReferencedParquetFiles(dir, "COPY \"t\" FROM 'missing.parquet';")
	if len(missing) != 1 {
		t.Errorf("checkReferencedParquetFiles: expected 1 missing, got %v", missing)
	}
}

func TestCheckReferencedParquetFiles_NonCopyLine(t *testing.T) {
	// Lines that are not COPY should be skipped
	missing := checkReferencedParquetFiles("/tmp", "SELECT * FROM t;\n-- comment\n  \n")
	if len(missing) != 0 {
		t.Errorf("checkReferencedParquetFiles non-copy: expected 0 missing, got %v", missing)
	}
}

// --- AutoBackup in-memory skip ---

func TestAutoBackup_InMemorySkips(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	// This should not panic and should log a warning
	db.AutoBackup(ctx, 100*time.Millisecond, t.TempDir(), 5)
	// If we get here without panic, it's fine
}
