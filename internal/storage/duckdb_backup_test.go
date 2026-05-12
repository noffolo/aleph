package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupRestore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	bakPath := filepath.Join(dir, "test_backup.duckdb")

	db, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(context.Background(), "CREATE TABLE items (id INTEGER, label VARCHAR)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO items VALUES (1, 'alpha'), (2, 'beta')")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Backup(context.Background(), bakPath); err != nil {
		t.Fatal("Backup failed:", err)
	}

	_, err = db.Exec(context.Background(), "INSERT INTO items VALUES (3, 'gamma')")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	db2, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	rows, err := db2.Query("SELECT count(*) FROM items")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var count int
	rows.Next()
	rows.Scan(&count)
	if count != 3 {
		t.Fatalf("expected 3 rows before restore, got %d", count)
	}
	rows.Close()

	if err := db2.Restore(bakPath); err != nil {
		t.Fatal("Restore failed:", err)
	}

	rows, err = db2.Query("SELECT label FROM items ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var l string
		rows.Scan(&l)
		labels = append(labels, l)
	}
	if len(labels) != 2 {
		t.Fatalf("expected 2 rows after restore, got %d: %v", len(labels), labels)
	}
	if labels[0] != "alpha" || labels[1] != "beta" {
		t.Fatalf("unexpected data after restore: %v", labels)
	}
}

func TestBackup_InMemoryFails(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Backup(context.Background(), "/tmp/nope.duckdb"); err == nil {
		t.Fatal("expected error backing up in-memory database")
	}
	if err := db.Restore("/tmp/nope.duckdb"); err == nil {
		t.Fatal("expected error restoring in-memory database")
	}
}

func TestAutoBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "autotest.duckdb")

	db, err := NewDuckDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(context.Background(), "CREATE TABLE t (x INTEGER)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO t VALUES (42)")
	if err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(dir, "backups")
	ctx, cancel := context.WithCancel(context.Background())
	go db.AutoBackup(ctx, 100*time.Millisecond, backupDir, 3)

	time.Sleep(350 * time.Millisecond)
	cancel()

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	var backupCount int
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".duckdb" {
			backupCount++
		}
	}
	if backupCount == 0 {
		t.Fatal("expected at least one backup file, got 0")
	}
}

func TestCleanOldBackups(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	os.MkdirAll(backupDir, 0o755)

	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < 5; i++ {
		ts := time.Date(2024, 1, 1, 0, 0, i, 0, time.UTC).Format("20060102T150405Z")
		name := filepath.Join(backupDir, "keeptest_backup_"+ts+".duckdb")
		os.WriteFile(name, []byte("fake"), 0o644)
	}

	db.cleanOldBackups(backupDir, "keeptest", 2)

	entries, _ := os.ReadDir(backupDir)
	var remaining int
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".duckdb" {
			remaining++
		}
	}
	if remaining != 2 {
		t.Fatalf("expected 2 backups after cleanup, got %d", remaining)
	}
}
