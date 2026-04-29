# Aleph-v2 Production Gate — Specifiche Tecniche

**Data**: 2026-04-29
**Piano riferimento**: `docs/superpowers/plans/29-apr-plan.md`
**Mandato**: Zero-Todo — nessun placeholder, nessun debito tecnico, nessun bug prima della chiusura Phase 2

---

## Indice

1. [SSRF Validator: unificazione fail-closed](#1-ssrf-validator-unificazione-fail-closed)
2. [Usage Tracking Subsystem](#2-usage-tracking-subsystem)
3. [Ontology Negotiation API (Protobuf)](#3-ontology-negotiation-api-protobuf)
4. [LLM Emerge Prompt Design](#4-llm-emerge-prompt-design)
5. [MemoryStore DuckDB Schema](#5-memorystore-duckdb-schema)
6. [Genesis Suggester: 3 Bound Types](#6-genesis-suggester-3-bound-types)
7. [Security Migration Patterns](#7-security-migration-patterns)
8. [Config Validation Contract](#8-config-validation-contract)

---

## 1. SSRF Validator: Unificazione Fail-Closed

**Wave**: W1-09
**File target**: `internal/ssrf/validator.go` (nuovo, unifica due meccanismi esistenti)

### Stato attuale (problema)

Due meccanismi SSRF separati che NON condividono codice:

1. **Transport-level** (`init()` in `engine.go`): `safeHTTPClient.Transport.DialContext` — blocca loopback/private/link-local a livello di connessione TCP
2. **URL-level** (`blockSSRF()` in `engine.go`): blocca private IPs, non-decimal IPs, short forms, internal TLDs

**Problema**: engine.go usa `safeHTTPClient` MA solo per alcune route. Altre usano `http.DefaultClient` → **fail-open** (nessuna protezione SSRF).

### Spec

```go
// internal/ssrf/validator.go

package ssrf

import "net/url"

// Config per SSRF validation
type Config struct {
    BlockPrivateRanges  bool // blocca 10.x, 172.16-31.x, 192.168.x, 127.x
    BlockLinkLocal      bool // blocca 169.254.x
    BlockLoopback       bool // blocca 127.x, ::1
    BlockInternalTLDs   bool // blocca .internal, .local, .corp
    BlockNonDecimalIPs  bool // blocca 0x7f, 0177, etc.
    AllowList           []string // override per range specifici
}

// Result del validation
type Result struct {
    Allowed bool
    Reason  string // motivo del blocco, per logging
    ResolvedIP string // IP risolto (se DNS lookup fatto)
}

// Validator è il punto di ingresso unico
type Validator struct {
    config Config
    // cache DNS risolti per performance
    // NOTA: non usare DNS resolver built-in (injection risk) — usarne uno con timeout
}

func NewValidator(config Config) *Validator { ... }

// ValidateURL verifica URL completo (host + IP resolution)
func (v *Validator) ValidateURL(rawURL string) Result { ... }

// SafeClient ritorna http.Client con Transport già configurato
// che chiama ValidateURL prima di ogni connessione
func (v *Validator) SafeClient() *http.Client { ... }
```

### Migrazione

1. Creare `internal/ssrf/validator.go` con `Validator` struct
2. Sostituire `safeHTTPClient` in `engine.go` con `validator.SafeClient()`
3. Sostituire chiamate a `blockSSRF()` con `validator.ValidateURL()`
4. Eliminare `safeHTTPClient` e `blockSSRF()` da engine.go
5. Aggiungere validator.Config al Config globale di app
6. Test: portare gli 80 test case esistenti da `ssrf_test.go` + `engine_extended_test.go`

---

## 2. Usage Tracking Subsystem

**Wave**: W1.5-04
**File target**: `internal/service/tracker/` (nuovo package)

### Scopo

Middleware che intercetta tool calls e le registra in DuckDB. Serve come base dati per:
- Genesis Suggester (W4-05): pattern d'uso → suggerimenti
- fixPerformance (W4-06): pattern detection nei dati d'uso
- Analytics future

### Spec

```go
// internal/service/tracker/tracker.go

package tracker

import (
    "context"
    "time"
)

// ToolUsage registra una singola esecuzione di tool
type ToolUsage struct {
    ID          string    `db:"id"`
    UserID      string    `db:"user_id"`
    ProjectID   string    `db:"project_id"`
    ToolName    string    `db:"tool_name"`
    InputHash   string    `db:"input_hash"`    // SHA-256 dell'input (per pattern detection, non loggare dati sensibili)
    DurationMs  int64     `db:"duration_ms"`
    Success     bool      `db:"success"`
    ErrorMsg    string    `db:"error_msg"`     // vuoto se success
    Timestamp   time.Time `db:"timestamp"`
}

// Tracker registra usage
type Tracker interface {
    Record(ctx context.Context, usage ToolUsage) error
    // Query patterns per Genesis (limitate nel tempo)
    MostUsedTools(ctx context.Context, userID string, limit int, since time.Time) ([]ToolUsageStat, error)
    ToolSequences(ctx context.Context, userID string, limit int) ([][]string, error)
}

// ToolUsageStat — statistica aggregata
type ToolUsageStat struct {
    ToolName    string  `db:"tool_name"`
    Count       int     `db:"count"`
    AvgDuration float64 `db:"avg_duration_ms"`
    SuccessRate float64 `db:"success_rate"`
}
```

### Schema DuckDB

```sql
CREATE TABLE IF NOT EXISTS tool_usage (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    project_id  VARCHAR NOT NULL,
    tool_name   VARCHAR NOT NULL,
    input_hash  VARCHAR,
    duration_ms BIGINT,
    success     BOOLEAN DEFAULT TRUE,
    error_msg   VARCHAR DEFAULT '',
    timestamp   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tool_usage_user_time ON tool_usage(user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_tool_usage_tool ON tool_usage(tool_name);
```

### Integrazione

- Middleware HTTP che wrappa gli handler di tool execution
- O chiamata esplicita dopo ogni `Act()` in `DecisionEngine`
- **NO** logging di dati sensibili (input_hash è hash, non input raw)

---

## 3. Ontology Negotiation API (Protobuf)

**Wave**: W2A-02
**File target**: `api/ontology.proto` (nuovo)

### Spec Protobuf

```protobuf
syntax = "proto3";
package aleph.ontology;
option go_package = "aleph-v2/api/ontology";

// --- Types ---

// OntologyDiff rappresenta una proposta di modifica
message OntologyDiff {
  string id = 1;
  string project_id = 2;
  // Versione parent (null per emergenza iniziale)
  string parent_version_id = 3;

  // Modifiche atomiche
  repeated ObjectAdd objects_add = 10;
  repeated ObjectModify objects_modify = 11;
  repeated ObjectRemove objects_remove = 12;
  repeated RelationAdd relations_add = 13;
  repeated RelationModify relations_modify = 14;
  repeated RelationRemove relations_remove = 15;

  // Source description (per LLM context)
  string source_description = 20;
  // LLM rationale dietro i suggerimenti
  string rationale = 21;
  // Confidence score LLM (0.0 - 1.0)
  float confidence = 22;

  // Stato proposta
  enum Status {
    PENDING = 0;
    ACCEPTED = 1;
    REJECTED = 2;
    MODIFIED = 3;
    SUPERSEDED = 4;
  }
  Status status = 23;
}

message ObjectAdd {
  string name = 1;
  string description = 2;
  repeated string properties = 3;
  map<string, string> type_hints = 4; // property_name → "text|number|datetime|boolean"
  string from_source = 5;
}

message ObjectModify {
  string name = 1;
  // Campi da aggiungere/modificare
  repeated string properties_add = 10;
  repeated string properties_remove = 11;
  map<string, string> type_hints_update = 12;
  string description_update = 13;
}

message ObjectRemove {
  string name = 1;
}

message RelationAdd {
  string name = 1;
  string from_object = 2;
  string to_object = 3;
  string on_property = 4;
  string relation_type = 5; // "fk" | "contains" | "references" | "derives_from"
}

message RelationModify {
  string name = 1;
  string on_property_update = 4;
  string relation_type_update = 5;
}

message RelationRemove {
  string name = 1;
}

// --- Services ---

service OntologyNegotiation {
  // LLM propone modifiche
  rpc Propose(OntologyDiff) returns (ProposeResponse);

  // User accetta
  rpc Accept(AcceptRequest) returns (AcceptResponse);

  // User rifiuta
  rpc Reject(RejectRequest) returns (RejectResponse);

  // User modifica la proposta
  rpc Modify(ModifyRequest) returns (ModifyResponse);

  // Storico versioni
  rpc ListVersions(ListVersionsRequest) returns (ListVersionsResponse);
}

message ProposeResponse {
  string diff_id = 1;
  string preview_core_aleph = 2; // anteprima del file generato
  repeated string warnings = 3;  // es: "2 objects hanno nomi simili a X"
}

message AcceptRequest {
  string diff_id = 1;
}
message AcceptResponse {
  string version_id = 1;
}

message RejectRequest {
  string diff_id = 1;
  string reason = 2; // opzionale, per migliorare LLM
}
message RejectResponse {
  bool success = 1;
}

message ModifyRequest {
  string diff_id = 1;
  OntologyDiff modified_diff = 2;
}
message ModifyResponse {
  string new_diff_id = 1;
  string preview_core_aleph = 2;
}

message ListVersionsRequest {
  string project_id = 1;
  int32 limit = 2;
}
message ListVersionsResponse {
  repeated VersionEntry versions = 1;
}
message VersionEntry {
  string version_id = 1;
  string parent_version_id = 2;
  string created_at = 3;
  string status = 4; // "accepted" | "rejected" | "superseded"
  string source_description = 5;
}
```

### DB Schema (DuckDB)

```sql
CREATE TABLE IF NOT EXISTS ontology_versions (
    version_id        VARCHAR PRIMARY KEY,
    project_id        VARCHAR NOT NULL,
    parent_version_id VARCHAR,
    diff_json         TEXT NOT NULL,       -- OntologyDiff serializzato
    core_aleph_snapshot TEXT NOT NULL,     -- core.aleph dopo applicazione
    status            VARCHAR DEFAULT 'pending',
    source_description VARCHAR,
    rationale         TEXT,
    confidence        FLOAT,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at       TIMESTAMP
);
```

---

## 4. LLM Emerge Prompt Design

**Wave**: W2B-01
**File target**: `internal/handler/project.go` (modifica `EmergeOntology`)

### Prompt Structure

```
System: Sei un ontologo esperto. Analizzi sorgenti dati e produci
un'ontologia strutturata. Regole:
- Ogni "source" (tabella/API/CSV) diventa un Object
- Colonne con nomi simili tra sorgenti diverse potrebbero essere relazioni
- Type inference: string → text, int/float → number, date/time → datetime, bool → boolean
- Relazioni: FK naming (user_id → users.id), name overlap (customer_name → customer.name)
- Output in formato DSL aleph (objects + relations)

User: [descrizione della sorgente, colonne, sample data]
```

### 3 Test Case per Convergenza

**TC1 — E-commerce semplice**: 2 tabelle (orders, customers) con FK `orders.customer_id → customers.id`

*Expected output*:
- objects: `orders` (properties: id, customer_id, total, status, created_at) + `customers` (properties: id, name, email)
- relations: `orders_belongs_to_customer` (from: orders.customer_id, to: customers.id)

**TC2 — API GitHub**: `/repos/{owner}/{repo}/issues` (issues, labels, assignees embedded)

*Expected output*:
- objects: `issues` (properties: number, title, state, created_at, labels[], assignee), `labels` (name, color), `users` (login, avatar_url)
- relations: `issue_has_labels`, `issue_assigned_to_user`

**TC3 — CSV misto con date e nested**: `transactions.csv` con campi data, importo, categoria, note

*Expected output*:
- objects: `transactions` (properties: data=datetime, importo=number, categoria=text, note=text)
- no relations (singola sorgente)
- factor suggestion: categoria potrebbe essere un Object separato se ci sono abbastanza valori distinti

### Convergence Gate

Il prompt è convergente quando:
1. Tutti e 3 i test case producono output validi (parseable dal DSL parser)
2. Nessun test case richiede più di 2 tentativi
3. L'output per TC1 include SEMPRE la relazione FK
4. L'output per TC2 include SEMPRE oggetti normalizzati (labels/users separati da issues)
5. L'output per TC3 NON inventa relazioni inesistenti

---

## 5. MemoryStore DuckDB Schema

**Wave**: W4-01, W4-02
**File target**: `internal/memory/memory.go` (riscrittura)

### Spec Interfaccia

```go
// internal/memory/memory.go

package memory

import "context"

type MemoryStore interface {
    // Insert memorizza un vettore embedding con metadati
    Insert(ctx context.Context, entry MemoryEntry) error

    // SearchSimilar ritorna i top-k vettori più simili (cosine similarity)
    SearchSimilar(ctx context.Context, query []float32, k int) ([]MemoryEntry, error)

    // SearchSimilarWithFilter come sopra ma con filtro per project_id
    SearchSimilarByProject(ctx context.Context, query []float32, projectID string, k int) ([]MemoryEntry, error)

    // Get ritorna un entry per ID
    Get(ctx context.Context, id string) (*MemoryEntry, error)

    // Delete rimuove un entry
    Delete(ctx context.Context, id string) error

    // Close rilascia risorse
    Close() error
}

type MemoryEntry struct {
    ID        string    `db:"id"`
    ProjectID string    `db:"project_id"`
    AgentID   string    `db:"agent_id"`
    Content   string    `db:"content"`     // testo originale
    Embedding []float32 `db:"embedding"`   // vettore FLOAT[]
    Metadata  string    `db:"metadata"`    // JSON opzionale
    CreatedAt time.Time `db:"created_at"`
}
```

### Schema DuckDB (VSS)

```sql
CREATE TABLE IF NOT EXISTS memory_entries (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL,
    agent_id    VARCHAR,
    content     TEXT NOT NULL,
    embedding   FLOAT[],
    metadata    JSON,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_memory_project ON memory_entries(project_id);
```

**Query VSS**: `array_cosine_similarity(embedding, ?::FLOAT[])` — se disponibile (P0-06).

### Fallback (W4-02, senza VSS)

```sql
-- Calcolare cosine similarity manualmente
SELECT *, 
  (embedding::FLOAT[] <=> ?::FLOAT[]) AS similarity
FROM memory_entries
WHERE project_id = ?
ORDER BY similarity DESC
LIMIT ?
```

NOTA: `<=>` è l'operatore di distanza coseno in DuckDB. Se non disponibile, usare formula SQL esplicita:
```sql
SELECT *, 
  (sum(a.v * b.v) / (sqrt(sum(a.v^2)) * sqrt(sum(b.v^2)))) AS cosine_similarity
FROM ...
```

---

## 6. Genesis Suggester: 3 Bound Types

**Wave**: W4-05
**File target**: `internal/genesis/suggester.go`

### Spec

```go
// internal/genesis/suggester.go

package genesis

// SuggestionType enum
type SuggestionType string
const (
    SuggestionToolAutomation  SuggestionType = "tool_automation"   // Ripetizione tool → script
    SuggestionSkillSuggestion SuggestionType = "skill_suggestion"   // Pattern → skill recommendation
    SuggestionCrossUser       SuggestionType = "cross_user_pattern" // Pattern cross-utente (admin)
)

type Suggestion struct {
    ID          string         `json:"id"`
    Type        SuggestionType `json:"type"`
    Title       string         `json:"title"`
    Description string         `json:"description"`
    Confidence  float64        `json:"confidence"`   // 0.0 - 1.0
    Source      string         `json:"source"`        // qual è il pattern che ha generato questo
    CreatedAt   time.Time      `json:"created_at"`
    Status      string         `json:"status"`        // "pending" | "applied" | "dismissed"
}

type Suggester struct {
    tracker tracker.Tracker // dipende da W1.5-04
    db      *sql.DB         // DuckDB
}

// Analyze analizza usage tracking e produce suggerimenti
func (s *Suggester) Analyze(ctx context.Context) ([]Suggestion, error) {
    // 1. tool_automation: se stesso tool chiamato >5 volte nell'ultima ora
    // 2. skill_suggestion: se pattern di 3+ tool consecutivi matcha una skill esistente
    // 3. cross_user_pattern: se pattern usato da 3+ utenti diversi può diventare skill built-in
}
```

### Regole di Suggerimento

| Tipo | Soglia | Azione |
|------|--------|--------|
| `tool_automation` | Stesso tool >5x/ora | "Crea uno script che esegua X automaticamente ogni N minuti" |
| `skill_suggestion` | Sequenza 3+ tool matcha skill | "Stai usando pattern Y — skill Z fa questo automaticamente" |
| `cross_user_pattern` | Pattern in 3+ utenti | "Pattern X usato da N utenti — creare skill built-in?" |

---

## 7. Security Migration Patterns

**Wave**: W1 (W1-04, W1-05, W1-06, W1-07, W1-08, W1-10)

### W1-04 — SQL Injection: Parameterize 5 Vectors

**Pattern generale**: Sostituire `fmt.Sprintf` per costruzione query con:

| Vector | Query Type | Fix |
|--------|-----------|-----|
| query.go info_schema | `SELECT column_name FROM information_schema.columns WHERE table_name = '%s'` | Parameterized query (`?` placeholder) |
| engine.go CREATE TABLE/VIEW | `CREATE VIEW "%s" AS SELECT ...` | Whitelist `validName()` + `sprig` escaping |
| query.go GetDataStats | `SELECT COUNT(*), AVG("%s")` FROM "%s" | Parameterize colonna (se dinamica → whitelist `validName()`) |
| query.go GetDataLineage | `WHERE table_name IN ('%s')` | Parameterize |
| compiler.go WHERE | `WHERE %s %s '%s'` | Usare DuckDB prepared statement + type assertions |

**Regola**: Ogni input che finisce in SQL deve passare da UNA di:
1. `?` placeholder per valori
2. `validName()` per identificatori (solo `[a-zA-Z_][a-zA-Z0-9_]*`)
3. Prepared statement parameter binding

### W1-05 — Code Execution: runDynamic blockedImports

**Lista blocchi aggiornata** (aggiungere):
```go
var blockedImports = map[string]bool{
    // esistenti...
    "unsafe":   true,
    "reflect":  true,
    "os":       true,
    "io":       true,    // os/io for codice malevolo
    "crypto":   true,    // crypto per mining
    "encoding": true,    // encoding/base64 per exfiltration
    "net":      true,    // per C2
    "syscall":  true,
    "embed":    true,
    "plugin":   true,
}
```

### W1-06 — API Key: sessionStorage → httpOnly Cookie

**Pattern**:
1. Backend: `Set-Cookie: aleph_api_key=<value>; HttpOnly; Secure; SameSite=Strict; Path=/api`
2. Frontend: rimuovere authSlice, leggere cookie implicitamente (browser lo invia)
3. Se serve per WebSocket/SSE: backend genera session token (JWT) associato a API key

### W1-07 — SHA-256 → Argon2

**Pattern**:
```go
import "github.com/alexedwards/argon2id"

hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
match, err := argon2id.ComparePasswordAndHash(password, hash)
```

### W1-10 — Email Credential Leak

**Fix**: invece di generare script Python con credenziali in tmpdir:
1. Usare Go nativo: `github.com/emersion/go-imap` (già best practice)
2. Se Python è necessario: passare credenziali via stdin + env vars, MAI in file su disco
3. Pulire tmpdir dopo esecuzione (defer os.RemoveAll)

---

## 8. Config Validation Contract

**Wave**: W1-08
**File target**: `internal/config/config.go`

### Regole di Validazione

```go
type ConfigValidator struct {
    RequiredKeys []string // fallisce se vuoto
    RegexKeys    map[string]string // key → regex pattern
    MinLengths   map[string]int
}

var productionRules = ConfigValidator{
    RequiredKeys: []string{
        "KEY_ENCRYPTION_KEY",    // MUST non vuoto in produzione
        "JWT_SECRET",
        "DB_CONNECTION_STRING",
    },
    MinLengths: map[string]int{
        "KEY_ENCRYPTION_KEY": 32,  // AES-256 richiede 32 byte
        "JWT_SECRET":         32,
    },
}
```

### Startup Behavior

- `KEY_ENCRYPTION_KEY` vuoto → **FATAL** (app non parte)
- `JWT_SECRET` vuoto → **FATAL**
- `DB_CONNECTION_STRING` vuota → **FATAL**
- Tutti gli altri warning → log + continue
- **MAI** default a stringa vuota (il default DEVE fallire)

---

## Appendice: Dipendenze tra Wave

```
P0 → W1 → W1.5 → W2A → W2B → W2C
               ↘       ↘
               W1.5 → W3 (parallelo a W2)
                              
W1.5 → W4 (dopo usage tracking)
W4 → W5

W1 → W6 (indipendente)
W1 → W7 (dopo CSP audit)
W7 → Release Gate
```
