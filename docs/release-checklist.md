# Aleph-v2 Release Gate Checklist

## Build Verification
- [ ] `go build ./...` — exit code 0
- [ ] `go test -race -count=1 ./...` — all pass
- [ ] `go vet ./...` — only pre-existing participle warnings in dsl/ast.go
- [ ] `npx tsc --noEmit` — zero errors (from frontend/)
- [ ] `npx vite build` — exit code 0 (from frontend/)
- [ ] `npx vitest run` — all pass (from frontend/)

## Security Verification
- [ ] CSP header has no `unsafe-inline`
- [ ] Rate limiting uses X-Forwarded-For
- [ ] CSRF middleware is active
- [ ] SSH/SSE endpoint has authentication
- [ ] SQL injection grep: zero fmt.Sprintf with user input in SQL
- [ ] API key storage: httpOnly cookie, not sessionStorage

## Docker Verification
- [ ] `docker compose config` — valid YAML
- [ ] all services have healthcheck
- [ ] Ollama pre-pulls nomic-embed-text and llama3

## CI/CD
- [ ] `.github/workflows/ci.yml` — all jobs valid
- [ ] `.github/workflows/security.yml` — gitleaks configured (if present)
