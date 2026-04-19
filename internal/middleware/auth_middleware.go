package middleware

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/auth"
)

func NewAuthMiddleware(authService *auth.AuthService) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := strings.TrimPrefix(req.Header().Get("Authorization"), "Bearer ")
			valid, projectID, err := authService.ValidateAPIKey(token)
			if err != nil || !valid {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
			}
			ctx = context.WithValue(ctx, "projectID", projectID)
			return next(ctx, req)
		}
	}
}
