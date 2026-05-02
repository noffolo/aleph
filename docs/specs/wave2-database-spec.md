# SPEC-05: DuckDB Concurrency & Database Hardening

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 2  
**Findings addressed**: D1-D12 (database cluster), C1-C6 (caching)  
**Depends on**: `docs/specs/wave0-auth-spec.md` (context-scoped transactions)  
**Related specs**: `docs/specs/wave1-injection-spec.md` (SQL injection fixes share storage paths), `docs/specs/wave4-concurrency-spec.md` (concurrent access patterns)  
**Status**: ✅ Approved — ready for execution

---

## 1. DuckDB Concurrency Model — Redesigned

### Current Design (Problematic)

```
Global RWMutex
├── Read queries → mu.RLock()    ← All reads share lock
├── Write queries → mu.Lock()    ← ALL writes block ALL reads
├── Semaphore (5) → limits concurrent queries
└── Result: reads blocked during any write
```

### New Design (Connection Pool + Dedicated Writer)

```
sql.DB Connection Pool (MaxOpen=10)
├── Read connections (pool, 0-10)
│   └── Read queries → db.QueryContext()  ← NO mutex, direct pool access
│
├── Write connection (dedicated, 1)
│   ├── writeMu.Mutex protects serial writes
│   └── Exec/TX → writeConn.ExecContext()
│
└── Concurrent reader throughput: 2x+ (Rill Data benchmark confirms)
```

### Implementation Interface

```go
// internal/storage/duckdb.go (redesigned)
type DuckDB struct {
    pool      *sql.DB        // Connection pool: MaxOpenConns = runtime.NumCPU()
    writeConn *sql.Conn      // Single reserved connection for writes
    writeMu   sync.Mutex     // Serializes writes (respects DuckDB single-writer constraint)
    
    path     string
    hasVSS   bool
    logger   *slog.Logger
    slowQueryThreshold time.Duration
}

// READS — no mutex, concurrent
func (d *DuckDB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
    return d.pool.QueryContext(ctx, query, args...)
}

func (d *DuckDB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
    return d.pool.QueryRowContext(ctx, query, args...)
}

// WRITES — serialized via writeMu, single write connection
func (d *DuckDB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
    d.writeMu.Lock()
    defer d.writeMu.Unlock()
    return d.writeConn.ExecContext(ctx, query, args...)
}

// TRANSACTIONS
func (d *DuckDB) BeginTX(ctx context.Context) (*TX, error) {
    d.writeMu.Lock()
    // Don't defer unlock here — TX.Commit/Rollback will unlock
    tx, err := d.writeConn.BeginTx(ctx, nil)
    if err != nil {
        d.writeMu.Unlock()
        return nil, err
    }
    return &TX{tx: tx, parentWriteMu: &d.writeMu}, nil
}

func (d *DuckDB) BeginReadTX(ctx context.Context) (*TX, error) {
    // No mutex — read transaction from pool
    tx, err := d.pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
    if err != nil {
        return nil, err
    }
    return &TX{tx: tx}, nil
}
```

### TX Struct

```go
type TX struct {
    tx           *sql.Tx
    parentWriteMu *sync.Mutex // nil for read-only transactions
    done         bool
}

func (tx *TX) Commit() error {
    defer func() {
        if tx.parentWriteMu != nil {
            tx.parentWriteMu.Unlock()
        }
        tx.done = true
    }()
    return tx.tx.Commit()
}

func (tx *TX) Rollback() error {
    defer func() {
        if tx.parentWriteMu != nil {
            tx.parentWriteMu.Unlock()
        }
        tx.done = true
    }()
    return tx.tx.Rollback()
}
```

### Initialization

```go
func NewDuckDB(path string, logger *slog.Logger) (*DuckDB, error) {
    pool, err := sql.Open("duckdb", path)
    if err != nil {
        return nil, err
    }
    
    // Pool configuration
    nCPU := runtime.NumCPU()
    pool.SetMaxOpenConns(nCPU)      // Reads scale with CPU
    pool.SetMaxIdleConns(nCPU)      // Keep connections warm
    pool.SetConnMaxLifetime(1 * time.Hour)
    pool.SetConnMaxIdleTime(15 * time.Minute)
    
    // Reserve write connection
    writeConn, err := pool.Conn(context.Background())
    if err != nil {
        pool.Close()
        return nil, err
    }
    
    return &DuckDB{
        pool:      pool,
        writeConn: writeConn,
        path:      path,
        logger:    logger,
    }, nil
}
```

---

## 2. MemoryStore Atomicity

### Current (Non-Atomic)

```go
// DELETE then INSERT — not atomic, not in transaction
delQ := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, tableName)
db.ExecContext(ctx, delQ, key)
insQ := fmt.Sprintf(`INSERT INTO %s (key, value, embedding) VALUES (?, ?, %s)`, tableName, arrayLiteral(embedding))
db.ExecContext(ctx, insQ, key, value)
```

### Fixed (Atomic Transaction)

```go
func (m *MemoryStore) Store(ctx context.Context, key, value string, embedding []float64) error {
    // Use storage.DuckDB.BeginTX for write transaction
    tx, err := m.db.BeginTX(ctx)
    if err != nil {
        return fmt.Errorf("memory store: begin tx: %w", err)
    }
    defer tx.Rollback()
    
    // DELETE existing
    delQ := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, m.tableName())
    if _, err := tx.ExecContext(ctx, delQ, key); err != nil {
        return fmt.Errorf("memory store: delete: %w", err)
    }
    
    // INSERT new
    insQ := fmt.Sprintf(`INSERT INTO %s (key, value, embedding) VALUES (?, ?, %s)`, 
        m.tableName(), m.arrayLiteral(embedding))
    if _, err := tx.ExecContext(ctx, insQ, key, value); err != nil {
        return fmt.Errorf("memory store: insert: %w", err)
    }
    
    return tx.Commit()
}
```

### Migration: Use storage.DuckDB wrapper

```go
// BEFORE: uses raw sql.DB (bypasses concurrency control)
type MemoryStore struct {
    db     *sql.DB    // ⚠️ raw, no semaphore, no mutex, no schema context
    schema string
    dim    int
}

// AFTER: uses storage.DuckDB wrapper
type MemoryStore struct {
    db     *storage.DuckDB  // ✅ proper concurrency, schema context
    schema string
    dim    int
}
```

---

## 3. DeleteProjectCascade — Two-Phase Atomicity

### Current (Ordering Bug)

```
1. DuckDB.DropProjectSchema()  ← Can't rollback if step 2 fails
2. PostgreSQL DELETE            ← Can fail
```

### Fixed (PostgreSQL First)

```
1. PostgreSQL BEGIN TX
2. PostgreSQL DELETE from 9 tables
3. PostgreSQL COMMIT
   → If COMMIT fails: ROLLBACK, return error (DuckDB still intact)
   → If COMMIT succeeds: proceed to DuckDB
4. DuckDB.DropProjectSchema()
   → If fails: log critical error, add to deferred cleanup queue
   → Cleanup queue retries DuckDB drop every 5 minutes
```

### Idempotent Drop

```go
func DeleteProjectCascadeWithDB(projectID string, d *storage.DuckDB, r *MetadataRepository) error {
    // Phase 1: PostgreSQL (atomic)
    if err := r.DeleteProjectCascade(projectID); err != nil {
        return fmt.Errorf("cascade: postgres phase failed: %w", err)
    }
    
    // Phase 2: DuckDB (best-effort, idempotent)
    storage.SanitizeProjectID(projectID)
    if err := storage.DropProjectSchema(d, projectID); err != nil {
        // Schema might already be dropped or never existed — idempotent
        d.logger.Error("cascade: duckdb schema drop failed, queued for cleanup",
            "projectID", projectID,
            "error", err,
        )
        cleanupQueue.Enqueue(projectID)
    }
    
    return nil
}

// Deferred cleanup
type CleanupQueue struct {
    mu   sync.Mutex
    items map[string]time.Time  // projectID → first failure time
}

func (q *CleanupQueue) Run(ctx context.Context, d *storage.DuckDB) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            q.retryAll(d)
        }
    }
}
```

---

## 4. DuckDBRegistry Concurrency

### Fix: Add Mutex

```go
type DuckDBRegistry struct {
    db     *sql.DB
    mu     sync.RWMutex
    logger *slog.Logger
}

func (r *DuckDBRegistry) RegisterComponent(ctx context.Context, meta ComponentMeta) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Check for duplicates
    existing, _ := r.getComponentByIDLocked(ctx, meta.ID)
    if existing != nil {
        return fmt.Errorf("component %q already exists", meta.ID)
    }
    
    // Insert
    _, err := r.db.ExecContext(ctx, insertQuery, /* 25 params */)
    return err
}

func (r *DuckDBRegistry) GetComponentByID(ctx context.Context, id string) (*Component, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.getComponentByIDLocked(ctx, id)
}
```

---

## 5. PostgreSQL Schema Hardening

### Migration: 000009_add_constraints

```sql
-- migrations/postgres/000009_add_constraints.up.sql

-- NOT NULL constraints
ALTER TABLE system_agents ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_skills ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_tasks ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_api_keys ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_notification_channels ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_chat_sessions ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_chat_history ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_ontology_versions ALTER COLUMN project_id SET NOT NULL;

-- Foreign keys
ALTER TABLE system_agents ADD CONSTRAINT fk_agents_project 
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;
ALTER TABLE system_skills ADD CONSTRAINT fk_skills_project 
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;
ALTER TABLE system_tasks ADD CONSTRAINT fk_tasks_project 
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;
ALTER TABLE system_api_keys ADD CONSTRAINT fk_apikeys_project 
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;
ALTER TABLE system_chat_history ADD CONSTRAINT fk_chat_agent 
    FOREIGN KEY (agent_id) REFERENCES system_agents(id) ON DELETE SET NULL;
ALTER TABLE system_chat_sessions ADD CONSTRAINT fk_sessions_project 
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

-- Indexes
CREATE INDEX idx_agents_project_status ON system_agents(project_id, status);
CREATE INDEX idx_skills_project ON system_skills(project_id);
CREATE INDEX idx_tasks_project_status ON system_tasks(project_id, status);
CREATE INDEX idx_chat_agent_created ON system_chat_history(agent_id, created_at);
CREATE INDEX idx_apikeys_project ON system_api_keys(project_id);
```

```sql
-- migrations/postgres/000009_add_constraints.down.sql
-- Reverse order: indexes → FKs → NOT NULL
DROP INDEX IF EXISTS idx_apikeys_project;
DROP INDEX IF EXISTS idx_chat_agent_created;
DROP INDEX IF EXISTS idx_tasks_project_status;
DROP INDEX IF EXISTS idx_skills_project;
DROP INDEX IF EXISTS idx_agents_project_status;

ALTER TABLE system_chat_sessions DROP CONSTRAINT IF EXISTS fk_sessions_project;
ALTER TABLE system_chat_history DROP CONSTRAINT IF EXISTS fk_chat_agent;
ALTER TABLE system_api_keys DROP CONSTRAINT IF EXISTS fk_apikeys_project;
ALTER TABLE system_tasks DROP CONSTRAINT IF EXISTS fk_tasks_project;
ALTER TABLE system_skills DROP CONSTRAINT IF EXISTS fk_skills_project;
ALTER TABLE system_agents DROP CONSTRAINT IF EXISTS fk_agents_project;

ALTER TABLE system_ontology_versions ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_chat_history ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_chat_sessions ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_notification_channels ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_api_keys ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_tasks ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_skills ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_agents ALTER COLUMN project_id DROP NOT NULL;
```

---

## 6. ToolCache Bounding

```go
type ToolCache struct {
    data    sync.Map
    ttl     time.Duration
    maxSize int             // NEW: default 500
    mu      sync.Mutex      // NEW: protects eviction
    keys    []string        // NEW: LRU tracking (ordered by access time)
}

func (c *ToolCache) Set(key string, value any) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if at capacity
    if len(c.keys) >= c.maxSize {
        oldest := c.keys[0]
        c.keys = c.keys[1:]
        c.data.Delete(oldest)
    }
    
    // Add new
    c.keys = append(c.keys, key)
    c.data.Store(key, cacheEntry{
        value:  value,
        expiry: time.Now().Add(c.ttl),
    })
}

// Background cleanup
func (c *ToolCache) startCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.evictExpired()
        }
    }
}
```

---

## 7. Verification

### Test Coverage

- [ ] `duckdb_concurrent_test.go` (NEW): 100 goroutines (50 read, 50 write) — reads never block
- [ ] `duckdb_tx_test.go` (NEW): Concurrent write TX — second blocks until first commits
- [ ] `memory_atomic_test.go` (NEW): 10 goroutines upsert same key — last write wins
- [ ] `registry_concurrent_test.go` (NEW): 100 RegisterComponent — no duplicates
- [ ] `project_cascade_test.go` (NEW): Cascade delete — all DuckDB + PG gone
- [ ] `backup_test.go` (expand): Backup during concurrent reads — consistent, reads succeed
- [ ] `cache_bounded_test.go` (NEW): 600 entries, max=500 → oldest 100 evicted

### Gate

```
go test -race -count=3 ./internal/storage/ ./internal/repository/ ./internal/memory/ ./internal/registry/
→ ALL pass, ZERO race detector warnings
→ go test -bench=. ./internal/storage/ — concurrency improvement confirmed

psql -c "\d+ system_agents"
→ NOT NULL on project_id, FK to system_projects, indexes present
```
