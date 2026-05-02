# SPEC-01: Auth Hardening — RBAC, JWT, Middleware Chain

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 0  
**Findings addressed**: R1-R10 (auth cluster), A5 (rate limiting)  
**Related specs**: `docs/specs/wave0-secrets-spec.md` (secrets shared with auth flow), `docs/specs/wave3-api-spec.md` (CSP/HSTS middleware chain)  
**Status**: ✅ Approved — ready for execution

---

## 1. RBAC Role Matrix

### Role Definitions (unchanged from current)

| Role | Type | Description |
|------|------|-------------|
| `RoleAdmin` | `"admin"` | Full project access: CRUD agents/tools/skills, manage API keys, delete projects |
| `RoleUser` | `"user"` | Standard access: create/read agents, execute tools, query data. Cannot manage keys or delete projects |
| `RoleReadOnly` | `"readonly"` | Read-only access: list agents/tools/skills, read data. Cannot create/modify/delete |

### Role Resolution

Roles are resolved in `internal/middleware/auth.go:roleFromEnvImpl` (line 81-93):
- `ALEPH_API_KEY_SECRET_BACKEND` env key → `RoleAdmin`
- Key prefix `user_` → `RoleUser`  
- Key prefix `ro_` → `RoleReadOnly`
- Default → `RoleUser`

### Permission Matrix

| Operation | Admin | User | ReadOnly | Unauthenticated |
|-----------|-------|------|----------|-----------------|
| `CreateApiKey` | ✅ | ❌ | ❌ | ❌ |
| `ListApiKeys` | ✅ | ❌ | ❌ | ❌ |
| `RevokeApiKey` | ✅ | ❌ | ❌ | ❌ |
| `DeleteProjectCascade` | ✅ | ❌ | ❌ | ❌ |
| `CreateAgent` | ✅ | ✅ | ❌ | ❌ |
| `UpdateAgent` | ✅ | ✅ | ❌ | ❌ |
| `DeleteAgent` | ✅ | ✅ | ❌ | ❌ |
| `CreateTool` | ✅ | ✅ | ❌ | ❌ |
| `RegisterComponent` | ✅ | ✅ | ❌ | ❌ |
| `UpdateComponentStatus` | ✅ | ✅ | ❌ | ❌ |
| `RunQuery` | ✅ | ✅ | ✅ | ❌ |
| `ListAgents` | ✅ | ✅ | ✅ | ❌ |
| `ListTools` | ✅ | ✅ | ✅ | ❌ |
| `GetComponentByID` | ✅ | ✅ | ✅ | ❌ |
| `Chat` | ✅ | ✅ | ✅ | ❌ |
| `StreamSSE` | ✅ | ✅ | ✅ | ❌ |

---

## 2. RBAC Enforcement Architecture

### Enforcement Points

```
Request → AuthMiddleware (JWT/API key validation)
       → Inject claims into context (userID, projectID, role)
       → RequireRole middleware (route-level)
       → Handler (NO auth checks — context is pre-validated)
```

### Implementation

```go
// internal/middleware/auth.go — already exists, wire it

// Route-level enforcement (chi router pattern)
r.Group(func(r chi.Router) {
    r.Use(RequireRole(middleware.RoleAdmin))
    r.Post("/api/v1/auth/apikeys", handler.CreateApiKey)
    r.Delete("/api/v1/auth/apikeys/{id}", handler.RevokeApiKey)
})

r.Group(func(r chi.Router) {
    r.Use(RequireRole(middleware.RoleAdmin, middleware.RoleUser))
    r.Post("/api/v1/agents", handler.CreateAgent)
    r.Put("/api/v1/agents/{id}", handler.UpdateAgent)
})

// Read endpoints — explicit, not default
r.Group(func(r chi.Router) {
    r.Use(RequireRole(middleware.RoleAdmin, middleware.RoleUser, middleware.RoleReadOnly))
    r.Get("/api/v1/agents", handler.ListAgents)
})
```

### ConnectRPC Enforcement

ConnectRPC uses interceptors, not chi middleware. RBAC check goes in `AuthInterceptor.WrapUnary` after authentication:

```go
// internal/middleware/auth_middleware.go
func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        // ... existing auth logic ...
        
        // RBAC check
        requiredRole := rbacRequiredRole(req.Spec().Procedure)
        if requiredRole != "" {
            if err := RequireRole(ctx, requiredRole); err != nil {
                return nil, connect.NewError(connect.CodePermissionDenied, err)
            }
        }
        
        return next(ctx, req)
    }
}

// Procedure → required role mapping
func rbacRequiredRole(procedure string) string {
    switch {
    case strings.HasPrefix(procedure, "/aleph.v1.AuthService/"):
        return RoleAdmin
    case isWriteOperation(procedure):
        return RoleUser  // or RoleAdmin — will be checked with RequireRole(RoleAdmin, RoleUser)
    default:
        return ""  // read operations — any authenticated role
    }
}
```

---

## 3. authSkipSet Fix

### Current (Vulnerable)

```go
var authSkipSet = map[string]bool{
    "/aleph.v1.AuthService/ListApiKeys":   true,
    "/aleph.v1.AuthService/CreateApiKey":  true,
    "/aleph.v1.AuthService/RevokeApiKey":  true,
}
```

### Fixed (Security)

```go
var authSkipSet = map[string]bool{}  // Empty — no endpoints bypass auth
```

### Handler Changes

`internal/api/handler/auth.go` — remove internal validation fallback:
- `CreateApiKey`: Remove `req.Msg.ProjectId` direct read → use `middleware.ProjectIDFromContext(ctx)`
- `ListApiKeys`: Same
- `DeleteApiKey`: Same  
- All three must have `RequireRole(RoleAdmin)` applied

---

## 4. JWT Claims Contract

### SessionToken (before)

```go
type SessionToken struct {
    UserID    string `json:"user_id"`
    ProjectID string `json:"project_id"`
    Role      string `json:"role"`
    Scopes    string `json:"scopes,omitempty"`
    jwt.RegisteredClaims
}
```

### SessionToken (after)

```go
type SessionToken struct {
    UserID    string `json:"user_id"`
    ProjectID string `json:"project_id"`
    Role      string `json:"role"`
    Scopes    string `json:"scopes,omitempty"`
    jwt.RegisteredClaims  // Uses: Issuer, Subject, Audience, ID, IssuedAt, ExpiresAt
}
```

### Required Claims

| Claim | Value | Validation |
|-------|-------|-----------|
| `iss` | `"aleph-v2"` | Must equal `"aleph-v2"` |
| `sub` | User identifier | Must be non-empty |
| `aud` | `["aleph-v2-api"]` | Must contain `"aleph-v2-api"` |
| `jti` | UUID v4 | Must be unique; checked against revocation list |
| `exp` | 1 hour from issuance | Standard `RegisteredClaims` expiry check |
| `iat` | Issuance timestamp | Standard |
| `nbf` | Issuance timestamp | Standard |

### Revocation List

```go
type TokenRevocationStore struct {
    mu       sync.RWMutex
    revoked  map[string]time.Time  // jti → revocation time
    ttl      time.Duration          // 1 hour (matches JWT TTL)
}

// Background cleanup: remove expired entries every 5 minutes
```

### Scopes Validation

```go
func ValidateScopes(required string, token string) bool {
    requiredSet := strings.Split(required, ",")
    tokenSet := strings.Split(token, ",")
    for _, r := range requiredSet {
        found := false
        for _, t := range tokenSet {
            if strings.TrimSpace(r) == strings.TrimSpace(t) {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    return true
}
```

---

## 5. SSE Auth Integration

### Route Registration (current)

```go
mux.HandleFunc("/api/v1/events", cfg.SSEHandler.Stream)
```

### Route Registration (fixed)

```go
sseGroup := r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(jwtKey))          // JWT + API key validation
    r.Use(RequireRole(RoleAdmin, RoleUser, RoleReadOnly))
    r.Use(RateLimitMiddleware(sseLimiter)) // 2 conns/IP, 100 events/min
    r.Use(SecurityHeaders)                 // CSP, X-Frame-Options
})
sseGroup.Get("/api/v1/events", cfg.SSEHandler.Stream)
```

### SSE Rate Limits

- Connections: 2 per IP address
- Events: 100 per minute per connection
- Connection timeout: 24h (then client must reconnect)

### SSE Authentication Flow (in handler)

```go
func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
    // Claims already injected by AuthMiddleware into context
    claims := middleware.ClaimsFromContext(r.Context())
    
    // SSE connection tracking
    if err := h.connectionTracker.Acquire(r.Context(), claims.ProjectID); err != nil {
        http.Error(w, "Too many SSE connections", http.StatusTooManyRequests)
        return
    }
    defer h.connectionTracker.Release(claims.ProjectID)
    
    // ... existing SSE logic ...
}
```

---

## 6. Rate Limiting Architecture

### Auth Endpoint Limits

| Endpoint | Limit | Key |
|----------|-------|-----|
| `POST /api/v1/auth/session` | 5 req/min | IP |
| `POST /aleph.v1.AuthService/CreateApiKey` | 10 req/min | IP + projectID |
| `POST /aleph.v1.AuthService/RevokeApiKey` | 10 req/min | IP + projectID |
| `GET /aleph.v1.AuthService/ListApiKeys` | 30 req/min | IP + projectID |

### Implementation

```go
// krishna-kudari/ratelimit with Redis backend
authLimiter, _ := goratelimit.New(
    redisURL,
    goratelimit.PerMinute(5),
    goratelimit.WithAlgorithm(goratelimit.SlidingWindow),
)

// Wire via middleware
mux.Post("/api/v1/auth/session", middleware.RateLimit(authLimiter, 
    middleware.KeyByRealIP)(sessionHandler.Create))
```

---

## 7. Verification

### Test Coverage

- [ ] `auth_test.go`: RBAC enforcement for admin/user/readonly/unauthenticated
- [ ] `auth_middleware_test.go`: authSkipSet empty, all endpoints require auth
- [ ] `jwt_test.go`: aud/sub/iss/jti validation, revocation, expired token rejection
- [ ] `ratelimit_test.go`: 6th request within 1 minute → 429
- [ ] `auth_integration_test.go`: Full middleware chain with all role tiers

### Manual Verification

```bash
# authSkipSet: create API key without auth → 401
curl -X POST http://localhost:8080/aleph.v1.AuthService/CreateApiKey -d '{"project_id":"test"}' -H "Content-Type: application/json"

# RBAC: readonly user tries to create agent → 403
curl -X POST http://localhost:8080/api/v1/agents -H "Authorization: Bearer <readonly_jwt>"

# JWT: expired token → 401 (not 500)
curl http://localhost:8080/api/v1/agents -H "Authorization: Bearer <expired_jwt>"

# Rate limit: 6 rapid login attempts → 429
for i in $(seq 1 6); do curl -X POST http://localhost:8080/api/v1/auth/session; done
```

### Gate

```
go test -race -count=1 ./internal/middleware/ ./internal/auth/ ./internal/api/handler/
npx vitest run src/store/
```
All pass. Manual curl checks pass.
