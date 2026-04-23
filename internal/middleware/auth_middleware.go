package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

var (
	errNoApiKey     = errors.New("missing api key: provide X-Aleph-Api-Key header")
	errInvalidApiKey = errors.New("invalid api key")
)

type ctxKey string

const ctxKeyProjectID ctxKey = "projectID"

type AuthInterceptor struct {
	metaRepo *repository.MetadataRepository
}

func NewAuthInterceptor(metaRepo *repository.MetadataRepository) *AuthInterceptor {
	return &AuthInterceptor{metaRepo: metaRepo}
}

func ProjectIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyProjectID).(string); ok {
		return v
	}
	return ""
}

func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if skipAuth(req.Spec().Procedure) {
			return next(ctx, req)
		}

		apiKey := extractApiKey(req.Header())
		if apiKey == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, errNoApiKey)
		}

		projectID, err := a.validateKey(apiKey)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = context.WithValue(ctx, ctxKeyProjectID, projectID)
		return next(ctx, req)
	}
}

func (a *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if skipAuth(conn.Spec().Procedure) {
			return next(ctx, conn)
		}

		apiKey := extractApiKey(conn.RequestHeader())
		if apiKey == "" {
			return connect.NewError(connect.CodeUnauthenticated, errNoApiKey)
		}

		projectID, err := a.validateKey(apiKey)
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = context.WithValue(ctx, ctxKeyProjectID, projectID)
		return next(ctx, conn)
	}
}

func (a *AuthInterceptor) validateKey(apiKey string) (string, error) {
	h := sha256.New()
	h.Write([]byte(apiKey))
	hashed := hex.EncodeToString(h.Sum(nil))

	projectID, err := a.metaRepo.ValidateAPIKey(hashed)
	if err != nil {
		return "", errInvalidApiKey
	}
	return projectID, nil
}

func extractApiKey(h http.Header) string {
	if key := h.Get("X-Aleph-Api-Key"); key != "" {
		return key
	}
	if auth := h.Get("Authorization"); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func skipAuth(procedure string) bool {
	return strings.Contains(procedure, "AuthService")
}
