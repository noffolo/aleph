package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	_ "github.com/marcboeker/go-duckdb"
)

type DuckDB struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
	sem  *semaphore.Weighted
}

func NewDuckDB(dbPath string) (*DuckDB, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	// Ottimizzazione per la Verità dei Dati: Modalità WAL e Resilienza
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	
	// Install and Load VSS for Vector Similarity Search (Predictive AI)
	db.Exec("INSTALL vss; LOAD vss;")

	// Optimize for a Data OS: allow concurrency but limit handles
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(1 * time.Hour)

	return &DuckDB{
		db:   db,
		path: dbPath,
		sem:  semaphore.NewWeighted(5), // Limit to 5 concurrent analytical queries
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
	// Explicitly clear DuckDB internal caches if supported by driver
	d.db.Exec("PRAGMA shrink_memory")
}

func (d *DuckDB) Close() error {
	return d.db.Close()
}

func (d *DuckDB) DB() *sql.DB {
	return d.db
}
