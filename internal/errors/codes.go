package errors

// Error codes as typed string constants
const (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = "ERR_NOT_FOUND"
	// ErrUnauthorized indicates authentication is required
	ErrUnauthorized = "ERR_UNAUTHORIZED"
	// ErrForbidden indicates insufficient permissions
	ErrForbidden = "ERR_FORBIDDEN"
	// ErrInternal indicates an unexpected internal error
	ErrInternal = "ERR_INTERNAL"
	// ErrValidation indicates invalid input
	ErrValidation = "ERR_VALIDATION"
	// ErrUnavailable indicates a service is temporarily unavailable
	ErrUnavailable = "ERR_UNAVAILABLE"
	// ErrDeadlineExceeded indicates an operation timed out
	ErrDeadlineExceeded = "ERR_DEADLINE_EXCEEDED"
	// ErrFailedPrecondition indicates a required condition was not met
	ErrFailedPrecondition = "ERR_FAILED_PRECONDITION"
	// ErrInvalidArgument indicates an invalid argument was provided
	ErrInvalidArgument = "ERR_INVALID_ARGUMENT"
)

// userMessages maps error codes to Italian user-facing messages
var userMessages = map[string]string{
	ErrNotFound:           "Risorsa non trovata",
	ErrUnauthorized:       "Autenticazione richiesta",
	ErrForbidden:          "Permessi insufficienti",
	ErrInternal:           "Errore interno del sistema",
	ErrValidation:         "Dati inseriti non validi",
	ErrUnavailable:        "Servizio temporaneamente non disponibile",
	ErrDeadlineExceeded:   "Operazione scaduta",
	ErrFailedPrecondition: "Condizione preliminare non soddisfatta",
	ErrInvalidArgument:    "Argomento non valido",
}

// GetUserMessage returns the Italian user-facing message for an error code
func GetUserMessage(code string) string {
	if msg, ok := userMessages[code]; ok {
		return msg
	}
	// Default fallback message
	return "Si è verificato un errore"
}

// NewNotFound creates a new APIError for resource not found
func NewNotFound(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrNotFound)
	}
	return NewAPIError(ErrNotFound, userMsg, err)
}

// NewUnauthorized creates a new APIError for authentication required
func NewUnauthorized(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrUnauthorized)
	}
	return NewAPIError(ErrUnauthorized, userMsg, err)
}

// NewForbidden creates a new APIError for insufficient permissions
func NewForbidden(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrForbidden)
	}
	return NewAPIError(ErrForbidden, userMsg, err)
}

// NewInternal creates a new APIError for internal error
func NewInternal(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrInternal)
	}
	return NewAPIError(ErrInternal, userMsg, err)
}

// NewValidation creates a new APIError for validation error
func NewValidation(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrValidation)
	}
	return NewAPIError(ErrValidation, userMsg, err)
}

// NewUnavailable creates a new APIError for service unavailable
func NewUnavailable(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrUnavailable)
	}
	return NewAPIError(ErrUnavailable, userMsg, err)
}

// NewDeadlineExceeded creates a new APIError for timeout
func NewDeadlineExceeded(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrDeadlineExceeded)
	}
	return NewAPIError(ErrDeadlineExceeded, userMsg, err)
}

// NewFailedPrecondition creates a new APIError for failed precondition
func NewFailedPrecondition(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrFailedPrecondition)
	}
	return NewAPIError(ErrFailedPrecondition, userMsg, err)
}

// NewInvalidArgument creates a new APIError for invalid argument
func NewInvalidArgument(userMsg string, err error) *APIError {
	if userMsg == "" {
		userMsg = GetUserMessage(ErrInvalidArgument)
	}
	return NewAPIError(ErrInvalidArgument, userMsg, err)
}