# Aleph-v2 Load Tests

k6-based load testing suite for Aleph-v2 API endpoints. Covers three scenarios:

| Test | Endpoint | Type | Auth |
|------|----------|------|------|
| `health.js` | `GET /api/v1/healthz` | REST (unary) | No |
| `query.js` | `POST /aleph.v1.QueryService/ExecuteQuery` | ConnectRPC (unary) | Yes |
| `chat.js` | `POST /aleph.v1.QueryService/Chat` | ConnectRPC (streaming) | Yes |

Target: **500 req/s, p95 < 1s** for unary endpoints; **100 req/s, p95 < 5s** for streaming.

---

## Prerequisites

### Install k6

**macOS (Homebrew):**
```bash
brew install k6
```

**Linux (APT):**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Docker:**
```bash
docker pull grafana/k6
```

**Other platforms:** [Download from k6.io](https://k6.io/docs/get-started/installation/)

---

## Setup

1. Ensure the Aleph-v2 backend is running:
   ```bash
   docker compose up -d
   # or locally:
   go run . &
   ```

2. Verify the server is reachable:
   ```bash
   curl http://localhost:8080/api/v1/healthz
   # → {"status":"ok"}
   ```

3. Set an API key in the Aleph UI (Settings → API Keys), or use the default `test-key` if one is configured.

---

## Usage

### Run all tests (default)

```bash
./deploy/load-tests/run.sh
```

### Run only health + query (quick mode, skips streaming)

```bash
./deploy/load-tests/run.sh --quick
```

### Run individual tests

```bash
# Health (baseline, no auth):
k6 run deploy/load-tests/health.js

# Query (unary RPC, authenticated):
k6 run deploy/load-tests/query.js

# Chat (streaming RPC, authenticated):
k6 run deploy/load-tests/chat.js
```

---

## Configuration

All settings are configurable via environment variables or CLI flags on `run.sh`.

### Env vars (all scripts)

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | Target server URL |
| `DURATION` | `30s` | Sustained load phase duration |
| `VUS` | `500` (health/query), `100` (chat) | Target virtual users |
| `API_KEY` | `test-key` | `X-Aleph-Api-Key` header value |
| `PROJECT_ID` | `default` | Project ID for query/chat tests |
| `OBJECT_TYPE` | `""` | Object type filter (query.js only) |
| `AGENT_ID` | `""` | Agent ID (chat.js only) |

### CLI flags (run.sh only)

| Flag | Description |
|------|-------------|
| `--quick` | Skip streaming chat test |
| `--vus N` | Override VUs for all tests |
| `--duration DUR` | Override test duration (e.g. `60s`, `5m`) |
| `--url URL` | Override target URL |
| `--api-key KEY` | Override API key |
| `--project ID` | Override project ID |
| `--help` | Show help |

### Examples

```bash
# Production test with custom parameters:
./deploy/load-tests/run.sh \
  --url https://api.aleph.example.com \
  --vus 1000 \
  --duration 120s \
  --api-key sk-prod-abc123

# Environment variables override:
VUS=200 BASE_URL=http://staging:8080 ./deploy/load-tests/run.sh

# Quick check with Docker k6:
docker run --rm -i grafana/k6 run - <deploy/load-tests/health.js \
  -e BASE_URL=http://host.docker.internal:8080
```

---

## Test Descriptions

### 1. health.js — Baseline

- **Endpoint:** `GET /api/v1/healthz`
- **Target:** p95 < 200ms, 0% errors
- **Why:** Measures raw server throughput with zero auth/processing overhead. Establishes the baseline for interpreting other test results.

### 2. query.js — ExecuteQuery

- **Endpoint:** `POST /aleph.v1.QueryService/ExecuteQuery`
- **Target:** p95 < 1s, < 1% errors
- **Payload:** JSON `{object_type, project_id, limit: 100}`
- **Why:** Tests the primary data access path — DuckDB query execution, serialization, and network transport. Includes a small `sleep(0-0.5s)` think-time to simulate realistic usage patterns.

### 3. chat.js — Chat (streaming)

- **Endpoint:** `POST /aleph.v1.QueryService/Chat`
- **Target:** p95 < 5s, < 2% errors
- **Payload:** JSON `{message, project_id, agent_id}`
- **Why:** Tests the heaviest endpoint — streaming responses through the LLM provider, tool call orchestration, and SSE transport. Uses a rotating set of sample prompts to vary the payload.

---

## Interpreting Results

k6 outputs summary metrics at the end of each test:

```text
http_req_duration..........: avg=45ms   min=2ms   med=30ms   max=950ms  p(90)=120ms  p(95)=200ms
http_req_failed............: 0.00%  ✓ 0   ✗ 15000
http_reqs..................: 15000  500.0/s
vus........................: 500    min=0   max=500
```

Key metrics:
- **http_req_duration p(95):** 95th percentile latency — the main SLA target
- **http_req_failed:** Error rate (4xx/5xx responses + network errors)
- **http_reqs:** Total requests and throughput (req/s)
- **vus:** Virtual user count (ramp-up/steady/ramp-down)

### Thresholds

All scripts have built-in thresholds that will cause a non-zero exit code if violated:

- Health: `p(95)<1000ms`, errors < 1%
- Query: `p(95)<1000ms`, errors < 1%
- Chat: `p(95)<5000ms`, errors < 2%

---

## CI Integration

These tests can run in CI. Example GitHub Actions snippet:

```yaml
- name: Load test
  run: |
    docker compose up -d
    sleep 5
    npm install -g k6
    ./deploy/load-tests/run.sh --quick
```

### k6 output formats

For CI, enable JSON or Prometheus output:

```bash
k6 run --out json=results.json deploy/load-tests/health.js
k6 run --out statsd deploy/load-tests/query.js
```

The Prometheus/Grafana stack already included in Aleph-v2's `deploy/` can ingest k6 metrics via StatsD.
