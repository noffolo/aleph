package errors

import (
	"context"
	stdErrors "errors"
	"fmt"
	"time"
)

// Context key for storing subsystem and operation information
// so the error handler interceptor can enrich APIError fields.
type contextKey string

const (
	// SubsystemContextKey stores the subsystem name (e.g. "duckdb", "llm", "sandbox")
	SubsystemContextKey contextKey = "subsystem"
	// OperationContextKey stores the operation name (e.g. "query", "insert", "execute")
	OperationContextKey contextKey = "operation"
)

// WithSubsystem annotates a context with subsystem metadata for error enrichment.
func WithSubsystem(ctx context.Context, subsystem, operation string) context.Context {
	ctx = context.WithValue(ctx, SubsystemContextKey, subsystem)
	ctx = context.WithValue(ctx, OperationContextKey, operation)
	return ctx
}

// SubsystemFromContext extracts the subsystem annotation from context.
func SubsystemFromContext(ctx context.Context) (subsystem, operation string) {
	if s, ok := ctx.Value(SubsystemContextKey).(string); ok {
		subsystem = s
	}
	if o, ok := ctx.Value(OperationContextKey).(string); ok {
		operation = o
	}
	return
}

// APIError represents an error that can be presented to users in Italian
// while maintaining technical details in English for logging.
type APIError struct {
	// Code is a machine-readable error code (e.g., "ERR_NOT_FOUND")
	Code string
	// Message is the user-facing message in Italian
	Message string
	// Details is optional additional context for the error
	Details map[string]interface{}
	// Err is the underlying technical error in English (optional)
	Err error
	// Subsystem identifies which subsystem generated the error
	// (e.g. "duckdb", "postgres", "llm", "sandbox", "mcp", "nlp", "ingestion", "handler").
	Subsystem string
	// Operation identifies the operation being performed
	// (e.g. "query", "insert", "delete", "discover", "execute", "validate").
	Operation string
	// Recoverable indicates whether the operation can be retried.
	Recoverable bool
	// RetryAfterMs is the suggested delay before retrying (0 = not retryable).
	RetryAfterMs int64
}

// NewAPIError creates a new APIError with the given code, user message, and underlying error.
// The user message should be in Italian.
func NewAPIError(code, userMsg string, err error) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Details:      nil,
		Subsystem:    "",
		Operation:    "",
		Recoverable:  false,
		RetryAfterMs: 0,
	}
}

// NewAPIErrorWithDetails creates a new APIError with additional details.
func NewAPIErrorWithDetails(code, userMsg string, details map[string]interface{}, err error) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Details:      details,
		Subsystem:    "",
		Operation:    "",
		Recoverable:  false,
		RetryAfterMs: 0,
	}
}

// NewAPIErrorWithMeta creates a new APIError with enriched metadata for structured error handling.
// subsystem identifies the source subsystem, operation identifies the failing action,
// recoverable indicates if retry is feasible, and retryAfter is the suggested wait before retry
// (0 means no retry suggested).
func NewAPIErrorWithMeta(code, userMsg string, err error, subsystem, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Details:      nil,
		Subsystem:    subsystem,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// RetryAfter returns the suggested retry delay as a time.Duration.
// Returns 0 if no retry is suggested (non-recoverable or no delay configured).
func (e *APIError) RetryAfter() time.Duration {
	if !e.Recoverable {
		return 0
	}
	return time.Duration(e.RetryAfterMs) * time.Millisecond
}

// Error returns the technical error message in English.
// If there's an underlying error, it formats it with the error code.
// This is suitable for logging and debugging.
func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

// UserMessage returns the user-facing message in Italian.
func (e *APIError) UserMessage() string {
	return e.Message
}

// Unwrap returns the underlying error for error chain inspection.
func (e *APIError) Unwrap() error {
	return e.Err
}

// IsAPIError checks if an error is an APIError.
func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

// AsAPIError attempts to extract an APIError from the error chain.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if err == nil {
		return nil, false
	}
	// Use errors.As to check the entire chain
	if ok := stdErrors.As(err, &apiErr); ok {
		return apiErr, true
	}
	return nil, false
}

// Wrap wraps an existing error with an APIError.
// This is useful when you have a technical error and want to add a user-facing message.
func Wrap(err error, code, userMsg string) *APIError {
	return NewAPIError(code, userMsg, err)
}

// WrapWithDetails wraps an existing error with an APIError and details.
func WrapWithDetails(err error, code, userMsg string, details map[string]interface{}) *APIError {
	return NewAPIErrorWithDetails(code, userMsg, details, err)
}

// Subsystem constants for use with typed constructors.
const (
	SubsystemLLM      = "llm"
	SubsystemDuckDB   = "duckdb"
	SubsystemPostgres = "postgres"
	SubsystemMCP      = "mcp"
	SubsystemNLP      = "nlp"
	SubsystemHandler  = "handler"
	SubsystemSandbox  = "sandbox"
	SubsystemIngest   = "ingestion"
)

// NewLLMError creates a new APIError tagged with the "llm" subsystem.
// recoverable indicates if retry is feasible; retryAfter is the suggested wait
// before retrying (0 means no retry delay suggested).
func NewLLMError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemLLM,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewDuckDBError creates a new APIError tagged with the "duckdb" subsystem.
func NewDuckDBError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemDuckDB,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewPostgresError creates a new APIError tagged with the "postgres" subsystem.
func NewPostgresError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemPostgres,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewMCPError creates a new APIError tagged with the "mcp" subsystem.
func NewMCPError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemMCP,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewNLPError creates a new APIError tagged with the "nlp" subsystem.
func NewNLPError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemNLP,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewHandlerError creates a new APIError tagged with the "handler" subsystem.
func NewHandlerError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemHandler,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewSandboxError creates a new APIError tagged with the "sandbox" subsystem.
func NewSandboxError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemSandbox,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}

// NewIngestionError creates a new APIError tagged with the "ingestion" subsystem.
func NewIngestionError(code, userMsg string, err error, operation string, recoverable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Code:         code,
		Message:      userMsg,
		Err:          err,
		Subsystem:    SubsystemIngest,
		Operation:    operation,
		Recoverable:  recoverable,
		RetryAfterMs: int64(retryAfter / time.Millisecond),
	}
}