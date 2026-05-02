package storage

import (
	"context"
	"database/sql"
	stdErrors "errors"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/safeident"
)

// DefaultSlowQueryThreshold is the default threshold for slow query logging.
const DefaultSlowQueryThreshold = 500 * time.Millisecond

// DuckDB wraps a *sql.DB connection pool with serialized write access.
// Reads use the connection pool directly (no mutex). Writes are serialized
// via writeMu and executed through the pool (db.ExecContext).
// This allows concurrent reads while ensuring write safety.
//
// Connection pool sizing:
//   - In-memory databases: 1 connection (DuckDB in-memory is per-connection)
//   - File-backed databases: runtime.NumCPU() max connections
type DuckDB struct {
	db                 *sql.DB
	writeMu            sync.Mutex
	poolMu             sync.Mutex // guards pool configuration (SetMaxOpenConns, SetMaxIdleConns)
	path               string
	HasVSS             bool
	slowQueryThreshold time.Duration
}

func NewDuckDB(dbPath string) (*DuckDB, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

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
	// File-backed databases use a pool sized to runtime.NumCPU().
	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		maxConns := runtime.NumCPU()
		db.SetMaxOpenConns(maxConns)
		db.SetMaxIdleConns(maxConns)
		db.SetConnMaxLifetime(1 * time.Hour)
	}

	return &DuckDB{
		db:                 db,
		path:               dbPath,
		HasVSS:             hasVSS,
		slowQueryThreshold: DefaultSlowQueryThreshold,
	}, nil
}

func (d *DuckDB) SetSlowQueryThreshold(dur time.Duration) {
	if dur <= 0 {
		dur = DefaultSlowQueryThreshold
	}
	d.slowQueryThreshold = dur
}

// logSlowQuery logs a warning if the query duration exceeds the configured threshold.
func (d *DuckDB) logSlowQuery(operation, query string, dur time.Duration) {
	if dur < d.slowQueryThreshold {
		return
	}
	// Truncate query to first 200 chars for readability
	queryPreview := query
	if len(queryPreview) > 200 {
		queryPreview = queryPreview[:200] + "..."
	}
	slog.Warn("slow duckdb query",
		"operation", operation,
		"duration", dur,
		"threshold", d.slowQueryThreshold,
		"query", queryPreview,
	)
}

// Query executes a read query using the connection pool.
// No mutex held — reads are fully concurrent.
func (d *DuckDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.db.Query(query, args...)
	d.logSlowQuery("Query", query, time.Since(start))
	return rows, err
}

// QueryContext executes a read query using the connection pool.
// No mutex held — reads are fully concurrent.
func (d *DuckDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.db.QueryContext(ctx, scopeQuery(ctx, query), args...)
	d.logSlowQuery("QueryContext", query, time.Since(start))
	return rows, err
}

// Exec executes a write query. Serialized via writeMu to prevent concurrent writes.
// Prefer ExecContext to avoid orphaned operations.
func (d *DuckDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	start := time.Now()
	res, err := d.db.ExecContext(context.TODO(), query, args...)
	d.logSlowQuery("Exec", query, time.Since(start))
	return res, err
}

// ExecContext executes a write query. Serialized via writeMu to prevent concurrent writes.
func (d *DuckDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	start := time.Now()
	res, err := d.db.ExecContext(ctx, scopeQuery(ctx, query), args...)
	d.logSlowQuery("ExecContext", query, time.Since(start))
	return res, err
}

func (d *DuckDB) Cleanup() {
	// No-op: DuckDB memory management differs from SQLite.
}

// Close closes the database pool. Safe to call multiple times; nil-guarded on d.db.
func (d *DuckDB) Close() error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return fmt.Errorf("duckdbClose: %w", err)
		}
	}
	return nil
}

// QueryRowContext executes a query returning at most one row using the pool.
// The returned *sql.Row will error on Scan if the query fails.
func (d *DuckDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := d.db.QueryRowContext(ctx, scopeQuery(ctx, query), args...)
	d.logSlowQuery("QueryRowContext", query, time.Since(start))
	return row
}

// QueryRowContextOrError is like QueryRowContext but returns (row, error) for callers
// that need to detect context cancellation early. The returned row is never nil.
func (d *DuckDB) QueryRowContextOrError(ctx context.Context, query string, args ...interface{}) (*sql.Row, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start := time.Now()
	row := d.db.QueryRowContext(ctx, scopeQuery(ctx, query), args...)
	d.logSlowQuery("QueryRowContextOrError", query, time.Since(start))
	return row, nil
}

// QueryRow executes a query returning at most one row using the pool.
func (d *DuckDB) QueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := d.db.QueryRow(query, args...)
	d.logSlowQuery("QueryRow", query, time.Since(start))
	return row
}

// DB returns the underlying *sql.DB connection pool.
func (d *DuckDB) DB() *sql.DB {
	return d.db
}

// TX wraps a *sql.Tx with schema context and optional write-lock tracking.
// For write transactions (BeginTX), writeMu is held for the entire transaction
// and released on Commit/Rollback.
// For read transactions (BeginReadTX), writeMu is nil and no cross-transaction lock is held.
//
// All query methods use an internal RWMutex for safety against concurrent
// use on the same TX handle. Each TX should be used by one goroutine at a time.
// Callers must call Commit or Rollback to release resources.
type TX struct {
	tx       *sql.Tx
	mu       sync.RWMutex
	writeMu  *sync.Mutex // non-nil for write transactions; released on Commit/Rollback
	schema   string
	done     bool

	slowQueryThreshold time.Duration
}

// BeginTX starts a new write transaction. Acquires writeMu to serialize
// with other write operations (Exec, ExecContext, other BeginTX calls).
// The transaction is serializable isolation. Schema from context is applied
// at transaction start.
// Call Commit or Rollback to release the write lock.
func (d *DuckDB) BeginTX(ctx context.Context) (*TX, error) {
	d.writeMu.Lock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.writeMu.Unlock()
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	schema, _ := SchemaFromContext(ctx)

	// Apply schema at transaction start if set.
	if schema != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET schema = %s", safeident.QuoteIdentifier(schema))); err != nil {
			_ = tx.Rollback()
			d.writeMu.Unlock()
			return nil, fmt.Errorf("set schema in tx: %w", err)
		}
	}

	return &TX{
		tx:       tx,
		schema:   schema,
		writeMu:  &d.writeMu,
		slowQueryThreshold: d.slowQueryThreshold,
	}, nil
}

// BeginReadTX starts a new transaction for read operations using the connection pool.
// No write lock is acquired — multiple read transactions can run concurrently
// with each other and with write transactions (the pool manages concurrency).
// Schema from context is applied at transaction start.
//
// Note: DuckDB driver does not support explicit ReadOnly transactions. Serializable
// isolation provides a consistent snapshot for reads without blocking writes.
func (d *DuckDB) BeginReadTX(ctx context.Context) (*TX, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin read tx: %w", err)
	}

	schema, _ := SchemaFromContext(ctx)

	// Apply schema at transaction start if set.
	if schema != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET schema = %s", safeident.QuoteIdentifier(schema))); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("set schema in read tx: %w", err)
		}
	}

	return &TX{
		tx:       tx,
		schema:   schema,
		writeMu:  nil,
		slowQueryThreshold: d.slowQueryThreshold,
	}, nil
}

// isSerializationError checks if the error is a serialization failure that warrants retry.
func isSerializationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "serialization") ||
		strings.Contains(errStr, "sqlite_busy") ||
		strings.Contains(errStr, "could not serialize")
}

// ExecWithRetry wraps ExecContext with up to 3 retries and exponential backoff.
// Backoff intervals: 10ms, 50ms, 250ms.
func (d *DuckDB) ExecWithRetry(ctx context.Context, query string, args ...any) (sql.Result, error) {
	const maxRetries = 3
	backoffMs := []int{10, 50, 250}

	for i := 0; i <= maxRetries; i++ {
		result, err := d.ExecContext(ctx, query, args...)
		if err == nil {
			return result, nil
		}
		if !isSerializationError(err) || i == maxRetries {
			return nil, err
		}
		time.Sleep(time.Duration(backoffMs[i]) * time.Millisecond)
	}
	return nil, stdErrors.New("exec with retry: exhausted all retries")
}

// ExecTx wraps BeginTX + fn + Commit/Rollback with retry on serialization failures.
// If serialization fails, the transaction is retried up to 3 times.
func (d *DuckDB) ExecTx(ctx context.Context, fn func(*TX) error) error {
	const maxRetries = 3
	backoffMs := []int{10, 50, 250}

	for i := 0; i <= maxRetries; i++ {
		tx, err := d.BeginTX(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		err = fn(tx)
		if err == nil {
			err = tx.Commit()
			if err == nil {
				return nil
			}
		}

		// Rollback on error, but only if not already committed.
		// tx.Rollback is idempotent (done flag).
		_ = tx.Rollback()

		if !isSerializationError(err) || i == maxRetries {
			return err
		}

		time.Sleep(time.Duration(backoffMs[i]) * time.Millisecond)
	}

	return stdErrors.New("exec tx: exhausted all retries")
}

// Commit commits the transaction and releases the write lock (if this is a
// write transaction). Safe to call multiple times — subsequent calls are no-ops.
func (t *TX) Commit() error {
	t.mu.Lock()
	if t.done {
		t.mu.Unlock()
		return nil
	}
	t.done = true
	t.mu.Unlock()

	err := t.tx.Commit()

	if t.writeMu != nil {
		t.writeMu.Unlock()
	}

	if err != nil {
		return fmt.Errorf("txCommit: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction and releases the write lock (if this is a
// write transaction). Safe to call multiple times — subsequent calls are no-ops.
func (t *TX) Rollback() error {
	t.mu.Lock()
	if t.done {
		t.mu.Unlock()
		return nil
	}
	t.done = true
	t.mu.Unlock()

	err := t.tx.Rollback()

	if t.writeMu != nil {
		t.writeMu.Unlock()
	}

	if err != nil {
		return fmt.Errorf("txRollback: %w", err)
	}
	return nil
}

// Query executes a query on the transaction.
func (t *TX) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := time.Now()
	rows, err := t.tx.Query(txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "Query", query, time.Since(start))
	return rows, err
}

// QueryContext executes a query on the transaction.
func (t *TX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := time.Now()
	rows, err := t.tx.QueryContext(ctx, txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "QueryContext", query, time.Since(start))
	return rows, err
}

// Exec executes a statement on the transaction.
func (t *TX) Exec(query string, args ...interface{}) (sql.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	start := time.Now()
	res, err := t.tx.Exec(txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "Exec", query, time.Since(start))
	return res, err
}

// ExecContext executes a statement on the transaction.
func (t *TX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	start := time.Now()
	res, err := t.tx.ExecContext(ctx, txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "ExecContext", query, time.Since(start))
	return res, err
}

// QueryRow executes a query that returns at most one row on the transaction.
func (t *TX) QueryRow(query string, args ...interface{}) *sql.Row {
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := time.Now()
	row := t.tx.QueryRow(txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "QueryRow", query, time.Since(start))
	return row
}

// QueryRowContext executes a query that returns at most one row on the transaction.
func (t *TX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := time.Now()
	row := t.tx.QueryRowContext(ctx, txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "QueryRowContext", query, time.Since(start))
	return row
}

// QueryRowContextOrError is like QueryRowContext but returns (row, error)
// for callers that need to handle nil gracefully.
func (t *TX) QueryRowContextOrError(ctx context.Context, query string, args ...interface{}) (*sql.Row, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := time.Now()
	row := t.tx.QueryRowContext(ctx, txScopeQuery(t.schema, query), args...)
	slowTxQueryLog(t.slowQueryThreshold, "QueryRowContextOrError", query, time.Since(start))
	return row, nil
}

func slowTxQueryLog(threshold time.Duration, operation, query string, dur time.Duration) {
	if dur < threshold {
		return
	}
	slog.Warn("slow tx query",
		"op", operation,
		"query", truncateQuery(query),
		"duration", dur,
		"threshold", threshold)
}

// truncateQuery returns the first 200 characters of a query string
// for logging purposes. Longer queries are truncated with "...".
func truncateQuery(query string) string {
	if len(query) > 200 {
		return query[:200] + "..."
	}
	return query
}

// txScopeQuery returns a query string that sets the schema if one was configured
// at transaction start. The schema SET is prepended once per query to ensure
// DuckDB's session-scoped schema setting is maintained across individual operations.
func txScopeQuery(schema string, query string) string {
	if schema == "" {
		return query
	}
	return fmt.Sprintf("SET schema = %s; %s", safeident.QuoteIdentifier(schema), query)
}
