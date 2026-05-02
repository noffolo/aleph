# Runbook — Aleph-v2

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Audience:** Site Reliability Engineers, DevOps, On-call engineers

This runbook contains step-by-step procedures for operating, monitoring, and troubleshooting each Aleph-v2 subsystem. Keep this document open during on-call shifts.

---

## Table of Contents

1. [Subsystem Overview](#1-subsystem-overview)
2. [Backend Go](#2-backend-go)
3. [Frontend React](#3-frontend-react)
4. [PostgreSQL Metadata](#4-postgresql-metadata)
5. [DuckDB Analytical](#5-duckdb-analytical)
6. [Python NLP Sidecar](#6-python-nlp-sidecar)
7. [Ollama LLM](#7-ollama-llm)
8. [Docker / Infrastructure](#8-docker--infrastructure)
9. [Security Incidents](#9-security-incidents)
10. [Performance Degradation](#10-performance-degradation)
11. [Data Recovery](#11-data-recovery)
12. [Escalation](#12-escalation)

---

## 1. Subsystem Overview

| Subsystem | Role | Criticality | Owner |
|-----------|------|-------------|-------|
| Backend Go | API, decision engine, sandbox | **Critical** | Backend team |
| Frontend React | User interface | High | Frontend team |
| PostgreSQL 16 | Metadata, auth, audit | **Critical** | Backend team |
| DuckDB | Analytical queries, embeddings | High | Backend team |
| Python NLP Sidecar | Sentiment, predictions, embeddings | Medium | ML team |
| Ollama | Local LLM inference | High | Infrastructure |
| Docker Compose | Orchestration | **Critical** | DevOps |

---

## 2. Backend Go

### 2.1 Health checks

```bash
# Readiness (503 during startup or drain)
curl -s http://localhost:8080/readyz | jq .

# Liveness (200 as long as process is alive)
curl -s http://localhost:8080/livez | jq .

# Full health
curl -s http://localhost:8080/api/v1/healthz | jq .

# Tool health
curl -s -H "X-Aleph-Api-Key: <key>" http://localhost:8080/api/v1/tools/health | jq .
```

Expected:
- `readyz`: `{"status":"ready"}` HTTP 200
- `livez`: `{"status":"alive"}` HTTP 200
- `healthz`: `{"status":"ok"}` HTTP 200

### 2.2 Restart procedure

```bash
# Graceful restart (Docker Compose)
docker compose restart aleph-backend

# Wait for readiness
watch -n 2 'curl -s http://localhost:8080/readyz'
# Stop when you see 200 OK
```

If the container keeps restarting, check logs:
```bash
docker compose logs --tail=100 aleph-backend
```

Common fatal causes:
- `KEY_ENCRYPTION_KEY` missing or wrong length
- `POSTGRES_DSN` unreachable
- Port 8080 already bound

### 2.3 Memory leak

Symptom: RSS grows continuously, OOM killed by Docker.

Diagnostic:
```bash
# Get memory profile
curl -s http://localhost:8080/debug/pprof/heap > heap.pprof
go tool pprof heap.pprof
```

Mitigation:
```bash
# Set memory limit in docker-compose.yml
services:
  aleph-backend:
    mem_limit: 1g
    memswap_limit: 1g
```

If leak persists, restart the container and file a bug with the heap profile attached.

### 2.4 High CPU

Symptom: `top` shows Go process at >80% CPU.

Diagnostic:
```bash
# CPU profile (30 seconds)
curl -s http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.pprof
go tool pprof cpu.pprof
```

Common causes:
- Infinite loop in PAORA Reflect (rare, caught by max 5 iterations)
- SSE broker fan-out under high load
- DuckDB query without LIMIT on large table

Mitigation:
1. Identify the hot goroutine from the profile
2. If it is a query, kill the connection: restart backend
3. If it is SSE, check active connections: `curl /metrics | grep aleph_sse_connections`

### 2.5 Circuit breaker triggered (NLP)

Symptom: `NLPService` returns synthetic fallback responses.

Check:
```bash
# Is the sidecar reachable?
docker compose exec aleph-backend wget -qO- http://aleph-nlp-sidecar:8001

# Is the sidecar healthy?
docker compose ps aleph-nlp-sidecar
```

If sidecar is down:
1. Restart it: `docker compose restart aleph-nlp-sidecar`
2. Wait 10s for health check
3. Verify backend reconnects: check logs for `NLP connection restored`

---

## 3. Frontend React

### 3.1 Blank page / 500 error

Checklist:
1. Backend is up: `curl http://localhost:8080/readyz`
2. `VITE_API_BASE_URL` is correct and reachable from browser
3. Browser console — any CORS errors? Update `CORS_ALLOWED_ORIGINS`
4. Network tab — API calls returning 401? API key missing or invalid

### 3.2 SSE disconnections

Symptom: Chat stops streaming, `/status` shows "SSE disconnected".

Diagnostic:
```bash
# Check active SSE connections
curl -s http://localhost:8080/metrics | grep aleph_sse_connections
```

Common causes:
- Reverse proxy timeout (nginx default 60s) — increase `proxy_read_timeout 300s`
- Bulkhead limit reached (default 100 concurrent SSE) — check metric `aleph_sse_connections`
- API key expired — client reconnects with 401

Mitigation:
```bash
# Increase SSE bulkhead in .env
# (requires code change in middleware/bulkhead.go or env var if exposed)
```

### 3.3 Slow UI

Symptom: Typing lag, slow re-renders.

Diagnostic:
- Browser DevTools → Performance → Record 5s of interaction
- Look for long `render` or `commit` phases

Common causes:
- Large dataset rendered without virtualization
- Zustand store update triggering too many re-renders
- D3 chart re-rendering on every message

Mitigation (user-side):
```
/theme minimal    # Disable scanline/glow effects
/clear            # Clear chat history if very long
```

Mitigation (ops-side): check if `React.memo` is applied to heavy components.

---

## 4. PostgreSQL Metadata

### 4.1 Connection pool exhaustion

Symptom: Backend logs `FATAL: sorry, too many clients already`.

Check:
```bash
docker compose exec aleph-db psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"
docker compose exec aleph-db psql -U postgres -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"
```

Mitigation:
```bash
# Increase max_connections in postgresql.conf
docker compose exec aleph-db psql -U postgres -c "ALTER SYSTEM SET max_connections = 200;"
docker compose restart aleph-db

# Or increase pool size in DSN (backend side)
# POSTGRES_DSN=postgres://...?pool_max_conns=30
```

### 4.2 Disk full

Symptom: PostgreSQL goes into read-only mode, backend returns 500.

Check:
```bash
docker compose exec aleph-db df -h /
docker compose exec aleph-db psql -U postgres -c "SELECT pg_database_size('aleph');"
```

Mitigation:
```bash
# Find large tables
docker compose exec aleph-db psql -U postgres -c "
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC LIMIT 10;"

# Truncate audit logs older than 90 days (if policy allows)
docker compose exec aleph-db psql -U postgres -c "
DELETE FROM audit_log WHERE created_at < NOW() - INTERVAL '90 days';"
```

Long-term: mount a larger volume for `aleph-pgdata`.

### 4.3 Slow queries

Symptom: API responses >2s, backend logs long query times.

Enable slow query log:
```bash
docker compose exec aleph-db psql -U postgres -c "
ALTER SYSTEM SET log_min_duration_statement = 1000;
SELECT pg_reload_conf();"
```

Check logs:
```bash
docker compose logs aleph-db | grep "duration:"
```

Common slow queries:
- `SELECT * FROM audit_log ORDER BY created_at DESC` without LIMIT
- `SELECT * FROM tools WHERE project_id = $1` with no index on `project_id`

Add index:
```bash
docker compose exec aleph-db psql -U postgres -c "
CREATE INDEX CONCURRENTLY idx_audit_created ON audit_log(created_at);
CREATE INDEX CONCURRENTLY idx_tools_project ON tools(project_id);"
```

### 4.4 Backup verification

Run weekly:
```bash
# Restore latest backup to a test container
docker run --rm -e POSTGRES_PASSWORD=test -v /backups:/backups postgres:16 \
  bash -c "pg_restore -U postgres -d aleph /backups/latest.sql"
```

If restore fails, alert immediately and check backup integrity.

---

## 5. DuckDB Analytical

### 5.1 Database corruption

Symptom: DuckDB queries return `Corrupt database` or segfault.

Check:
```bash
docker compose exec aleph-backend ls -lh /app/data/aleph.duckdb
docker compose exec aleph-backend duckdb /app/data/aleph.duckdb "PRAGMA integrity_check;"
```

Mitigation:
```bash
# Restore from backup
docker compose cp ./aleph.duckdb.backup aleph-backend:/app/data/aleph.duckdb
docker compose restart aleph-backend
```

If no backup exists, re-ingest all raw data. This is why automated backups are critical.

### 5.2 Query timeout

Symptom: `ExecuteQuery` returns `ERR_TIMEOUT` after 30s.

Diagnostic:
```bash
# Check running queries
docker compose exec aleph-backend duckdb /app/data/aleph.duckdb "
SELECT * FROM duckdb_queries();"
```

Mitigation:
1. Ask user to add `LIMIT` to their query
2. For scheduled analytics, run heavy queries during off-peak hours
3. Consider creating materialized views for common aggregations

### 5.3 Storage growth

DuckDB files can grow large with many append operations.

Check:
```bash
docker compose exec aleph-backend ls -lh /app/data/aleph.duckdb
docker compose exec aleph-backend duckdb /app/data/aleph.duckdb "
SELECT table_name, estimated_size FROM information_schema.tables;"
```

Compact:
```bash
# Vacuum and checkpoint
docker compose exec aleph-backend duckdb /app/data/aleph.duckdb "
CHECKPOINT;
VACUUM;"
```

---

## 6. Python NLP Sidecar

### 6.1 Sidecar crash

Symptom: `AnalyzeSentiment` and `StreamPredictions` return synthetic fallback.

Check:
```bash
docker compose ps aleph-nlp-sidecar
docker compose logs --tail=50 aleph-nlp-sidecar
```

Common causes:
- ONNX model file missing from `./nlp/models/`
- Out of memory (ONNX model uses ~400MB)
- gRPC port conflict

Mitigation:
```bash
# Restart
docker compose restart aleph-nlp-sidecar

# Verify
docker compose exec aleph-backend python3 -c "
import grpc, socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
result = s.connect_ex(('aleph-nlp-sidecar', 8001))
print('Port open' if result == 0 else 'Port closed')
"
```

### 6.2 Model loading failure

Symptom: Logs show `ONNXRuntimeError` or `Model file not found`.

Check:
```bash
docker compose exec aleph-nlp-sidecar ls -la /app/onnx_model/
```

Expected files:
- `model.onnx`
- `vocab.txt` (if applicable)

If missing:
```bash
# Re-download or restore from backup
docker compose cp ./nlp/models/ aleph-nlp-sidecar:/app/onnx_model/
docker compose restart aleph-nlp-sidecar
```

### 6.3 High latency

Symptom: NLP calls take >5s.

Diagnostic:
```bash
# Time a direct gRPC call
docker compose exec aleph-backend python3 -c "
import grpc, time
from api.proto.aleph.nlp.v1 import nlp_pb2, nlp_pb2_grpc
ch = grpc.insecure_channel('aleph-nlp-sidecar:8001')
stub = nlp_pb2_grpc.NLPServiceStub(ch)
start = time.time()
resp = stub.AnalyzeSentiment(nlp_pb2.AnalyzeSentimentRequest(text='test'))
print(f'Latency: {(time.time()-start)*1000:.0f}ms')
"
```

Common causes:
- CPU throttling (ONNX is CPU-intensive)
- DuckDB query inside sentiment analysis (should be instant on cached data)

Mitigation: ensure the sidecar has dedicated CPU cores, not shared with Ollama.

---

## 7. Ollama LLM

### 7.1 Model not found

Symptom: Backend logs `model 'llama3' not found`.

Check:
```bash
# List pulled models
curl http://localhost:11434/api/tags | jq '.models[].name'

# Or from inside Docker
docker compose exec ollama ollama list
```

Mitigation:
```bash
# Pull model
docker compose exec ollama ollama pull llama3

# Verify
curl http://localhost:11434/api/generate -d '{"model":"llama3","prompt":"hello"}'
```

### 7.2 OOM during inference

Symptom: Ollama process killed, backend switches to degraded mode.

Check:
```bash
docker compose logs ollama | grep -i "killed\|oom\|memory"
free -h
```

Mitigation:
1. Use a smaller model: `ollama pull llama3:8b` instead of `70b`
2. Add swap: `swapon /swapfile`
3. Increase Docker memory limit in `docker-compose.yml`:
   ```yaml
   ollama:
     deploy:
       resources:
         limits:
           memory: 8G
   ```

### 7.3 Degraded mode

Symptom: Agent responses are keyword-based instead of LLM-generated.

Check backend logs:
```bash
docker compose logs aleph-backend | grep -i "degraded\|heuristic\|provider"
```

If `provider nil` or `error LLM`: Ollama is unreachable. Follow Ollama restart procedure above.

If Ollama is up but slow: the backend may have circuit-breaker-open. Wait 30s for recovery, or restart backend.

---

## 8. Docker / Infrastructure

### 8.1 Container won’t start

```bash
# Inspect
docker compose ps
docker compose logs --tail=50 <service>

# Common fixes
docker compose down
docker compose build --no-cache <service>
docker compose up -d <service>
```

### 8.2 Volume full

```bash
# Check disk
docker system df -v

# Prune (careful — removes unused volumes)
docker volume prune

# If aleph-pgdata is full, expand the volume or move to larger disk
```

### 8.3 Network issues

Symptom: Services cannot reach each other (for example, backend cannot reach DB).

Check:
```bash
# List networks
docker network ls
docker network inspect aleph-v2_aleph-network

# Test connectivity
docker compose exec aleph-backend ping -c 3 aleph-db
docker compose exec aleph-backend wget -qO- http://aleph-db:5432
```

Mitigation:
```bash
# Recreate network
docker compose down
docker network rm aleph-v2_aleph-network || true
docker compose up -d
```

### 8.4 Image pull failure

Symptom: `docker compose up` fails with `pull access denied`.

Check:
```bash
docker login
# Or verify image exists on registry
docker pull <image:tag>
```

If using a private registry, ensure `docker-compose.yml` references the correct image and credentials are configured.

---

## 9. Security Incidents

### 9.1 Suspected API key compromise

1. **Revoke immediately**
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/keys/revoke \
     -H "X-Aleph-Api-Key: <admin-key>" \
     -H "Content-Type: application/json" \
     -d '{"key_id": "<compromised-key-id>"}'
   ```

2. **Check audit logs**
   ```bash
   docker compose exec aleph-db psql -U postgres -c "
   SELECT * FROM audit_log WHERE details->>'key_id' = '<compromised-key-id>' ORDER BY created_at DESC;"
   ```

3. **Rotate `KEY_ENCRYPTION_KEY`**
   - Generate new key
   - Update `.env` and Docker secrets
   - Restart backend (this re-encrypts existing keys)
   - Notify all users to regenerate their API keys

4. **File incident report**

### 9.2 Sandbox escape attempt

Symptom: `SecurityScanner` logs show blocked patterns, or suspicious tool code.

1. **Block the tool**
   ```bash
   curl -X DELETE http://localhost:8080/api/v1/tools/<tool-id> \
     -H "X-Aleph-Api-Key: <admin-key>"
   ```

2. **Check sandbox logs**
   ```bash
   docker compose logs aleph-backend | grep -i "sandbox\|security\|blocked"
   ```

3. **Review tool code** via `LibraryService > GetAssetContent`

4. **If escape succeeded**: isolate the host, preserve logs, contact security team.

### 9.3 DDoS / Rate limit abuse

Symptom: `ERR_RATE_LIMIT` spikes, high `aleph_request_errors_total`.

Check:
```bash
curl -s http://localhost:8080/metrics | grep rate_limit
```

Mitigation:
1. If legitimate traffic: increase limits temporarily in `.env`
2. If abusive IP: block at reverse proxy level:
   ```nginx
   deny 192.0.2.100;
   ```
3. If distributed: enable Cloudflare or similar DDoS protection

---

## 10. Performance Degradation

### 10.1 General diagnostic flow

```bash
# 1. Check error rate
curl -s http://localhost:8080/metrics | grep request_errors

# 2. Check latency percentiles
curl -s http://localhost:8080/metrics | grep request_duration_seconds

# 3. Check active connections
curl -s http://localhost:8080/metrics | grep sse_connections

# 4. Check resource usage
docker stats --no-stream

# 5. Check slow query log (PostgreSQL)
docker compose logs aleph-db | grep "duration:"
```

### 10.2 High latency checklist

| Symptom | Likely cause | Action |
|---------|--------------|--------|
| All endpoints slow | CPU/memory pressure | Check `docker stats`, scale up |
| Chat streaming slow | Ollama overloaded | Check Ollama queue, use smaller model |
| Query endpoints slow | DuckDB lock contention | Check concurrent queries, add LIMIT |
| SSE drops | Bulkhead limit | Increase limit or add load balancer |
| First request slow | Cold start / JIT | Pre-warm after restart |

### 10.3 Scaling

Docker Compose is single-host. For horizontal scaling, move to Kubernetes:

- Backend: 3+ replicas with shared PostgreSQL and DuckDB (read replicas)
- Frontend: static files on CDN or nginx replicas
- NLP sidecar: 2+ replicas behind gRPC load balancer
- Ollama: dedicated GPU nodes

See [`docs/deployment-guide.md`](./deployment-guide.md) for Kubernetes migration notes.

---

## 11. Data Recovery

### 11.1 Full disaster recovery

Scenario: Host failure, all data lost, backups intact.

1. **Provision new host**
2. **Clone repository**
   ```bash
   git clone <repo-url> && cd aleph-v2
   ```
3. **Restore secrets**
   ```bash
   # From password manager / vault
   echo "$KEY_ENCRYPTION_KEY" > .env
   echo "$POSTGRES_PASSWORD" >> .env
   ```
4. **Restore PostgreSQL**
   ```bash
   docker compose up -d aleph-db
   sleep 10
   cat /backups/postgres-latest.sql | docker compose exec -T aleph-db psql -U postgres
   ```
5. **Restore DuckDB**
   ```bash
   docker compose cp /backups/aleph.duckdb.backup aleph-backend:/app/data/aleph.duckdb
   ```
6. **Restore projects**
   ```bash
   docker compose cp /backups/projects/ aleph-backend:/app/data/
   ```
7. **Start everything**
   ```bash
   docker compose up -d
   ```
8. **Verify**
   ```bash
   curl http://localhost:8080/readyz
   curl http://localhost:5174
   ```

### 11.2 Partial recovery (single project)

If only one project is corrupted:

```bash
# From backup
docker compose cp /backups/projects/<project-name>/ aleph-backend:/app/data/projects/<project-name>/

# Re-ingest metadata
curl -X POST http://localhost:8080/api/v1/projects/<project-id>/emerge \
  -H "X-Aleph-Api-Key: <key>"
```

### 11.3 Point-in-time recovery

PostgreSQL supports PITR if WAL archiving is enabled. For Docker Compose, enable in `postgresql.conf`:

```
wal_level = replica
archive_mode = on
archive_command = 'cp %p /backups/wal/%f'
```

Recovery:
```bash
# Stop DB, restore base backup, replay WAL up to target time
# See PostgreSQL documentation for full procedure
```

---

## 12. Escalation

### When to escalate

| Situation | Escalate to | Contact |
|-----------|-------------|---------|
| Sandbox escape | Security team | #security-incident |
| Data breach | Security + Legal | #security-incident + legal@ |
| Core algorithm bug | ML/Backend lead | @ml-lead |
| Frontend framework crash | Frontend lead | @frontend-lead |
| Infrastructure outage | DevOps/SRE | #infrastructure |
| Vendor outage (Ollama, OpenAI) | Product + Comms | @product-manager |

### Incident response checklist

1. **Detect** — alert fires or user reports
2. **Assess** — classify severity (P1=critical, P2=high, P3=medium, P4=low)
3. **Contain** — stop the bleeding (restart, block IP, revoke key)
4. **Diagnose** — follow runbook procedures above
5. **Resolve** — apply fix, verify recovery
6. **Communicate** — post status update to `#incidents`
7. **Post-mortem** — within 48 hours for P1/P2

### Useful commands cheat sheet

```bash
# All logs
docker compose logs -f

# Specific service
docker compose logs -f aleph-backend

# Resource usage
docker stats

# Database size
docker compose exec aleph-db psql -U postgres -c "SELECT pg_size_pretty(pg_database_size('aleph'));"

# Active connections
docker compose exec aleph-db psql -U postgres -c "SELECT count(*) FROM pg_stat_activity WHERE state='active';"

# Backend goroutines
curl -s http://localhost:8080/debug/pprof/goroutine > goroutines.prof

# Force restart everything
docker compose down && docker compose up -d
```

---

## Reference

- [`docs/deployment-guide.md`](./deployment-guide.md) — Full deployment instructions
- [`docs/developer-onboarding.md`](./developer-onboarding.md) — Development setup
- [`docs/api-reference.md`](./api-reference.md) — API endpoints
- [`docs/manuale-tecnico.md`](./manuale-tecnico.md) — Technical manual (Italian)
