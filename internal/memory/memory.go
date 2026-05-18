package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/safeident"
	"github.com/ff3300/aleph-v2/internal/storage"
)

// SearchResult holds a single result from a vector similarity or text search.
type SearchResult struct {
	Key   string
	Value []byte
	Score float64
}

// MemEntry holds a single entry returned by List.
type MemEntry struct {
	Key       string
	Value     []byte
	Embedding []float32
}

// MemoryStore provides embedding-augmented memory for the decision engine.
// Backed by DuckDB's native FLOAT[] arrays and array_cosine_similarity for VSS.
// Writes are routed through the DuckDB wrapper (serialized via writeMu);
// reads use the connection pool directly via QueryContext.
type MemoryStore struct {
	db      *storage.DuckDB
	schema  string
	dim     int
	init    sync.Once
	initErr error
}

// expectedEmbedDim is the expected embedding dimension for nomic-embed-text.
const expectedEmbedDim = 768

// NewMemoryStore creates a new MemoryStore backed by the given DuckDB wrapper.
// Writes are serialized through the wrapper's writeMu; reads use the connection pool.
func NewMemoryStore(db *storage.DuckDB, schema string, embeddingDim int) (*MemoryStore, error) {
	if db == nil {
		return nil, fmt.Errorf("memory: db is nil")
	}
	if embeddingDim <= 0 {
		return nil, fmt.Errorf("memory: embedding dimension must be positive, got %d", embeddingDim)
	}
	if schema != "" {
		if err := safeident.ValidateIdentifier(schema); err != nil {
			return nil, fmt.Errorf("memory: invalid schema name: %w", err)
		}
	}
	if embeddingDim != expectedEmbedDim {
		slog.Warn("memory: embedding dimension differs from nomic-embed-text default",
			"expected", expectedEmbedDim, "got", embeddingDim)
	}
	return &MemoryStore{db: db, schema: schema, dim: embeddingDim}, nil
}

// Close is a no-op; the underlying DB is owned by the caller.
func (m *MemoryStore) Close() error {
	return nil
}

// Store stores a key-value pair with its embedding vector.
// Uses DELETE + INSERT because DuckDB's FLOAT[] columns do not support
// INSERT OR REPLACE or ON CONFLICT DO UPDATE. Transactions cannot be used
// because DuckDB's PRIMARY KEY enforcement doesn't see intra-transaction
// DELETEs (constraint check occurs before the delete takes effect).
// Instead, we retry the full DELETE+INSERT on constraint violation.
func (m *MemoryStore) Store(ctx context.Context, key string, value []byte, embedding []float32) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	delQ := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, m.tableName())
	insQ := fmt.Sprintf(
		`INSERT INTO %s (key, value, embedding) VALUES (?, ?, %s)`,
		m.tableName(), m.arrayLiteral(embedding),
	)

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if _, err := m.db.ExecContext(ctx, delQ, key); err != nil {
			return fmt.Errorf("memory Store delete: %w", err)
		}
		if _, err := m.db.ExecContext(ctx, insQ, key, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("memory Store: failed after %d retries", maxRetries)
}

// Search performs vector similarity search using DuckDB's array_cosine_similarity.
func (m *MemoryStore) Search(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(
		`SELECT key, value, array_cosine_similarity(embedding, %s) AS score FROM %s ORDER BY score DESC LIMIT ?`,
		m.arrayLiteral(queryEmbedding), m.tableName(),
	)
	rows, err := m.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("memory Search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Key, &r.Value, &r.Score); err != nil {
			return nil, fmt.Errorf("memory Search scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory Search rows: %w", err)
	}
	return results, nil
}

// SearchText performs a fallback text search using ILIKE on both key and value.
func (m *MemoryStore) SearchText(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(
		`SELECT key, value FROM %s WHERE key ILIKE '%%' || ? || '%%' OR value::VARCHAR ILIKE '%%' || ? || '%%' LIMIT ?`,
		m.tableName(),
	)
	rows, err := m.db.QueryContext(ctx, q, query, query, limit)
	if err != nil {
		return nil, fmt.Errorf("memory SearchText: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Key, &r.Value); err != nil {
			return nil, fmt.Errorf("memory SearchText scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory SearchText rows: %w", err)
	}
	return results, nil
}

// Get retrieves a value by key. Returns (value, true) if found, (nil, false) otherwise.
func (m *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool) {
	q := fmt.Sprintf(`SELECT value FROM %s WHERE key = ?`, m.tableName())
	row := m.db.QueryRowContext(ctx, q, key)
	var value []byte
	if err := row.Scan(&value); err != nil {
		return nil, false
	}
	return value, true
}

// Delete removes a key-value pair from the store.
func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	q := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, m.tableName())
	if _, err := m.db.ExecContext(ctx, q, key); err != nil {
		return fmt.Errorf("memory Delete: %w", err)
	}
	return nil
}

// List returns paginated entries ordered by key.
func (m *MemoryStore) List(ctx context.Context, limit, offset int) ([]MemEntry, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`SELECT key, value, embedding FROM %s ORDER BY key LIMIT ? OFFSET ?`, m.tableName())
	rows, err := m.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("memory List: %w", err)
	}
	defer rows.Close()

	var entries []MemEntry
	for rows.Next() {
		var e MemEntry
		var rawEmbedding []any
		if err := rows.Scan(&e.Key, &e.Value, &rawEmbedding); err != nil {
			return nil, fmt.Errorf("memory List scan: %w", err)
		}
		e.Embedding = make([]float32, len(rawEmbedding))
		for i, v := range rawEmbedding {
			switch val := v.(type) {
			case float32:
				e.Embedding[i] = val
			case float64:
				e.Embedding[i] = float32(val)
			default:
				return nil, fmt.Errorf("memory List scan: unexpected embedding element type %T at index %d", v, i)
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory List rows: %w", err)
	}
	return entries, nil
}

// tableName returns the schema-qualified table name.
func (m *MemoryStore) tableName() string {
	if m.schema != "" {
		return safeident.QuoteIdentifier(m.schema) + ".memory_store"
	}
	return "memory_store"
}

// arrayLiteral formats a []float32 as a DuckDB array literal with bracket syntax.
func (m *MemoryStore) arrayLiteral(embedding []float32) string {
	if len(embedding) == 0 {
		return fmt.Sprintf("CAST(ARRAY[] AS FLOAT[%d])", m.dim)
	}
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return fmt.Sprintf("[%s]::FLOAT[%d]", strings.Join(parts, ","), m.dim)
}

// ensureTable creates the memory_store table if it does not yet exist.
// Uses sync.Once with up to maxInitAttempts retries on failure.
func (m *MemoryStore) ensureTable(ctx context.Context) error {
	const maxInitAttempts = 3
	m.init.Do(func() {
		q := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			key VARCHAR PRIMARY KEY,
			value BLOB,
			embedding FLOAT[%d],
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`, m.tableName(), m.dim)
		for attempt := 1; attempt <= maxInitAttempts; attempt++ {
			_, m.initErr = m.db.ExecContext(ctx, q)
			if m.initErr == nil {
				return
			}
			slog.Warn("memory: table creation attempt failed",
				"attempt", attempt, "max", maxInitAttempts, "error", m.initErr)
			if attempt < maxInitAttempts {
				select {
				case <-ctx.Done():
					m.initErr = ctx.Err()
					return
				case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
				}
			}
		}
	})
	return m.initErr
}
