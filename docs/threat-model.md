# Aleph-v2 Threat Model

**Version:** 2.0 — 30 Apr 2026

---

## 1. System Overview

Aleph-v2 is a Decision Intelligence platform with three main components:

| Component | Technology | Role |
|-----------|-----------|------|
| Frontend | React/TypeScript/Vite/Tailwind | Web UI for analysts |
| Backend | Go + ConnectRPC | API server, orchestration, auth |
| NLP Sidecar | Python + gRPC + ONNX/Prophet/GBM | NLP inference, forecasting |

Data stores: PostgreSQL (metadata), DuckDB (analytical), filesystem (drop/ directory for file ingestion).

---

## 2. Data Flow Diagram

```
┌─────────────┐     HTTPS      ┌──────────────┐    ConnectRPC     ┌──────────────┐
│   Browser   │ ────────────── │   Frontend    │ ──────────────── │   Backend     │
│  (User)     │               │  (Vite/React) │                  │   (Go)       │
└─────────────┘               └──────────────┘                  └──────────────┘
                                                                  │ │ │ │
                                                    ┌─────────────┘ │ │ └──────────────┐
                                                    │               │ │                │
                                              gRPC  ▼          SQL ▼  ▼          HTTP  ▼
                                              ┌──────────┐  ┌──────────┐      ┌──────────┐
                                              │   NLP     │  │PostgreSQL│      │ External │
                                              │ Sidecar   │  │ + DuckDB │      │ Services │
                                              │ (Python)  │  │          │      │ (LLM etc)│
                                              └──────────┘  └──────────┘      └──────────┘
```

**Key data flows:**

1. **Browser → Frontend → Backend**: User requests carry API keys (X-Aleph-Api-Key or httpOnly cookie). Backend authenticates via argon2id verification, extracts project ID and role (RBAC: admin/user/readonly).
2. **Backend → NLP Sidecar**: gRPC calls for NLP inference. No auth between backend and sidecar (trusted internal network).
3. **Backend → PostgreSQL/DuckDB**: SQL queries via database/sql. Parameterized queries used; legacy concat sites migrated.
4. **Backend → External Services**: HTTP calls to Ollama, OpenAI, Anthropic, GitHub APIs. Protected by SSRF validation (DNS-resolving).
5. **File Ingestion**: User drops files in `drop/` → file watcher creates IngestionTask → engine processes.

---

## 3. Trust Boundaries

| Boundary | Trust Level | Protocols | Risk |
|----------|------------|-----------|------|
| Browser ↔ Frontend | Untrusted → Semi-trusted | HTTPS | XSS, CSRF, credential theft |
| Frontend ↔ Backend | Semi-trusted → Trusted | HTTPS + ConnectRPC | API key exfiltration, injection |
| Backend ↔ NLP Sidecar | Trusted ↔ Trusted | gRPC (plaintext) | Sidecar compromise, replay |
| Backend ↔ PostgreSQL | Trusted ↔ Trusted | SQL/TLS | SQL injection (legacy sites) |
| Backend ↔ External APIs | Trusted → Untrusted | HTTPS | SSRF, credential leak, MITM |
| Backend ↔ DuckDB | Trusted ↔ Trusted | Embedded SQL | SQL injection, schema isolation |
| OS ↔ drop/ directory | Semi-trusted → Trusted | Filesystem | Path traversal, malicious payloads |

---

## 4. Attack Surface

### 4.1 API Endpoints

| Endpoint Category | Attack Vector | Current Mitigation |
|-----------------|--------------|-------------------|
| `/api/v1/query` | SQL injection via project/table names | validName, resolveTableName, parameterized queries |
| `/api/v1/tools` | Tool execution escape | Sandbox validation, import blocklist |
| `/api/v1/mcp/*` | MCP discovery SSRF | DNS-resolving SSRF validator |
| `/api/v1/ingest` | Malicious file payloads | File type validation, size limits |
| `/api/v1/agents` | API key exfiltration | AES-256-GCM encryption at rest |
| `/api/v1/auth/*` | Credential brute force | Rate limiting, argon2id hashing |
| SSE streams | Unauthenticated streaming | API key validation on connect |

### 4.2 Authentication & Authorization

- **Auth**: API key via X-Aleph-Api-Key header or Authorization Bearer. Keys stored as argon2id hashes.
- **RBAC**: Three roles — admin, user, readonly. Stored in `system_api_keys.role` column. Fallback role derivation from key prefix (`user_` → user, `ro_` → readonly) or env var (`ALEPH_API_KEY_SECRET_BACKEND` match → admin).
- **SSE**: Header-based auth (X-Aleph-Api-Key). No query-param auth (prevents logging exposure).
- **CSRF**: Origin/Referer validation on state-changing requests.

### 4.3 Tool Execution

- **Sandbox**: Go/Python code compiled and executed in sandbox with import blocklist
- **Gaps**: `exec_sandbox.go:ExecuteTool` does not call `ValidateGoCode`/`ValidatePythonCode`
- **Gaps**: `compiler_tool.go` generates `http.Client` without SSRF validation
- **Gaps**: Command allowlist includes `curl` (SSRF risk) and `pip` (supply chain risk)

### 4.4 NLP Sidecar

- No authentication between backend and sidecar
- Runs as separate Docker container on shared Docker network
- Compromise of sidecar grants access to all NLP inference paths
- Circuit breaker monitors health but uses polling

### 4.5 Data at Rest

| Data | Storage | Encryption |
|------|---------|-----------|
| API keys (system) | PostgreSQL | argon2id hash |
| Agent API keys | PostgreSQL | AES-256-GCM (KEY_ENCRYPTION_KEY) |
| User data | DuckDB | None (project-scoped schema isolation) |
| Chat history | PostgreSQL | None |
| RBAC role | PostgreSQL | Plaintext column (`admin`/`user`/`readonly`) |

---

## 5. Threat Categories (STRIDE)

| Threat | Category | Impact | Likelihood | Current Control |
|--------|----------|--------|-----------|----------------|
| API key theft via XSS | Spoofing/Info Disclosure | High | Medium | httpOnly cookies, CSP |
| SQL injection in queries | Tampering | Critical | Low | Parameterized queries, validName guards |
| SSRF via tool execution | Repudiation/Info Disclosure | High | Medium | SSRF validator with DNS resolution |
| Brute-force API key | Spoofing | Medium | Low | Rate limiting, argon2id |
| RBAC privilege escalation | Elevation of Privilege | High | Low | Role stored in DB, prefix fallback |
| NLP sidecar takeover | Elevation of Privilege | High | Low | Docker network isolation |
| File ingestion path traversal | Tampering | Medium | Low | File type + size validation |
| Session hijack | Spoofing | High | Medium | Secure httpOnly cookies, CSRF |
| LLM prompt injection | Tampering | Medium | Medium | System prompts, sandbox validation |
| DuckDB schema escape | Elevation of Privilege | High | Low | Project-scoped schemas, safe identifiers |

---

## 6. Sandbox Threat Model (Detailed)

### 6.1 Attack Vectors

#### Code Execution Attacks
- **Arbitrary System Command Execution**: `os/exec`, `subprocess`, `os.system`
- **Network Access**: `net`, `socket` packages for HTTP requests or data exfiltration
- **Filesystem Escalation**: `syscall`, `unsafe` to bypass restrictions
- **Dynamic Code Loading**: `reflect`, `eval`, `exec`, `__import__`
- **Memory Corruption**: `unsafe` packages for direct memory manipulation

#### Resource Exhaustion Attacks
- **CPU Overload**: Infinite loops, computational bombs
- **Memory Exhaustion**: Large array allocation, memory leaks
- **Disk Space**: Writing large files to `/tmp` or mounted volumes
- **Process Forking**: Unlimited child process creation

#### Data Exfiltration Attacks
- **Network Exfiltration**: Sending data via network packages
- **DNS Tunneling**: Covert channels via DNS queries
- **File-Based Exfiltration**: Writing then reading data from files
- **Environment Variable Leaks**: Reading sensitive env variables

#### Privilege Escalation Attacks
- **Container Escape**: Breaking out of Docker isolation
- **Host System Calls**: Using `syscall` to interact with host kernel
- **Capability Abuse**: Exploiting retained Linux capabilities

### 6.2 Sandbox Mitigations

#### Language-Level Restrictions

**Go Import Blocklist**: `unsafe`, `reflect`, `os/*`, `io/*`, `crypto/*`, `encoding/*`, `net/*`, `syscall`, `embed`, `plugin`, `os/exec`, `os/signal`. AST-based parsing with string-scan fallback.

**Python Import/Call Blocklist**: `subprocess`, `socket`, `ctypes`, `__import__`, `eval()`, `exec()`. Regex pattern matching on source lines.

#### Container Isolation (Docker)

- `read_only: true` — read-only filesystem
- `tmpfs: /tmp:rw,noexec,nosuid` — noexec temp directory
- `cap_drop: ALL` + `cap_add: CHOWN, SETGID, SETUID`
- `security_opt: no-new-privileges:true`

#### Runtime Restrictions

- Go sandbox: 60s timeout via `context.WithTimeout`
- HTTP fetches: 30s timeout
- Ollama embedding: 30s timeout
- gRPC health: 3s timeout per check

---

## 7. Mitigations Summary

| Mitigation | Scope | Status |
|-----------|-------|--------|
| Argon2id password hashing | Auth | ✅ |
| AES-256-GCM key encryption | Agent keys | ✅ |
| RBAC (admin/user/readonly) | API access | ✅ |
| Rate limiting per IP | All endpoints | ✅ |
| CSRF (Origin/Referer check) | State-changing requests | ✅ |
| CSP without unsafe-inline | Frontend | ✅ |
| SSRF validation with DNS | MCP, external calls | ✅ (partial — 7 HTTP clients unprotected) |
| Sandbox import blocklist | Tool execution | ✅ (partial — ExecuteTool gap) |
| File watcher + validation | Ingestion | ✅ |
| Docker network isolation | NLP sidecar | ✅ |
| govulncheck in CI | Go dependencies | ✅ |
| npm audit in CI | Frontend dependencies | ✅ |
| Trivy container scan in CI | Docker images | ✅ |

---

## 8. Open Risks

1. **Unprotected HTTP clients**: 7+ HTTP clients (agent.go, ollama.go, openai.go, anthropic.go, embed.go, notification.go, compiler_tool.go) lack SSRF validation. Wrap with `mcp.ValidateSSRF`.
2. **ExecuteTool gap**: `exec_sandbox.go:ExecuteTool` bypasses `ValidateGoCode`/`ValidatePythonCode`.
3. **NLP no auth**: Sidecar trusts all gRPC calls from Docker network.
4. **DuckDB no encryption**: Analytical data at rest is unencrypted.
5. **Command allowlist**: `curl` (SSRF) and `pip` (supply chain) allowed.
6. **RBAC role column**: Stored as plaintext. No role-based rate limiting yet.

---

## 9. Incident Response

### Detection Points
1. Sandbox validation failures logged at ERROR level
2. Container resource limit violations (OOM kills)
3. Unexpected network traffic from sandbox containers
4. CI security scan failures (govulncheck, npm audit, trivy)

### Response Procedures
1. Blocked import → reject execution and log incident
2. Container escape detection → stop all sandbox containers and investigate
3. Resource exhaustion → restart with tighter limits
4. CI vulnerability found → block merge, create security issue, patch within SLA

---

## 10. Review Cadence

This threat model should be reviewed:
- After any architecture change (new services, data stores, endpoints)
- After security incidents
- Quarterly as part of security review