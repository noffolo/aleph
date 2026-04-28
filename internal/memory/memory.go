package memory

import (
	"database/sql"
	"fmt"
)

// MemoryStore provides embedding-augmented memory for the decision engine.
// Currently a stub; full implementation deferred to a future wave.
type MemoryStore struct {
	db     *sql.DB
	schema string
	dim    int
}

// NewMemoryStore creates a new MemoryStore backed by the given DB.
// Returns an error if the DB is nil or unreachable, allowing graceful degradation.
func NewMemoryStore(db *sql.DB, schema string, embeddingDim int) (*MemoryStore, error) {
	if db == nil {
		return nil, fmt.Errorf("memory: db is nil")
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("memory: db unreachable: %w", err)
	}
	return &MemoryStore{db: db, schema: schema, dim: embeddingDim}, nil
}

func (m *MemoryStore) Close() error {
	return nil
}