package middleware

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type AuthInterceptor struct {
	metaRepo *repository.MetadataRepository
}

func NewAuthInterceptor(metaRepo *repository.MetadataRepository) *AuthInterceptor {
	return &AuthInterceptor{metaRepo: metaRepo}
}

func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if skipAuth(req.Spec().Procedure) {
			return next(ctx, req)
		}

		apiKey := extractApiKey(req.Header())
		if apiKey == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoAPIKey)
		}

		projectID, err := a.validateKey(apiKey)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = projectIDToContext(ctx, projectID)
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
			return connect.NewError(connect.CodeUnauthenticated, ErrNoAPIKey)
		}

		projectID, err := a.validateKey(apiKey)
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = projectIDToContext(ctx, projectID)
		return next(ctx, conn)
	}
}

func (a *AuthInterceptor) validateKey(apiKey string) (string, error) {
	return ValidateAPIKey(a.metaRepo, apiKey)
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
