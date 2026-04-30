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
