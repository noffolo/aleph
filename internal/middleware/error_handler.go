package middleware

import (
	"context"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/errors"
)

// ErrorHandlerInterceptor is a ConnectRPC interceptor that catches unhandled errors
// and wraps them in structured APIError types for consistent error responses.
type ErrorHandlerInterceptor struct{}

// NewErrorHandlerInterceptor creates a new error handler interceptor.
func NewErrorHandlerInterceptor() *ErrorHandlerInterceptor {
	return &ErrorHandlerInterceptor{}
}

// WrapUnary implements connect.Interceptor for unary RPC calls.
func (i *ErrorHandlerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return nil, i.wrapError(ctx, err)
		}
		return resp, nil
	}
}

// WrapStreamingClient implements connect.Interceptor for client-side streaming.
func (i *ErrorHandlerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements connect.Interceptor for server-side streaming.
func (i *ErrorHandlerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		if err != nil {
			return i.wrapError(ctx, err)
		}
		return nil
	}
}

// wrapError converts raw errors into structured APIError types.
// If the error is already an APIError or connect.Error with APIError details,
// it passes through unchanged. Otherwise, it wraps the error with context.
func (i *ErrorHandlerInterceptor) wrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// Attempt to extract subsystem/operation from context
	subsystem, operation := errors.SubsystemFromContext(ctx)

	// If already an APIError, enrich with context metadata if not already set
	if apiErr, ok := errors.AsAPIError(err); ok {
		if apiErr.Subsystem == "" {
			apiErr.Subsystem = subsystem
		}
		if apiErr.Operation == "" {
			apiErr.Operation = operation
		}
		code := i.apiErrorCodeToConnectCode(apiErr.Code)
		return connect.NewError(code, apiErr)
	}

	// Already a connect.Error - check if it contains APIError details
	if connectErr, ok := err.(*connect.Error); ok {
		return i.wrapConnectError(connectErr, subsystem, operation)
	}

	// Unknown error - wrap as internal error
	apiErr := errors.NewInternal("unexpected error occurred", err)
	if subsystem != "" {
		apiErr.Subsystem = subsystem
	}
	if operation != "" {
		apiErr.Operation = operation
	}
	return connect.NewError(connect.CodeInternal, apiErr)
}

// wrapConnectError wraps a connect.Error with structured APIError details.
func (i *ErrorHandlerInterceptor) wrapConnectError(err *connect.Error, subsystem, operation string) *connect.Error {
	code := err.Code()
	msg := err.Message()

	var apiErr *errors.APIError
	switch code {
	case connect.CodeInvalidArgument:
		apiErr = errors.NewInvalidArgument(msg, err)
	case connect.CodeNotFound:
		apiErr = errors.NewNotFound(msg, err)
	case connect.CodeAlreadyExists:
		apiErr = errors.NewFailedPrecondition(msg, err)
	case connect.CodePermissionDenied:
		apiErr = errors.NewForbidden(msg, err)
	case connect.CodeUnauthenticated:
		apiErr = errors.NewUnauthorized(msg, err)
	case connect.CodeResourceExhausted:
		apiErr = errors.NewUnavailable(msg, err)
	case connect.CodeFailedPrecondition:
		apiErr = errors.NewFailedPrecondition(msg, err)
	case connect.CodeAborted:
		apiErr = errors.NewFailedPrecondition(msg, err)
	case connect.CodeOutOfRange:
		apiErr = errors.NewInvalidArgument(msg, err)
	case connect.CodeUnimplemented:
		apiErr = errors.NewUnavailable(msg, err)
	case connect.CodeInternal:
		apiErr = errors.NewInternal(msg, err)
	case connect.CodeUnavailable:
		apiErr = errors.NewUnavailable(msg, err)
	case connect.CodeDeadlineExceeded:
		apiErr = errors.NewDeadlineExceeded(msg, err)
	default:
		apiErr = errors.NewInternal(msg, err)
	}

	// Enrich with subsystem/operation from context
	if subsystem != "" {
		apiErr.Subsystem = subsystem
	}
	if operation != "" {
		apiErr.Operation = operation
	}

	return connect.NewError(code, apiErr)
}

// apiErrorCodeToConnectCode maps APIError codes to ConnectRPC codes.
func (i *ErrorHandlerInterceptor) apiErrorCodeToConnectCode(code string) connect.Code {
	switch code {
	case errors.ErrNotFound:
		return connect.CodeNotFound
	case errors.ErrUnauthorized:
		return connect.CodeUnauthenticated
	case errors.ErrForbidden:
		return connect.CodePermissionDenied
	case errors.ErrValidation, errors.ErrInvalidArgument:
		return connect.CodeInvalidArgument
	case errors.ErrFailedPrecondition:
		return connect.CodeFailedPrecondition
	case errors.ErrDeadlineExceeded:
		return connect.CodeDeadlineExceeded
	case errors.ErrUnavailable:
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}
