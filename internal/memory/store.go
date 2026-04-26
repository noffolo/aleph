package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MemoryEntry represents a single stored memory with embedding and metadata.
type MemoryEntry struct {
	ID        string    `json:"id"`
	Namespace string    `json:"namespace"`
	Content   string    `json:"content"`
	Source    string    `json:"source"` // "chat", "document", "tool_result"
	Metadata  string    `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Score     float64   `json:"score,omitempty"`
}

// MemoryStore provides vector-based memory storage and retrieval using DuckDB's
// built-in list_cosine_similarity. Zero additional runtime dependencies.
type MemoryStore struct {
	db        *sql.DB
	schema    string
	mu        sync.RWMutex
	dimension int
}

// NewMemoryStore creates a MemoryStore backed by the given DuckDB connection.
// The schema parameter is the DuckDB schema used for table creation.
// dimension is the embedding vector size (e.g. 768 for nomic-embed-text).
func NewMemoryStore(db *sql.DB, schema string, dimension int) (*MemoryStore, error) {
	s := &MemoryStore{
		db:        db,
		schema:    schema,
		dimension: dimension,
	}
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("memory store init: %w", err)
	}
	return s, nil
}

func (s *MemoryStore) init() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.memory_entries (
			id          VARCHAR PRIMARY KEY,
			namespace   VARCHAR NOT NULL,
			content     VARCHAR NOT NULL,
			source      VARCHAR NOT NULL DEFAULT 'chat',
			metadata    VARCHAR DEFAULT '{}',
			embedding   FLOAT[%d],
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_memory_namespace
			ON %s.memory_entries (namespace);
	`, s.schema, s.dimension, s.schema)

	_, err := s.db.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("create memory_entries table: %w", err)
	}
	return nil
}

// Insert stores a memory entry with its embedding vector.
func (s *MemoryStore) Insert(ctx context.Context, id, namespace, content, source, metadata string, embedding []float32) error {
	metaStr := metadata
	if metaStr == "" {
		metaStr = "{}"
	}

	embedJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	// DuckDB does not support INSERT OR REPLACE for FLOAT[] arrays,
	// so we DELETE first, then INSERT.
	delQ := fmt.Sprintf(`DELETE FROM %s.memory_entries WHERE id = ?`, s.schema)
	if _, err = s.db.ExecContext(ctx, delQ, id); err != nil {
		return fmt.Errorf("delete before insert: %w", err)
	}

	now := time.Now()
	insQ := fmt.Sprintf(`
		INSERT INTO %s.memory_entries (id, namespace, content, source, metadata, embedding, created_at)
		VALUES (?, ?, ?, ?, ?, CAST(? AS FLOAT[%d]), ?)
	`, s.schema, s.dimension)

	_, err = s.db.ExecContext(ctx, insQ, id, namespace, content, source, metaStr, string(embedJSON), now)
	if err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}
	return nil
}

// Search finds the top-k most similar memories in the given namespace using
// cosine similarity on DuckDB FLOAT[] arrays.
func (s *MemoryStore) Search(ctx context.Context, namespace string, queryEmbedding []float32, limit int) ([]MemoryEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	embedJSON, err := json.Marshal(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("marshal query embedding: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, namespace, content, source, metadata, created_at,
			list_cosine_similarity(embedding, CAST(? AS FLOAT[%d])) AS score
		FROM %s.memory_entries
		WHERE namespace = ?
			AND embedding IS NOT NULL
		ORDER BY score DESC
		LIMIT ?
	`, s.dimension, s.schema)

	rows, err := s.db.QueryContext(ctx, q, string(embedJSON), namespace, limit)
	if err != nil {
		return nil, fmt.Errorf("search memory: %w", err)
	}
	defer rows.Close()

	var results []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var metaStr string
		if err := rows.Scan(&e.ID, &e.Namespace, &e.Content, &e.Source, &metaStr, &e.CreatedAt, &e.Score); err != nil {
			return nil, fmt.Errorf("scan memory row: %w", err)
		}
		e.Metadata = metaStr
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id, namespace string) error {
	query := fmt.Sprintf(`DELETE FROM %s.memory_entries WHERE id = ? AND namespace = ?`, s.schema)
	_, err := s.db.ExecContext(ctx, query, id, namespace)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

// DeleteNamespace removes all memory entries in the given namespace.
func (s *MemoryStore) DeleteNamespace(ctx context.Context, namespace string) error {
	query := fmt.Sprintf(`DELETE FROM %s.memory_entries WHERE namespace = ?`, s.schema)
	_, err := s.db.ExecContext(ctx, query, namespace)
	if err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

// Count returns the number of memory entries in the given namespace (or total if namespace is empty).
func (s *MemoryStore) Count(ctx context.Context, namespace string) (int, error) {
	var q string
	var args []interface{}
	if namespace != "" {
		q = fmt.Sprintf(`SELECT COUNT(*) FROM %s.memory_entries WHERE namespace = ?`, s.schema)
		args = append(args, namespace)
	} else {
		q = fmt.Sprintf(`SELECT COUNT(*) FROM %s.memory_entries`, s.schema)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count memory: %w", err)
	}
	return count, nil
}

// ListNamespaces returns all unique namespaces with entry counts.
func (s *MemoryStore) ListNamespaces(ctx context.Context) (map[string]int, error) {
	query := fmt.Sprintf(`SELECT namespace, COUNT(*) as cnt FROM %s.memory_entries GROUP BY namespace ORDER BY cnt DESC`, s.schema)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var ns string
		var cnt int
		if err := rows.Scan(&ns, &cnt); err != nil {
			return nil, fmt.Errorf("scan namespace row: %w", err)
		}
		result[ns] = cnt
	}
	return result, nil
}

// LogAccess records a memory access for audit/telemetry.
func (s *MemoryStore) LogAccess(ctx context.Context, operation string, namespace string, details string) {
	slog.Debug("memory store access",
		"operation", operation,
		"namespace", namespace,
		"details", details,
	)
}
