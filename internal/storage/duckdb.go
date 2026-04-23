package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	_ "github.com/marcboeker/go-duckdb"
)

type DuckDB struct {
	db     *sql.DB
	path   string
	mu     sync.RWMutex
	sem    *semaphore.Weighted
	HasVSS bool
}

func NewDuckDB(dbPath string) (*DuckDB, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	// SQLite PRAGMAs (journal_mode, synchronous) intentionally omitted:
	// DuckDB uses a different storage/persistence model where these do not apply.
	
	// Install and Load VSS for Vector Similarity Search (Predictive AI)
	hasVSS := true
	if _, err := db.Exec("INSTALL vss;"); err != nil {
		log.Printf("[DuckDB] VSS extension install failed: %v (vector search unavailable)", err)
		hasVSS = false
	} else if _, err := db.Exec("LOAD vss;"); err != nil {
		log.Printf("[DuckDB] VSS extension load failed: %v (vector search unavailable)", err)
		hasVSS = false
	}

	// Optimize for a Data OS: allow concurrency but limit handles
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(1 * time.Hour)

	return &DuckDB{
		db:     db,
		path:   dbPath,
		sem:    semaphore.NewWeighted(5),
		HasVSS: hasVSS,
	}, nil
}

func (d *DuckDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.Query(query, args...)
}

func (d *DuckDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !d.sem.TryAcquire(1) {
		return nil, fmt.Errorf("duckdb resource exhausted: too many concurrent queries")
	}
	defer d.sem.Release(1)

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.QueryContext(ctx, query, args...)
}

func (d *DuckDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db.Exec(query, args...)
}

func (d *DuckDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !d.sem.TryAcquire(1) {
		return nil, fmt.Errorf("duckdb resource exhausted: too many concurrent queries")
	}
	defer d.sem.Release(1)

	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db.ExecContext(ctx, query, args...)
}

func (d *DuckDB) Cleanup() {
	// PRAGMA shrink_memory intentionally omitted: it is SQLite-specific.
	// DuckDB memory management differs; refer to DuckDB docs for equivalents.
}

func (d *DuckDB) Close() error {
	return d.db.Close()
}

func (d *DuckDB) QueryRow(query string, args ...interface{}) *sql.Row {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.QueryRow(query, args...)
}

func (d *DuckDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !d.sem.TryAcquire(1) {
		return d.db.QueryRowContext(ctx, "SELECT 'duckdb resource exhausted'")
	}
	defer d.sem.Release(1)
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *DuckDB) DB() *sql.DB {
	return d.db
}
