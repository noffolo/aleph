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
	db       *sql.DB
	path     string
	mu       sync.RWMutex
	backupMu sync.Mutex
	sem      *semaphore.Weighted
	HasVSS   bool
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

	// :memory: databases are per-connection in DuckDB; limit to single connection
	// to prevent different pool connections from seeing isolated in-memory instances.
	// File-backed databases can safely use the connection pool.
	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		// Optimize for a Data OS: allow concurrency but limit handles
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(1 * time.Hour)
	}

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
	return d.db.QueryContext(ctx, scopeQuery(ctx, query), args...)
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
	return d.db.ExecContext(ctx, scopeQuery(ctx, query), args...)
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
	return d.db.QueryRowContext(ctx, scopeQuery(ctx, query), args...)
}

func (d *DuckDB) DB() *sql.DB {
	return d.db
}

// TX wraps a *sql.Tx with the DuckDB semaphore and schema context.
// All query methods use an internal RLock for safety against concurrent use.
// Callers must call Commit or Rollback to release resources.
type TX struct {
	tx       *sql.Tx
	mu       sync.RWMutex
	schema   string
	sem      *semaphore.Weighted
	parentMu *sync.RWMutex
	isReadTx bool
	done     bool
}

// BeginTX starts a new transaction. Acquires the semaphore (blocks until available)
// and the write lock. Schema from context is applied at transaction start.
// Call Commit or Rollback to release resources.
func (d *DuckDB) BeginTX(ctx context.Context) (*TX, error) {
	if err := d.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire semaphore for begin tx: %w", err)
	}

	d.mu.Lock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.mu.Unlock()
		d.sem.Release(1)
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	schema, _ := SchemaFromContext(ctx)

	// Apply schema at transaction start if set.
	if schema != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET schema = '%s'", schema)); err != nil {
			_ = tx.Rollback()
			d.mu.Unlock()
			d.sem.Release(1)
			return nil, fmt.Errorf("set schema in tx: %w", err)
		}
	}

	return &TX{
		tx:       tx,
		schema:   schema,
		sem:      d.sem,
		parentMu: &d.mu,
		isReadTx: false,
	}, nil
}

// BeginReadTX starts a new read-only transaction.
// Acquires the semaphore (blocks until available) and the read lock,
// allowing concurrent reads. Schema from context is applied at transaction start.
func (d *DuckDB) BeginReadTX(ctx context.Context) (*TX, error) {
	if err := d.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire semaphore for begin read tx: %w", err)
	}

	d.mu.RLock()

	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		d.mu.RUnlock()
		d.sem.Release(1)
		return nil, fmt.Errorf("begin read tx: %w", err)
	}

	schema, _ := SchemaFromContext(ctx)

	// Apply schema at transaction start if set.
	if schema != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET schema = '%s'", schema)); err != nil {
			_ = tx.Rollback()
			d.mu.RUnlock()
			d.sem.Release(1)
			return nil, fmt.Errorf("set schema in read tx: %w", err)
		}
	}

	return &TX{
		tx:       tx,
		schema:   schema,
		sem:      d.sem,
		parentMu: &d.mu,
		isReadTx: true,
	}, nil
}

// Commit commits the transaction and releases the acquired locks and semaphore.
// Safe to call multiple times — subsequent calls are no-ops.
func (t *TX) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return nil
	}
	t.done = true

	err := t.tx.Commit()

	// Release the parent DuckDB-level lock.
	if t.isReadTx {
		t.parentMu.RUnlock()
	} else {
		t.parentMu.Unlock()
	}
	t.sem.Release(1)

	return err
}

// Rollback rolls back the transaction and releases the acquired locks and semaphore.
// Safe to call multiple times — subsequent calls are no-ops.
func (t *TX) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return nil
	}
	t.done = true

	err := t.tx.Rollback()

	// Release the parent DuckDB-level lock.
	if t.isReadTx {
		t.parentMu.RUnlock()
	} else {
		t.parentMu.Unlock()
	}
	t.sem.Release(1)

	return err
}

// Query executes a query on the transaction with a read lock.
func (t *TX) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tx.Query(txScopeQuery(t.schema, query), args...)
}

// QueryContext executes a query on the transaction with a read lock.
func (t *TX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tx.QueryContext(ctx, txScopeQuery(t.schema, query), args...)
}

// Exec executes a statement on the transaction with a write lock.
func (t *TX) Exec(query string, args ...interface{}) (sql.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tx.Exec(txScopeQuery(t.schema, query), args...)
}

// ExecContext executes a statement on the transaction with a write lock.
func (t *TX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tx.ExecContext(ctx, txScopeQuery(t.schema, query), args...)
}

// QueryRow executes a query that returns at most one row on the transaction.
func (t *TX) QueryRow(query string, args ...interface{}) *sql.Row {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tx.QueryRow(txScopeQuery(t.schema, query), args...)
}

// QueryRowContext executes a query that returns at most one row on the transaction.
func (t *TX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tx.QueryRowContext(ctx, txScopeQuery(t.schema, query), args...)
}

// txScopeQuery returns a query string that sets the schema if one was configured
// at transaction start. The schema SET is prepended once per query to ensure
// DuckDB's session-scoped schema setting is maintained across individual operations.
func txScopeQuery(schema string, query string) string {
	if schema == "" {
		return query
	}
	return fmt.Sprintf("SET schema = '%s'; %s", schema, query)
}
