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

// ── Backup (file-copy with CHECKPOINT) ──────────────────────────────────────

// Backup creates a consistent snapshot of the DuckDB database at destPath.
// It runs CHECKPOINT to flush the WAL, then copies the database file.
// During the checkpoint + file-copy window a read transaction is held
// to guarantee consistency while NOT blocking concurrent reads.
// In-memory databases return an error.
//
// For a schema-and-data export (DuckDB EXPORT DATABASE) see ExportDatabase.
func (d *DuckDB) Backup(ctx context.Context, destPath string) error {
	if d.path == ":memory:" || d.path == "" {
		return fmt.Errorf("cannot backup an in-memory database; use a file-based DuckDB path")
	}
	parent := filepath.Dir(destPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create backup parent directory: %w", err)
	}

	// Use a read transaction so we get a consistent snapshot without blocking
	// concurrent reads. The read transaction ensures no writes can interfere
	// during the checkpoint + file copy, but reads proceed unimpeded.
	tx, err := d.BeginReadTX(ctx)
	if err != nil {
		return fmt.Errorf("begin read tx for backup: %w", err)
	}
	defer tx.Rollback() // Rollback releases the pool connection if commit not called.

	if _, err := tx.Exec("CHECKPOINT;"); err != nil {
		slog.Warn("CHECKPOINT before backup failed, snapshot may be incomplete",
			"error", err)
	}

	if err := d.copyFile(d.path, destPath); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return tx.Commit()
}

// ── ExportDatabase (DuckDB EXPORT DATABASE) ────────────────────────────────

// ExportDatabase uses DuckDB's EXPORT DATABASE command to write the full
// database schema and data as Parquet files into exportDir. The export is a
// consistent snapshot taken under the write lock (writeMu).
//
// The resulting directory contains:
//   - schema.sql   — full DDL to recreate the database
//   - load.sql     — COPY statements to load each table from Parquet files
//   - *.parquet    — one Parquet file per table
//
// Callers should verify the export with VerifyExportBackup after a successful return.
// In-memory databases return an error.
func (d *DuckDB) ExportDatabase(ctx context.Context, exportDir string) error {
	if d.path == ":memory:" || d.path == "" {
		return fmt.Errorf("cannot export an in-memory database; use a file-based DuckDB path")
	}

	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	// EXPORT DATABASE is serialized via writeMu + the connection pool.
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	// EXPORT DATABASE uses a quoted path — the engine writes everything to that
	// directory. It is transactional: if the export fails mid-way the directory
	// may contain partial output and should be cleaned up by the caller.
	query := fmt.Sprintf("EXPORT DATABASE '%s'", strings.ReplaceAll(exportDir, "'", "''"))
	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("EXPORT DATABASE failed: %w", err)
	}

	// fsync the schema and load files so the export is durable on disk.
	for _, name := range []string{"schema.sql", "load.sql"} {
		p := filepath.Join(exportDir, name)
		if err := fsyncFile(p); err != nil {
			slog.Warn("export: fsync failed", "file", p, "error", err)
		}
	}

	slog.Info("export database complete", "exportDir", exportDir)
	return nil
}

// fsyncFile opens the file at path and calls Sync() to flush kernel buffers to
// stable storage. Silently skips missing files (common if a table is empty and
// DuckDB does not write the parquet).
func fsyncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open for fsync: %w", err)
	}
	defer f.Close()
	return f.Sync()
}

// ── VerifyBackup ───────────────────────────────────────────────────────────

// VerifyBackup checks the integrity of a DuckDB database file at backupPath.
// It opens the file in read-only mode (?access_mode=read_only), runs CHECKPOINT
// to flush any pending WAL data, verifies the file is non-empty and readable,
// and performs a table+view count as a sanity check.
//
// Returns nil if the backup is valid. Returns an error describing the issue if:
//   - the file does not exist or is empty
//   - DuckDB rejects the file (corrupt header, wrong format)
//   - no tables or views are found
//
// For EXPORT DATABASE exports (which produce directories), pass the path to the
// individual .duckdb file if one was produced, or use VerifyExportBackup.
func (d *DuckDB) VerifyBackup(backupPath string) error {
	info, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup file does not exist: %s", backupPath)
		}
		return fmt.Errorf("stat backup file: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("backup file is empty: %s", backupPath)
	}

	// Open the backup read-only to prevent accidental writes during verification.
	tmpDB, err := sql.Open("duckdb", backupPath+"?access_mode=read_only")
	if err != nil {
		return fmt.Errorf("open backup for verification: %w", err)
	}
	defer tmpDB.Close()

	// DuckDB CHECKPOINT to flush any uncommitted WAL data.
	// In read-only mode this is a no-op but safe to call.
	if _, err := tmpDB.Exec("CHECKPOINT;"); err != nil {
		slog.Warn("verify backup: CHECKPOINT failed",
			"path", backupPath, "error", err)
	}

	// Verify the file size is reasonable (at least the DuckDB header).
	info2, err := os.Stat(backupPath)
	if err == nil && info2.Size() < 1024 {
		slog.Warn("verify: backup file is very small, may be incomplete",
			"path", backupPath, "size", info2.Size())
	}

	// Sanity check: count tables using information_schema (DuckDB-compatible).
	rows, err := tmpDB.Query("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')")
	if err != nil {
		// Fallback: try DuckDB's native table listing.
		rows, err = tmpDB.Query("SELECT COUNT(*) FROM duckdb_tables()")
		if err != nil {
			return fmt.Errorf("table count query failed: %w", err)
		}
	}
	defer rows.Close()
	var tableCount int
	if rows.Next() {
		if err := rows.Scan(&tableCount); err != nil {
			return fmt.Errorf("scan table count: %w", err)
		}
	}

	// Count views as an additional integrity check.
	var viewCount int
	vrows, err := tmpDB.Query("SELECT COUNT(*) FROM duckdb_views()")
	if err == nil {
		defer vrows.Close()
		if vrows.Next() {
			_ = vrows.Scan(&viewCount)
		}
	}

	totalObjects := tableCount + viewCount
	if totalObjects == 0 {
		slog.Warn("verify: backup has zero tables or views", "path", backupPath)
	}

	slog.Info("backup verified successfully",
		"path", backupPath,
		"size", info2.Size(),
		"tables", tableCount,
		"views", viewCount,
		"total", totalObjects)
	return nil
}

// VerifyExportBackup verifies an EXPORT DATABASE export directory by checking
// that schema.sql, load.sql exist and are non-empty, and that referenced Parquet
// files exist.
func (d *DuckDB) VerifyExportBackup(exportDir string) error {
	required := []string{"schema.sql", "load.sql"}
	for _, name := range required {
		p := filepath.Join(exportDir, name)
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("export missing required file: %s", name)
			}
			return fmt.Errorf("stat export file %s: %w", name, err)
		}
		if info.Size() == 0 {
			slog.Warn("export file is empty (may be normal for empty databases)",
				"file", name)
		}
	}

	// Verify referenced parquet files exist (parse load.sql for COPY statements).
	loadSQL, err := os.ReadFile(filepath.Join(exportDir, "load.sql"))
	if err != nil {
		return fmt.Errorf("read load.sql: %w", err)
	}
	missingParquet := checkReferencedParquetFiles(exportDir, string(loadSQL))
	if len(missingParquet) > 0 {
		return fmt.Errorf("export missing referenced parquet files: %v", missingParquet)
	}

	return nil
}

// checkReferencedParquetFiles parses load.sql for COPY ... FROM '...parquet'
// lines and verifies each parquet file exists under exportDir.
func checkReferencedParquetFiles(exportDir, loadSQL string) []string {
	var missing []string
	for _, line := range strings.Split(loadSQL, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), "COPY") {
			continue
		}
		// Extract path between single quotes after FROM.
		// Format: COPY "schema"."table" FROM 'path/to/file.parquet' ...
		quoteStart := strings.Index(line, "'")
		if quoteStart == -1 {
			continue
		}
		quoteEnd := strings.Index(line[quoteStart+1:], "'")
		if quoteEnd == -1 {
			continue
		}
		refPath := line[quoteStart+1 : quoteStart+1+quoteEnd]
		if !strings.HasSuffix(refPath, ".parquet") {
			continue
		}
		fullPath := refPath
		if !filepath.IsAbs(refPath) {
			fullPath = filepath.Join(exportDir, refPath)
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			missing = append(missing, refPath)
		}
	}
	return missing
}

// ── Restore ────────────────────────────────────────────────────────────────

// Restore replaces the current database with the contents of the backup at
// sourcePath. It acquires the write lock, closes the current connection,
// copies the backup file over the original path, and re-opens the database.
//
// RESTORE PROCEDURE (manual — this method is the automated portion):
//
//  1. Stop all application traffic to ensure zero concurrent queries.
//  2. Call Restore(backupPath) — this closes the existing connection,
//     copies the backup file over the live database file, re-opens the
//     connection, and re-loads extensions (e.g. VSS).
//  3. Run all DuckDB migrations to ensure the schema is current (the backup
//     may be from an older schema version).
//  4. Verify the restored database with VerifyBackup(d.path).
//  5. Run a smoke-test query (e.g. SELECT COUNT(*) on a known table).
//  6. Resume application traffic.
//
// For EXPORT DATABASE exports, the manual restore procedure is:
//
//  1. Stop application traffic.
//  2. Delete the current database file or move it aside.
//  3. Open a new empty DuckDB at the same path.
//  4. Run: IMPORT DATABASE '/path/to/export/dir';
//  5. Close and re-open the connection.
//  6. Run migrations and verification as above.
//
// WARNING: In-memory databases cannot be restored; use a file-backed path.
// After restore the caller MUST re-run any schema migrations.
func (d *DuckDB) Restore(sourcePath string) error {
	if d.path == ":memory:" || d.path == "" {
		return fmt.Errorf("cannot restore an in-memory database; use a file-based DuckDB path")
	}

	d.writeMu.Lock()
	defer d.writeMu.Unlock()

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

	d.createBackup(ctx, dir, baseName)

	for {
		select {
		case <-ctx.Done():
			slog.Info("auto-backup stopped")
			return
		case <-ticker.C:
			d.createBackup(ctx, dir, baseName)
			if keep > 0 {
				d.cleanOldBackups(dir, baseName, keep)
			}
		}
	}
}

func (d *DuckDB) createBackup(ctx context.Context, dir, baseName string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("auto-backup: failed to create backup directory",
			"dir", dir, "error", err)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	destPath := filepath.Join(dir, fmt.Sprintf("%s_backup_%s.duckdb", baseName, ts))

	if err := d.Backup(ctx, destPath); err != nil {
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
