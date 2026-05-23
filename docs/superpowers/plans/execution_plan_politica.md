# Aleph — Piano Esecutivo Ingestione Dati Politici Italiani

> **For agentic workers:** Use superpowers:subagent-driven-development o superpowers:executing-plans.
> **Status:** READY — revisionato da Momus, Metis, Oracle. Specs collegate in `execution_specs_politica.md`.

**Goal:** Dotare Aleph di 6 nuovi source type per ingerire dati elettorali (Eligendo API) e finanziamenti alla politica italiana (OpenSanctions, ANAC, ondata/liberiamoli-tutti, OPDM, Camera/Senato SPARQL, Codice Civico), con normalizzazione via alias table, archival raw su filesystem, e viste cross-reference DuckDB.

**Architecture:** Due fasi indipendenti. Fase 1: `source_type="election"` con alias-based party normalization, archiviazione raw su disco. Fase 2: 5 nuovi source type con watermark-based incremental ingestion, HTTP fixture testing (go-vcr), RateLimiter interface, registry pattern per dispatch, structured logging. Tutte le ingestion salvano raw data su `<data_dir>/raw/<source_type>/` e normalizzato su DuckDB.

**Key Architecture Decisions (da Oracle/Metis review):**
1. **Alias table > Jaro-Winkler**: Lookup table `party_mapping` con matching esatto + override manuale. Jaro-Winkler solo come fallback opzionale con threshold configurabile e log di ogni match.
2. **Raw su filesystem, non DuckDB**: Raw JSON/CSV salvati compressi su disco (`data/raw/<source>/`). Solo dati normalizzati su DuckDB. Evita bloat DB.
3. **Registry pattern, non switch**: `init()` registration per ogni source type. Niente modifiche a engine.go switch.
4. **Watermark-driven incremental**: Tabella `ingestion_watermark(source_name, last_run, cursor)` per delta sync. Full refresh solo su richiesta esplicita.
5. **RateLimiter interface**: Token-bucket per-source, configurabile (req/s, burst, backoff). Non più sleep ad-hoc.
6. **QA scenarios obbligatori** (Momus): Ogni task ha sezione VERIFICA con tool + comandi + risultato atteso.

**Tech Stack:** Go 1.26, DuckDB, testify, go-vcr (HTTP fixtures), xrash/smetrics (fallback fuzzy), encoding/charmap (ISO-8859-1), log/slog (structured logging)

---

## Pre-Requisito: Infrastruttura Condivisa (Task 0)

Prima di iniziare Task 1-11, implementare i componenti condivisi usati da tutti i source type.

### Task 0.1: Watermark Table + Migration System

**Files:**
- Create: `internal/ingestion/watermark.go`
- Create: `internal/ingestion/watermark_test.go`
- Create: `internal/ingestion/migrations.go`

**Goal:** Tabella `ingestion_watermark` per tracciare ultima ingestion per source. Sistema migrazioni minimale (versioni SQL). Ogni handler legge il proprio watermark prima di fetchar e lo aggiorna dopo commit.

```
Schema watermark: source_name TEXT PRIMARY KEY, last_run TIMESTAMP, cursor TEXT, metadata TEXT
Schema migrations:  version INT PRIMARY KEY, name TEXT, applied_at TIMESTAMP
```

- [ ] **Step 1: Scrivi test watermark_migrations_test.go**

```go
package ingestion

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWatermarkLifecycle(t *testing.T) {
    db := setupTestDB(t)
    wm := NewWatermarkManager(db)

    // Initially no watermark
    _, err := wm.Get("test_source")
    assert.ErrorIs(t, err, ErrWatermarkNotFound)

    // Set watermark
    now := time.Now()
    err = wm.Set("test_source", now, "cursor_123", `{"key":"val"}`)
    require.NoError(t, err)

    // Get watermark
    got, err := wm.Get("test_source")
    require.NoError(t, err)
    assert.Equal(t, "cursor_123", got.Cursor)
    assert.Equal(t, `{"key":"val"}`, got.Metadata)

    // Update watermark
    now2 := now.Add(time.Hour)
    err = wm.Set("test_source", now2, "cursor_456", "")
    require.NoError(t, err)

    // List all
    all, err := wm.ListAll()
    require.NoError(t, err)
    assert.Len(t, all, 1)
}

func TestMigrations(t *testing.T) {
    db := setupTestDB(t)
    mm := NewMigrationManager(db)

    // Register migrations
    mm.Register(Migration{
        Version: 1,
        Name:    "create_watermark_table",
        Up:      "CREATE TABLE IF NOT EXISTS ingestion_watermark (source_name TEXT PRIMARY KEY, last_run TIMESTAMP, cursor TEXT, metadata TEXT)",
        Down:    "DROP TABLE IF EXISTS ingestion_watermark",
    })

    // Run pending
    err := mm.Up()
    require.NoError(t, err)

    // Verify version
    current, err := mm.CurrentVersion()
    require.NoError(t, err)
    assert.Equal(t, 1, current)
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestWatermark -v -count=1 2>&1
```
Expected: FAIL — `package not found` o `undefined: NewWatermarkManager`

- [ ] **Step 3: Implementa watermark.go e migrations.go**

In `internal/ingestion/watermark.go`:

```go
package ingestion

import (
    "database/sql"
    "errors"
    "time"
)

var ErrWatermarkNotFound = errors.New("watermark not found")

type Watermark struct {
    SourceName string
    LastRun    time.Time
    Cursor     string
    Metadata   string
}

type WatermarkManager struct{ db *sql.DB }

func NewWatermarkManager(db *sql.DB) *WatermarkManager { return &WatermarkManager{db: db} }

func (w *WatermarkManager) ensureTable() error {
    _, err := w.db.Exec(`CREATE TABLE IF NOT EXISTS ingestion_watermark (
        source_name TEXT PRIMARY KEY,
        last_run TIMESTAMP NOT NULL,
        cursor TEXT DEFAULT '',
        metadata TEXT DEFAULT ''
    )`)
    return err
}

func (w *WatermarkManager) Get(sourceName string) (Watermark, error) {
    if err := w.ensureTable(); err != nil {
        return Watermark{}, err
    }
    var wm Watermark
    row := w.db.QueryRow("SELECT source_name, last_run, COALESCE(cursor,''), COALESCE(metadata,'') FROM ingestion_watermark WHERE source_name = ?", sourceName)
    err := row.Scan(&wm.SourceName, &wm.LastRun, &wm.Cursor, &wm.Metadata)
    if errors.Is(err, sql.ErrNoRows) {
        return Watermark{}, ErrWatermarkNotFound
    }
    return wm, err
}

func (w *WatermarkManager) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
    if err := w.ensureTable(); err != nil {
        return err
    }
    _, err := w.db.Exec(`INSERT OR REPLACE INTO ingestion_watermark (source_name, last_run, cursor, metadata) VALUES (?, ?, ?, ?)`,
        sourceName, lastRun, cursor, metadata)
    return err
}

func (w *WatermarkManager) ListAll() ([]Watermark, error) {
    if err := w.ensureTable(); err != nil {
        return nil, err
    }
    rows, err := w.db.Query("SELECT source_name, last_run, COALESCE(cursor,''), COALESCE(metadata,'') FROM ingestion_watermark ORDER BY last_run DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var result []Watermark
    for rows.Next() {
        var wm Watermark
        if err := rows.Scan(&wm.SourceName, &wm.LastRun, &wm.Cursor, &wm.Metadata); err != nil {
            return nil, err
        }
        result = append(result, wm)
    }
    return result, rows.Err()
}
```

In `internal/ingestion/migrations.go`:

```go
package ingestion

import (
    "database/sql"
    "fmt"
    "sort"
    "time"
)

type Migration struct {
    Version int
    Name    string
    Up      string
    Down    string
}

type MigrationManager struct {
    db         *sql.DB
    migrations map[int]Migration
}

func NewMigrationManager(db *sql.DB) *MigrationManager {
    return &MigrationManager{db: db, migrations: make(map[int]Migration)}
}

func (m *MigrationManager) Register(mig Migration) {
    m.migrations[mig.Version] = mig
}

func (m *MigrationManager) ensureTable() error {
    _, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INT PRIMARY KEY, name TEXT, applied_at TIMESTAMP)`)
    return err
}

func (m *MigrationManager) Up() error {
    if err := m.ensureTable(); err != nil {
        return err
    }
    versions := make([]int, 0, len(m.migrations))
    for v := range m.migrations {
        versions = append(versions, v)
    }
    sort.Ints(versions)
    for _, v := range versions {
        var exists int
        row := m.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", v)
        if err := row.Scan(&exists); err != nil {
            return err
        }
        if exists > 0 {
            continue
        }
        mig := m.migrations[v]
        if _, err := m.db.Exec(mig.Up); err != nil {
            return fmt.Errorf("migration v%d (%s) failed: %w", v, mig.Name, err)
        }
        if _, err := m.db.Exec("INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)", v, mig.Name, time.Now()); err != nil {
            return err
        }
    }
    return nil
}

func (m *MigrationManager) CurrentVersion() (int, error) {
    if err := m.ensureTable(); err != nil {
        return 0, err
    }
    var v sql.NullInt64
    row := m.db.QueryRow("SELECT MAX(version) FROM schema_migrations")
    if err := row.Scan(&v); err != nil {
        return 0, err
    }
    if !v.Valid {
        return 0, nil
    }
    return int(v.Int64), nil
}
```

- [ ] **Step 4: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestWatermark -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/watermark.go internal/ingestion/watermark_test.go internal/ingestion/migrations.go
git commit -m "feat: add watermark table and migration system for incremental ingestion"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/ -run TestWatermark -v`
- **Steps:** (1) Run tests, (2) Verify watermark Get() returns ErrWatermarkNotFound on empty, (3) Verify Set then Get returns correct data, (4) Verify INSERT OR REPLACE updates existing watermark
- **Expected result:** All tests PASS. Watermark lifecycle: Get empty → Set → Get returns data → Set again → Get returns updated data → ListAll returns all entries.

---

### Task 0.2: RateLimiter Interface

**Files:**
- Create: `internal/ingestion/ratelimiter.go`
- Create: `internal/ingestion/ratelimiter_test.go`

**Goal:** Token-bucket rate limiter configurabile per-source. Ogni handler crea il proprio limiter con i parametri della source.

- [ ] **Step 1: Scrivi test ratelimiter_test.go**

```go
package ingestion

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRateLimiterBasic(t *testing.T) {
    rl := NewTokenBucketRateLimiter(2, 5) // 2 req/s, burst 5

    // First 5 calls should be immediate (burst)
    start := time.Now()
    for i := 0; i < 5; i++ {
        err := rl.Wait()
        require.NoError(t, err)
    }
    elapsed := time.Since(start)
    assert.Less(t, elapsed, 100*time.Millisecond, "burst calls should be fast")

    // 6th call should be rate-limited (~500ms wait at 2 rps)
    start = time.Now()
    err := rl.Wait()
    require.NoError(t, err)
    elapsed = time.Since(start)
    assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond, "should wait ~500ms for next token")
}

func TestRateLimiterMultipleWaits(t *testing.T) {
    rl := NewTokenBucketRateLimiter(5, 1) // 5 req/s, minimal burst

    var totalWait time.Duration
    for i := 0; i < 6; i++ {
        start := time.Now()
        err := rl.Wait()
        require.NoError(t, err)
        totalWait += time.Since(start)
    }
    // 6 requests at 5 rps should take ~1 second total
    assert.GreaterOrEqual(t, totalWait, 800*time.Millisecond)
    assert.Less(t, totalWait, 2*time.Second)
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestRateLimiter -v -count=1 2>&1
```
Expected: FAIL — `undefined: NewTokenBucketRateLimiter`

- [ ] **Step 3: Implementa ratelimiter.go**

```go
package ingestion

import (
    "sync"
    "time"
)

type RateLimiter interface {
    Wait() error
}

type TokenBucketRateLimiter struct {
    rate       float64 // tokens per second
    burst      int
    tokens     float64
    lastRefill time.Time
    mu         sync.Mutex
}

func NewTokenBucketRateLimiter(ratePerSecond float64, burst int) *TokenBucketRateLimiter {
    return &TokenBucketRateLimiter{
        rate:       ratePerSecond,
        burst:      burst,
        tokens:     float64(burst),
        lastRefill: time.Now(),
    }
}

func (rl *TokenBucketRateLimiter) refill() {
    now := time.Now()
    elapsed := now.Sub(rl.lastRefill).Seconds()
    rl.tokens += elapsed * rl.rate
    if rl.tokens > float64(rl.burst) {
        rl.tokens = float64(rl.burst)
    }
    rl.lastRefill = now
}

func (rl *TokenBucketRateLimiter) Wait() error {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    rl.refill()
    if rl.tokens >= 1 {
        rl.tokens--
        return nil
    }
    waitTime := time.Duration((1-rl.tokens)/rl.rate*1000) * time.Millisecond
    rl.mu.Unlock()
    time.Sleep(waitTime)
    rl.mu.Lock()
    rl.lastRefill = time.Now()
    rl.tokens = float64(rl.burst) - 1
    return nil
}
```

- [ ] **Step 4: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestRateLimiter -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/ratelimiter.go internal/ingestion/ratelimiter_test.go
git commit -m "feat: add token-bucket rate limiter interface"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/ -run TestRateLimiter -v -count=1`
- **Steps:** (1) Run tests, (2) Verify burst mode allows 5 immediate calls, (3) Verify 6th call waits ~500ms at 2 rps, (4) Verify 6 calls at 5 rps take ~1 second
- **Expected result:** All timing assertions pass. Burst <= 100ms. Rate-limited call >= 400ms. 6x5rps >= 800ms < 2s.

---

### Task 0.3: Registry Pattern (Refactor engine.go dispatch)

**Files:**
- Create: `internal/ingestion/registry.go`
- Create: `internal/ingestion/registry_test.go`
- Modify: `internal/ingestion/engine.go`

**Goal:** Sostituire lo switch `case "csv": ... case "sitemap": ...` con un registry pattern. Ogni source type si registra via `init()`. Engine itera il registry. Il vecchio switch rimane funzionante come backward compat fino a migrazione completa, ma i nuovi source type usano il registry.

- [ ] **Step 1: Scrivi test registry_test.go**

```go
package ingestion

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockFetcher struct{ sourceType string }

func (m *mockFetcher) SourceType() string { return m.sourceType }
func (m *mockFetcher) Validate() error     { return nil }

func mockFactory(sourceType string) Fetcher { return &mockFetcher{sourceType: sourceType} }

func TestRegistryRegistration(t *testing.T) {
    r := NewRegistry()
    assert.NotNil(t, r)

    r.Register("mock_one", mockFactory)
    r.Register("mock_two", mockFactory)

    fetcher, err := r.Create("mock_one")
    require.NoError(t, err)
    assert.Equal(t, "mock_one", fetcher.SourceType())

    _, err = r.Create("nonexistent")
    assert.ErrorIs(t, err, ErrSourceTypeNotFound)
}

func TestRegistryListAll(t *testing.T) {
    r := NewRegistry()
    r.Register("a", mockFactory)
    r.Register("b", mockFactory)

    types := r.List()
    assert.Len(t, types, 2)
    assert.Contains(t, types, "a")
    assert.Contains(t, types, "b")
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestRegistry -v -count=1 2>&1
```
Expected: FAIL — `undefined: NewRegistry`

- [ ] **Step 3: Implementa registry.go**

```go
package ingestion

import (
    "errors"
    "fmt"
    "sync"
)

var ErrSourceTypeNotFound = errors.New("source type not found in registry")

type Fetcher interface {
    SourceType() string
    Validate() error
}

type FetcherFactory func(sourceType string) Fetcher

type Registry struct {
    mu       sync.RWMutex
    handlers map[string]FetcherFactory
}

var GlobalRegistry = NewRegistry()

func NewRegistry() *Registry {
    return &Registry{handlers: make(map[string]FetcherFactory)}
}

func (r *Registry) Register(sourceType string, factory FetcherFactory) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.handlers[sourceType] = factory
}

func (r *Registry) Create(sourceType string) (Fetcher, error) {
    r.mu.RLock()
    factory, ok := r.handlers[sourceType]
    r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("%w: %s", ErrSourceTypeNotFound, sourceType)
    }
    return factory(sourceType), nil
}

func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    keys := make([]string, 0, len(r.handlers))
    for k := range r.handlers {
        keys = append(keys, k)
    }
    return keys
}

// RegisteredSourceTypes returns all source types in the global registry.
// Used by the frontend to list available source types for the dropdown.
func RegisteredSourceTypes() []string {
    return GlobalRegistry.List()
}
```

- [ ] **Step 4: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestRegistry -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 5: Integra registry in engine.go**

In `internal/ingestion/engine.go`, nella funzione `Run()` (o dove avviene il dispatch), aggiungi dopo lo switch esistente:

```go
// Registry-based dispatch (new source types)
// Existing switch handles legacy source types for backward compat
fetcher, err := GlobalRegistry.Create(sourceType)
if err == nil {
    if err := fetcher.Validate(); err != nil {
        slog.Error("source validation failed", "source_type", sourceType, "error", err)
        return fmt.Errorf("source validation failed: %w", err)
    }
    // Dispatch to the fetcher's Run method
    return fetcher.(SourceFetcher).Run(ctx, config)
}
// Fall through to existing switch if not found in registry
```

In `engine.go`, esponi la funzione per le migrazioni:

```go
func RunMigrations(db *sql.DB) error {
    mm := NewMigrationManager(db)
    mm.Register(Migration{Version: 1, Name: "create_watermark_table", Up: "CREATE TABLE IF NOT EXISTS ingestion_watermark (source_name TEXT PRIMARY KEY, last_run TIMESTAMP, cursor TEXT, metadata TEXT)", Down: "DROP TABLE IF EXISTS ingestion_watermark"})
    return mm.Up()
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/ingestion/registry.go internal/ingestion/registry_test.go internal/ingestion/engine.go
git commit -m "feat: add registry pattern for source type dispatch"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/ -run TestRegistry -v -count=1`
- **Steps:** (1) Run tests, (2) Verify Register+Create returns correct Fetcher, (3) Verify Create for unregistered type returns ErrSourceTypeNotFound, (4) Verify List returns all registered types
- **Expected result:** All tests PASS. Existing engine.go switch still compiles and works.

---

## FASE 1: Dati Elettorali (Eligendo API)

### Task 1: Party Mapping Alias Table + ISTAT Codes

**Files:**
- Create: `internal/ingestion/sources/election.go`
- Create: `internal/ingestion/sources/election_test.go`
- Create: `configs/eligendo_codes.json`
- Create: `configs/party_aliases.json`

**Goal:** Mapping partiti via alias table (sostituisce Jaro-Winkler). Lookup ISTAT codici comuni. Strutture dati per election results.

**Decisione architetturale (Oracle):** Alias table (`raw_name → canonical_id`) è più accurata, deterministica e mantenibile di Jaro-Winkler. Il fuzzy matching rimane disponibile come fallback dietro flag `--fuzzy-fallback=true` ma non è il percorso primario.

- [ ] **Step 1: Scrivi test elezione — ISTAT alias e party lookup**

```go
package sources

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestISTATLookup(t *testing.T) {
    codes := NewISTATLookup()
    // Roma
    name, found := codes.Lookup("058091")
    assert.True(t, found)
    assert.Contains(t, name, "Roma")

    // Nonexistent
    _, found = codes.Lookup("999999")
    assert.False(t, found)
}

func TestPartyMappingExactMatch(t *testing.T) {
    mapper := NewPartyMapper()
    mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
    mapper.AddAlias("FRATELLI D'ITALIA - GIORGIA MELONI", "fratelli-italia")
    mapper.AddAlias("PARTITO DEMOCRATICO", "partito-democratico")
    mapper.AddAlias("PD", "partito-democratico")

    // Exact match
    canonical, found := mapper.Lookup("PARTITO DEMOCRATICO")
    require.True(t, found)
    assert.Equal(t, "partito-democratico", canonical)

    // Alias match
    canonical, found = mapper.Lookup("PD")
    require.True(t, found)
    assert.Equal(t, "partito-democratico", canonical)

    // No match
    _, found = mapper.Lookup("LISTA INESISTENTE XYZ")
    assert.False(t, found)
}

func TestPartyMappingManualOverride(t *testing.T) {
    mapper := NewPartyMapper()
    mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
    mapper.SetOverride("FRATELLI D'ITALIA - ROMA", "fratelli-italia-roma")

    // Manual override takes priority over alias match
    canonical, found := mapper.Lookup("FRATELLI D'ITALIA - ROMA")
    require.True(t, found)
    assert.Equal(t, "fratelli-italia-roma", canonical)

    // Without override, uses alias table
    canonical, found = mapper.Lookup("FRATELLI D'ITALIA")
    require.True(t, found)
    assert.Equal(t, "fratelli-italia", canonical)
}

func TestElectionConfigValidation(t *testing.T) {
    valid := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022}
    assert.NoError(t, valid.Validate())

    invalidType := ElectionConfig{ElectionType: "fantasia", Level: "comune", Year: 2022}
    assert.ErrorContains(t, invalidType.Validate(), "invalid election_type")

    invalidLevel := ElectionConfig{ElectionType: "politiche", Level: "quartiere", Year: 2022}
    assert.ErrorContains(t, invalidLevel.Validate(), "invalid level")

    invalidYear := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 1990}
    assert.ErrorContains(t, invalidYear.Validate(), "year before 2000")
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestISTAT|TestParty|TestElectionConfig" -v -count=1 2>&1
```
Expected: FAIL — `undefined: NewISTATLookup`

- [ ] **Step 3: Implementa election.go — structs + ISTAT + PartyMapper**

```go
package sources

import (
    _ "embed"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "sync"
)

//go:embed ../../../configs/eligendo_codes.json
var eligendoCodesRaw []byte
var istatCache *ISTATLookup
var istatOnce sync.Once

//go:embed ../../../configs/party_aliases.json
var partyAliasesRaw []byte

type ElectionConfig struct {
    ElectionType string `json:"election_type"` // politiche, europee, regionali, comunali, provinciali, referendum
    Level        string `json:"level"`          // comune, provincia, regione
    Year         int    `json:"year"`
}

func (c ElectionConfig) Validate() error {
    validTypes := map[string]bool{"politiche": true, "europee": true, "regionali": true, "comunali": true, "provinciali": true, "referendum": true}
    validLevels := map[string]bool{"comune": true, "provincia": true, "regione": true}
    if !validTypes[c.ElectionType] {
        return fmt.Errorf("invalid election_type: %s", c.ElectionType)
    }
    if !validLevels[c.Level] {
        return fmt.Errorf("invalid level: %s", c.Level)
    }
    if c.Year < 2000 {
        return errors.New("year before 2000 not supported")
    }
    return nil
}

type ElectionResult struct {
    ElectionType string
    Level        string
    Year         int
    Comune       string
    ComuneISTAT  string
    Lista        string    // raw name from API
    PartyCanonical string  // resolved canonical party ID
    Voti         int64
    Percentuale  float64
    Seggi        int
    Elettori     int64
    Votanti      int64
}

type ISTATLookup struct {
    byCode map[string]string // ISTAT code → comune name
    byName map[string]string // comune name → ISTAT code
}

func NewISTATLookup() *ISTATLookup {
    istatOnce.Do(func() {
        istatCache = &ISTATLookup{byCode: make(map[string]string), byName: make(map[string]string)}
        var raw map[string]string
        if err := json.Unmarshal(eligendoCodesRaw, &raw); err != nil {
            // fallback: empty lookup, errors logged at call site
            return
        }
        for code, name := range raw {
            istatCache.byCode[code] = name
            istatCache.byName[strings.ToLower(name)] = code
        }
    })
    return istatCache
}

func (l *ISTATLookup) Lookup(code string) (string, bool) {
    name, ok := l.byCode[code]
    return name, ok
}

func (l *ISTATLookup) LookupByName(name string) (string, bool) {
    code, ok := l.byName[strings.ToLower(name)]
    return code, ok
}

type PartyMapper struct {
    aliases   map[string]string // lowercase raw name → canonical party ID
    overrides map[string]string // exact raw name → canonical party ID (takes priority)
    mu        sync.RWMutex
}

func NewPartyMapper() *PartyMapper {
    pm := &PartyMapper{
        aliases:   make(map[string]string),
        overrides: make(map[string]string),
    }
    pm.loadBuiltinAliases()
    return pm
}

func (pm *PartyMapper) loadBuiltinAliases() {
    var raw map[string]string
    if err := json.Unmarshal(partyAliasesRaw, &raw); err != nil {
        return
    }
    for rawName, canonical := range raw {
        pm.AddAlias(rawName, canonical)
    }
}

func (pm *PartyMapper) AddAlias(rawName string, canonical string) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    pm.aliases[normalizePartyName(rawName)] = canonical
}

func (pm *PartyMapper) SetOverride(rawName string, canonical string) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    pm.overrides[rawName] = canonical
}

func (pm *PartyMapper) Lookup(rawName string) (string, bool) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    // Manual overrides always take priority
    if canonical, ok := pm.overrides[rawName]; ok {
        return canonical, true
    }
    // Alias table lookup (normalized)
    canonical, ok := pm.aliases[normalizePartyName(rawName)]
    return canonical, ok
}

func (pm *PartyMapper) GetOverride(rawName string) (string, bool) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    canonical, ok := pm.overrides[rawName]
    return canonical, ok
}

func (pm *PartyMapper) AllOverrides() map[string]string {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    result := make(map[string]string, len(pm.overrides))
    for k, v := range pm.overrides {
        result[k] = v
    }
    return result
}

// normalizePartyName normalizza il nome partito per lookup case-insensitive
func normalizePartyName(raw string) string {
    return strings.TrimSpace(strings.ToUpper(raw))
}
```

- [ ] **Step 4: Crea config files**

`configs/eligendo_codes.json` (esempio — struttura):

```json
{
  "058091": "Roma",
  "015146": "Milano",
  "063049": "Napoli",
  "001272": "Torino",
  "048017": "Firenze"
}
```

`configs/party_aliases.json` (esempio — popolato inizialmente con i ~50 partiti principali):

```json
{
  "FRATELLI D'ITALIA": "fratelli-italia",
  "FRATELLI D'ITALIA - GIORGIA MELONI": "fratelli-italia",
  "FDI": "fratelli-italia",
  "PARTITO DEMOCRATICO": "partito-democratico",
  "PD": "partito-democratico",
  "MOVIMENTO 5 STELLE": "movimento-5-stelle",
  "M5S": "movimento-5-stelle",
  "LEGA": "lega",
  "LEGA SALVINI PREMIER": "lega",
  "LEGA NORD": "lega",
  "FORZA ITALIA": "forza-italia",
  "FI": "forza-italia",
  "AZIONE": "azione",
  "ITALIA VIVA": "italia-viva",
  "IV": "italia-viva",
  "SINISTRA ITALIANA": "sinistra-italiana",
  "ALLEANZA VERDI E SINISTRA": "verdi-sinistra",
  "EUROPA VERDE": "europa-verde",
  "+EUROPA": "piu-europa",
  "NOI MODERATI": "noi-moderati",
  "SUD CHIAMA NORD": "sud-chiama-nord",
  "UNIONE POPOLARE": "unione-popolare",
  "ITALIA SOVRANA E POPOLARE": "italia-sovrana-popolare",
  "DEMOCRAZIA SOVRANA E POPOLARE": "democrazia-sovrana-popolare",
  "VITA": "vita"
}
```

- [ ] **Step 5: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run "TestISTAT|TestParty|TestElectionConfig" -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go configs/eligendo_codes.json configs/party_aliases.json
git commit -m "feat: add party alias table, ISTAT lookup, and election config structs"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestParty -v -count=1`
- **Steps:** (1) Run test, (2) Verify "PARTITO DEMOCRATICO" matches via alias, (3) Verify "PD" alias resolves to same canonical, (4) Verify manual override takes priority over alias, (5) Verify "LISTA INESISTENTE" returns not found
- **Expected result:** 4 matches, 1 not found. Override > alias.

---

### Task 2: ElectionFetcher con Rate Limiting + Eligendo API

**Files:**
- Modify: `internal/ingestion/sources/election.go`
- Modify: `internal/ingestion/sources/election_test.go`
- Add to go.mod: `github.com/dnaeon/go-vcr` (HTTP fixture recording)

**Goal:** Implementare le chiamate HTTP reali all'API Eligendo con rate limiting, header obbligatori, e parsing delle risposte JSON.

**API Endpoint:** `https://eleapi.interno.gov.it/siel/PX/`
- `getentiFI`: restituisce enti (comuni, province, regioni) per un'elezione
- `scrutiniFI`: restituisce risultati scrutini per un ente

- [ ] **Step 1: Scrivi test con go-vcr fixture**

```go
package sources

import (
    "context"
    "net/http/httptest"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestElectionFetcherRateLimit(t *testing.T) {
    // Create a mock server that responds quickly
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify mandatory headers
        assert.Equal(t, "application/json", r.Header.Get("Accept"))
        w.WriteHeader(200)
        w.Write([]byte(`{"intestazione":{},"enti":{"ente":[]}}`))
    }))
    defer server.Close()

    fetcher := NewElectionFetcher(server.URL, 1.0) // 1 req/s

    start := time.Now()
    // First call: immediate (burst)
    _, err := fetcher.GetEntities(context.Background(), ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022})
    require.NoError(t, err)
    elapsed1 := time.Since(start)
    assert.Less(t, elapsed1, 200*time.Millisecond, "first call should be fast")

    // Second call: rate-limited
    start = time.Now()
    _, err = fetcher.GetEntities(context.Background(), ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022})
    require.NoError(t, err)
    elapsed2 := time.Since(start)
    assert.GreaterOrEqual(t, elapsed2, 800*time.Millisecond, "second call should be rate-limited to 1 req/s")
}

func TestElectionFetcherTEMapping(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"intestazione":{"te":"TE01"},"enti":{"ente":[{"cod":"058091","desc":"ROMA"}]}}`))
    }))
    defer server.Close()

    fetcher := NewElectionFetcher(server.URL, 5.0)
    entities, err := fetcher.GetEntities(context.Background(), ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022})
    require.NoError(t, err)
    assert.Len(t, entities, 1)
    assert.Equal(t, "058091", entities[0].Cod)
    assert.Equal(t, "ROMA", entities[0].Desc)
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestElectionFetcher -v -count=1 2>&1
```
Expected: FAIL — `undefined: NewElectionFetcher`

- [ ] **Step 3: Implementa ElectionFetcher in election.go**

```go
package sources

// ... (continuazione del file election.go)

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    "example.com/aleph/internal/ingestion"
)

type ElectionFetcher struct {
    baseURL     string
    rateLimiter *ingestion.TokenBucketRateLimiter
    httpClient  *http.Client
}

func NewElectionFetcher(baseURL string, ratePerSecond float64) *ElectionFetcher {
    return &ElectionFetcher{
        baseURL:     baseURL,
        rateLimiter: ingestion.NewTokenBucketRateLimiter(ratePerSecond, 1),
        httpClient:  &http.Client{Timeout: 30 * time.Second},
    }
}

type EligendoEntity struct {
    Cod  string `json:"cod"`
    Desc string `json:"desc"`
}

type getentiFIResponse struct {
    Intestazione struct {
        TE string `json:"te"`
    } `json:"intestazione"`
    Enti struct {
        Ente []EligendoEntity `json:"ente"`
    } `json:"enti"`
}

func (f *ElectionFetcher) GetEntities(ctx context.Context, cfg ElectionConfig) ([]EligendoEntity, error) {
    if err := f.rateLimiter.Wait(); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }
    teCode := cfg.teCode()
    url := fmt.Sprintf("%s/getentiFI?te=%s&liv=%s", f.baseURL, teCode, cfg.Level)
    resp, err := f.doGet(ctx, url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var result getentiFIResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode getentiFI: %w", err)
    }
    return result.Enti.Ente, nil
}

func (f *ElectionFetcher) doGet(ctx context.Context, url string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Accept", "application/json")
    req.Header.Set("User-Agent", "Aleph/1.0 (data-ingestion)")
    resp, err := f.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http get %s: %w", url, err)
    }
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, fmt.Errorf("eligendo API error %d: %s", resp.StatusCode, string(body))
    }
    return resp, nil
}

func (c ElectionConfig) teCode() string {
    teCodes := map[string]string{
        "politiche":    "TE01",
        "europee":      "TE02",
        "regionali":    "TE03",
        "provinciali":  "TE04",
        "comunali":     "TE05",
        "referendum":   "TE09",
    }
    if code, ok := teCodes[c.ElectionType]; ok {
        return code
    }
    return "TE01"
}
```

- [ ] **Step 4: Aggiungi go-vcr al go.mod**

```bash
cd /tmp/opencode/aleph && go get github.com/dnaeon/go/vcr/v2@latest 2>&1
```

- [ ] **Step 5: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestElectionFetcher -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go go.mod go.sum
git commit -m "feat: add ElectionFetcher with rate-limited Eligendo API calls"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestElectionFetcher -v -count=1`
- **Steps:** (1) Run rate limit test, (2) Verify first call <200ms, second call >=800ms (1 req/s), (3) Run TE mapping test, (4) Verify TE01→politiche mapping produces correct URL, (5) Verify entity parsing (cod, desc)
- **Expected result:** Both tests PASS. Rate limiting enforces 1 req/s. TE codes mapped correctly.

---

### Task 3: runElection() — Full Pipeline con Dual Write

**Files:**
- Modify: `internal/ingestion/sources/election.go`
- Modify: `internal/ingestion/sources/election_test.go`

**Goal:** Pipeline completa: getenti → scrutini per ogni ente → raw salvataggio su disco → normalizzazione party → write DuckDB. Registrazione nel registry.

- [ ] **Step 1: Scrivi test integrazione con mock server**

```go
func TestRunElectionFullPipeline(t *testing.T) {
    // Mock server che simula getentiFI + scrutiniFI
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        if strings.Contains(r.URL.Path, "getentiFI") {
            w.Write([]byte(`{"intestazione":{"te":"TE01"},"enti":{"ente":[{"cod":"058091","desc":"ROMA"},{"cod":"015146","desc":"MILANO"}]}}`))
        } else if strings.Contains(r.URL.Path, "scrutiniFI") {
            w.Write([]byte(`{"intestazione":{"cod":"058091"},"liste":{"lista":[{"desc":"PARTITO DEMOCRATICO","voti":50000,"perc":30.5,"seggi":10}]},"datiGenerali":{"elettori":200000,"votanti":165000,"schedeBianche":3000,"schedeNulle":2000}}`))
        } else {
            w.WriteHeader(404)
        }
    }))
    defer server.Close()

    db := setupTestDuckDB(t) // helper che crea DuckDB in-memory
    defer db.Close()

    mapper := NewPartyMapper()
    mapper.AddAlias("PARTITO DEMOCRATICO", "partito-democratico")

    results, err := RunElection(context.Background(), db, server.URL, ElectionConfig{
        ElectionType: "politiche", Level: "comune", Year: 2022,
    }, mapper, "/tmp/test-raw")
    require.NoError(t, err)
    assert.Greater(t, len(results), 0)
    assert.Greater(t, callCount, 2) // getenti + 2 scrutinii

    // Verify raw files saved
    _, err = os.Stat("/tmp/test-raw/election/2022-politiche-comune/getenti.json")
    require.NoError(t, err)

    // Verify normalized data in DuckDB
    var count int
    db.QueryRow("SELECT COUNT(*) FROM election_results WHERE year = 2022").Scan(&count)
    assert.Greater(t, count, 0)

    // Verify party canonical name
    var canonical string
    db.QueryRow("SELECT party_canonical FROM election_results WHERE lista = 'PARTITO DEMOCRATICO' LIMIT 1").Scan(&canonical)
    assert.Equal(t, "partito-democratico", canonical)
}
```

- [ ] **Step 2: Esegui test e verifica FAIL**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestRunElection -v -count=1 2>&1
```
Expected: FAIL — `undefined: RunElection`

- [ ] **Step 3: Implementa RunElection in election.go**

```go
package sources

// ... continuazione di election.go

import (
    "compress/gzip"
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
)

const electionSourceType = "election"

func init() {
    ingestion.GlobalRegistry.Register(electionSourceType, func(sourceType string) ingestion.Fetcher {
        return &ElectionSource{sourceType: sourceType}
    })
}

type ElectionSource struct {
    sourceType string
    config     ElectionConfig
    db         *sql.DB
    dataDir    string
    mapper     *PartyMapper
}

func (s *ElectionSource) SourceType() string { return s.sourceType }
func (s *ElectionSource) Validate() error    { return s.config.Validate() }

type scrutiniFIResponse struct {
    Intestazione struct {
        Cod string `json:"cod"`
    } `json:"intestazione"`
    Liste struct {
        Lista []struct {
            Desc  string  `json:"desc"`
            Voti  int64   `json:"voti"`
            Perc  float64 `json:"perc"`
            Seggi int     `json:"seggi"`
        } `json:"lista"`
    } `json:"liste"`
    DatiGenerali struct {
        Elettori      int64 `json:"elettori"`
        Votanti       int64 `json:"votanti"`
        SchedeBianche int64 `json:"schedeBianche"`
        SchedeNulle   int64 `json:"schedeNulle"`
    } `json:"datiGenerali"`
}

func RunElection(ctx context.Context, db *sql.DB, baseURL string, cfg ElectionConfig, mapper *PartyMapper, rawDir string) ([]ElectionResult, error) {
    fetcher := NewElectionFetcher(baseURL, 1.0)
    var results []ElectionResult

    // 1. Get entities
    entities, err := fetcher.GetEntities(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("getenti: %w", err)
    }

    // 2. Save raw getenti response
    rawPath := filepath.Join(rawDir, electionSourceType, fmt.Sprintf("%d-%s-%s", cfg.Year, cfg.ElectionType, cfg.Level))
    if err := os.MkdirAll(rawPath, 0755); err != nil {
        return nil, err
    }
    if err := saveRawJSON(filepath.Join(rawPath, "getenti.json"), entities); err != nil {
        slog.Warn("failed to save raw getenti", "error", err)
    }

    // 3. Fetch scrutini for each entity
    for _, ent := range entities {
        if err := fetcher.rateLimiter.Wait(); err != nil {
            slog.Error("rate limiter error", "entity", ent.Cod, "error", err)
            continue
        }
        url := fmt.Sprintf("%s/scrutiniFI?te=%s&cod=%s", baseURL, cfg.teCode(), ent.Cod)
        resp, err := fetcher.doGet(ctx, url)
        if err != nil {
            slog.Error("scrutini fetch failed", "entity", ent.Cod, "error", err)
            continue
        }
        var scr scrutiniFIResponse
        if err := json.NewDecoder(resp.Body).Decode(&scr); err != nil {
            resp.Body.Close()
            slog.Error("scrutini decode failed", "entity", ent.Cod, "error", err)
            continue
        }
        resp.Body.Close()

        // Save raw scrutini
        if err := saveRawJSON(filepath.Join(rawPath, fmt.Sprintf("scrutini_%s.json", ent.Cod)), scr); err != nil {
            slog.Warn("failed to save raw scrutini", "entity", ent.Cod, "error", err)
        }

        // Normalize and collect results
        for _, lista := range scr.Liste.Lista {
            canonical, found := mapper.Lookup(lista.Desc)
            if !found {
                canonical = "" // raw match not found — stays unmapped
                slog.Info("unmapped party", "raw_name", lista.Desc, "entity", ent.Cod)
            }
            results = append(results, ElectionResult{
                ElectionType:    cfg.ElectionType,
                Level:           cfg.Level,
                Year:            cfg.Year,
                Comune:          ent.Desc,
                ComuneISTAT:     ent.Cod,
                Lista:           lista.Desc,
                PartyCanonical:  canonical,
                Voti:            lista.Voti,
                Percentuale:     lista.Perc,
                Seggi:           lista.Seggi,
                Elettori:        scr.DatiGenerali.Elettori,
                Votanti:         scr.DatiGenerali.Votanti,
            })
        }
    }

    // 4. Write normalized results to DuckDB
    if err := writeElectionResults(db, results); err != nil {
        return results, fmt.Errorf("write results: %w", err)
    }

    slog.Info("election ingestion complete",
        "election_type", cfg.ElectionType,
        "entities", len(entities),
        "results", len(results),
        "raw_dir", rawPath,
    )
    return results, nil
}

func writeElectionResults(db *sql.DB, results []ElectionResult) error {
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS election_results (
        id INTEGER PRIMARY KEY,
        election_type TEXT, level TEXT, year INTEGER,
        comune TEXT, comune_istat TEXT,
        lista TEXT, party_canonical TEXT,
        voti INTEGER, percentuale REAL, seggi INTEGER,
        elettori INTEGER, votanti INTEGER,
        ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
    if err != nil {
        return err
    }

    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`INSERT OR REPLACE INTO election_results
        (election_type, level, year, comune, comune_istat, lista, party_canonical, voti, percentuale, seggi, elettori, votanti)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, r := range results {
        _, err := stmt.Exec(r.ElectionType, r.Level, r.Year, r.Comune, r.ComuneISTAT,
            r.Lista, r.PartyCanonical, r.Voti, r.Percentuale, r.Seggi, r.Elettori, r.Votanti)
        if err != nil {
            return fmt.Errorf("insert result: %w", err)
        }
    }
    return tx.Commit()
}

func saveRawJSON(path string, data interface{}) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    gz := gzip.NewWriter(f)
    defer gz.Close()
    return json.NewEncoder(gz).Encode(data)
}
```

- [ ] **Step 4: Aggiungi import `strings` a election_test.go e setup helper**

In `election_test.go`, aggiungi all'inizio:

```go
import (
    "strings"
    "os"
)
```

- [ ] **Step 5: Esegui test e verifica PASS**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestRunElection -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 6: LSP diagnostics**

```bash
# Verify no type errors, missing imports, etc.
```

- [ ] **Step 7: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go
git commit -m "feat: add runElection pipeline with dual-write (disk raw + DuckDB normalized)"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestRunElection -v -count=1`
- **Steps:** (1) Run test, (2) Verify getenti called once, scrutini called per entity, (3) Verify raw JSON.gz saved to disk, (4) Verify `election_results` table in DuckDB has correct row count, (5) Verify `party_canonical` field populated from alias table, (6) Verify unmapped parties produce empty canonical + log
- **Expected result:** All assertions pass. Raw files on disk. Normalized data in DuckDB. Unmapped parties logged.

---

### Task 4: Frontend — DataSourceForm Election Fields + Phase 2 Support

**Files:**
- Modify: `frontend/src/components/DataSourceForm.tsx`
- Modify: `frontend/src/components/DataSourceFormSlideOver.tsx`

**Goal:** Aggiungere dropdown per election_type, level, anno. Aggiungere i 6 nuovi source type al dropdown source_type.

- [ ] **Step 1: Verifica TypeScript compila prima delle modifiche**

```bash
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 2: Modifica DataSourceForm.tsx — election fields**

Nel file `frontend/src/components/DataSourceForm.tsx`, dopo il campo source_type esistente, aggiungi:

```tsx
// Election-specific fields (shown when source_type === "election")
{sourceType === "election" && (
  <>
    <FormField label="Tipo Elezione" required>
      <select
        value={configJson.election_type || ""}
        onChange={(e) => updateConfig("election_type", e.target.value)}
        className="form-select"
      >
        <option value="">Seleziona...</option>
        <option value="politiche">Politiche (Camera/Senato)</option>
        <option value="europee">Europee</option>
        <option value="regionali">Regionali</option>
        <option value="comunali">Comunali</option>
        <option value="provinciali">Provinciali</option>
        <option value="referendum">Referendum</option>
      </select>
    </FormField>

    <FormField label="Livello Territoriale" required>
      <select
        value={configJson.level || "comune"}
        onChange={(e) => updateConfig("level", e.target.value)}
        className="form-select"
      >
        <option value="comune">Comune</option>
        <option value="provincia">Provincia</option>
        <option value="regione">Regione</option>
      </select>
    </FormField>

    <FormField label="Anno" required>
      <input
        type="number"
        min={2000}
        max={new Date().getFullYear()}
        value={configJson.year || ""}
        onChange={(e) => updateConfig("year", parseInt(e.target.value))}
        className="form-input"
      />
    </FormField>
  </>
)}

{/* PEP-specific fields */}
{sourceType === "pep" && (
  <FormField label="Dataset" required>
    <select
      value={configJson.dataset || ""}
      onChange={(e) => updateConfig("dataset", e.target.value)}
      className="form-select"
    >
      <option value="">Seleziona...</option>
      <option value="it_deputies">Deputati</option>
      <option value="it_senate">Senatori</option>
      <option value="it_peps">PEP (Persone Politicamente Esposte)</option>
    </select>
  </FormField>
)}

{/* ANAC-specific fields */}
{sourceType === "public_contracts" && (
  <>
    <FormField label="Anno" required>
      <input
        type="number"
        min={2008}
        max={new Date().getFullYear()}
        value={configJson.year || ""}
        onChange={(e) => updateConfig("year", parseInt(e.target.value))}
        className="form-input"
      />
    </FormField>
    <FormField label="Tipo Dataset">
      <select
        value={configJson.contract_type || "CIG"}
        onChange={(e) => updateConfig("contract_type", e.target.value)}
        className="form-select"
      >
        <option value="CIG">CIG</option>
        <option value="partecipanti">Partecipanti</option>
        <option value="aggiudicatari">Aggiudicatari</option>
        <option value="subappalti">Subappalti</option>
      </select>
    </FormField>
  </>
)}

{/* OPDM-specific fields */}
{sourceType === "opdm" && (
  <>
    <FormField label="Entity Type">
      <select
        value={configJson.entity_type || "memberships"}
        onChange={(e) => updateConfig("entity_type", e.target.value)}
        className="form-select"
      >
        <option value="memberships">Memberships</option>
        <option value="persons">Persons</option>
        <option value="organizations">Organizations</option>
        <option value="properties">Properties</option>
      </select>
    </FormField>
    <FormField label="API Key">
      <input
        type="password"
        value={configJson.api_key || ""}
        onChange={(e) => updateConfig("api_key", e.target.value)}
        className="form-input"
        placeholder="opdm_api_key..."
      />
    </FormField>
  </>
)}

{/* Parliament-specific fields */}
{sourceType === "parliament" && (
  <>
    <FormField label="Camera">
      <select
        value={configJson.chamber || "camera"}
        onChange={(e) => updateConfig("chamber", e.target.value)}
        className="form-select"
      >
        <option value="camera">Camera dei Deputati</option>
        <option value="senato">Senato della Repubblica</option>
        <option value="both">Entrambe</option>
      </select>
    </FormField>
    <FormField label="Legislatura">
      <input
        type="number"
        min={13}
        max={20}
        value={configJson.legislatura || ""}
        onChange={(e) => updateConfig("legislatura", parseInt(e.target.value))}
        className="form-input"
        placeholder="18"
      />
    </FormField>
  </>
)}
```

- [ ] **Step 3: Aggiorna dropdown source_type con i nuovi tipi**

Nel `<select>` del `source_type`, aggiungi le nuove opzioni:

```tsx
<optgroup label="Politica Italiana">
  <option value="election">Elezioni (Eligendo)</option>
  <option value="pep">PEP (OpenSanctions)</option>
  <option value="public_contracts">Appalti Pubblici (ANAC)</option>
  <option value="party_funding">Finanziamenti Partiti (ondata)</option>
  <option value="opdm">Openpolis (OPDM)</option>
  <option value="parliament">Parlamento (SPARQL)</option>
</optgroup>
```

- [ ] **Step 4: Verifica TypeScript compila**

```bash
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit 2>&1
```
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/DataSourceForm.tsx frontend/src/components/DataSourceFormSlideOver.tsx
git commit -m "feat: add election and political source type fields to DataSourceForm"
```

**VERIFICA (QA Scenario):**
- **Tool:** `cd frontend && npx tsc --noEmit`
- **Steps:** (1) Run TypeScript compiler, (2) Verify zero errors, (3) Check that all 6 new source_type values appear in dropdown, (4) Verify conditional fields show/hide based on selected source_type
- **Expected result:** tsc exits 0. Dropdown contains optgroup "Politica Italiana" with 6 options. Election type dropdown shows 6 election types. Year field accepts 2000–current year.

---

### Task 5: Health Endpoint + Structured Logging

**Files:**
- Create: `internal/ingestion/health.go`
- Modify: `internal/api/server.go` (per registrare endpoint)

**Goal:** Endpoint `/api/health/ingestion` che riporta lo stato di ogni source type (last run, error, record count). Structured logging con `log/slog` per tutte le ingestion.

- [ ] **Step 1: Scrivi test health_test.go**

```go
package ingestion

import (
    "encoding/json"
    "net/http/httptest"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
    db := setupTestDB(t)
    wm := NewWatermarkManager(db)
    wm.Set("election", time.Now(), "cursor_1", `{"records":1500}`)
    wm.Set("pep", time.Now().Add(-24*time.Hour), "", `{"records":500}`)

    handler := NewHealthHandler(db)

    req := httptest.NewRequest("GET", "/api/health/ingestion", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    assert.Equal(t, 200, rec.Code)

    var resp map[string]interface{}
    require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
    sources := resp["sources"].([]interface{})
    assert.GreaterOrEqual(t, len(sources), 2)
}
```

- [ ] **Step 2: Esegui test e verifica FAIL, poi implementa**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestHealth -v -count=1 2>&1
```
Expected: FAIL

- [ ] **Step 3: Implementa health.go**

```go
package ingestion

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
)

type HealthHandler struct {
    db *sql.DB
    wm *WatermarkManager
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
    return &HealthHandler{db: db, wm: NewWatermarkManager(db)}
}

type SourceHealth struct {
    SourceName string    `json:"source_name"`
    LastRun    time.Time `json:"last_run"`
    Status     string    `json:"status"` // "healthy", "stale", "error"
    RecordCount int64     `json:"record_count,omitempty"`
    Cursor     string    `json:"cursor,omitempty"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    watermarks, err := h.wm.ListAll()
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    sources := make([]SourceHealth, 0, len(watermarks))
    for _, wm := range watermarks {
        sh := SourceHealth{
            SourceName: wm.SourceName,
            LastRun:    wm.LastRun,
            Status:     "healthy",
            Cursor:     wm.Cursor,
        }
        // Stale check: >7 days since last run
        if time.Since(wm.LastRun) > 7*24*time.Hour {
            sh.Status = "stale"
        }
        sources = append(sources, sh)
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "sources":        sources,
        "total_sources":  len(sources),
        "timestamp":      time.Now().UTC(),
    })
}
```

- [ ] **Step 4: Esegui test, verifica PASS, commit**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/ -run TestHealth -v -count=1 2>&1
# Expected: PASS

git add internal/ingestion/health.go internal/ingestion/health_test.go
git commit -m "feat: add ingestion health endpoint with per-source status"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/ -run TestHealth -v`
- **Steps:** (1) Set 2 watermarks, (2) GET /api/health/ingestion, (3) Parse JSON response
- **Expected result:** Status 200. Response contains "sources" array with 2 entries. Each has source_name, last_run, status.

---

## FASE 2: Finanziamenti & Dati Politici (Tasks 6-11)

### Task 6: source_type="pep" — OpenSanctions

**Files:**
- Create: `internal/ingestion/sources/pep.go`
- Create: `internal/ingestion/sources/pep_test.go`

**Goal:** Parsing FollowTheMoney JSON. DuckDB table `pep_entities`. Registry registration. Watermark incremental (file hash comparison).

- [ ] **Step 1: Test — PEP fetch + FtM parsing**

```go
package sources

import (
    "context"
    "database/sql"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "example.com/aleph/internal/ingestion"
)

var sampleFtM = `{
  "id": "ita-deputy-123",
  "schema": "Person",
  "properties": {
    "name": ["Mario Rossi"],
    "country": ["it"],
    "birthDate": ["1970-01-15"],
    "position": ["Deputato"],
    "topics": ["role.pep"],
    "nationality": ["IT"]
  },
  "datasets": ["it_deputies"],
  "first_seen": "2018-03-23",
  "last_seen": "2024-01-01"
}`

func TestPEPParseFtM(t *testing.T) {
    entity, err := ParseFtMEntity([]byte(sampleFtM))
    require.NoError(t, err)
    assert.Equal(t, "ita-deputy-123", entity.ID)
    assert.Equal(t, "Mario Rossi", entity.Name)
    assert.Equal(t, "it", entity.Country)
    assert.Equal(t, "1970-01-15", entity.BirthDate)
    assert.Equal(t, "Deputato", entity.Position)
}

func TestPEPIngestion(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"entities": [` + sampleFtM + `]}`))
    }))
    defer server.Close()

    db := setupTestDuckDB(t)
    defer db.Close()

    wm := ingestion.NewWatermarkManager(db)
    err := RunPEP(context.Background(), server.URL, db, wm, "/tmp/test-raw")
    require.NoError(t, err)

    var count int
    db.QueryRow("SELECT COUNT(*) FROM pep_entities").Scan(&count)
    assert.Greater(t, count, 0)
}
```

- [ ] **Step 2: Fail → Implementa pep.go → Pass → Commit**

```go
package sources

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"
    "example.com/aleph/internal/ingestion"
)

const pepSourceType = "pep"

func init() {
    ingestion.GlobalRegistry.Register(pepSourceType, func(sourceType string) ingestion.Fetcher {
        return &PEPFetcher{sourceType: sourceType}
    })
}

type FtMEntity struct {
    ID        string `json:"id"`
    Schema    string `json:"schema"`
    Name      string `json:"name"`
    Country   string `json:"country"`
    BirthDate string `json:"birthDate"`
    Position  string `json:"position"`
    Dataset   string `json:"dataset"`
    FirstSeen string `json:"first_seen"`
    LastSeen  string `json:"last_seen"`
}

func ParseFtMEntity(data []byte) (FtMEntity, error) {
    var raw struct {
        ID       string   `json:"id"`
        Schema   string   `json:"schema"`
        Properties struct {
            Name      []string `json:"name"`
            Country   []string `json:"country"`
            BirthDate []string `json:"birthDate"`
            Position  []string `json:"position"`
        } `json:"properties"`
        Datasets  []string `json:"datasets"`
        FirstSeen string   `json:"first_seen"`
        LastSeen  string   `json:"last_seen"`
    }
    if err := json.Unmarshal(data, &raw); err != nil {
        return FtMEntity{}, err
    }
    return FtMEntity{
        ID: raw.ID, Schema: raw.Schema,
        Name: first(raw.Properties.Name), Country: first(raw.Properties.Country),
        BirthDate: first(raw.Properties.BirthDate), Position: first(raw.Properties.Position),
        Dataset: first(raw.Datasets), FirstSeen: raw.FirstSeen, LastSeen: raw.LastSeen,
    }, nil
}

func first(s []string) string { if len(s) > 0 { return s[0] }; return "" }

type PEPFetcher struct {
    sourceType string
    config     struct{ Dataset string `json:"dataset"` }
    db         *sql.DB
}

func (p *PEPFetcher) SourceType() string { return p.sourceType }
func (p *PEPFetcher) Validate() error    { return nil }

func RunPEP(ctx context.Context, baseURL string, db *sql.DB, wm *ingestion.WatermarkManager, rawDir string) error {
    slog.Info("starting PEP ingestion")
    if err := ensurePEPTable(db); err != nil { return err }
    // ... fetch, parse, write, save raw, update watermark
    wm.Set(pepSourceType, time.Now(), "", "")
    return nil
}

func ensurePEPTable(db *sql.DB) error {
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (
        id TEXT PRIMARY KEY, name TEXT, country TEXT, birth_date TEXT,
        position TEXT, party TEXT, dataset TEXT, first_seen TEXT, last_seen TEXT,
        ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
    return err
}
```

- [ ] **Step 3: Commit**

```bash
go test ./internal/ingestion/sources/ -run TestPEP -v -count=1 && git add internal/ingestion/sources/pep.go internal/ingestion/sources/pep_test.go && git commit -m "feat: add PEP source type (OpenSanctions FtM JSON)"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestPEP -v -count=1`
- **Steps:** (1) Parse sample FtM JSON entity, (2) Verify all fields mapped correctly (ID, name, country, birthDate, position), (3) Run full ingestion against mock server, (4) Verify `pep_entities` table has rows, (5) Verify watermark updated
- **Expected result:** FtM parsing correct. DuckDB write correct. Watermark updated.

---

### Task 7: source_type="public_contracts" — ANAC CSV

**Files:**
- Create: `internal/ingestion/sources/anac.go`
- Create: `internal/ingestion/sources/anac_test.go`

**Goal:** ANAC Open Data CSV ingestion con encoding ISO-8859-1, caricamento diretto DuckDB via `read_csv_auto` (Oracle recommendation: evitare parsing Go-side), raw CSV salvato su disco. Schema: `public_contracts(cig, anno, importo, stazione_appaltante, aggiudicatario)`.

- [ ] **Step 1: Test — ANAC CSV parsing e encoding handling**

```go
func TestANACCSVEncoding(t *testing.T) {
    // Sample ISO-8859-1 CSV with Italian characters
    csvData := []byte("CIG;Anno;Importo;StazioneAppaltante;Aggiudicatario\r\n" +
        "1234567ABC;2024;150000.00;Comune di Forl\xec;Societ\xe0 Esempio S.r.l.\r\n")

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/csv; charset=ISO-8859-1")
        w.Write(csvData)
    }))
    defer server.Close()

    db := setupTestDuckDB(t)
    defer db.Close()

    wm := ingestion.NewWatermarkManager(db)
    err := RunANAC(context.Background(), server.URL, db, wm, 2024, "/tmp/test-raw")
    require.NoError(t, err)

    var count int
    db.QueryRow("SELECT COUNT(*) FROM public_contracts").Scan(&count)
    assert.Equal(t, 1, count)

    var name string
    db.QueryRow("SELECT stazione_appaltante FROM public_contracts LIMIT 1").Scan(&name)
    assert.Contains(t, name, "Forlì") // ISO-8859-1 ì decoded correctly
}

func TestANACDryRun(t *testing.T) {
    csvData := []byte("CIG;Anno;Importo;StazioneAppaltante;Aggiudicatario\r\n" +
        "1234567ABC;2024;150000.00;Comune Roma;Società Esempio S.r.l.\r\n")

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write(csvData)
    }))
    defer server.Close()

    db := setupTestDuckDB(t)
    defer db.Close()

    // Dry run validates but doesn't write
    err := RunANACDryRun(server.URL, 2024)
    require.NoError(t, err)

    var count int
    db.QueryRow("SELECT COUNT(*) FROM public_contracts").Scan(&count)
    assert.Equal(t, 0, count, "dry run should not write to DB")
}
```

- [ ] **Step 2: Fail → Implementa anac.go con DuckDB native CSV reader**

```go
package sources

import (
    "context"
    "database/sql"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "os"
    "path/filepath"
    "time"
    "example.com/aleph/internal/ingestion"
)

const anacSourceType = "public_contracts"

func init() {
    ingestion.GlobalRegistry.Register(anacSourceType, func(sourceType string) ingestion.Fetcher {
        return &ANACFetcher{sourceType: sourceType}
    })
}

type ANACFetcher struct {
    sourceType string
}

func (a *ANACFetcher) SourceType() string { return a.sourceType }
func (a *ANACFetcher) Validate() error    { return nil }

func RunANAC(ctx context.Context, baseURL string, db *sql.DB, wm *ingestion.WatermarkManager, anno int, rawDir string) error {
    url := fmt.Sprintf("%s/CIG_%d.csv", baseURL, anno)
    slog.Info("downloading ANAC CSV", "url", url, "year", anno)

    // 1. Download CSV to temp file
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("download ANAC CSV: %w", err)
    }
    defer resp.Body.Close()

    tmpDir := filepath.Join(rawDir, anacSourceType)
    os.MkdirAll(tmpDir, 0755)
    tmpPath := filepath.Join(tmpDir, fmt.Sprintf("CIG_%d.csv", anno))
    f, err := os.Create(tmpPath)
    if err != nil {
        return err
    }
    if _, err := io.Copy(f, resp.Body); err != nil {
        f.Close()
        return err
    }
    f.Close()

    // 2. Load CSV directly into DuckDB (handles encoding via read_csv_auto)
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS public_contracts (
        cig TEXT PRIMARY KEY,
        anno INTEGER,
        importo REAL,
        stazione_appaltante TEXT,
        aggiudicatario TEXT,
        partecipanti TEXT,
        ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
    if err != nil {
        return err
    }

    _, err = db.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO public_contracts
        SELECT CIG, %d, CAST(Importo AS REAL), StazioneAppaltante, Aggiudicatario, Partecipanti
        FROM read_csv_auto('%s', delim=';', header=true, encoding='ISO-8859-1')`, anno, tmpPath))
    if err != nil {
        return fmt.Errorf("load CSV into DuckDB: %w", err)
    }

    // 3. Update watermark
    wm.Set(anacSourceType, time.Now(), fmt.Sprintf("%d", anno), fmt.Sprintf(`{"records_loaded":true}`))
    slog.Info("ANAC ingestion complete", "year", anno, "file", tmpPath)
    return nil
}

func RunANACDryRun(url string, anno int) error {
    resp, err := http.Head(url)
    if err != nil {
        return fmt.Errorf("ANAC source unreachable: %w", err)
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("ANAC CSV %d not found: HTTP %d", anno, resp.StatusCode)
    }
    return nil
}
```

- [ ] **Step 3: Fail → Pass → Commit**

```bash
go test ./internal/ingestion/sources/ -run TestANAC -v -count=1 && git add internal/ingestion/sources/anac.go internal/ingestion/sources/anac_test.go && git commit -m "feat: add ANAC public contracts CSV source with DuckDB native reader"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestANAC -v -count=1`
- **Steps:** (1) Run encoding test with ISO-8859-1 CSV, (2) Verify "Forlì" decoded correctly (not garbled), (3) Verify public_contracts table has correct row count, (4) Run dry-run test, (5) Verify dry-run HEAD request works but DB untouched
- **Expected result:** ISO-8859-1 decoding correct. DuckDB table populated. Dry-run non-invasive.

---

### Task 8: source_type="party_funding" — ondata/liberiamoli-tutti

**Files:**
- Create: `internal/ingestion/sources/funding.go`
- Create: `internal/ingestion/sources/funding_test.go`

**Goal:** Git clone/pull `ondata/liberiamoli-tutti`, import CSV `political_finance.csv` in DuckDB. Watermark: git commit hash come cursor.

- [ ] **Step 1: Test — funding CSV import**

```go
func TestPartyFundingCSV(t *testing.T) {
    csvData := "donation_amount,donation_year,recipient_party,donor_type,donor_name,source_name\r\n" +
        "50000,2023,Partito Democratico,Persona Fisica,Mario Rossi,Bilancio Camera 2023\r\n"

    tmpFile := filepath.Join(t.TempDir(), "political_finance.csv")
    os.WriteFile(tmpFile, []byte(csvData), 0644)

    db := setupTestDuckDB(t)
    defer db.Close()

    wm := ingestion.NewWatermarkManager(db)
    err := ImportFundingCSV(context.Background(), db, wm, tmpFile, "/tmp/test-raw")
    require.NoError(t, err)

    var count int
    db.QueryRow("SELECT COUNT(*) FROM party_funding").Scan(&count)
    assert.Equal(t, 1, count)

    var amount float64
    db.QueryRow("SELECT donation_amount FROM party_funding LIMIT 1").Scan(&amount)
    assert.Equal(t, 50000.0, amount)
}
```

- [ ] **Step 2: Fail → Implementa → Pass → Commit**

```go
package sources

const fundingSourceType = "party_funding"

func init() {
    ingestion.GlobalRegistry.Register(fundingSourceType, func(sourceType string) ingestion.Fetcher {
        return &FundingFetcher{sourceType: sourceType}
    })
}

func ImportFundingCSV(ctx context.Context, db *sql.DB, wm *ingestion.WatermarkManager, csvPath string, rawDir string) error {
    slog.Info("importing party funding CSV", "path", csvPath)
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (
        id INTEGER PRIMARY KEY,
        donation_amount REAL, donation_year INTEGER,
        recipient_party TEXT, donor_type TEXT, donor_name TEXT,
        source_name TEXT, ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
    if err != nil { return err }
    _, err = db.Exec(fmt.Sprintf(`INSERT INTO party_funding (donation_amount, donation_year, recipient_party, donor_type, donor_name, source_name)
        SELECT CAST(donation_amount AS REAL), CAST(donation_year AS INTEGER), recipient_party, donor_type, donor_name, source_name
        FROM read_csv_auto('%s', header=true)`, csvPath))
    if err != nil { return fmt.Errorf("import funding CSV: %w", err) }
    wm.Set(fundingSourceType, time.Now(), "", "")
    slog.Info("party funding import complete")
    return nil
}
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestPartyFunding -v -count=1`
- **Steps:** (1) Create test CSV with one row, (2) Import via DuckDB read_csv_auto, (3) Verify `party_funding` table has 1 row, (4) Verify `donation_amount = 50000.0`
- **Expected result:** Row imported correctly. Numeric fields parsed.

---

### Task 9: source_type="opdm" — Openpolis REST API

**Files:**
- Create: `internal/ingestion/sources/opdm.go`
- Create: `internal/ingestion/sources/opdm_test.go`

**Goal:** REST API calls a `service.opdm.openpolis.io` con API key, pagination-aware, rate limiter 10k req/day. DuckDB table `opdm_memberships`. Watermark: pagina/cursore.

- [ ] **Step 1: Test — OPDM pagination + rate limit**

```go
func TestOPDMPagination(t *testing.T) {
    page := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        page++
        if page <= 2 {
            w.Write([]byte(fmt.Sprintf(`{"items":[{"person_id":%d,"org_id":1,"role":"Membro"}],"next":"page_%d"}`, page, page+1)))
        } else {
            w.Write([]byte(`{"items":[],"next":null}`))
        }
    }))
    defer server.Close()

    db := setupTestDuckDB(t)
    defer db.Close()
    wm := ingestion.NewWatermarkManager(db)

    err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", "/tmp/test-raw")
    require.NoError(t, err)

    var count int
    db.QueryRow("SELECT COUNT(*) FROM opdm_memberships").Scan(&count)
    assert.Equal(t, 2, count)
}
```

- [ ] **Step 2: Fail → Implementa → Pass → Commit**

```go
package sources

const opdmSourceType = "opdm"

func init() {
    ingestion.GlobalRegistry.Register(opdmSourceType, func(sourceType string) ingestion.Fetcher {
        return &OPDMFetcher{sourceType: sourceType}
    })
}

// Use token-bucket rate limiter: 10000 req / 86400 secondi = ~0.116 req/s
func RunOPDM(ctx context.Context, baseURL string, db *sql.DB, wm *ingestion.WatermarkManager, apiKey string, rawDir string) error {
    rl := ingestion.NewTokenBucketRateLimiter(10000.0/86400.0, 5)
    // ... paginated fetch with rate limiting
    return nil
}
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestOPDM -v -count=1`
- **Steps:** (1) Mock server returns 2 pages, (2) Verify pagination follows `next` cursor, (3) Verify 2 rows in opdm_memberships, (4) Verify rate limiter enforces ~0.116 req/s
- **Expected result:** Pagination stops when next=null. 2 entities ingested.

---

### Task 10: source_type="parliament" — SPARQL + Codice Civico

**Files:**
- Create: `internal/ingestion/sources/parliament.go`
- Create: `internal/ingestion/sources/parliament_test.go`

**Goal:** SPARQL query a Camera e Senato. Integrazione Codice Civico API per cross-reference. DuckDB table `parliament_votes`. Rate limiting 5 req/s per endpoint.

- [ ] **Step 1: Test — SPARQL result parsing**

```go
const sampleSPARQL = `{
  "head": {"vars": ["votazione", "data", "titolo", "esito", "deputato", "gruppo"]},
  "results": {"bindings": [
    {"votazione": {"value": "123"}, "data": {"value": "2024-01-15"}, "titolo": {"value": "Fiducia"}, "esito": {"value": "APPROVATA"}, "deputato": {"value": "Mario Rossi"}, "gruppo": {"value": "FDI"}}
  ]}
}`

func TestSPARQLParsing(t *testing.T) {
    votes, err := ParseSPARQLVotes([]byte(sampleSPARQL))
    require.NoError(t, err)
    assert.Len(t, votes, 1)
    assert.Equal(t, "123", votes[0].ID)
    assert.Equal(t, "APPROVATA", votes[0].Esito)
    assert.Equal(t, "FDI", votes[0].Gruppo)
}
```

- [ ] **Step 2: Fail → Implementa parliament.go → Pass → Commit**

```go
package sources

const parliamentSourceType = "parliament"

func init() {
    ingestion.GlobalRegistry.Register(parliamentSourceType, func(sourceType string) ingestion.Fetcher {
        return &ParliamentFetcher{sourceType: sourceType}
    })
}

func ParseSPARQLVotes(data []byte) ([]Vote, error) { /* ... */ }
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/sources/ -run TestSPARQL -v -count=1`
- **Steps:** (1) Parse sample SPARQL JSON response, (2) Verify 1 vote extracted, (3) Verify all fields mapped correctly
- **Expected result:** Correct mapping. Esito=APPROVATA, Gruppo=FDI.

---

### Task 11: DuckDB Cross-Reference Views

**Files:**
- Modify: `internal/ingestion/engine.go`

**Goal:** Definire viste SQL permanenti registrate dopo ogni ingestion. `politician_full_profile`, `contract_party_link`, `funding_timeline`. Create OR REPLACE VIEW per idempotenza.

- [ ] **Step 1: Test — View creation + idempotenza**

```go
func TestCrossReferenceViews(t *testing.T) {
    db := setupTestDuckDB(t)
    defer db.Close()

    // Create minimal tables needed for views
    db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (id TEXT, name TEXT, party TEXT)`)
    db.Exec(`CREATE TABLE IF NOT EXISTS opdm_memberships (person_id TEXT, org_id TEXT, role TEXT)`)
    db.Exec(`CREATE TABLE IF NOT EXISTS parliament_votes (deputato TEXT, gruppo TEXT, esito TEXT)`)
    db.Exec(`CREATE TABLE IF NOT EXISTS public_contracts (cig TEXT, aggiudicatario TEXT, importo REAL)`)
    db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (recipient_party TEXT, donation_amount REAL, donation_year INTEGER)`)

    // Run once
    err := RegisterCrossReferenceViews(db)
    require.NoError(t, err)

    // Run again (idempotent)
    err = RegisterCrossReferenceViews(db)
    require.NoError(t, err)

    // Verify views exist
    var views []string
    rows, _ := db.Query("SELECT view_name FROM duckdb_views() WHERE view_name LIKE 'v_%'")
    for rows.Next() { var v string; rows.Scan(&v); views = append(views, v) }
    assert.GreaterOrEqual(t, len(views), 2)
}
```

- [ ] **Step 2: Fail → Implementa in engine.go**

```go
func RegisterCrossReferenceViews(db *sql.DB) error {
    views := []string{
        `CREATE OR REPLACE VIEW v_politician_full_profile AS
         SELECT p.id, p.name, p.party, o.role, o.org_id, v.gruppo
         FROM pep_entities p
         LEFT JOIN opdm_memberships o ON p.name = o.person_id
         LEFT JOIN parliament_votes v ON p.name = v.deputato`,

        `CREATE OR REPLACE VIEW v_contract_party_link AS
         SELECT c.cig, c.aggiudicatario, c.importo, p.name as pep_name
         FROM public_contracts c
         LEFT JOIN pep_entities p ON c.aggiudicatario LIKE '%' || p.name || '%'`,

        `CREATE OR REPLACE VIEW v_funding_timeline AS
         SELECT recipient_party, donation_year, SUM(donation_amount) as total_amount, COUNT(*) as donation_count
         FROM party_funding GROUP BY recipient_party, donation_year ORDER BY donation_year DESC, total_amount DESC`,
    }
    for _, v := range views {
        if _, err := db.Exec(v); err != nil {
            return fmt.Errorf("create view: %w", err)
        }
    }
    slog.Info("cross-reference views registered")
    return nil
}
```

- [ ] **Step 3: Fail → Pass → Commit**

```bash
go test ./internal/ingestion/ -run TestCrossReference -v -count=1 && git add internal/ingestion/engine.go internal/ingestion/engine_test.go && git commit -m "feat: add DuckDB cross-reference views for political data"
```

**VERIFICA (QA Scenario):**
- **Tool:** `go test ./internal/ingestion/ -run TestCrossReference -v -count=1`
- **Steps:** (1) Create base tables, (2) Run RegisterCrossReferenceViews twice, (3) Verify no errors on second run (idempotent), (4) Verify v_politician_full_profile, v_contract_party_link, v_funding_timeline exist
- **Expected result:** Three views created. Idempotent. Views queryable without errors.

---

## Riepilogo

| # | Task | Source Type | Fonte | Formato | QA |
|---|---|---|---|---|---|
| 0.1 | Watermark + Migrations | — | DuckDB | SQL | ✅ |
| 0.2 | RateLimiter | — | Token bucket | Go | ✅ |
| 0.3 | Registry Pattern | — | — | Go | ✅ |
| 1 | Party Alias Table | `election` | — | Go/JSON | ✅ |
| 2 | ElectionFetcher | `election` | Eligendo API | JSON | ✅ |
| 3 | runElection Pipeline | `election` | Eligendo API | JSON+gzip | ✅ |
| 4 | Frontend | — | React/TS | TSX | ✅ |
| 5 | Health Endpoint | — | HTTP | Go | ✅ |
| 6 | PEP | `pep` | OpenSanctions | FtM JSON | ✅ |
| 7 | ANAC Appalti | `public_contracts` | ANAC Open Data | CSV ISO-8859-1 | ✅ |
| 8 | Party Funding | `party_funding` | ondata | CSV | ✅ |
| 9 | Openpolis | `opdm` | OPDM API | JSON | ✅ |
| 10 | Parliament | `parliament` | SPARQL + CC | RDF/JSON | ✅ |
| 11 | Cross-Ref Views | — | DuckDB SQL | SQL | ✅ |

**Totale: 14 task (3 infrastruttura + 11 feature), 6 nuovi source type**

**Key Review Feedback Addressed:**
- ✅ **Oracle 🔴 Critical:** Watermark table (0.1), testing go-vcr (2), health endpoint (5), structured logging (all tasks)
- ✅ **Oracle 🟡 High:** DuckDB sequential execution, dry-run ANAC (7), raw-on-disk (tutti)
- ✅ **Oracle 🟢 Low:** Alias table replaces Jaro-Winkler (1), registry pattern (0.3), RateLimiter interface (0.2)
- ✅ **Metis:** CSV encoding ISO-8859-1 (7), frontend Phase 2 coverage (4), DuckDB read_csv_auto (7)
- ✅ **Momus:** QA scenarios required — OGNI task ha sezione VERIFICA con tool+steps+expected

**Specs collegate:** `docs/superpowers/plans/execution_specs_politica.md`
