# Aleph-v2 Chaos Drills

**Version:** 1.0 — 30 Apr 2026

---

## Purpose

This document defines manual chaos drills for Aleph-v2 to validate resilience patterns. These drills simulate component failures to verify that the system degrades gracefully and recovers correctly.

> **Note:** No automated chaos tooling (chaos-mesh, litmus, etc.) is required. These are manual drills executed by operators.

---

## Drill 1: NLP Sidecar Down

### Objective
Verify the backend degrades gracefully when the NLP sidecar becomes unavailable.

### Expected Behavior
- Query endpoint returns results without NLP enrichment (degraded quality, no forecasting)
- Circuit breaker opens after consecutive failures (default: 5)
- Health check endpoint reports sidecar as unhealthy
- Logs record connection failures at WARN level (not ERROR)
- Fallback heuristic mode activates in Decision Engine (PAORA cycle)

### Manual Test Steps

1. Start the full stack: `docker compose up -d`
2. Verify sidecar is healthy: `curl http://localhost:8001/health`
3. Stop the sidecar: `docker compose stop aleph-nlp`
4. Send a query: `curl -X POST http://localhost:8080/api/v1/query -H "X-Aleph-Api-Key: $KEY" -d '{"query": "test"}'`
5. Verify response includes results without NLP enrichment
6. Check backend logs for WARN-level connection failure messages (not ERROR)
7. Verify health endpoint reports NLP as unhealthy: `curl http://localhost:8080/healthz`
8. Restart sidecar: `docker compose start aleph-nlp`
9. Verify circuit breaker closes after recovery (wait ~60s for health checks)
10. Send another query and verify NLP enrichment resumes

### Pass Criteria
- [ ] No 500 errors (only degraded responses)
- [ ] Circuit breaker opens and closes correctly
- [ ] Health check reflects sidecar status
- [ ] Logs do not spew ERROR on expected failures

---

## Drill 2: Database Latency Injection

### Objective
Verify timeout and retry behavior under database latency.

### Expected Behavior
- Queries time out gracefully (configurable timeout, default 30s)
- Retry logic fires for transient connection errors (up to 3 retries with backoff)
- Rate limiter prevents connection pool exhaustion
- Client receives clear error messages, not raw SQL errors

### Manual Test Steps

1. Start the full stack
2. Add artificial latency to PostgreSQL:
   ```sql
   -- Connect to psql and add a rule-based delay (requires pg_delay extension)
   -- Alternative: use network throttling
   sudo tc qdisc add dev eth0 netem delay 5000ms
   ```
3. Send a query: `curl -X POST http://localhost:8080/api/v1/query -H "X-Aleph-Api-Key: $KEY" -d '{"query": "test"}'`
4. Verify response either:
   - Returns within timeout with delayed results, OR
   - Returns 504 Gateway Timeout with clear error message
5. Check retry logs: `docker compose logs aleph-backend | grep "retry"`
6. Remove latency: `sudo tc qdisc del dev eth0 netem`
7. Verify system returns to normal response times

### Alternative (No Network Access)
1. Set `DB_MAX_OPEN_CONNS=1` and `DB_MAX_IDLE_CONNS=1` in `.env`
2. Send 5 concurrent queries
3. Verify some get 503 Service Unavailable (bulkhead)
4. Restore original connection settings

### Pass Criteria
- [ ] No hung connections (all requests terminate within timeout)
- [ ] Retry logic fires correctly
- [ ] Bulkhead prevents pool exhaustion
- [ ] Clear error messages (no raw SQL leaks)

---

## Drill 3: MCP Discovery Failure

### Objective
Verify cache fallback when MCP discovery service is unavailable.

### Expected Behavior
- MCP discovery uses cached tool list when refresh fails
- Stale cache is better than no response (serve cached data up to TTL)
- Background retry continues without blocking user requests
- Logs record discovery failures at WARN level

### Manual Test Steps

1. Start the full stack
2. Verify tools are registered: `curl http://localhost:8080/api/v1/tools -H "X-Aleph-Api-Key: $KEY"`
3. Block discovery endpoints (add to `/etc/hosts`):
   ```bash
   echo "127.0.0.1 registry.example.com" | sudo tee -a /etc/hosts
   ```
   OR set `MCP_REGISTRY_URL=http://nonexistent:9999` in `.env` and restart
4. Wait for discovery cache TTL to expire (default: 5 minutes)
5. Request tools list: `curl http://localhost:8080/api/v1/tools -H "X-Aleph-Api-Key: $KEY"`
6. Verify response includes cached tools (possibly stale)
7. Check logs for discovery failure WARN messages
8. Restore discovery URL and restart (or remove /etc/hosts entry)
9. Verify new tools are discovered after recovery

### Pass Criteria
- [ ] Cached tools served during outage
- [ ] No 500 errors from discovery failures
- [ ] Recovery happens automatically after service restoration

---

## Drill 4: High Resource Usage (CPU/Memory)

### Objective
Verify circuit breaker and resource limits under load.

### Expected Behavior
- Bulkhead rejects requests when concurrent limit exceeded
- Circuit breaker opens after failure threshold
- OOM killer or resource limits prevent system crash
- System recovers after load subsides

### Manual Test Steps

1. Start the full stack
2. Generate load: `for i in $(seq 1 100); do curl -X POST http://localhost:8080/api/v1/query -H "X-Aleph-Api-Key: $KEY" -d '{"query":"test"}' & done`
3. Monitor response codes — should see mix of 200 and 503 (bulkhead)
4. Check backend logs for circuit breaker state changes
5. Wait for load to subside (all background curls finish)
6. Verify system returns to normal: single curl returns 200

### Pass Criteria
- [ ] No system crash
- [ ] 503s returned during overload (not 500s)
- [ ] System recovers to normal within 30s after load stops

---

## Drill 5: RBAC Enforcement

### Objective
Verify role-based access control works correctly under anomalous conditions.

### Expected Behavior
- Read-only keys cannot write data
- User keys cannot manage system settings
- Admin keys retain full access
- Invalid/expired keys return 401

### Manual Test Steps

1. Create three API keys with different roles:
   ```sql
   INSERT INTO system_api_keys (id, project_id, label, key, role)
   VALUES
     ('ro_test_', 'test', 'readonly-test', '<argon2id-hash>', 'readonly'),
     ('user_tes', 'test', 'user-test', '<argon2id-hash>', 'user'),
     ('adm_test', 'test', 'admin-test', '<argon2id-hash>', 'admin');
   ```
2. With readonly key: attempt to POST to write endpoints
   - Expected: 403 Forbidden
3. With user key: attempt to access admin-only endpoints
   - Expected: 403 Forbidden
4. With admin key: attempt all operations
   - Expected: 200 OK
5. With expired/invalid key: attempt any operation
   - Expected: 401 Unauthorized

### Pass Criteria
- [ ] Role restrictions enforced correctly
- [ ] No privilege escalation possible
- [ ] Clear error messages (no information leakage)

---

## Drill 6: Rollback Procedure

### Trigger
Any drill that reveals a critical failure requiring system rollback.

### Steps

1. **Identify the failed deployment version**:
   ```bash
   docker compose ps  # Note current image tags
   git log --oneline -5  # Note current commit
   ```

2. **Stop all services**:
   ```bash
   docker compose down
   ```

3. **Checkout previous known-good version**:
   ```bash
   git checkout <previous-stable-tag>
   ```

4. **Rebuild and restart**:
   ```bash
   docker compose up --build -d
   ```

5. **Verify rollback**:
   ```bash
   docker compose ps
   curl http://localhost:8080/healthz
   ```

6. **Document the incident** in `docs/incidents/` with:
   - What triggered the rollback
   - Steps taken
   - Recovery time
   - Root cause (once identified)

### Database Rollback (if needed)

If a migration caused the issue:
```bash
# PostgreSQL rollback
psql -U aleph -d aleph -f migrations/postgres/000006_api_key_role.down.sql

# DuckDB rollback
duckdb data/aleph.duckdb < migrations/duckdb/000007_api_key_role.down.sql
```

---

## Drill Schedule

| Drill | Frequency | Owner | Last Run |
|-------|-----------|-------|----------|
| NLP Sidecar Down | Monthly | Platform team | — |
| DB Latency | Monthly | Platform team | — |
| MCP Discovery Failure | Monthly | Platform team | — |
| High Resource Usage | Quarterly | Platform team | — |
| RBAC Enforcement | Monthly | Security team | — |

---

## Incident Logging

After each drill, record results in `docs/incidents/`:
- Date and time
- Drill performed
- Pass/fail criteria results
- Any unexpected behavior
- Follow-up actions