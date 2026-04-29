package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// ErrNoAPIKey is returned when no API key is provided in the request.
var ErrNoAPIKey = errors.New("missing api key: provide X-Aleph-Api-Key header")

// ErrInvalidAPIKey is returned when the provided API key is invalid.
var ErrInvalidAPIKey = errors.New("invalid api key")

type authCtxKey string

const authCtxProjectID authCtxKey = "projectID"

// ValidateAPIKey validates an API key against the repository using argon2id.
// The first 8 characters of the API key are used as the key ID to look up
// the stored argon2id hash, then the full key is verified against that hash.
// Returns the associated project ID on success.
func ValidateAPIKey(metaRepo *repository.MetadataRepository, apiKey string) (string, error) {
	if len(apiKey) < 8 {
		return "", ErrInvalidAPIKey
	}

	keyID := apiKey[:8]

	storedHash, projectID, err := metaRepo.GetAPIKeyByID(keyID)
	if err != nil {
		return "", ErrInvalidAPIKey
	}

	ok, err := auth.VerifyAPIKey(apiKey, storedHash)
	if err != nil || !ok {
		return "", ErrInvalidAPIKey
	}

	return projectID, nil
}

// ProjectIDFromContext retrieves the authenticated project ID from the context.
// Returns empty string if not found.
func ProjectIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authCtxProjectID).(string); ok {
		return v
	}
	return ""
}

// projectIDToContext stores the project ID in the request context.
func projectIDToContext(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, authCtxProjectID, projectID)
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

// AuthMiddleware returns an HTTP middleware that validates the API key from
// the X-Aleph-Api-Key header (or Authorization: Bearer) against the repository.
// On success, the project ID is stored in the request context and can be
// retrieved via ProjectIDFromContext. On failure, a 401 response is returned.
func AuthMiddleware(metaRepo *repository.MetadataRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := ExtractAPIKeyFromHeader(r)
		if apiKey == "" {
			apiKey = ExtractAPIKeyFromCookie(r)
		}
		if apiKey == "" {
			http.Error(w, ErrNoAPIKey.Error(), http.StatusUnauthorized)
			return
		}

		projectID, err := ValidateAPIKey(metaRepo, apiKey)
		if err != nil {
			http.Error(w, ErrInvalidAPIKey.Error(), http.StatusUnauthorized)
			return
		}

		ctx := projectIDToContext(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
