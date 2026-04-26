package humanecosystems

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/storage"
)

// DuckDBLayer provides DuckDB integration for human ecosystems tools.
// It wraps storage.DuckDB with schema-aware context propagation.
type DuckDBLayer struct {
	db   *storage.DuckDB
	mu   sync.RWMutex
	path string
}

// NewDuckDBLayer creates a new DuckDBLayer backed by a storage.DuckDB instance.
// When db is nil the layer gracefully degrades and returns synthetic results.
func NewDuckDBLayer(db *storage.DuckDB) *DuckDBLayer {
	return &DuckDBLayer{db: db}
}

// QueryContext executes a read-only SQL query with context and schema scope.
// When the underlying DuckDB is nil it returns a synthetic result.
func (d *DuckDBLayer) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		slog.Warn("DuckDBLayer: no DuckDB instance available, returning synthetic result",
			"query", query)
		return nil, fmt.Errorf("duckdb not available")
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdb query: %w", err)
	}
	return rows, nil
}

// ExecContext executes a write SQL statement with context and schema scope.
func (d *DuckDBLayer) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("duckdb not available")
	}

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdb exec: %w", err)
	}
	return result, nil
}

// IsAvailable returns true when the underlying DuckDB instance is ready.
func (d *DuckDBLayer) IsAvailable() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db != nil
}

// SchemaContext returns a context scoped to the given project schema.
func (d *DuckDBLayer) SchemaContext(ctx context.Context, projectID string) context.Context {
	if projectID == "" {
		return ctx
	}
	if err := storage.SanitizeProjectID(projectID); err != nil {
		slog.Warn("SchemaContext: invalid projectID rejected", "projectID", projectID, "error", err)
		return ctx
	}
	return storage.ContextWithSchema(ctx, "project_"+projectID)
}

// SyntheticDuckDBLayer returns a DuckDBLayer that always returns synthetic
// results. Useful for testing or when the real database is not available.
func SyntheticDuckDBLayer() *DuckDBLayer {
	return &DuckDBLayer{db: nil}
}

// syntheticRowCount returns a synthetic result count for tools that
// gracefully degrade when DuckDB is unavailable.
func syntheticRowCount() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"count":        0,
			"is_synthetic": true,
			"message":      "DuckDB unavailable — data may not reflect real ecosystem state",
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
}
