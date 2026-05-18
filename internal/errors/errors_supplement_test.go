package errors

import (
	"context"
	"testing"
	"time"

	stderrors "errors"
	"github.com/stretchr/testify/assert"
)

func TestWithSubsystem(t *testing.T) {
	ctx := context.Background()
	annotated := WithSubsystem(ctx, SubsystemLLM, "generate")
	assert.NotEqual(t, ctx, annotated)
}

func TestSubsystemFromContext(t *testing.T) {
	t.Run("with values", func(t *testing.T) {
		ctx := WithSubsystem(context.Background(), SubsystemDuckDB, "query")
		sub, op := SubsystemFromContext(ctx)
		assert.Equal(t, SubsystemDuckDB, sub)
		assert.Equal(t, "query", op)
	})

	t.Run("without values", func(t *testing.T) {
		sub, op := SubsystemFromContext(context.Background())
		assert.Empty(t, sub)
		assert.Empty(t, op)
	})
}

func TestNewAPIErrorWithMeta(t *testing.T) {
	err := NewAPIErrorWithMeta(ErrInternal, "Errore grave", stderrors.New("underlying"),
		SubsystemDuckDB, "query", true, 5*time.Second)
	assert.Equal(t, ErrInternal, err.Code)
	assert.Equal(t, "Errore grave", err.Message)
	assert.Equal(t, SubsystemDuckDB, err.Subsystem)
	assert.Equal(t, "query", err.Operation)
	assert.True(t, err.Recoverable)
	assert.Equal(t, int64(5000), err.RetryAfterMs)
	assert.NotNil(t, err.Err)
}

func TestAPIError_RetryAfter(t *testing.T) {
	t.Run("recoverable with delay", func(t *testing.T) {
		e := NewAPIErrorWithMeta("TEST", "msg", nil, "sub", "op", true, 2*time.Second)
		assert.Equal(t, 2*time.Second, e.RetryAfter())
	})
	t.Run("recoverable zero delay", func(t *testing.T) {
		e := &APIError{Recoverable: true, RetryAfterMs: 0}
		assert.Equal(t, time.Duration(0), e.RetryAfter())
	})
	t.Run("not recoverable", func(t *testing.T) {
		e := &APIError{Recoverable: false, RetryAfterMs: 5000}
		assert.Equal(t, time.Duration(0), e.RetryAfter())
	})
}

func TestNewAPIErrorWithMeta_NonRecoverable(t *testing.T) {
	err := NewAPIErrorWithMeta(ErrValidation, "Dati non validi", nil,
		SubsystemHandler, "validate", false, 0)
	assert.False(t, err.Recoverable)
	assert.Equal(t, int64(0), err.RetryAfterMs)
	assert.Equal(t, SubsystemHandler, err.Subsystem)
}

func TestSubsystemConstructors(t *testing.T) {
	tests := []struct {
		name      string
		ctor      func() *APIError
		subsystem string
	}{
		{
			"NewLLMError",
			func() *APIError {
				return NewLLMError(ErrInternal, "LLM failure", stderrors.New("timeout"),
					"complete", true, 3*time.Second)
			},
			SubsystemLLM,
		},
		{
			"NewDuckDBError",
			func() *APIError {
				return NewDuckDBError(ErrNotFound, "Record mancante", stderrors.New("no rows"),
					"select", false, 0)
			},
			SubsystemDuckDB,
		},
		{
			"NewPostgresError",
			func() *APIError {
				return NewPostgresError(ErrInternal, "Postgres crash", stderrors.New("conn refused"),
					"connect", true, 5*time.Second)
			},
			SubsystemPostgres,
		},
		{
			"NewMCPError",
			func() *APIError {
				return NewMCPError(ErrUnavailable, "Tool server spento", stderrors.New("eof"),
					"call", true, 1*time.Second)
			},
			SubsystemMCP,
		},
		{
			"NewNLPError",
			func() *APIError {
				return NewNLPError(ErrInternal, "NLP model error", stderrors.New("onnx segfault"),
					"predict", false, 0)
			},
			SubsystemNLP,
		},
		{
			"NewHandlerError",
			func() *APIError {
				return NewHandlerError(ErrValidation, "Input non valido", stderrors.New("bad field"),
					"parse", false, 0)
			},
			SubsystemHandler,
		},
		{
			"NewSandboxError",
			func() *APIError {
				return NewSandboxError(ErrForbidden, "Esecuzione bloccata", stderrors.New("seccomp"),
					"execute", false, 0)
			},
			SubsystemSandbox,
		},
		{
			"NewIngestionError",
			func() *APIError {
				return NewIngestionError(ErrDeadlineExceeded, "Import troppo lento", stderrors.New("timeout"),
					"fetch", true, 10*time.Second)
			},
			SubsystemIngest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctor()
			assert.NotNil(t, err)
			assert.Equal(t, tt.subsystem, err.Subsystem)
			assert.NotEmpty(t, err.Code)
			assert.NotEmpty(t, err.Message)
		})
	}
}
