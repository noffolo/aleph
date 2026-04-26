package errors

import (
	stdErrors "errors"
	"fmt"
)

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
}

// NewAPIError creates a new APIError with the given code, user message, and underlying error.
// The user message should be in Italian.
func NewAPIError(code, userMsg string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: userMsg,
		Err:     err,
		Details: nil,
	}
}

// NewAPIErrorWithDetails creates a new APIError with additional details.
func NewAPIErrorWithDetails(code, userMsg string, details map[string]interface{}, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: userMsg,
		Err:     err,
		Details: details,
	}
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