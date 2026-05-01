package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	stderrors "errors"
)

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(ErrNotFound, "Risorsa non trovata", nil)
	assert.NotNil(t, err)
	assert.Equal(t, ErrNotFound, err.Code)
	assert.Equal(t, "Risorsa non trovata", err.Message)
	assert.Nil(t, err.Err)
	assert.Nil(t, err.Details)
}

func TestNewAPIError_WithUnderlying(t *testing.T) {
	underlying := fmt.Errorf("underlying db error")
	err := NewAPIError(ErrInternal, "Errore interno", underlying)
	assert.Equal(t, underlying, err.Err)
	assert.True(t, IsAPIError(err))
}

func TestNewAPIErrorWithDetails(t *testing.T) {
	details := map[string]interface{}{"field": "name", "reason": "too short"}
	err := NewAPIErrorWithDetails(ErrValidation, "Dati non validi", details, nil)
	assert.Equal(t, details, err.Details)
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		expect string
	}{
		{"with underlying", NewAPIError(ErrNotFound, "msg", fmt.Errorf("db error")), "ERR_NOT_FOUND: db error"},
		{"without underlying", NewAPIError(ErrForbidden, "msg", nil), "ERR_FORBIDDEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.err.Error())
		})
	}
}

func TestAPIError_UserMessage(t *testing.T) {
	err := NewAPIError(ErrUnauthorized, "Autenticazione fallita", nil)
	assert.Equal(t, "Autenticazione fallita", err.UserMessage())
}

func TestAPIError_Unwrap(t *testing.T) {
	underlying := fmt.Errorf("original error")
	err := NewAPIError(ErrInternal, "msg", underlying)
	assert.Equal(t, underlying, err.Unwrap())
	assert.Nil(t, NewAPIError(ErrInternal, "msg", nil).Unwrap())
}

func TestIsAPIError(t *testing.T) {
	assert.True(t, IsAPIError(NewAPIError(ErrNotFound, "", nil)))
	assert.False(t, IsAPIError(fmt.Errorf("plain error")))
	assert.False(t, IsAPIError(nil))
}

func TestAsAPIError(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		_, ok := AsAPIError(nil)
		assert.False(t, ok)
	})

	t.Run("direct APIError", func(t *testing.T) {
		apiErr := NewAPIError(ErrNotFound, "msg", nil)
		as, ok := AsAPIError(apiErr)
		assert.True(t, ok)
		assert.Equal(t, apiErr, as)
	})

	t.Run("wrapped APIError", func(t *testing.T) {
		apiErr := NewAPIError(ErrInternal, "msg", nil)
		wrapped := fmt.Errorf("wrapper: %w", apiErr)
		as, ok := AsAPIError(wrapped)
		assert.True(t, ok)
		assert.Equal(t, apiErr, as)
	})

	t.Run("plain error", func(t *testing.T) {
		_, ok := AsAPIError(fmt.Errorf("plain"))
		assert.False(t, ok)
	})
}

func TestWrap(t *testing.T) {
	underlying := fmt.Errorf("original")
	err := Wrap(underlying, ErrNotFound, "Risorsa non trovata")
	assert.Equal(t, underlying, err.Err)
	assert.Equal(t, ErrNotFound, err.Code)
}

func TestWrapWithDetails(t *testing.T) {
	details := map[string]interface{}{"detail": "value"}
	err := WrapWithDetails(fmt.Errorf("orig"), ErrValidation, "msg", details)
	assert.Equal(t, details, err.Details)
}

func TestGetUserMessage(t *testing.T) {
	assert.Equal(t, "Resource not found", GetUserMessage(ErrNotFound))
	assert.Equal(t, "Authentication required", GetUserMessage(ErrUnauthorized))
	assert.Equal(t, "An error occurred", GetUserMessage("NONEXISTENT"))
}

func TestErrorCodeConstructors(t *testing.T) {
	tests := []struct {
		name     string
		ctor     func(string, error) *APIError
		expected string
	}{
		{"NewNotFound", NewNotFound, ErrNotFound},
		{"NewUnauthorized", NewUnauthorized, ErrUnauthorized},
		{"NewForbidden", NewForbidden, ErrForbidden},
		{"NewInternal", NewInternal, ErrInternal},
		{"NewValidation", NewValidation, ErrValidation},
		{"NewUnavailable", NewUnavailable, ErrUnavailable},
		{"NewDeadlineExceeded", NewDeadlineExceeded, ErrDeadlineExceeded},
		{"NewFailedPrecondition", NewFailedPrecondition, ErrFailedPrecondition},
		{"NewInvalidArgument", NewInvalidArgument, ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with custom message", func(t *testing.T) {
			err := tt.ctor("Custom message", nil)
			assert.Equal(t, tt.expected, err.Code)
			assert.Equal(t, "Custom message", err.Message)
		})
		t.Run(tt.name+" with default message", func(t *testing.T) {
			err := tt.ctor("", nil)
			assert.Equal(t, tt.expected, err.Code)
			assert.NotEmpty(t, err.Message)
		})
	}
}

func TestErrorChain(t *testing.T) {
	// Verify errors.As works through wrapping
	apiErr := NewAPIError(ErrNotFound, "not found", nil)
	wrapped := fmt.Errorf("wrap1: %w", fmt.Errorf("wrap2: %w", apiErr))

	var extracted *APIError
	ok := stderrors.As(wrapped, &extracted)
	assert.True(t, ok)
	assert.Equal(t, ErrNotFound, extracted.Code)
}

func TestAPIErrorWithNilDetails(t *testing.T) {
	err := NewAPIError(ErrNotFound, "msg", nil)
	assert.Nil(t, err.Details)
}

func TestAPIErrorWithErrorOnly(t *testing.T) {
	err := NewAPIErrorWithDetails(ErrInvalidArgument, "bad arg",
		map[string]interface{}{"arg": "value"}, fmt.Errorf("root cause"))
	assert.Equal(t, "ERR_INVALID_ARGUMENT: root cause", err.Error())
}
