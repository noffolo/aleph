package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

const (
	HeaderApiKey = "X-Aleph-Api-Key"
)

func NewAuthInterceptor(metaRepo *repository.MetadataRepository) connect.Interceptor {
	return &authInterceptor{metaRepo: metaRepo}
}

type authInterceptor struct {
	metaRepo *repository.MetadataRepository
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure
		if strings.Contains(procedure, "ListProjects") || 
		   strings.Contains(procedure, "CreateProject") ||
		   strings.Contains(procedure, "AuthService") {
			return next(ctx, req)
		}

		apiKey := req.Header().Get(HeaderApiKey)
		if apiKey == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing API Key"))
		}

		if err := i.validateKey(apiKey); err != nil {
			return nil, err
		}

		return next(ctx, req)
	}
}

func (i *authInterceptor) validateKey(apiKey string) error {
	if len(apiKey) < 8 {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid API Key format"))
	}
	id := apiKey[:8]
	
	hsh := sha256.Sum256([]byte(apiKey))
	incomingHash := hex.EncodeToString(hsh[:])

	var storedHash string
	err := i.metaRepo.DB().QueryRow("SELECT key FROM system_api_keys WHERE id = ?", id).Scan(&storedHash)
	if err != nil || subtle.ConstantTimeCompare([]byte(storedHash), []byte(incomingHash)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid API Key"))
	}
	return nil
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		procedure := conn.Spec().Procedure
		if strings.Contains(procedure, "Chat") {
			apiKey := conn.RequestHeader().Get(HeaderApiKey)
			if apiKey == "" {
				return connect.NewError(connect.CodeUnauthenticated, errors.New("missing API Key"))
			}
			if err := i.validateKey(apiKey); err != nil {
				return err
			}
		}
		return next(ctx, conn)
	}
}
