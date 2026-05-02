# SPEC-06: API Security — CSP, HSTS, CSRF, Security Headers

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 3, tasks W3-1, W3-2, W3-3  
**Findings addressed**: A1-A10 (API security cluster)  
**Depends on**: `docs/specs/wave0-auth-spec.md` (JWT/SameSite cookies for CSRF token), `docs/specs/wave0-secrets-spec.md` (no secrets in client)  
**Related specs**: `docs/specs/wave3-frontend-spec.md` (CSP needs frontend asset hashes)  
**Status**: ✅ Approved — ready for execution

---

## 1. CSP Policy — Strict Mode

### Current (Vulnerable)

```
Content-Security-Policy: default-src 'self'; script-src 'self'; 
style-src 'self' 'unsafe-inline'; img-src 'self' data:; 
font-src 'self'; connect-src 'self' ws://localhost:8080 ws://localhost:5173; 
frame-ancestors 'none'; base-uri 'self'; form-action 'self'
```

### Target (Production - Strict)

```
Content-Security-Policy: default-src 'self';
script-src 'self';
style-src 'self';
img-src 'self' data:;
font-src 'self';
connect-src 'self';
frame-ancestors 'none';
object-src 'none';
worker-src 'self';
base-uri 'self';
form-action 'self';
upgrade-insecure-requests;
block-all-mixed-content;
```

### Key Changes

| Directive | Before | After | Reason |
|-----------|--------|-------|--------|
| `style-src` | `'self' 'unsafe-inline'` | `'self'` | Block CSS injection |
| `connect-src` | `'self' ws://localhost:8080 ws://localhost:5173` | `'self'` | Remove dev hosts from production |
| `object-src` | *(not set)* | `'none'` | Block plugin embedding |
| `worker-src` | *(not set)* | `'self'` | Restrict Web Workers |
| `upgrade-insecure-requests` | *(not set)* | *(present)* | Auto-upgrade HTTP→HTTPS |
| `block-all-mixed-content` | *(not set)* | *(present)* | Block mixed content |

### Frontend: vite-plugin-csp-guard

```ts
// vite.config.ts
import csp from "vite-plugin-csp-guard"
import { definePolicy, self } from "csp-toolkit"

export default defineConfig({
  plugins: [
    react(),
    csp({
      dev: { run: true },
      policy: definePolicy({
        "default-src": [self],
        "script-src": [self],
        "style-src": [self],
        "font-src": [self, "https://fonts.gstatic.com"],
        "img-src": [self, "data:"],
        "connect-src": [self],
        "frame-ancestors": ["'none'"],
        "object-src": ["'none'"],
        "worker-src": [self],
        "base-uri": [self],
        "form-action": [self],
      }),
      build: {
        sri: true  // Subresource Integrity
      }
    }),
  ],
})
```

### Go Server: Security Headers Middleware

```go
// internal/middleware/security.go
func SecurityHeaders() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Security-Policy",
                "default-src 'self'; "+
                "script-src 'self'; "+
                "style-src 'self'; "+
                "img-src 'self' data:; "+
                "font-src 'self'; "+
                "connect-src 'self'; "+
                "frame-ancestors 'none'; "+
                "object-src 'none'; "+
                "worker-src 'self'; "+
                "base-uri 'self'; "+
                "form-action 'self'; "+
                "upgrade-insecure-requests; "+
                "block-all-mixed-content; ")
            next.ServeHTTP(w, r)
        })
    }
}
```

### CSS-in-JS Strategy

The app uses Tailwind CSS (not styled-components), so removing `unsafe-inline` for styles is safe. Tailwind generates external CSS files via PostCSS.

---

## 2. HSTS Configuration

### Policy

```
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

- `max-age=31536000` → 1 year
- `includeSubDomains` → All subdomains also HTTPS-only
- `preload` → Eligible for browser HSTS preload list

### unrolled/secure Integration

```go
import "github.com/unrolled/secure"

func NewSecureMiddleware(isProduction bool) func(http.Handler) http.Handler {
    secureMiddleware := secure.New(secure.Options{
        // Host validation
        AllowedHosts:          []string{"aleph\\.example\\.com"},
        AllowedHostsAreRegex:  true,
        HostsProxyHeaders:     []string{"X-Forwarded-Host"},
        
        // HTTPS
        SSLRedirect:           isProduction,
        SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
        
        // HSTS (1 year)
        STSSeconds:            31536000,
        STSIncludeSubdomains:  true,
        STSPreload:            true,
        
        // Clickjacking
        FrameDeny:             true,
        
        // MIME sniffing
        ContentTypeNosniff:    true,
        
        // XSS filter (legacy browsers)
        BrowserXssFilter:      true,
        
        // Referrer policy
        ReferrerPolicy:        "strict-origin-when-cross-origin",
        
        // Permissions policy (disable unnecessary)
        PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
        
        // Dev mode
        IsDevelopment:         !isProduction,
    })
    
    return secureMiddleware.Handler
}
```

### Additional Security Headers

```go
// Added in SecurityHeaders middleware
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
```

### Middleware Order

```
Recovery → RequestID → SecurityHeaders (CSP + HSTS + X-* headers)
         → CSRFProtection → RateLimitMiddleware → AuthMiddleware
         → RBAC → Handler
```

---

## 3. CSRF Hardening

### Current Issues

1. No Origin/No Referer → **ALLOW** (bypass)
2. Referer matching → `strings.HasPrefix` (subdomain spoofing)

### Fix 1: Require Origin or Referer for Mutating Requests

```go
func CSRFProtection(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // GET, HEAD, OPTIONS — never CSRF
            if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
                next.ServeHTTP(w, r)
                return
            }
            
            origin := r.Header.Get("Origin")
            referer := r.Header.Get("Referer")
            
            // FIX: No Origin AND no Referer for mutating request → REJECT
            if origin == "" && referer == "" {
                http.Error(w, "Forbidden: missing Origin/Referer", http.StatusForbidden)
                return
            }
            
            // Check Origin
            if origin != "" {
                for _, allowed := range allowedOrigins {
                    if origin == allowed {  // EXACT match (not HasPrefix)
                        next.ServeHTTP(w, r)
                        return
                    }
                }
            }
            
            // Check Referer (exact origin extraction)
            if referer != "" {
                refURL, err := url.Parse(referer)
                if err == nil {
                    for _, allowed := range allowedOrigins {
                        if fmt.Sprintf("%s://%s", refURL.Scheme, refURL.Host) == allowed {
                            next.ServeHTTP(w, r)
                            return
                        }
                    }
                }
            }
            
            http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
        })
    }
}
```

### Fix 2: CSRF Token Pattern (for browser-based clients)

For SPA clients that can handle cookies, add double-submit cookie pattern:

```go
// Server: Set CSRF token cookie on first authenticated request
func SetCSRFMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Generate and set token if not present
            if _, err := r.Cookie("csrf_token"); err != nil {
                token := generateCSRFToken()
                http.SetCookie(w, &http.Cookie{
                    Name:     "csrf_token",
                    Value:    token,
                    Path:     "/",
                    Secure:   true,
                    HttpOnly: false,  // Client must read this
                    SameSite: http.SameSiteStrictMode,
                })
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Verify: X-CSRF-Token header matches cookie
func CSRFVerifyMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
                next.ServeHTTP(w, r)
                return
            }
            
            cookie, err := r.Cookie("csrf_token")
            if err != nil {
                http.Error(w, "Forbidden: missing CSRF token", http.StatusForbidden)
                return
            }
            
            header := r.Header.Get("X-CSRF-Token")
            if cookie.Value == "" || cookie.Value != header {
                http.Error(w, "Forbidden: CSRF token mismatch", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4. Rate Limiting (Auth-Specific)

See SPEC-01 (wave0-auth-spec.md) Section 6 for rate limiting architecture. In production, rate limiting should be applied at both levels:

1. **Application level** (Go): `krishna-kudari/ratelimit` for fine-grained per-endpoint limits
2. **Reverse proxy level** (Nginx): `limit_req_zone` for global anti-DDoS

---

## 5. Verification

### Test Coverage

- [ ] `security_test.go` (expand): CSP header content, no `unsafe-inline`, no `ws://localhost`
- [ ] `csrf_test.go` (expand): No-Origin POST → 403; valid Origin POST → 200; HasPrefix spoofing → 403
- [ ] `security_headers_test.go` (NEW): HSTS header present; all X-* headers present

### Manual Verification

```bash
# CSP: no unsafe-inline in production
curl -I https://localhost:8443 | grep Content-Security-Policy
# → No "unsafe-inline" in output

# HSTS: max-age=31536000; includeSubDomains; preload
curl -I https://localhost:8443 | grep Strict-Transport-Security
# → max-age=31536000; includeSubDomains; preload

# CSRF: no-origin POST → 403
curl -X POST http://localhost:8080/api/v1/agents -d '{}'
# → 403 Forbidden: missing Origin/Referer

# CSRF: valid origin POST → 200 (after auth)
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Origin: https://aleph.example.com" \
  -H "Cookie: aleph_jwt=..."
# → 200 OK

# Frame-ancestors: none
curl -I https://localhost:8443 | grep frame-ancestors
# → frame-ancestors 'none'
```
