# Aleph-v2 Debug Analysis — Known Issues

## CRITICAL (Build Failures)

### 1. Unused imports in `internal/auth/auth_service.go`
- Imports `crypto/rand`, `encoding/base64`, `fmt`, `golang.org/x/crypto/bcrypt` are all unused
- This prevents `go build`, `go vet`, and `go test` from succeeding
- The `AuthService` is a stub with mock validation, but imports suggest real auth was planned

### 2. Auth middleware references non-existent `AuthService` 
- `internal/middleware/auth_middleware.go:12` calls `NewAuthMiddleware(authService *auth.AuthService)`
- BUT `internal/auth/auth_service.go` exports `NewAuthService()` not `NewAuthMiddleware()`
- The middleware uses `authService.ValidateAPIKey()` which exists, but the constructor name mismatch means it can't be instantiated properly

### 3. `app.go:88` uses `middleware.NewAuthInterceptor()` which doesn't exist
- `internal/middleware/auth_middleware.go` defines `NewAuthMiddleware()`, NOT `NewAuthInterceptor()`
- `app.go:88` calls `middleware.NewAuthInterceptor(a.metaRepo)` — this function doesn't exist
- The middleware takes `*auth.AuthService` as param, not `*repository.MetadataRepository`

## HIGH (Architectural / Runtime Issues)

### 4. PostgreSQL hard dependency with no Docker service
- `app.go:58` calls `storage.NewPostgres(cfg.PostgresDSN)` which does `db.Ping()`
- Default DSN: `postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable`
- `docker-compose.yml` has NO PostgreSQL service — only alpine for DuckDB volume
- The app FAILS TO START without a running PostgreSQL instance
- Only `cmd/aleph-server/` (Docker entry) works without PG

### 5. Two divergent entry points with different service sets
- `main.go` (root) — Full app with 10 Connect services, needs PG
- `cmd/aleph-server/main.go` — Docker-only with 3 services (Registry, Sandbox, Project), DuckDB only
- No shared initialization logic; completely different architectures
- Docker Compose uses `cmd/aleph-server/Dockerfile` but the README says `http://localhost:5173` (which is the `main.go` port)

### 6. DuckDB VSS extension may fail silently
- `storage/duckdb.go:35` runs `INSTALL vss; LOAD vss;` but ignores errors
- VSS requires specific DuckDB builds; if unavailable, vector search features break silently

### 7. Circuit Breaker for NLP has broken `StreamPredictions` signature
- `breaker.go:32` returns `(*connect.ServerStreamForClient[...], error)` — this is a **client** stream type
- But `nlp.go:43` calls `h.nlpClient.StreamPredictions(ctx, req)` which expects a **server stream** for proxying
- The Circuit Breaker wraps a client but `StreamPredictions` in the handler is called on the server side with `*connect.ServerStream` — incompatible types

### 8. `cmd/aleph-server/main.go` Dockerfile references `go:1.25-bookworm`
- Go 1.25 doesn't exist yet as a Docker image (current stable is 1.22/1.23)
- Docker build will fail

## MEDIUM (Logic Bugs)

### 9. SQL injection in `ingestion/engine.go:78`
- `fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN...", task.Id)` — task.Id is user-controlled
- Double-quote escaping is insufficient; no sanitization

### 10. SQL injection in `query/handler.go:161`
- Direct string formatting of column/table names from schema into SQL queries
- `GetDataStats` and `EmergeOntology` build SQL from schema data without parameterization

### 11. `EmergeOntology` has double `rows.Close()` (project.go:185,188)
- `rows.Close()` called twice in the loop — harmless but indicates copy-paste bug

### 12. Auth mock is dangerously permissive
- `auth_service.go:16` — any string longer than 6 chars is valid
- No actual key verification, no project isolation

### 13. `GetChatHistory` queries PostgreSQL with `?` placeholders
- But other handlers (ingestion, agent) use `$1` PostgreSQL positional params
- `?` works with DuckDB but NOT with PostgreSQL/pgx

### 14. NLP sidecar `main.py:53` references `time.time()` without importing `time`

### 15. NLP sidecar proto files are manually written (not generated)
- `nlp/nlp_pb2.py` and `nlp/nlp_pb2_grpc.py` are hand-crafted, not from `protoc`
- May be out of sync with Go proto definitions

## LOW (Code Quality / Missing Features)

### 16. `internal/workflow/` directory is empty
### 17. `internal/service/library/` directory is empty
### 18. `internal/api/middleware/` directory exists but is unused (different from `internal/middleware/`)
### 19. Registry service handlers `GetComponent` and `UpdateComponentStatus` return `CodeUnimplemented`
### 20. Sandbox handler returns empty responses (no actual execution)
### 21. No `.env` file exists (only `.env.example`)
### 22. `go:1.25.0` in go.mod — Go 1.25 doesn't exist yet
### 23. Frontend ConnectRPC client version mismatch: `@connectrpc/connect:1.4.0` vs backend `connectrpc.com/connect v1.19.1`
