# Aleph-v2 Security Model

## Authentication
- **API Key**: `X-Aleph-Api-Key` header, `Authorization: Bearer` header, or `aleph_session` httpOnly cookie
- **API Key Storage**: Argon2id hashed (SHA-256 legacy fallback detected via `hash_algorithm` column). First 8 hex chars = key ID.
- **Session**: httpOnly+Secure+SameSite=Strict cookie `aleph_session` (POST/DELETE `/api/v1/auth/session`)
- **SSE**: Validated via `isAuthenticatedForSSE()` — validates `X-Aleph-Api-Key` header against repository
- **Encryption-at-rest**: API keys encrypted with AES-256-GCM via `KEY_ENCRYPTION_KEY` env var (64 hex chars) — **required**, startup FATAL without it

## Authorization
- Auth middleware (`AuthMiddleware` + `AuthInterceptor`) checks every Connect RPC call + raw HTTP routes
- SSE endpoint fail-closed (returns 401 without valid key)
- CORS restricted to explicitly configured origins (`CORS_ALLOWED_ORIGINS` env var)

## Web Security (W7)
- **CSP**: `default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws://localhost:*;` — no `unsafe-inline` in style-src
- **CSRF**: Origin/Referer validation middleware on all non-GET requests (allows CLI/internal clients without headers)
- **Rate Limiting**: Per-IP via `golang.org/x/time/rate`. Uses `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` chain. Per-endpoint categories: Chat (10/min), Health (100/min), Default (500/min)
- **Security Headers**: X-Content-Type-Options: nosniff, X-Frame-Options: DENY, Referrer-Policy: same-origin

## Network Security
- CORS: explicit allowed origins, explicit header whitelist (`Content-Type`, `Authorization`, `X-Aleph-Api-Key`, `X-Request-Id`, `X-Project-Id`)
- NLP gRPC: cleartext (h2c only, TLS-ready for production)
- SSRF: Unified `internal/ssrf/validator.go` — DNS-resolving DialContext, redirect re-validation, TLS 1.2 min, blocks private IPs and bypass forms

## Data Protection
- SQL injection defense: parameterized queries (`$1`/`$2`/`?`) + `validName()` regex check on identifiers (`^[a-zA-Z_][a-zA-Z0-9_]*$`)
- AES-256-GCM encryption for API keys at rest via `internal/crypto/aesgcm.go`
- Input size limits on all HTTP endpoints
- Audit logging for all tool operations (`internal/middleware/audit.go`)
- Sandbox code execution blocked via import blocklist (`os/exec`, `syscall`, `net`, `unsafe`, `reflect`, etc.)

## Release Gate
See `docs/release-checklist.md` for the full release verification checklist (build, security, Docker, CI/CD).

---

## Responsible Disclosure Policy

Aleph-v2 takes security seriously. If you discover a security vulnerability,
please follow this disclosure process.

### Reporting a Vulnerability

**Do not open a public GitHub issue.** Instead, send your report to:

```
🔒 security@aleph-dataos.dev
```

If you need encrypted communication, use our PGP key:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
[PGP key placeholder — obtain the real key from the maintainer]
-----END PGP PUBLIC KEY BLOCK-----
```

### What to Include

Please provide as much of the following as possible:

1. **Description** — What type of vulnerability (XSS, SQLi, RCE, etc.)
2. **Steps to reproduce** — Minimal, complete, reproducible steps or a proof of concept
3. **Impact** — What an attacker could achieve
4. **Affected versions** — Git commit hash, release tag, or version number
5. **Suggested fix** — If you have one (optional but appreciated)
6. **Your name/alias** — For acknowledgment (optional)

### Response Timeline

| Phase | Timeframe |
|-------|-----------|
| Acknowledgment | Within 48 hours of report |
| Initial assessment | Within 5 business days |
| Fix development | Depends on severity (typically 7-30 days) |
| Fix release | Coordinated with reporter |
| Public disclosure | After fix is deployed |

### Scope

**In scope:**
- The Go backend (Aleph-v2 server)
- The React/TypeScript frontend
- The Python NLP sidecar
- Docker configurations and deployment scripts

**Out of scope:**
- Third-party services (Ollama, PostgreSQL, DuckDB — report to their maintainers)
- Dependency vulnerabilities already tracked by govulncheck / Trivy
- Theoretical attacks without practical exploit
- Social engineering attacks against project maintainers

### Safe Harbor

We participate in coordinated disclosure. We will not pursue legal action
against researchers who:

- Follow this disclosure policy
- Make a good-faith effort to avoid privacy violations, data destruction,
  and service interruption
- Do not exploit vulnerabilities beyond what is necessary to demonstrate
  the issue
- Give us reasonable time to fix the issue before public disclosure

### Vulnerability Disclosure Process

1. **Report received** → Acknowledged within 48h
2. **Triage** → Severity assessment within 5 business days
3. **Fix** → Developed in a private fork
4. **Release** → Security advisory published with CVE (if applicable)
5. **Credit** → Reporter acknowledged in CHANGELOG (unless anonymity requested)

### Security Advisories

Security advisories are published on the GitHub repository's
[Security Advisories](https://github.com/ff3300/aleph-v2/security/advisories)
page and tagged releases.

---

## Audit Reports

Security audit reports are maintained in the `audit/` directory:

| Report | Date | Tool |
|--------|------|------|
| `audit/gosec-report.md` | 2026-05-02 | gosec v2.26.1 |
| `audit/govulncheck-report.md` | 2026-05-02 | govulncheck v1.3.0 |
| `audit/trivy-report.md` | 2026-05-02 | Trivy v0.69.3 |
| `audit/zap-config.md` | 2026-05-02 | Configuration guide |

---

*Last updated: 2 May 2026*
