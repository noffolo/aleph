package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Backup creates a consistent snapshot of the DuckDB database at destPath.
// It runs CHECKPOINT to flush the WAL, then copies the database file.
// Only one backup runs at a time (via backupMu). During the checkpoint +
// file-copy window the read lock (mu.RLock) prevents concurrent writes
// without blocking concurrent reads. In-memory databases return an error.
func (d *DuckDB) Backup(destPath string) error {
	if d.path == ":memory:" || d.path == "" {
		return fmt.Errorf("cannot backup an in-memory database; use a file-based DuckDB path")
	}
	parent := filepath.Dir(destPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create backup parent directory: %w", err)
	}

	// Serialize backups against each other.
	d.backupMu.Lock()
	defer d.backupMu.Unlock()

	// Hold the read lock during CHECKPOINT + file copy so no writes can
	// modify the database while we capture the snapshot. Other readers
	// (the vast majority of traffic) proceed in parallel.
	d.mu.RLock()
	defer d.mu.RUnlock()

	// CHECKPOINT flushes the DuckDB WAL into the main database file so our
	// file copy captures a consistent on-disk state.
	if _, err := d.db.Exec("CHECKPOINT;"); err != nil {
		slog.Warn("CHECKPOINT before backup failed, snapshot may be incomplete",
			"error", err)
	}
	return d.backupFileCopyLocked(destPath)
}

// Restore replaces the current database with the contents of the backup at
// sourcePath. It acquires the write lock, closes the current connection,
// copies the backup file over the original path, and re-opens the database.
// Callers MUST ensure no other goroutines execute queries during restore.
// After restore the caller should re-run any schema migrations.
func (d *DuckDB) Restore(sourcePath string) error {
	if d.path == ":memory:" || d.path == "" {
		return fmt.Errorf("cannot restore an in-memory database; use a file-based DuckDB path")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.db.Close(); err != nil {
		return fmt.Errorf("close existing DuckDB connection for restore: %w", err)
	}

	if err := d.copyFile(sourcePath, d.path); err != nil {
		return fmt.Errorf("copy backup to original path: %w", err)
	}

	newDB, err := sql.Open("duckdb", d.path)
	if err != nil {
		return fmt.Errorf("re-open DuckDB after restore: %w", err)
	}
	d.db = newDB

	if d.HasVSS {
		if _, err := d.db.Exec("LOAD vss;"); err != nil {
			slog.Warn("failed to LOAD vss after restore", "error", err)
			d.HasVSS = false
		}
	}

	return nil
}

// AutoBackup launches a background goroutine that creates periodic consistent
// backups of the DuckDB database. It runs until ctx is cancelled.
//
//   - interval: how often to create a backup (e.g. 15*time.Minute)
//   - dir:      directory in which to store backup files
//   - keep:     maximum number of recent backups to retain (≤0 keeps all)
func (d *DuckDB) AutoBackup(ctx context.Context, interval time.Duration, dir string, keep int) {
	if d.path == ":memory:" || d.path == "" {
		slog.Warn("AutoBackup skipped: in-memory database has no file to back up")
		return
	}

	baseName := strings.TrimSuffix(filepath.Base(d.path), ".duckdb")
	if baseName == "" {
		baseName = strings.TrimSuffix(filepath.Base(d.path), ".db")
	}
	if baseName == "" {
		baseName = "aleph"
	}

	slog.Info("auto-backup starting",
		"interval", interval,
		"dir", dir,
		"keep", keep,
		"base", baseName)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	d.createBackup(dir, baseName)

	for {
		select {
		case <-ctx.Done():
			slog.Info("auto-backup stopped")
			return
		case <-ticker.C:
			d.createBackup(dir, baseName)
			if keep > 0 {
				d.cleanOldBackups(dir, baseName, keep)
			}
		}
	}
}

func (d *DuckDB) createBackup(dir, baseName string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("auto-backup: failed to create backup directory",
			"dir", dir, "error", err)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	destPath := filepath.Join(dir, fmt.Sprintf("%s_backup_%s.duckdb", baseName, ts))

	if err := d.Backup(destPath); err != nil {
		slog.Error("auto-backup failed",
			"dest", destPath, "error", err)
		return
	}
	slog.Debug("auto-backup created", "dest", destPath)

	metaPath := destPath + ".meta"
	_ = os.WriteFile(metaPath, []byte(fmt.Sprintf(
		"source=%s\ntime=%s\n", d.path, ts)), 0o644)
}

func (d *DuckDB) cleanOldBackups(dir, baseName string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Error("auto-backup: read backup directory", "dir", dir, "error", err)
		return
	}

	prefix := baseName + "_backup_"
	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) &&
			strings.HasSuffix(e.Name(), ".duckdb") {
			backups = append(backups, filepath.Join(dir, e.Name()))
		}
	}

	if len(backups) <= keep {
		return
	}
	sort.Strings(backups)

	toRemove := backups[:len(backups)-keep]
	for _, f := range toRemove {
		if err := os.Remove(f); err != nil {
			slog.Warn("auto-backup: remove old backup", "file", f, "error", err)
		}
		_ = os.Remove(f + ".meta")
	}
	slog.Debug("auto-backup: cleaned old backups", "removed", len(toRemove))
}

func (d *DuckDB) backupFileCopyLocked(destPath string) error {
	return d.copyFile(d.path, destPath)
}

func (d *DuckDB) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}
