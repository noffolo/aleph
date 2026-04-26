# Aleph-v2 Security Model

## Authentication
- **API Key**: `X-Aleph-Api-Key` header or `api_key` query param
- **SSE**: Validated via `isAuthenticatedForSSE()` — rejects keys shorter than 6 chars or without `aleph_` prefix
- **Encryption-at-rest**: API keys encrypted with AES-256-GCM via `KEY_ENCRYPTION_KEY` env var (64 hex chars) — **required**, startup fails without it

## Authorization
- Auth middleware checks every Connect RPC call
- SSE endpoint now fail-closed (returns 401 without valid key)
- CORS restricted to explicitly configured origins (`CORS_ALLOWED_ORIGINS` env var)

## Network Security
- CORS: explicit allowed origins, explicit header whitelist (`Content-Type`, `Authorization`, `X-Aleph-Api-Key`, `X-Request-Id`, `X-Project-Id`)
- NLP gRPC: cleartext (h2c only, TLS-ready for production)
- Sandbox: Docker `network_mode: none`, `read_only: true` with command and path blocklists

## Data Protection
- SQL injection defense: parameterized queries + `validName()` regex check on identifiers
- Input size limits on all HTTP endpoints
- Audit logging for all tool operations

## Migration Security
- DuckDB + PostgreSQL migrations run on startup
- Root `migrations/` directory marked dead/kept for reference
