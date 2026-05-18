package storage

import (
	"context"
	"database/sql"
)

// DBExecutor is a minimal interface for executing SQL queries against a database.
// It exposes the three core database/sql execution methods — ExecContext,
// QueryContext, and QueryRowContext — without requiring the full *sql.DB surface.
//
// DuckDB is the canonical implementation. Callers in handler, decision,
// middleware, memory, tools, diagnostic, and repository packages accept
// DBExecutor instead of *DuckDB, enabling testing with :memory: DuckDB
// without importing the storage package directly.
type DBExecutor interface {
	// ExecContext executes a query that does not return rows (INSERT, UPDATE,
	// DELETE, DDL). The arguments are for placeholder parameters in the query.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryContext executes a query that returns rows, typically a SELECT.
	// The caller must close the returned *sql.Rows.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRowContext executes a query that returns at most one row.
	// Errors are deferred until Scan is called on the returned *sql.Row.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
