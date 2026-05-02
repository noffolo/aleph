# Aleph-v2 Release Gate Checklist

## Ship Gate Status

| Check | Status |
|-------|--------|
| `go build ./...` | ✅ |
| `go test -race -count=1 ./...` | ✅ |
| `go vet ./...` | ✅ |
| `npx tsc --noEmit` (frontend/) | ✅ |
| `npx vite build` (frontend/) | ✅ |
| `npx vitest run` (frontend/) | ✅ |
| Docker compose config | ✅ |
| Playwright E2E | ✅ |

## Build Verification
- [x] `go build ./...` — exit code 0
- [x] `go test -race -count=1 ./...` — all pass
- [x] `go vet ./...` — only pre-existing participle warnings in dsl/ast.go
- [x] `npx tsc --noEmit` — zero errors (from frontend/)
- [x] `npx vite build` — exit code 0 (from frontend/)
- [x] `npx vitest run` — all pass (from frontend/)

## Security Verification
- [x] CSP header has no `unsafe-inline`
- [x] Rate limiting uses X-Forwarded-For
- [x] CSRF middleware is active
- [x] SSH/SSE endpoint has authentication
- [x] SQL injection grep: zero fmt.Sprintf with user input in SQL
- [x] API key storage: httpOnly cookie, not sessionStorage

## Docker Verification
- [x] `docker compose config` — valid YAML
- [x] all services have healthcheck
- [x] Ollama pre-pulls nomic-embed-text and llama3

## CI/CD
- [x] `.github/workflows/ci.yml` — all jobs valid
- [x] `.github/workflows/security.yml` — gitleaks configured (if present)

## Pre-Flight Production Deploy Checklist
- [ ] `KEY_ENCRYPTION_KEY` env var is set (32 bytes, base64)
- [ ] `.env` file is present and readable (not committed in repo)
- [ ] Postgres connectivity test: `pg_isready -h $POSTGRES_HOST -p $POSTGRES_PORT`
- [ ] Ollama health: `curl http://$OLLAMA_HOST:11434/api/tags` returns 200
- [ ] DuckDB data volume mounted and writable (`/data` in container)
- [ ] Disk space: at least 5 GB free for container images and data
- [ ] Port availability: 5173, 8080, 5432, 9090, 3000 not in use
- [ ] TLS/SSL certificates configured if exposing publicly
- [ ] Alertmanager Slack webhook and email SMTP validated
- [ ] Backup volume snapshot created before deploy

## Rollback Procedure (Docker Compose)
1. `docker compose down` — stop all running services
2. `git fetch --tags && git checkout <previous-stable-tag>`
3. Verify env vars match the target release
4. `docker compose pull` — ensure images for that tag exist
5. `docker compose up -d --build` — rebuild and restart
6. Run smoke tests (build, health endpoints, basic query)
7. If rollback fails: restore Docker volume from backup snapshot

## Fase 2 Stability Engine Verification
- [ ] k6 load test: 500 req/s, p95 < 1 s (Fase 2), < 500 ms (Fase 4)
- [ ] DuckDB concurrency: 0 deadlock in 1000 parallel queries, 10 min stress
- [ ] LLM budget controls: circuit breaker, token/cost metrics, alerts
- [ ] Rate limiter: memory-safe TTL cleanup, LRU eviction, 100k IP test
- [ ] NLP watchdog: auto-restart with max 3 restarts / 5 min, graceful shutdown
- [ ] Build verification: `go test -race`, `npx tsc`, `npx vite build` all pass
