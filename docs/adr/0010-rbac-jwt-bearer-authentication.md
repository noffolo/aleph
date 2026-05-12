# ADR-0010: RBAC + JWT Bearer Authentication

## Status

Accepted

## Context

Aleph Data OS API requires authentication for all operations except health checks and public endpoints. Additionally, multi-tenant usage (multiple users/projects on the same instance) requires role-based access control — not all users should have administrative privileges.

Requirements:

- **Stateless auth**: The API serves tool execution clients that cannot maintain session state
- **API-key compatible**: Programmatic access via Bearer token in HTTP headers
- **Role granularity**: At minimum: admin (full system control), user (normal operations), viewer (read-only)
- **Route-level authorization**: Different routes require different roles
- **Token lifecycle**: Short-lived access tokens with refresh capability
- **Multi-tenant project isolation**: Users should only access projects they are members of

Options considered:

| Option | Stateful | API Compatible | Role Support | Complexity |
|--------|----------|----------------|--------------|------------|
| Session cookies | Yes | No | Manual | Low |
| OAuth2 + PKCE | No | Yes | Via scopes | High |
| JWT Bearer | No | Yes | Via claims | Medium |
| API Key only | No | Yes | No | Low |

## Decision

Use **JWT Bearer tokens with RS256 signing** for authentication, combined with **role-based access control** for authorization.

### Authentication Flow

1. **Login**: User submits credentials → backend validates → issues access token (15min) + refresh token (7d)
2. **Access**: All protected API calls include `Authorization: Bearer <jwt>` header
3. **Validation**: `AuthMiddleware` validates JWT signature, checks expiry, extracts claims
4. **Refresh**: `POST /api/auth/refresh` with valid refresh token issues new access token
5. **Logout**: Refresh token revoked server-side (blacklist)

### JWT Structure

```json
{
  "sub": "user-uuid",
  "role": "admin|user|viewer",
  "project_ids": ["proj-uuid-1", "proj-uuid-2"],
  "iat": 1700000000,
  "exp": 1700000900
}
```

### Roles

| Role | Permissions |
|------|------------|
| `admin` | Full system access, user management, configuration changes |
| `user` | Normal operations: create tools, execute queries, manage own agents |
| `viewer` | Read-only: view dashboards, inspect tool outputs, no mutations |

### Authorization Pipeline

Middleware stack (applied in order):
1. `AuthMiddleware` — Validates JWT, populates request context with claims
2. `RoleMiddleware` — Checks if the user's role satisfies the route's required role
3. `ProjectMiddleware` — (optional) Verifies project membership for project-scoped routes

### Implementation

- JWT generation and validation: `internal/auth/` using `github.com/golang-jwt/jwt/v5`
- Middleware: `internal/middleware/auth.go`
- Refresh token blacklist: in-memory with Redis planned for multi-instance deployments
- API key support: long-lived API keys (AES-256-GCM encrypted) that map to a user identity and role

## Consequences

### Positive
- Stateless authentication — no session store required for access tokens
- API-key compatible via Bearer header — works with any HTTP client
- Role checks at middleware layer — consistent enforcement across all routes
- Fine-grained access control per route and per project
- Short-lived access tokens (15min) limit exposure window if a token leaks

### Negative
- Token revocation requires a blacklist (or short TTLs) — JWT cannot be invalidated server-side
- JWT size overhead for large claim sets (e.g., many project IDs)
- Refresh token rotation adds complexity — old refresh tokens must be tracked
- RS256 signing requires public/private key management
- Middleware stacking adds per-request latency (validating JWT + checking roles)

## Compliance

- All protected routes behind `AuthMiddleware` middleware in middleware stack
- Roles checked via `RoleMiddleware` with route-specific required role annotations
- JWT generation and signing via `internal/auth/` package
- Refresh token endpoint at `POST /api/auth/refresh`
- New routes specify required role in route registration
- No unprotected mutation endpoints (except public health check at `/healthz`)
- API keys stored encrypted via `internal/crypto/` (see ADR-0008)

## Notes

- JWT library: https://github.com/golang-jwt/jwt
- OWASP JWT best practices: https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html
- Related ADRs: ADR-0002 (ConnectRPC over HTTP/2), ADR-0008 (Argon2id + AES-256-GCM for Security)
