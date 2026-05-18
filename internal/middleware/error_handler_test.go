package middleware

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	apierrors "github.com/ff3300/aleph-v2/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandlerInterceptor_New(t *testing.T) {
	i := NewErrorHandlerInterceptor()
	assert.NotNil(t, i)
}

func TestErrorHandlerInterceptor_wrapError(t *testing.T) {
	interceptor := &ErrorHandlerInterceptor{}

	t.Run("nil error", func(t *testing.T) {
		assert.Nil(t, interceptor.wrapError(context.Background(), nil))
	})

	t.Run("already APIError", func(t *testing.T) {
		apiErr := apierrors.NewNotFound("resource missing", errors.New("not found"))
		ctx := apierrors.WithSubsystem(context.Background(), apierrors.SubsystemDuckDB, "select")
		result := interceptor.wrapError(ctx, apiErr)
		connectErr, ok := result.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeNotFound, connectErr.Code())
	})

	t.Run("already connect.Error", func(t *testing.T) {
		connectErr := connect.NewError(connect.CodeInternal, errors.New("boom"))
		result := interceptor.wrapError(context.Background(), connectErr)
		cErr, ok := result.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInternal, cErr.Code())
	})

	t.Run("unknown error becomes internal", func(t *testing.T) {
		result := interceptor.wrapError(context.Background(), errors.New("random crash"))
		cErr, ok := result.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInternal, cErr.Code())
	})

	t.Run("unknown error with subsystem context", func(t *testing.T) {
		ctx := apierrors.WithSubsystem(context.Background(), apierrors.SubsystemSandbox, "execute")
		result := interceptor.wrapError(ctx, errors.New("segfault"))
		cErr, ok := result.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInternal, cErr.Code())
	})

	t.Run("APIError from chain", func(t *testing.T) {
		apiErr := apierrors.NewDuckDBError(apierrors.ErrInternal, "db error", nil, "query", false, 0)
		wrapped := fmt.Errorf("wrapper: %w", apiErr)
		result := interceptor.wrapError(context.Background(), wrapped)
		cErr, ok := result.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInternal, cErr.Code())
	})
}

func TestErrorHandlerInterceptor_wrapConnectError(t *testing.T) {
	interceptor := &ErrorHandlerInterceptor{}

	tests := []struct {
		name         string
		code         connect.Code
		expectedCode connect.Code
	}{
		{"InvalidArgument", connect.CodeInvalidArgument, connect.CodeInvalidArgument},
		{"NotFound", connect.CodeNotFound, connect.CodeNotFound},
		{"PermissionDenied", connect.CodePermissionDenied, connect.CodePermissionDenied},
		{"Unauthenticated", connect.CodeUnauthenticated, connect.CodeUnauthenticated},
		{"Internal", connect.CodeInternal, connect.CodeInternal},
		{"Unavailable", connect.CodeUnavailable, connect.CodeUnavailable},
		{"DeadlineExceeded", connect.CodeDeadlineExceeded, connect.CodeDeadlineExceeded},
		{"AlreadyExists", connect.CodeAlreadyExists, connect.CodeAlreadyExists},
		{"ResourceExhausted", connect.CodeResourceExhausted, connect.CodeResourceExhausted},
		{"Aborted", connect.CodeAborted, connect.CodeAborted},
		{"OutOfRange", connect.CodeOutOfRange, connect.CodeOutOfRange},
		{"Unimplemented", connect.CodeUnimplemented, connect.CodeUnimplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := connect.NewError(tt.code, errors.New("original error"))
			result := interceptor.wrapConnectError(orig, "test-subsystem", "test-operation")
			assert.Equal(t, tt.expectedCode, result.Code())
		})
	}

	t.Run("unknown code falls to internal", func(t *testing.T) {
		orig := connect.NewError(connect.CodeUnknown, errors.New("weird"))
		result := interceptor.wrapConnectError(orig, "", "")
		assert.Equal(t, connect.CodeUnknown, result.Code())
	})
}

func TestErrorHandlerInterceptor_apiErrorCodeToConnectCode(t *testing.T) {
	interceptor := &ErrorHandlerInterceptor{}

	tests := []struct {
		apiCode  string
		connCode connect.Code
	}{
		{apierrors.ErrNotFound, connect.CodeNotFound},
		{apierrors.ErrUnauthorized, connect.CodeUnauthenticated},
		{apierrors.ErrForbidden, connect.CodePermissionDenied},
		{apierrors.ErrValidation, connect.CodeInvalidArgument},
		{apierrors.ErrInvalidArgument, connect.CodeInvalidArgument},
		{apierrors.ErrFailedPrecondition, connect.CodeFailedPrecondition},
		{apierrors.ErrDeadlineExceeded, connect.CodeDeadlineExceeded},
		{apierrors.ErrUnavailable, connect.CodeUnavailable},
		{"UNKNOWN_CODE", connect.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.apiCode, func(t *testing.T) {
			assert.Equal(t, tt.connCode, interceptor.apiErrorCodeToConnectCode(tt.apiCode))
		})
	}
}

func TestErrorHandlerInterceptor_WrapUnary(t *testing.T) {
	interceptor := NewErrorHandlerInterceptor()

	t.Run("passes through success", func(t *testing.T) {
		wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{ Name string }{Name: "ok"}), nil
		})
		resp, err := wrapped(context.Background(), nil)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("wraps error", func(t *testing.T) {
		wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errors.New("something broke")
		})
		_, err := wrapped(context.Background(), nil)
		assert.Error(t, err)
		_, ok := err.(*connect.Error)
		assert.True(t, ok)
	})
}

func TestErrorHandlerInterceptor_WrapStreamingHandler(t *testing.T) {
	interceptor := NewErrorHandlerInterceptor()

	t.Run("passes through success", func(t *testing.T) {
		wrapped := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
			return nil
		})
		err := wrapped(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("wraps error", func(t *testing.T) {
		wrapped := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
			return errors.New("stream broke")
		})
		err := wrapped(context.Background(), nil)
		assert.Error(t, err)
	})
}
