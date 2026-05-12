package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// Role represents the access level of an authenticated request.
type Role string

const (
	// RoleAdmin has full access to all endpoints.
	RoleAdmin Role = "admin"
	// RoleUser can read and write data but cannot manage keys or system settings.
	RoleUser Role = "user"
	// RoleReadOnly can only read data.
	RoleReadOnly Role = "readonly"
)

// ErrNoAPIKey is returned when no API key is provided in the request.
var ErrNoAPIKey = errors.New("missing api key: provide X-Aleph-Api-Key header")

// ErrInvalidAPIKey is returned when the provided API key is invalid.
var ErrInvalidAPIKey = errors.New("invalid api key")

// ErrForbidden is returned when the role lacks permission for the operation.
var ErrForbidden = errors.New("insufficient permissions for this operation")

type authCtxKey string

const (
	authCtxProjectID authCtxKey = "projectID"
	authCtxRole      authCtxKey = "role"
)

// ValidateAPIKey validates an API key against the repository using argon2id.
// The first 8 characters of the API key are used as the key ID to look up
// the stored argon2id hash, then the full key is verified against that hash.
// Returns the associated project ID and role on success.
func ValidateAPIKey(metaRepo *repository.MetadataRepository, apiKey string) (string, Role, error) {
	if len(apiKey) < 8 {
		return "", "", ErrInvalidAPIKey
	}

	// Bootstrap key check: if the key matches ALEPH_API_KEY_SECRET_BACKEND,
	// skip DB lookup (the bootstrap key is not stored in the database).
	if role := bootstrapRole(apiKey); role != "" {
		return "bootstrap", role, nil
	}

	keyID := apiKey[:8]

	storedHash, projectID, role, err := metaRepo.GetAPIKeyByID(keyID)
	if err != nil {
		return "", "", ErrInvalidAPIKey
	}

	ok, err := auth.VerifyAPIKey(apiKey, storedHash)
	if err != nil || !ok {
		return "", "", ErrInvalidAPIKey
	}

	// Derive effective role: stored DB role takes precedence.
	effectiveRole := Role(role)
	if effectiveRole == "" {
		effectiveRole = roleFromEnv(apiKey)
	}

	return projectID, effectiveRole, nil
}

// roleFromEnvFn is the function variable for determining role from env/prefix.
// Overridable in tests.
var roleFromEnvFn = roleFromEnvImpl

// roleFromEnv determines role using the overridable function.
func roleFromEnv(apiKey string) Role {
	return roleFromEnvFn(apiKey)
}

// bootstrapRole checks if the key matches the ALEPH_API_KEY_SECRET_BACKEND env var.
func bootstrapRole(apiKey string) Role {
	return roleFromEnvImpl(apiKey)
}

func roleFromEnvImpl(apiKey string) Role {
	backendKey := os.Getenv("ALEPH_API_KEY_SECRET_BACKEND")
	if backendKey != "" && apiKey == backendKey {
		return RoleAdmin
	}
	if strings.HasPrefix(apiKey, "user_") {
		return RoleUser
	}
	if strings.HasPrefix(apiKey, "ro_") {
		return RoleReadOnly
	}
	return RoleUser
}

// ProjectIDFromContext retrieves the authenticated project ID from the context.
// Returns empty string if not found.
func ProjectIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authCtxProjectID).(string); ok {
		return v
	}
	return ""
}

// RoleFromContext retrieves the authenticated role from the context.
// Returns RoleUser if no role is found (safe default).
func RoleFromContext(ctx context.Context) Role {
	if v, ok := ctx.Value(authCtxRole).(Role); ok {
		return v
	}
	return RoleUser
}

// projectIDToContext stores the project ID and role in the request context.
func projectIDToContext(ctx context.Context, projectID string, role Role) context.Context {
	ctx = context.WithValue(ctx, authCtxProjectID, projectID)
	ctx = context.WithValue(ctx, authCtxRole, role)
	return ctx
}

// RequireRole returns an error if the context role is not in the allowed set.
func RequireRole(ctx context.Context, allowed ...Role) error {
	current := RoleFromContext(ctx)
	for _, r := range allowed {
		if current == r {
			return nil
		}
	}
	return ErrForbidden
}

// RequireRoleHTTP returns HTTP middleware that enforces RBAC for chi/mux routes.
// It extracts the role from context (set by AuthMiddleware) and returns 403
// if the role is not in the allowed set.
func RequireRoleHTTP(allowed ...Role) func(http.Handler) http.Handler {
	allowedSet := make(map[Role]bool, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if !allowedSet[role] {
				http.Error(w, ErrForbidden.Error(), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireProjectRole ensures the project in the URL matches the authenticated project.
// Returns 403 if the projects don't match.
func RequireProjectRole() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authProjectID := ProjectIDFromContext(r.Context())
			if authProjectID == "" {
				next.ServeHTTP(w, r)
				return
			}
			urlProject := r.PathValue("project_id")
			if urlProject == "" {
				urlProject = r.URL.Query().Get("project_id")
			}
			if urlProject != "" && urlProject != authProjectID {
				http.Error(w, "project access denied", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IsAdmin returns true if the context role is admin.
func IsAdmin(ctx context.Context) bool {
	return RoleFromContext(ctx) == RoleAdmin
}

// ExtractAPIKeyFromHeader extracts an API key from the request headers.
// Checks X-Aleph-Api-Key first, then Authorization: Bearer.
func ExtractAPIKeyFromHeader(r *http.Request) string {
	return ExtractAPIKey(r.Header)
}

// ExtractAPIKeyFromCookie extracts an API key from the aleph_session cookie.
func ExtractAPIKeyFromCookie(r *http.Request) string {
	c, err := r.Cookie("aleph_session")
	if err != nil {
		return ""
	}
	return c.Value
}

// ExtractAPIKey extracts an API key from an http.Header.
// Checks X-Aleph-Api-Key first, then Authorization: Bearer.
func ExtractAPIKey(h http.Header) string {
	if key := h.Get("X-Aleph-Api-Key"); key != "" {
		return key
	}
	if auth := h.Get("Authorization"); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// AuthMiddleware returns an HTTP middleware that validates authentication from
// JWT cookie (aleph_jwt) or legacy X-Aleph-Api-Key header (deprecated).
// On success, the project ID and role are stored in the request context via
// ProjectIDFromContext and RoleFromContext. On failure, a 401 response is returned.
func AuthMiddleware(metaRepo *repository.MetadataRepository, jwtSecret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, role, err := authenticateHTTP(r, metaRepo, jwtSecret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := projectIDToContext(r.Context(), projectID, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateHTTP resolves auth from JWT cookie, then legacy API key (deprecated).
func authenticateHTTP(r *http.Request, metaRepo *repository.MetadataRepository, jwtSecret []byte) (string, Role, error) {
	// 1. JWT from aleph_jwt cookie
	if jwtStr := jwtFromCookie(r.Header); jwtStr != "" {
		return validateJWT(jwtStr, jwtSecret)
	}

	// 2. Legacy API key (deprecated)
	apiKey := ExtractAPIKeyFromHeader(r)
	if apiKey == "" {
		apiKey = ExtractAPIKeyFromCookie(r)
	}
	if apiKey != "" {
		return ValidateAPIKey(metaRepo, apiKey)
	}

	return "", "", ErrNoAPIKey
}
