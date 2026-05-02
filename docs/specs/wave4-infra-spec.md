# SPEC-09: Infrastructure Hardening — Docker, Nginx, Rollback Strategy

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 4, tasks W4-6 through W4-10  
**Findings addressed**: INF2-INF13 (infrastructure cluster), Q5-Q14 (code quality)  
**Depends on**: All W0-W3 (infra hardening is the final layer)  
**Related specs**: `docs/specs/wave4-concurrency-spec.md` (graceful shutdown shares Docker entrypoint), `docs/specs/wave3-api-spec.md` (nginx proxies the hardened API)  
**Status**: ✅ Approved — ready for execution

---

## 1. Docker Image Optimization

### Target: Multi-Stage Build

```dockerfile
# Dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /aleph ./cmd/aleph/

# Stage 2: Runtime (minimal)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /aleph /usr/local/bin/aleph
RUN adduser -D -h /home/aleph aleph
USER aleph
WORKDIR /home/aleph
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/aleph"]
```

### Size Target

| Layer | Before | After |
|-------|--------|-------|
| Go binary | ~40MB | ~15MB (with `-ldflags="-s -w"`) |
| Docker image | ~800MB (with build tools) | ~30MB (alpine + binary) |
| Total | ~1GB+ | < 50MB |

### .dockerignore

```
.git/
.github/
.vscode/
node_modules/
frontend/
*.md
.gitignore
.env
.env.*
secrets/*.key
tmp/
```

### CI Build

```yaml
# .github/workflows/build.yml
- name: Build Docker image
  run: |
    docker build -t aleph-v2:latest .
    docker tag aleph-v2:latest ghcr.io/${{ github.repository }}:${{ github.sha }}
    docker push ghcr.io/${{ github.repository }}:${{ github.sha }}
```

---

## 2. Nginx TLS Configuration

### nginx.conf

```nginx
server {
    listen 80;
    server_name aleph.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name aleph.example.com;

    # TLS
    ssl_certificate     /etc/nginx/certs/fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/m;
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;
    limit_conn_zone   $binary_remote_addr zone=conn:10m;

    # Logging
    access_log /var/log/nginx/aleph-access.log;
    error_log  /var/log/nginx/aleph-error.log;

    location /api/v1/auth/session {
        limit_req zone=auth burst=5 nodelay;
        proxy_pass http://aleph-backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://aleph-backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://aleph-frontend:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 3. Service Exposure Hardening

### PostgreSQL

```yaml
# docker-compose.yml
services:
  db:
    image: postgres:16-alpine
    ports:
      - "127.0.0.1:5432:5432"    # ⚠️ FIX: was "5432:5432" (0.0.0.0)
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    command: |
      -c ssl=on
      -c ssl_cert_file=/etc/ssl/certs/server.crt
      -c ssl_key_file=/etc/ssl/private/server.key
      -c password_encryption=scram-sha-256
    volumes:
      - pgdata:/var/lib/postgresql/data
```

### pg_hba.conf

```
# TYPE  DATABASE  USER  ADDRESS       METHOD
local   all       all                 scram-sha-256
host    all       all   127.0.0.1/32  scram-sha-256
host    all       all   172.16.0.0/12 scram-sha-256  # Docker network
# host  all       all   0.0.0.0/0     trust  ← NEVER
```

### Ollama

```yaml
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "127.0.0.1:11434:11434"  # ⚠️ FIX: bind to localhost only
    environment:
      - OLLAMA_HOST=127.0.0.1
    volumes:
      - ollama_data:/root/.ollama
```

### Firewall Rules (iptables)

```bash
# Deny external access to internal services
iptables -A INPUT -p tcp --dport 5432 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 5432 -j DROP

iptables -A INPUT -p tcp --dport 11434 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 11434 -j DROP
```

---

## 4. Rollback Strategy

### Pre-Deploy Backup

```yaml
# CI/CD: before deploy
- name: Backup databases
  run: |
    docker exec aleph-db pg_dump -U aleph aleph > backup-$(date +%Y%m%d-%H%M).sql
    docker exec aleph-backend cp /data/aleph.duckdb /data/aleph.duckdb.backup-$(date +%Y%m%d-%H%M)
```

### Rollback Procedure

```bash
# 1. Stop new version
docker compose stop backend frontend

# 2. Restore databases from backup
docker exec aleph-db psql -U aleph aleph < backup-20260502-1200.sql
docker exec aleph-backend cp /data/aleph.duckdb.backup-20260502-1200 /data/aleph.duckdb

# 3. Start previous version
git checkout <previous-tag>
docker compose up -d backend frontend

# 4. Verify
curl http://localhost:8080/healthz
# → {"status":"ok","version":"<previous-version>"}
```

### Migration Rollbacks

All migrations must have `.down.sql`:

```
migrations/
├── duckdb/
│   ├── 000001_init.up.sql
│   ├── 000001_init.down.sql    # DROP TABLE components, system_features
│   ├── 000002_tool_metadata.up.sql
│   ├── 000002_tool_metadata.down.sql
│   └── ...
├── postgres/
│   ├── 000001_init.up.sql
│   ├── 000001_init.down.sql
│   ├── 000009_add_constraints.up.sql
│   ├── 000009_add_constraints.down.sql   # ← NEW (see SPEC-05)
│   └── ...
```

---

## 5. Code Quality Grooming

### Dead Code Removal

| File | Status | Action |
|------|--------|--------|
| `internal/middleware/circuitbreaker.go` | Not wired | Wire into interceptor chain OR remove |
| `internal/tools/finance/` | Stub | Implement OR remove (deferred — keep) |
| `internal/tools/osint/` | Stub | Implement OR remove (deferred — keep) |
| `internal/tools/humanecosystems/` | Stub | Implement OR remove (deferred — keep) |
| `internal/tools/adaptation/` | Stub | Implement OR remove (deferred — keep) |

### Error Wrapping Standardization

```go
// ❌ INCONSISTENT
errors.Wrap(err, "context")
fmt.Errorf("context: %v", err)
fmt.Errorf("context: %w", err)

// ✅ STANDARD
fmt.Errorf("context: %w", err)  // Preserves error chain (Go 1.13+)
```

### golangci-lint Configuration

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - misspell
    - unconvert

issues:
  exclude-rules:
    - path: internal/dsl/ast.go
      linters: [govet]
      text: "structtag"
```

### CI Integration

```yaml
# .github/workflows/lint.yml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: latest
    args: --timeout=5m
```

---

## 6. Verification

### Test Coverage

- [ ] `shutdown_test.go` (NEW): SIGTERM → 5 second graceful shutdown → all goroutines exit
- [ ] `docker compose config` validation: No 0.0.0.0 binding for PostgreSQL, Ollama
- [ ] Playwright smoke test: Login → create agent → send message → verify response

### Gate

```
go test -race -count=3 ./...
→ ALL pass, ZERO race warnings

docker compose config
→ No 0.0.0.0 ports except 80/443

grep -rn "context.Background()" internal/ --include="*.go" | grep -v "_test.go" | grep -v "main.go"
→ < 5 remaining (all justified)

npx tsc --noEmit
→ 0 errors

npx vite build
→ Successful, CSP headers clean
```
