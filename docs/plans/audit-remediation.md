# Aleph-v2 Audit Remediation Plan

**Plan version**: 1.0  
**Date**: 2 May 2026  
**Audit reference**: `docs/reports/report-1.md` (111 findings, 22 CRITICAL, 38 HIGH, 39 MEDIUM, 12 LOW)  
**Goal**: Close all 22 CRITICAL and 38 HIGH findings. Address medium/low where overlapping with critical paths.  
**Estimated timeline**: 15 working days (5 waves × 3 days)

---

## Executive Summary

The Aleph-v2 audit identified 111 findings across 10 security & quality clusters. This plan organizes remediation into **5 dependency-ordered waves**, each self-contained with explicit test coverage and verification gates. No wave proceeds until its predecessor's gate passes.

### Critical Path

```
Secrets & Auth (W0) → Injection & Sandbox (W1) → Database (W2) → API & Frontend (W3) → Concurrency & Infra (W4)
```

### Why This Order

| Wave | Dependency | Rationale |
|------|-----------|-----------|
| W0 | None | Cannot secure anything without proper auth + secrets management |
| W1 | W0 auth | Sandbox scoping needs RBAC enforcement; injection fixes need context propagation from auth |
| W2 | W0 auth | Database atomicity requires context-scoped transactions |
| W3 | W0 secrets | Frontend can't remove `apiKey` from Zustand until server-side auth is solid; CSP needs auth path finalized |
| W4 | All above | Goroutine leaks often tied to missing context deadlines set in W1-W3; infra hardening is orthogonal |

---

## Wave 0: Foundation — Secrets & Auth (Days 1–2)

**Objective**: Secure the authentication surface and eliminate credential leakage.  
**Findings addressed**: R1-R10 (auth), L1-L5+L8 (secrets), A5 (rate limiting), INF1 (plaintext secrets)  
**Priority**: 🔴 CRITICAL — gates all other work

### Work Strategy

**Delegation**: Fire 2 parallel agents: `backend-engineer` for Go auth/RBAC changes, `fullstack-guardian` for secrets migration (Go + Python subprocess). Wait for both, then run integration tests before marking done.

### Tasks

#### W0-1: Secrets Management Migration
- [ ] Install `gosecrets` for encrypted credential storage
- [ ] Remove all plaintext secrets from `.env` → `secrets/development.enc`
- [ ] Remove `MASTER_KEY`, `DB_PASSWORD`, `SMTP_PASSWORD`, `ALEPH_API_KEY_SECRET_BACKEND`, `JWT_SECRET`, `KEY_ENCRYPTION_KEY` from plaintext
- [ ] Add `.gitignore` entries for `secrets/*.key`
- [ ] Add CI pipeline env vars: `GOSECRETS_ENV=production`, `GOSECRETS_PRODUCTION_KEY` (from GitHub Secrets)
- [ ] Update `internal/app/app.go` config loading to use `gosecrets.Load()` instead of `os.Getenv()`
- [ ] Write migration doc for onboarding new developers

#### W0-2: Remove apiKey from Frontend State
- [ ] Remove `apiKey` field from `authSlice.ts`
- [ ] Replace Zustand `apiKey` reads with httpOnly cookie session auth (`POST /api/v1/auth/session`)
- [ ] Remove `apiKey` from `setProjectContext()` 
- [ ] Remove `apiKey` plaintext display from `SetupWizard.tsx` (line 164)
- [ ] Fix API key masking in `SettingsView.tsx` to show first 4 chars (prefix), not last 4
- [ ] Verify `useSSE.ts` still authenticates via cookie or X-Aleph-Api-Key header (not Zustand state)
- [ ] Verify `client.ts` `createSession` uses credentials only (no api_key in body)

#### W0-3: RBAC Enforcement
- [ ] Wire `RequireRole` into ALL handler groups:
  - Admin-only routes (CreateApiKey, DeleteApiKey, DeleteProjectCascade, tool management): `RequireRole(RoleAdmin)`
  - Write routes (CreateAgent, UpdateTool, etc.): `RequireRole(RoleAdmin, RoleUser)`
  - Read routes (ListAgents, GetTool, etc.): `RequireRole(RoleAdmin, RoleUser, RoleReadOnly)`
- [ ] Wire `IsAdmin` into admin-specific handler logic where partial admin access is needed
- [ ] Add `RequireProjectRole` middleware that verifies user has >= required role IN the target project
- [ ] Apply RBAC to ConnectRPC interceptor chain (before `authInterceptor` or as post-auth step)
- [ ] Document role matrix in `docs/AUTH.md`

#### W0-4: Fix authSkipSet
- [ ] Remove `CreateApiKey`, `ListApiKeys`, `RevokeApiKey` from `authSkipSet`
- [ ] Verify `AuthHandler` uses `ValidateAPIKey` internally (it already does — validate)
- [ ] Add `RequireRole(RoleAdmin)` to all 3 endpoint handlers
- [ ] Ensure admin API keys are marked with admin role in JWT

#### W0-5: JWT Hardening
- [ ] Add `aud` (audience) claim validation — token bound to `aleph-v2-api`
- [ ] Add `sub` (subject) claim validation — token bound to user identity
- [ ] Add `iss` (issuer) claim verification — reject tokens from unknown issuers
- [ ] Add `jti` (JWT ID) claim — enables revocation
- [ ] Implement token revocation list (in-memory TTL cache with PostgreSQL persistence)
- [ ] Validate `scopes` claim on ConnectRPC endpoints
- [ ] Reduce JWT TTL from 24h → 1h with refresh token mechanism (or remove refresh entirely — use API keys for long-lived access)

#### W0-6: Rate Limiting on Auth Endpoints
- [ ] Install and wire `krishna-kudari/ratelimit` for auth-specific rate limiting
- [ ] Auth endpoints (`/api/v1/auth/session`): 5 req/min/IP (brute-force prevention)
- [ ] API key endpoints (`CreateApiKey`, `ListApiKeys`, `RevokeApiKey`): 10 req/min/IP
- [ ] Verify Redis-backed rate limiter works in multi-instance deployment
- [ ] Keep existing dual-key rate limiter for general API

#### W0-7: Secure Credential Passing to Python Subprocesses
- [ ] Refactor `ingestion/engine.go` `runEmailFetch()` (line 882) → use stdin pipe for credentials instead of env vars
- [ ] Refactor `internal/dsl/compiler_tool.go` `apiConnectorPythonTemplate` → remove `urllib.request` import from template
- [ ] Add `--stdin-credentials` flag to NLP sidecar Docker image
- [ ] Verify no other subprocess execution paths pass credentials via env

#### W0-8: SSE Auth Chain Integration
- [ ] Move SSE handler (`/api/v1/events`) INTO the main middleware chain
- [ ] Alternative: Apply `RateLimitMiddleware` + `SecurityHeaders` + `CSRFProtection` manually to SSE handler
- [ ] SSE-specific rate limit: 2 connections/IP, 100 events/min
- [ ] Add SSE connection tracking (active connections per project)

### Test Strategy — Wave 0

| Layer | Test Type | What |
|-------|-----------|------|
| **Unit** | Go | `auth_test.go`: RBAC enforcement for all role combinations (admin/user/readonly/unauthenticated) |
| **Unit** | Go | `auth_middleware_test.go`: authSkipSet removal validation (verify these endpoints now REQUIRE auth) |
| **Unit** | Go | `jwt_test.go`: aud/sub/iss/jti validation, reject expired/revoked tokens |
| **Unit** | Go | `ratelimit_test.go`: 5 req/min/IP → 6th returns 429 (auth endpoints) |
| **Unit** | TS | `authSlice.test.ts`: apiKey field removed, no secrets in Zustand |
| **Integration** | Go | `auth_integration_test.go`: Full middleware chain → handler → response for all RBAC tiers |
| **Integration** | Go | `secrets_test.go`: `gosecrets.Load()` succeeds in test env, fails gracefully when key missing |
| **E2E** | Playwright | Login flow with rate limiting (verify 429 response after 5 rapid attempts) |
| **E2E** | Playwright | API key creation/deletion requires admin session |

**Verification Gate**: 
- `go test -race -count=1 ./internal/middleware/ ./internal/auth/ ./internal/api/handler/` — ALL pass
- `npx vitest run src/store/` — ALL pass
- `go vet ./internal/middleware/ ./internal/auth/` — clean
- Manual: `curl -H "X-Aleph-Api-Key: ..." POST /api/v1/auth/apikeys` returns 401 without JWT session cookie

---

## Wave 1: Core Security — Injection & Sandbox (Days 3–5)

**Objective**: Eliminate code injection vectors and harden the sandbox to production-grade isolation.  
**Findings addressed**: I1-I6 (injection), S1-S10 (sandbox isolation)  
**Priority**: 🔴 CRITICAL

### Work Strategy

**Delegation**: Fire 3 parallel agents:
1. `backend-engineer` for SQL/DSN injection fixes (Go)
2. `backend-engineer` + `secure-code-guardian` for sandbox isolation (Go + Docker config)
3. `golang-pro` for CommandAllowlist tightening

Inject ← sandbox dependency: Injection fixes complete before sandbox tests run (sandbox code path uses validated inputs).

### Tasks

#### W1-1: SQL Injection Hardening
- [ ] `internal/repository/metadata.go`: Fix DSN double-escaping (constructing DSN with `fmt.Sprintf` before passing to sql.Open — should use parsed Config struct)
- [ ] `internal/repository/metadata.go` line 709-726: Validate table names in `DeleteProjectCascade` loop against hardcoded allowlist (no `fmt.Sprintf` with variable table names)
- [ ] `internal/repository/metadata.go` line 364: `fmt.Sprintf` for tool cache key — parameterize or validate
- [ ] `internal/storage/duckdb.go` line 159-170: Add `validName()` regex check on schema names passed to `scopeQuery`
- [ ] `internal/storage/context.go` line 32: `SanitizeProjectID` already validates — verify it's called in ALL code paths
- [ ] `internal/ingestion/engine.go` line 796-797: `resolveTableName(task.Id)` — validate task.Id with regex OR use UUID parsing
- [ ] `internal/ingestion/engine.go` line 808-809: `fmt.Sprintf` for temp file dir — use `os.MkdirTemp` prefix, not user input
- [ ] Audit ALL remaining `fmt.Sprintf` calls that construct SQL — parameterize or validate inputs

#### W1-2: DSL Injection
- [ ] `internal/dsl/compiler_tool.go`: Audit all template interpolation — no user input reaches SQL without parameterization
- [ ] `internal/dsl/parser.go`: Validate DSL input before parsing (length limits, character blacklisting)
- [ ] `internal/genesis/sandbox.go`: Convert `checkDangerousPatterns` from `strings.Contains` to Go AST parsing (`go/parser` + `go/ast`)

#### W1-3: Sandbox Isolation — Namespaces + Seccomp
- [ ] Replace `ExecSandbox` default implementation with namespace-isolated execution:
  - PID namespace (`CLONE_NEWPID`) — process sees only itself
  - Mount namespace (`CLONE_NEWNS`) — no access to host filesystem
  - Network namespace (`CLONE_NEWNET`) — no outbound connections
  - User namespace (`CLONE_NEWUSER`) — root inside = nobody outside
  - UTS namespace (`CLONE_NEWUTS`) — isolated hostname
  - IPC namespace (`CLONE_NEWIPC`) — no shared memory
- [ ] Add seccomp-bpf filter via `elastic/go-seccomp-bpf`:
  - Allow: `read`, `write`, `exit`, `exit_group`, `sigreturn`, `futex`, `brk`, `mmap`, `mprotect`
  - Block: `ptrace`, `mount`, `umount2`, `socket`, `connect`, `bind`, `init_module`, `finit_module`, `execveat`
  - Default-deny policy with explicit allowlist
  - Set `PR_SET_NO_NEW_PRIVS` before filter load
- [ ] Add cgroups v2 resource limits:
  - Memory: 256MB per execution
  - CPU: 0.5 core (50ms/100ms period)
  - PIDs: 32 max
  - 30-second CPU time limit
- [ ] Ensure `ExecSandbox` is NEVER used as fallback — if namespaces/seccomp unavailable, return error, don't execute

#### W1-4: ContainerSandbox Fallback Removal
- [ ] `internal/sandbox/container_sandbox.go` line 86-91: Remove fallback to ExecSandbox when Docker unavailable
- [ ] Instead: return `ErrContainerUnavailable` with clear instructions for enabling Docker/gVisor
- [ ] Add startup health check that verifies Docker daemon is running AND user has Docker permissions
- [ ] Document minimum Docker version and configuration required

#### W1-5: gVisor Integration (ContainerSandbox enhancement)
- [ ] Install `runsc` runtime binary in Docker image
- [ ] Add `HostConfig.Runtime = "runsc"` to `executeInContainer()` Docker config
- [ ] Verify gVisor runtime works with current Docker socket access
- [ ] Add startup check: `docker info` confirms `runtimes: runsc` present
- [ ] If gVisor missing, degrade to plain Docker (with namespaces/seccomp) → log warning

#### W1-6: Python Blocklist Expansion
- [ ] Add to `validation.go` Python blocklist:
  - `importlib` (module level + import)
  - `runpy` 
  - `pickle`
  - `shutil`
  - `os` (full module — block `import os` entirely)
  - `subprocess` (via any alias)
  - `code` (interactive interpreter)
  - `builtins` (access to `__import__`)
  - `compile` + `eval` + `exec` in any form
  - `open()` with network schemas (http, ftp, s3)
  - `requests`, `httpx`, `urllib3` (network clients — unless explicitly allowed per-tool)
  - `socket` via `from socket import *`
- [ ] Add `getattr`, `__getattribute__`, `__dict__`, `__class__` dynamic access detection
- [ ] Add AST-based validation as secondary check (not just regex)

#### W1-7: Go Blocklist Expansion
- [ ] Align `internal/sandbox/validation.go` (9 entries) with `internal/ingestion/engine.go` blocklist (45+ entries):
  - Add: `os`, `plugin`, `runtime`, `crypto/*`, `encoding/*`, `io` (to match ingestion)
  - Add: `debug/*`, `internal/*` packages
  - Add: `net/*` (all net subpackages)
  - Add: `mime/multipart`, `text/template` (code execution vectors)
  - Keep `sync`, `context`, `fmt`, `math`, `sort`, `strings`, `time`, `unicode` as safe defaults
- [ ] Use `go/parser` for import validation (not just string matching)

#### W1-8: CommandAllowlist Tightening
- [ ] Remove from allowlist: `curl`, `pip`, `python3`, `python`, `git`, `make`
- [ ] Replace with sandbox-safe alternatives:
  - File copying → Go `io.Copy` or `shutil` from sandbox code
  - Package installation → pre-approved pip packages list, installed at sandbox build time
  - Code execution → ONLY via sandbox runtimes (never raw process execution)
  - Git operations → Go `go-git` library (no shell out)
- [ ] Add argument whitelisting for remaining commands (e.g., `ls` only `-l`, `-a`, `-la`)
- [ ] Add output size limits (10MB max) for allowed commands

#### W1-9: runDynamic Hardening
- [ ] `internal/ingestion/engine.go` line 827: Change `os.Getenv("PATH")` → hardcoded `PATH=/usr/bin:/bin` (or empty PATH)
- [ ] Add network namespace isolation to `runDynamic` execution
- [ ] Add seccomp filter identical to sandbox
- [ ] Use `resolveTableName` regex validation (already exists — verify)

#### W1-10: Genesis Sandbox AST Conversion
- [ ] `internal/genesis/sandbox.go` line 114-133: Replace `checkDangerousPatterns` (strings.Contains) with:
  1. `go/parser.ParseFile` to get AST
  2. `go/ast.Walk` to visit all ImportSpec nodes
  3. Compare import paths against the expanded Go blocklist
  4. Check for `os.Remove`, `os.RemoveAll`, `os.Chmod` AST nodes
  5. Check for `plugin.Open`, `runtime.CPUProfile`, `net.Listen`, `net.Dial` calls
- [ ] Keep obfuscation detection (`detectObfuscation`) as secondary check

### Test Strategy — Wave 1

| Layer | Test Type | What |
|-------|-----------|------|
| **Unit** | Go | `validation_test.go` (573 existing): Expand with new blocklist entries, AST bypass test cases, importlib bypass attempt |
| **Unit** | Go | `duckdb_test.go`: SQL injection fuzz tests — 1000 random inputs against `SanitizeProjectID`, verify no bypass |
| **Unit** | Go | `sandbox/namespace_test.go` (NEW): Table-driven tests: Python/Go code that tries to escape namespace isolation — all must fail |
| **Unit** | Go | `sandbox/seccomp_test.go` (NEW): Test that blocked syscalls return `EPERM` (ptrace, mount, socket, init_module) |
| **Unit** | Go | `sandbox/limit_test.go` (NEW): Memory limit (256MB) exceeded → OOM kill; PID limit (32) exceeded → `EAGAIN` |
| **Unit** | Go | `genesis/sandbox_test.go` (NEW): AST validation correctly catches `os.Remove`, `plugin.Open`, `net.Listen` in parsed Go code |
| **Integration** | Go | `sandbox/escape_test.go` (NEW): Real Python scripts attempting 10 known sandbox escapes — all must be blocked |
| **Integration** | Go | `sandbox/firecracker_test.go` (optional, if Firecracker path implemented) |
| **Unit** | Go | `allowlist_test.go` (NEW): Verify `curl`, `pip`, `git`, `make` removed; verify argument whitelisting works |
| **Fuzz** | Go | `sandbox/fuzz_test.go` (573 existing + expand): Fuzz blocklist with malformed Python/Go code — no panics, no bypasses |

**Verification Gate**:
- `go test -race -count=1 ./internal/sandbox/ ./internal/genesis/ ./internal/ingestion/` — ALL pass (ZERO skipped)
- `go vet ./internal/sandbox/ ./internal/genesis/ ./internal/ingestion/` — clean
- Manual: Execute `try_escaping_sandbox.py` script that uses importlib → must be rejected
- Manual: Execute Go code importing `os/exec` → must be rejected
- Manual: Execute `echo "hello"` via ExecCommandContext → must be rejected (if echo not on new allowlist)

---

## Wave 2: Data Layer — Database Hardening (Days 6–8)

**Objective**: Fix concurrency, atomicity, and schema integrity across DuckDB and PostgreSQL.  
**Findings addressed**: D1-D12 (database), C1-C6 (caching)  
**Priority**: 🔴 CRITICAL (D1, D2, D3, D4, D6) / 🟡 HIGH (D5, D7, D8)

### Work Strategy

**Delegation**: Fire 2 parallel agents:
1. `backend-engineer` + `database-optimizer` for DuckDB concurrency rewrite
2. `backend-engineer` for PostgreSQL constraints + schema deduplication

Wait for both, then run integration/race tests.

### Tasks

#### W2-1: DuckDB Concurrency — Remove Global RWMutex for Reads
- [ ] `internal/storage/duckdb.go`: Replace `mu sync.RWMutex` pattern with:
  - Connection pool: `SetMaxOpenConns(runtime.NumCPU())`, `SetMaxIdleConns(runtime.NumCPU())`
  - Dedicated write connection: `writeConn *sql.Conn` (one connection for all writes)
  - `writeMu sync.Mutex` for serializing writes only (not reads)
  - `Query()` / `QueryRow()`: NO mutex — use `db.QueryContext()` with connection pool
  - `Exec()`: use `writeMu.Lock()` + `writeConn.ExecContext()`
  - `BeginTX()` (write): acquire `writeMu` + `writeConn.BeginTx()`
  - `BeginReadTX()`: NO mutex — use `db.BeginTx()` from pool
  - `QueryContext()` / `ExecContext()`: Deprecate semaphore pattern in favor of pool
- [ ] Benchmark: 100 concurrent reads → verify 2x+ improvement over RWMutex

#### W2-2: DuckDB TX Struct — Fix LocK Ordering
- [ ] `internal/storage/duckdb.go` TX struct: Fix `parentMu` lock ordering in `Commit()` and `Rollback()`
  - Current: `Commit()` holds `tx.mu` → acquires `parentMu` → releases `tx.mu` → releases `parentMu`
  - Fix: `tx.mu` is sufficient (TX is single-use); `parentMu` only for write connection sync
  - OR: Document ordering clearly if `parentMu` still needed for `BeginTX` coordination
- [ ] Ensure `Close()` method guards against nil `d.db` before calling `d.db.Close()` (W1-03 fix validation)

#### W2-3: MemoryStore Atomicity
- [ ] `internal/memory/memory.go` line 75-91: Wrap DELETE + INSERT in DuckDB transaction
- [ ] Use `m.db.BeginTx(ctx, nil)` instead of two separate `ExecContext` calls
- [ ] Verify transaction works with `FLOAT[768]` array columns (DUCKDB limitation: arrays in tx are fine, just not `INSERT OR REPLACE`)
- [ ] Switch from `db.ExecContext` (raw) to `storage.DuckDB.ExecContext` (proper wrapper with schema context)
- [ ] Add error handling: if INSERT fails after DELETE, ROLLBACK to preserve old entry

#### W2-4: DeleteProjectCascade Atomicity
- [ ] `internal/repository/metadata.go` line 740-753: Reverse order — PostgreSQL transaction FIRST, then DuckDB schema drop
- [ ] If PostgreSQL tx succeeds but DuckDB drop fails: log critical error, mark project for cleanup, DON'T leave inconsistent state
- [ ] Add `DeleteProjectCascade` to a deferred cleanup queue (background goroutine retries DuckDB drops)
- [ ] Add idempotency: if DuckDB schema already dropped, don't fail

#### W2-5: DuckDBRegistry Concurrency Protection
- [ ] `internal/registry/duckdb_registry.go`: Add `mu sync.RWMutex` to `DuckDBRegistry`
- [ ] All read methods (`GetComponentByID`, `ListComponents`) acquire `RLock()`
- [ ] `RegisterComponent` acquires `Lock()`
- [ ] `UpdateComponentStatus` acquires `Lock()`
- [ ] OR: Migrate to use `*storage.DuckDB` wrapper which already has concurrency protection
- [ ] Add duplicate `id` detection in `RegisterComponent` (return error, don't silently overwrite)

#### W2-6: QueryRowContext Silent Failure
- [ ] `internal/storage/duckdb.go` line 159-170: Audit ALL callers of `QueryRowContext`:
  - `registry/duckdb_registry.go` line 110 → add nil check OR use `QueryRowContextOrError`
  - `repository/metadata.go` lines 99, 137, 222, 482, 578, 646 → add nil checks
  - Any caller doing `row.Scan(&dest)` on nil → add guard
- [ ] Option: Make `QueryRowContext` NEVER return nil — allocate a dummy `*sql.Row` that errors on Scan, OR change signature to `(*sql.Row, error)`

#### W2-7: PostgreSQL Schema Hardening
- [ ] Add NOT NULL constraints to all `project_id` columns across all `system_*` tables
- [ ] Add FOREIGN KEY constraints:
  - `system_agents.project_id → system_projects.id`
  - `system_skills.project_id → system_projects.id`
  - `system_tasks.project_id → system_projects.id`
  - `system_chat_history.agent_id → system_agents.id`
  - `system_chat_sessions.project_id → system_projects.id`
- [ ] Add indexes on:
  - `system_agents(project_id, status)`
  - `system_skills(project_id)`
  - `system_tasks(project_id, status)`
  - `system_chat_history(agent_id, created_at)`
  - `system_api_keys(project_id)`
- [ ] Add migration `postgres/000009_add_constraints.up.sql` with rollback

#### W2-8: Schema Deduplication
- [ ] Remove `createTableSQL` constant from `duckdb_registry.go` (line 48-74 — already commented out at line 79-80)
- [ ] Remove `migrations/000001_init_schema.up.sql` (monolithic legacy migration)
- [ ] Remove duplicate `system_tools` definition from `duckdb/000001_init.up.sql` (keep only in `duckdb/000004_system_tables.up.sql`)
- [ ] Ensure `duckdb/000004_system_tables.up.sql` uses `TEXT` (matching PostgreSQL) or document `VARCHAR` choice
- [ ] Add `docs/SCHEMA.md` documenting single source of truth per table

#### W2-9: ToolCache Bounding
- [ ] `internal/repository/cache.go`: Add `maxSize int` (default: 500) to `ToolCache`
- [ ] In `Set()`: if `len(cache) >= maxSize`, evict LRU entry before inserting
- [ ] Add background cleanup goroutine (every 5 minutes) to evict expired entries
- [ ] Option: Replace `ToolCache` with `embeddingCache` LRU pattern from `memory/embed.go` (with O(n) → O(1) improvement using linked list)

#### W2-10: Backup & Recovery Fixes
- [ ] `internal/storage/duckdb_backup.go` line 36: Use `BeginReadTX` instead of `mu.Lock()` (backup should not block reads)
- [ ] `internal/storage/duckdb_backup.go` line 139: Replace `PRAGMA integrity_check` (SQLite) with DuckDB equivalent (`CHECKPOINT` + verify file size)
- [ ] `internal/storage/duckdb_backup.go` line 143: Replace `sqlite_master` query with `information_schema.tables`
- [ ] Add backup verification: after backup, open in read-only connection and run `SELECT COUNT(*) FROM main_views`
- [ ] Add `context.Context` propagation to `ExportDatabase` (line 74 uses `context.Background()`)

### Test Strategy — Wave 2

| Layer | Test Type | What |
|-------|-----------|------|
| **Unit** | Go | `duckdb_concurrent_test.go` (NEW): 100 goroutines → 50 read + 50 write → verify reads never block, writes are serialized correctly |
| **Unit** | Go | `duckdb_tx_test.go` (NEW): `BeginTX` → INSERT → `BeginTX` (concurrent) → verify second blocks until first commits |
| **Unit** | Go | `memory_atomic_test.go` (NEW): 10 goroutines doing UPDATE (DELETE+INSERT) to same key → verify all succeed, last write wins atomically |
| **Unit** | Go | `registry_concurrent_test.go` (NEW): 100 concurrent `RegisterComponent` → verify no duplicates, no panics, mutex holds |
| **Integration** | Go | `project_cascade_test.go` (NEW): Create project with agents/skills/tasks → DeleteProjectCascade → verify all DuckDB schemas + PG rows gone |
| **Integration** | Go | `backup_test.go` (expand existing): Backup during concurrent reads → verify backup is consistent, reads don't fail |
| **Unit** | Go | `cache_bounded_test.go` (NEW): Add 600 entries to ToolCache(max=500) → verify oldest 100 evicted |

**Verification Gate**:
- `go test -race -count=3 ./internal/storage/ ./internal/repository/ ./internal/memory/ ./internal/registry/` — ALL pass, ZERO race detector warnings
- `go test -bench=. ./internal/storage/` — verify concurrency benchmark improvement
- PostgresSQL: `psql -c "\d+ system_agents"` shows NOT NULL + FK constraints + indexes

---

## Wave 3: External Surface — API & Frontend (Days 9–12)

**Objective**: Harden the API security perimeter and eliminate frontend anti-patterns.  
**Findings addressed**: A1-A10 (API), F1-F10 (frontend), L2-L4 (data leakage)  
**Priority**: 🟡 HIGH

### Work Strategy

**Delegation**: Fire 3 parallel agents:
1. `backend-engineer` for CSP/HSTS/CSRF middleware (Go)
2. `frontend-engineer` for Zustand cleanup, form fixes, accessibility (React/TS)
3. `frontend-engineer` for test fixes (vitest)

Frontend agents are independent of each other. Backend agent must complete before frontend CSP changes can be verified.

### Tasks

#### W3-1: CSP Hardening
- [ ] `internal/middleware/security.go` line 17: Remove `'unsafe-inline'` from `style-src`
- [ ] Install `vite-plugin-csp-guard` for hash-based CSP generation
- [ ] `vite.config.ts`: Add strict CSP policy:
  ```ts
  defaultSrc: [self],
  scriptSrc: [self],  // No unsafe-inline
  styleSrc: [self],   // No unsafe-inline (use CSS modules or extracted .css)
  connectSrc: [self],  // Remove hardcoded ws://localhost:*
  frameAncestors: ["'none'"],
  objectSrc: ["'none'"],
  workerSrc: [self],
  ```
- [ ] `internal/middleware/security.go` line 20: Remove hardcoded `ws://localhost:8080 ws://localhost:5173` from `connect-src`
- [ ] Add `upgrade-insecure-requests` directive
- [ ] Add `block-all-mixed-content` directive

#### W3-2: HSTS + Security Headers
- [ ] Install `github.com/unrolled/secure` middleware
- [ ] Configure:
  - `SSLRedirect: true`
  - `STSSeconds: 31536000` (1 year)
  - `STSIncludeSubdomains: true`
  - `STSPreload: true`
  - `FrameDeny: true` (replaces `frame-ancestors 'none'`)
  - `ContentTypeNosniff: true`
  - `BrowserXssFilter: true`
  - `ReferrerPolicy: "strict-origin-when-cross-origin"`
  - `PermissionsPolicy: "geolocation=(), microphone=(), camera=()"`
  - `IsDevelopment: false` (set from config)
- [ ] Wire into HTTP middleware chain BEFORE CSP middleware

#### W3-3: CSRF Hardening
- [ ] `internal/middleware/csrf.go` line 28-30: Change no-Origin/no-Referer behavior:
  - Keep allow for GET/HEAD/OPTIONS
  - For POST/PUT/DELETE: REJECT if no Origin AND no Referer (require one)
  - OR: Require custom header (`X-Aleph-CSRF-Token`) for mutating requests from browser
- [ ] `internal/middleware/csrf.go` line 43: Replace `strings.HasPrefix` with exact origin matching for Referer
- [ ] Add CSRF token pattern:
  - Server sets `SameSite=Strict` cookie with random token
  - Client reads cookie and sends in `X-CSRF-Token` header
  - Middleware validates token matches cookie

#### W3-4: Remove window.__ALEPH_STORE__ from Production
- [ ] `frontend/src/App.tsx` line 26: Wrap in `import.meta.env.DEV` check
- [ ] `frontend/src/store/useStore.ts` line 74-76: Wrap in `import.meta.env.DEV` check
- [ ] Both gated behind `__ALEPH_DEV_TOOLS__` feature flag (opt-in)
- [ ] Remove `apiKey` field from `authSlice` (W0-2 already done — verify)

#### W3-5: Fix AbortController Gaps
- [ ] `frontend/src/App.tsx` line 50: Add AbortController to `listProjects` effect
- [ ] `frontend/src/App.tsx` line 56-73: Add AbortController to `executeQuery` + `getDataStats` effect
- [ ] `frontend/src/App.tsx` line 76-90: Add AbortController to `getChatHistory` effect
- [ ] Cleanup: abort all 3 controllers in useEffect return function

#### W3-6: Fix Chat Streaming Memory
- [ ] `frontend/src/hooks/useAppActions.ts` line 205-207: Replace `store.setChat([...chat])` with:
  - Immutable update via `immer` (`store.setChat(produce(chat, draft => { draft.push(token) }))`)
  - OR direct mutation + re-render trigger (if Zustand supports it)
  - OR `store.getState().chat.push(token); store.setState({})` (force re-render)

#### W3-7: Replace useStore.getState() in Render Paths
- [ ] Audit 136 call sites across 29 files
- [ ] Prioritize component render paths (not hook actions)
- [ ] Replace `useStore.getState().field` with `useStore(s => s.field)` subscriptions
- [ ] Add `React.memo` to components that only need shallow comparison
- [ ] Add `useCallback` to event handlers passed as props

#### W3-8: Fix Form Accessibility
- [ ] Convert all 5 SlideOver forms from `<button onClick>` → `<form onSubmit>`:
  - `AgentFormSlideOver.tsx`
  - `ToolFormSlideOver.tsx`
  - `SkillFormSlideOver.tsx`
  - `DataSourceFormSlideOver.tsx`
  - `ComponentFormSlideOver.tsx`
- [ ] Add `<button type="submit">` inside `<form>`
- [ ] Add native HTML5 validation attributes (`required`, `minlength`, `pattern`)
- [ ] Add `aria-describedby` for error messages
- [ ] Add Enter key submission support

#### W3-9: Fix SlideOverPanel Accessibility
- [ ] Add backdrop/overlay click-to-close handler
- [ ] Add `inert` attribute to background elements (modern alternative to `aria-hidden`)
- [ ] Add `aria-describedby` pointing to panel content
- [ ] Add `aria-live="polite"` announcement on open/close
- [ ] Replace emoji icon `⛶` with proper SVG icon (with `aria-label`)

#### W3-10: Remove console.error from Production
- [ ] Create `frontend/src/lib/errorReporter.ts` — structured error service:
  ```ts
  export const reportError = (context: string, error: unknown) => {
    if (import.meta.env.DEV) {
      console.error(`[${context}]`, error)
    }
    // In production: send to error tracking service OR toast only
    useStore.getState().addToast({ type: 'error', message: t('errors.default') })
  }
  ```
- [ ] Replace all 7 `console.error` sites:
  - `ToolIntelligenceView.tsx:34`
  - `AlephErrorBoundary.tsx:22` (keep for dev, gate for prod)
  - `ToolConfigPanel.tsx:18`
  - `OracleView.tsx:72`
  - `DashboardView.tsx:60`
  - `InlineErrorBoundary.tsx:21`
  - `useCursorPagination.ts:38`

#### W3-11: Fix Vitest Failures (9 tests in 4 files)
- [ ] Fix mock `useStore.getState()` to return full `AppState` shape (not `{}`)
- [ ] `useAgentActions.test.ts` (3 failures): Add `agents`, `pendingCrud`, `setPendingCrud` to mock
- [ ] `useToolActions.test.ts` (2 failures): Add `setPendingCrud`, `tools`, `pendingCrud` to mock
- [ ] Verify all 216 tests pass with `npx vitest run`

### Test Strategy — Wave 3

| Layer | Test Type | What |
|-------|-----------|------|
| **Unit** | Go | `security_test.go` (existing): Expand CSP header validation, HSTS header presence, CSRF token flow |
| **Unit** | Go | `csrf_test.go` (expand existing): No-Origin POST → 403, valid Origin POST → 200, HasPrefix spoofing → 403 |
| **Unit** | TS | `authSlice.test.ts`: apiKey removed (verified), no __ALEPH_STORE__ in prod build |
| **Unit** | TS | `useAgentActions.test.ts`: All 9 previously-failing tests now pass |
| **Unit** | TS | `useToolActions.test.ts`: All tests pass |
| **Integration** | TS | `App.test.tsx`: Verify AbortController cleanup on unmount (no fetch after component gone) |
| **E2E** | Playwright | `forms.spec.ts` (NEW): Tab through all 5 forms, verify Enter submits, verify HTML5 validation |
| **E2E** | Playwright | `accessibility.spec.ts` (NEW): Tab through SlideOverPanel, verify focus trap, backdrop click, Escape close |
| **E2E** | Playwright | `csp.spec.ts` (NEW): Verify unsafe-inline is NOT in CSP header; verify frame-ancestors 'none' |
| **Security** | Manual | `git grep "console.error" frontend/src/` → only dev-gated or error boundary |
| **Security** | Manual | `git grep "__ALEPH_STORE__" frontend/src/` → only in dev-gated code |

**Verification Gate**:
- `npx vitest run` → ALL 216+ tests pass, ZERO failures
- `npx tsc --noEmit` → 0 errors
- `npx vite build` → succeeds, CSP headers in index.html have no unsafe-inline
- Manual: Open production build → `window.__ALEPH_STORE__` is `undefined`
- Manual: `curl -I https://localhost:8443` → `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`

---

## Wave 4: Reliability — Concurrency & Infrastructure (Days 13–15)

**Objective**: Fix goroutine leaks, race conditions, and harden deployment.  
**Findings addressed**: G1-G6 (concurrency), INF2-INF13 (infrastructure), Q1-Q12 (code quality)  
**Priority**: 🟡 HIGH (concurrency) / 🟢 MEDIUM (infra, polish)

### Work Strategy

**Delegation**: Fire 2 parallel agents:
1. `backend-engineer` + `golang-pro` for concurrency fixes (goroutine leaks, context deadlines)
2. `devops-engineer` for Docker/CI/CD/nginx hardening

Code quality fixes (context.Background, json.Unmarshal) can run as background grooming in any wave. Prioritize concurrency race fixes first.

### Tasks

#### W4-1: Goroutine Leak Fixes
- [ ] `internal/api/handler/chat_session.go`: Verify `Chat()` goroutine lifecycle — all paths (success/error/timeout) close channels and call `wg.Done()`
- [ ] `internal/mcp/discovery.go`: MCP discovery goroutine — add context cancellation, don't silently fail
- [ ] `internal/sandbox/dev_mode.go` line 115: File watcher goroutine — add `ctx.Done()` check in select loop
- [ ] `internal/middleware/ratelimit.go` line 55-74: Cleanup goroutine — verify `ticker.Stop()` on shutdown
- [ ] `internal/storage/duckdb_backup.go` line 312-349: AutoBackup goroutine — add `ctx.Done()` for graceful shutdown
- [ ] Audit: `grep -rn "go func" internal/` → all goroutines have defer/recover, context cancellation, and known lifetime

#### W4-2: Context.Background() → Context Propagation
- [ ] `internal/storage/duckdb_backup.go` line 74: `ExportDatabase` uses `context.Background()` → pass ctx from caller
- [ ] `internal/registry/duckdb_registry.go`: `RegisterComponent`, `UpdateComponentStatus`, `ListComponents` — accept ctx parameter
- [ ] `internal/repository/metadata.go`: All methods accepting context should propagate (most already do — verify)
- [ ] `internal/api/handler/nlp.go`: `UpdateTrustScore` already takes ctx — verify no child goroutine uses `context.Background()`
- [ ] `internal/app/app.go`: Startup code — use `context.WithTimeout` for initialization (30s) instead of Background

#### W4-3: json.Unmarshal Error Handling
- [ ] Audit `grep -rn "json.Unmarshal\|json.NewDecoder" internal/ | grep -v "_test.go"` → all sites check and propagate errors
- [ ] `internal/sandbox/validation.go`: JSON parsing for tool metadata — add error return (currently some paths swallow)
- [ ] `internal/dsl/compiler_tool.go`: JSON template parsing — add error return
- [ ] No `_ = json.Unmarshal` or `_ = json.NewDecoder.Decode` anywhere

#### W4-4: Race on nextID
- [ ] `internal/repository/metadata.go` or wherever `nextID` is generated: Replace `int` + `++` with `atomic.AddInt64` or UUID
- [ ] If using sequential IDs: use PostgreSQL `SERIAL` or `BIGSERIAL` instead of application-level counter
- [ ] If using UUIDs: ensure `uuid.New()` (not deterministic, not counter)

#### W4-5: Context Deadline Enforcement
- [ ] Audit all `exec.CommandContext(ctx, ...)` calls — verify ctx is NOT `context.Background()`
- [ ] Add deadlines to all network calls (HTTP, TCP, Docker API)
- [ ] `internal/sandbox/exec_sandbox.go`: Already has 30s timeout via `execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)` — verify

#### W4-6: Docker Image Optimization
- [ ] Multi-stage build: Go binary → `scratch` (or `alpine:3.20` minimal)
- [ ] Remove build tools (gcc, git, make) from final image
- [ ] Add `.dockerignore` (already exists — `docs/reports/` says "already done" in W7 — verify)
- [ ] Add healthcheck: `HEALTHCHECK --interval=30s CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1`
- [ ] Reduce image size target: < 50MB (Go binary) + < 30MB (Docker layer)

#### W4-7: Nginx TLS + Reverse Proxy
- [ ] Add TLS termination (cert-manager or nginx config)
- [ ] Add `ssl_certificate` + `ssl_certificate_key` directives
- [ ] Redirect HTTP → HTTPS (301)
- [ ] Add rate limiting in nginx: `limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/m`
- [ ] Add `proxy_set_header X-Forwarded-Proto https` for backend
- [ ] Document in `docs/DEPLOY.md`

#### W4-8: PostgreSQL and Ollama Exposure
- [ ] PostgreSQL: bind to `127.0.0.1` only if accessed locally; `172.x.x.x` for Docker network (internal); never `0.0.0.0`
- [ ] PostgreSQL: require `pg_hba.conf` to use `scram-sha-256` (not `trust` or `md5`)
- [ ] Ollama: bind to `127.0.0.1` only (never `0.0.0.0`)
- [ ] Ollama: add API key authentication via reverse proxy (see Ollama docs for `OLLAMA_ORIGINS`)
- [ ] Add firewall rules (iptables/ufw) to deny external access to DB and Ollama ports

#### W4-9: Rollback Strategy
- [ ] Add `migrations/duckdb/*.down.sql` rollback for all up migrations (verify existing)
- [ ] Add `migrations/postgres/*.down.sql` rollback for all up migrations (verify existing)
- [ ] Document rollback procedure in `docs/DEPLOY.md`: which migration to revert to, how to rollback DuckDB (file copy) vs PostgreSQL (down migration)
- [ ] Add pre-deploy backup step in CI/CD

#### W4-10: Code Quality Grooming
- [ ] Remove dead code: `circuitbreaker.go` (standalone, not wired) — either wire it or remove it
- [ ] Remove dead code: `internal/tools/{finance,osint,humanecosystems,adaptation}` stub trees — either implement or remove
- [ ] Standardize error wrapping: use `fmt.Errorf("context: %w", err)` consistently (some files use `errors.Wrap`, some use `fmt.Errorf`)
- [ ] Add `nolint` comments with justification where lint rules are intentionally violated
- [ ] Fix ALL `go vet` warnings (except previously deferred PEG struct tags in `dsl/ast.go`)
- [ ] Add `golangci-lint` to CI (start with `gofmt`, `govet`, `errcheck`, `staticcheck`, `unused`)

### Test Strategy — Wave 4

| Layer | Test Type | What |
|-------|-----------|------|
| **Unit** | Go | `goroutine_leak_test.go` (NEW): Start/stop app 10 times, verify goroutine count returns to baseline |
| **Unit** | Go | `context_propagation_test.go` (NEW): Verify no `context.Background()` in handler code paths (static analysis via `grep` + test assertion) |
| **Unit** | Go | `race_nextid_test.go` (NEW): 100 goroutines incrementing nextID → verify no data race (with `-race` flag) |
| **Integration** | Go | `shutdown_test.go` (NEW): SIGTERM → app.Close() → all goroutines exit within 5 seconds |
| **Integration** | Docker | `docker compose config` → validates TLS certs, PostgreSQL binding, Ollama binding |
| **E2E** | Playwright | Smoke test: login → create agent → send message → verify response (full user journey) |

**Verification Gate**:
- `go test -race -count=3 ./...` → ALL pass, ZERO race detector warnings
- `go test -run TestShutdown ./internal/app/` → goroutine count returns to baseline
- `grep -rn "context.Background()" internal/ --include="*.go" | grep -v "_test.go" | grep -v "app.go"` → fewer than 5 remaining (justified with comments)
- `docker compose config` → no ports bound to `0.0.0.0` except 80/443
- `npx tsc --noEmit` → 0 errors (test files excluded from strict check)

---

## Dependency Map

```
W0 (Secrets & Auth)
 ├─→ W0-1 (Secrets)      ──── independent
 ├─→ W0-2 (Frontend Keys) ──── depends on W0-1
 ├─→ W0-3 (RBAC)          ──── independent
 ├─→ W0-4 (authSkipSet)   ──── independent
 ├─→ W0-5 (JWT)           ──── independent
 ├─→ W0-6 (Rate Limiting) ──── independent
 ├─→ W0-7 (Subprocess Creds) ─ depends on W0-1
 └─→ W0-8 (SSE)           ──── depends on W0-3 (auth chain)
      │
      ▼
W1 (Injection & Sandbox)
 ├─→ W1-1 (SQL Injection)  ──── independent
 ├─→ W1-2 (DSL Injection)  ──── independent
 ├─→ W1-3 (Namespace+Seccomp) ─ independent (new sandbox code)
 ├─→ W1-4 (ContainerFallback) ─ depends on W1-3
 ├─→ W1-5 (gVisor)         ──── depends on W1-4
 ├─→ W1-6 (Python Blocklist) ── independent
 ├─→ W1-7 (Go Blocklist)   ──── independent
 ├─→ W1-8 (CommandAllowlist) ── depends on W1-3 (sandbox exec model)
 ├─→ W1-9 (runDynamic)      ──── depends on W1-3 (isolation pattern)
 └─→ W1-10 (Genesis AST)    ──── independent
      │
      ▼
W2 (Database)
 ├─→ W2-1 (Concurrency)     ──── independent (rewrite)
 ├─→ W2-2 (TX Lock Ordering) ─── depends on W2-1
 ├─→ W2-3 (MemoryStore)     ──── depends on W2-1 (using proper wrapper)
 ├─→ W2-4 (DeleteCascade)   ──── independent
 ├─→ W2-5 (Registry)        ──── depends on W2-1 (concurrency pattern)
 ├─→ W2-6 (QueryRowContext)  ──── independent
 ├─→ W2-7 (PG Constraints)  ──── independent (new migrations)
 ├─→ W2-8 (Schema Dedup)    ──── independent
 ├─→ W2-9 (ToolCache)       ──── independent
 └─→ W2-10 (Backup)         ──── independent
      │
      ▼
W3 (API & Frontend)
 ├─→ W3-1 (CSP)             ──── depends on W0-2 (no secrets in client)
 ├─→ W3-2 (HSTS+Headers)    ──── independent
 ├─→ W3-3 (CSRF)            ──── depends on W0-5 (JWT hardened, session cookies SameSite)
 ├─→ W3-4 (Remove __STORE__)─ depends on W0-2 (apiKey already removed)
 ├─→ W3-5 (AbortController)  ─── independent
 ├─→ W3-6 (Chat Streaming)   ─── independent
 ├─→ W3-7 (getState audit)   ─── independent (large refactor)
 ├─→ W3-8 (Form Accessibility)─ ─ independent
 ├─→ W3-9 (SlideOver accessibility)─ independent
 ├─→ W3-10 (console.error)   ──── independent
 └─→ W3-11 (Vitest Fixes)    ──── independent
      │
      ▼
W4 (Concurrency & Infra)
 ├─→ W4-1 (Goroutine Leaks)  ──── independent
 ├─→ W4-2 (Context Propagation)─ independent
 ├─→ W4-3 (json.Unmarshal)   ──── independent
 ├─→ W4-4 (nextID Race)      ──── independent
 ├─→ W4-5 (Context Deadline)  ──── independent
 ├─→ W4-6 (Docker Optimize)  ──── independent
 ├─→ W4-7 (Nginx TLS)        ──── independent
 ├─→ W4-8 (DB/Ollama Exposure)─ ─ independent
 ├─→ W4-9 (Rollback Strategy)─ ─ independent
 └─→ W4-10 (Code Quality)    ──── independent (aggregate of all wave fixes)
```

---

## Per-Wave Test Coverage Matrix

| Wave | Unit Tests | Integration Tests | E2E Tests | Fuzz Tests | Static Analysis | Manual |
|------|-----------|-------------------|-----------|------------|-----------------|--------|
| **W0** | 15+ new | 3 new | 2 new | 0 | go vet, golangci-lint | curl auth checks |
| **W1** | 20+ new | 4 new | 0 | 1000+ fuzz cases | go vet, AST validation | Python escape attempts |
| **W2** | 15+ new | 3 new | 0 | 0 | go vet, -race ×3 | psql schema verification |
| **W3** | 10 new (TS: 9 fixed + 5 new) | 2 new | 3 new | 0 | tsc --noEmit, CSP headers | window.__STORE__ |
| **W4** | 8 new | 2 new | 1 new | 0 | golangci-lint, grep audits | docker compose config |
| **TOTAL** | ~73 new tests | ~14 new tests | ~6 new E2E | ~1000 fuzz | 5 tools | ~8 manual checks |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| W0 auth changes break existing API integrations | Medium | High | Run full E2E suite before wave passes; provide migration guide for API consumers |
| W1 sandbox isolation breaks legitimate tool execution | Medium | High | Add development mode flag (opt-in namespace isolation); test with all existing tools |
| W2 concurrency rewrite introduces deadlocks | Low | Critical | -race ×3 runs; dedicated deadlock stress test (100 concurrent operations × 100 iterations) |
| W3 CSP strict mode breaks inline styles | Medium | Medium | Migrate to CSS modules; vite-plugin-csp-guard auto-generates hashes for known scripts |
| W3 form refactoring introduces submission bugs | Low | Low | Playwright E2E tests for all 5 forms cover submit/cancel/validation flows |
| W4 goroutine leak fixes accidentally close essential goroutines | Low | High | Graceful shutdown test; 30-day soak test in staging |
| Schema migration rollbacks fail | Low | High | Always test .down.sql on staging first; keep pre-migration backup |
| Production deployment during remediation | N/A | N/A | ALL waves deployed to staging first; wave gates prevent wave N before wave N-1 verified |

---

## Acceptance Criteria

### Per Wave

| Wave | Gate |
|------|------|
| **W0** | `go test -race -count=1 ./internal/middleware/ ./internal/auth/ ./internal/api/handler/` ALL pass; `npx vitest run src/store/` ALL pass; manual curl shows 401 for unauthenticated protected endpoints |
| **W1** | `go test -race -count=1 ./internal/sandbox/ ./internal/genesis/ ./internal/ingestion/` ALL pass; manual sandbox escape attempts all fail; Python `importlib` bypass blocked |
| **W2** | `go test -race -count=3 ./internal/storage/ ./internal/repository/ ./internal/memory/ ./internal/registry/` ALL pass, ZERO race warnings; PostgreSQL constraints verified |
| **W3** | `npx vitest run` ALL 216+ pass; `npx tsc --noEmit` 0 errors; `npx vite build` succeeds; CSP has no `unsafe-inline`; Playwright E2E ALL pass |
| **W4** | `go test -race -count=3 ./...` ALL pass; `docker compose config` secure; `grep context.Background` only in startup/shutdown |

### Final

- [ ] All 22 CRITICAL findings addressed (fixed or justified exception)
- [ ] All 38 HIGH findings addressed
- [ ] Build: `go build ./...` ✅ | `go vet ./...` ✅ | `npx tsc --noEmit` ✅ | `npx vite build` ✅
- [ ] Test: `go test -race -count=1 ./...` ALL pass | `npx vitest run` ALL pass
- [ ] Playwright E2E: All smoke tests pass
- [ ] Security headers: CSP strict mode, HSTS 1yr, CSRF token enforced
- [ ] Sandbox: gVisor or namespace+seccomp isolation enforced; zero-isolation ExecSandbox deprecated
- [ ] Database: Concurrent reads not blocked by writes; race detector clean
- [ ] Frontend: Zero `useStore.getState()` in render paths; zero `console.error` in production; all forms use `<form onSubmit>`

---

## Specification Files

Each wave has a corresponding specification document in `docs/specs/`:

```
docs/specs/
├── wave0-auth-spec.md       — RBAC matrix, JWT claims, auth middleware chain
├── wave0-secrets-spec.md    — gosecrets config, subprocess credential contract
├── wave1-sandbox-spec.md    — Sandbox isolation matrix, blocklists (Go + Python), seccomp profile
├── wave1-injection-spec.md  — SQL injection vectors fixed, DSL validation rules
├── wave2-database-spec.md   — DuckDB concurrency model, TX patterns, PostgreSQL constraints
├── wave3-api-spec.md        — CSP policy, HSTS config, CSRF token flow
├── wave3-frontend-spec.md   — Zustand store contract, form accessibility, error reporting API
├── wave4-concurrency-spec.md— Goroutine lifecycle, context propagation rules
└── wave4-infra-spec.md      — Docker security, nginx config, rollback strategy
```

### Cross-Reference Map

| Spec | Plan Reference | Depends On |
|------|---------------|------------|
| `docs/specs/wave0-auth-spec.md` | Wave 0 tasks W0-3, W0-4, W0-5, W0-6, W0-8 | — |
| `docs/specs/wave0-secrets-spec.md` | Wave 0 tasks W0-1, W0-2, W0-7 | — |
| `docs/specs/wave1-sandbox-spec.md` | Wave 1 tasks W1-3, W1-4, W1-5, W1-6, W1-7, W1-8 | W0 auth |
| `docs/specs/wave1-injection-spec.md` | Wave 1 tasks W1-1, W1-2, W1-9 | W0 auth |
| `docs/specs/wave2-database-spec.md` | Wave 2 tasks W2-1 through W2-10 | W0 auth |
| `docs/specs/wave3-api-spec.md` | Wave 3 tasks W3-1, W3-2, W3-3 | W0 secrets + auth |
| `docs/specs/wave3-frontend-spec.md` | Wave 3 tasks W3-4 through W3-11 | W0 secrets |
| `docs/specs/wave4-concurrency-spec.md` | Wave 4 tasks W4-1 through W4-5 | W1-W3 context propagation |
| `docs/specs/wave4-infra-spec.md` | Wave 4 tasks W4-6 through W4-10 | All above |

**See also**: `docs/reports/report-1.md` (source audit, 111 findings)
