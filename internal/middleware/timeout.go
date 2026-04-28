package middleware

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// TimeoutConfig defines timeout durations per operation domain.
type TimeoutConfig struct {
	// DBTimeout is for database operations (Query, Exec, etc.)
	DBTimeout time.Duration
	// LLMTimeout is for LLM provider calls (Ollama, Anthropic, OpenAI)
	LLMTimeout time.Duration
	// NLPTimeout is for NLP sentiment analysis and classification
	NLPTimeout time.Duration
	// ExternalHTTPTimeout is for external HTTP requests (web scraping, API calls)
	ExternalHTTPTimeout time.Duration
	// DefaultTimeout is used when no specific domain matches
	DefaultTimeout time.Duration
}

// DefaultTimeoutConfig provides the recommended timeouts for the Aleph system.
var DefaultTimeoutConfig = TimeoutConfig{
	DBTimeout:           10 * time.Second,
	LLMTimeout:          5 * time.Minute,
	NLPTimeout:          30 * time.Second,
	ExternalHTTPTimeout: 30 * time.Second,
	DefaultTimeout:      5 * time.Minute,
}

// timeoutKey is a private context key for timeout middleware.
type timeoutKey struct{}

// TimeoutInterceptor implements Connect RPC interceptors with domain-specific timeouts.
type TimeoutInterceptor struct {
	config TimeoutConfig
}

// NewTimeoutInterceptor creates a new timeout interceptor with the given config.
// If config is nil, DefaultTimeoutConfig will be used.
func NewTimeoutInterceptor(config *TimeoutConfig) *TimeoutInterceptor {
	if config == nil {
		cfg := DefaultTimeoutConfig
		config = &cfg
	}
	return &TimeoutInterceptor{config: *config}
}

// WrapUnary adds a timeout context to unary RPC calls.
func (t *TimeoutInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		timeout := t.timeoutForProcedure(req.Spec().Procedure)
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		ctx = context.WithValue(ctx, timeoutKey{}, timeout)
		return next(ctx, req)
	}
}

// WrapStreamingHandler adds a timeout context to streaming handler calls.
func (t *TimeoutInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		timeout := t.timeoutForProcedure(conn.Spec().Procedure)
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		ctx = context.WithValue(ctx, timeoutKey{}, timeout)
		return next(ctx, conn)
	}
}

// WrapStreamingClient is a no-op for client-side streaming.
func (t *TimeoutInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// timeoutForProcedure determines the appropriate timeout duration based on the procedure name.
func (t *TimeoutInterceptor) timeoutForProcedure(procedure string) time.Duration {
	// Chat is an LLM operation even though it's under QueryService
	if strings.Contains(procedure, "/Chat") {
		return t.config.LLMTimeout
	}

	// Check for DB operations
	if strings.Contains(procedure, "QueryService") ||
		strings.Contains(procedure, "ProjectService") ||
		strings.Contains(procedure, "RegistryService") ||
		strings.Contains(procedure, "LibraryService") {
		return t.config.DBTimeout
	}

	// Check for LLM operations
	if strings.Contains(procedure, "AgentService") ||
		strings.Contains(procedure, "SkillService") ||
		strings.Contains(procedure, "ToolService") {
		return t.config.LLMTimeout
	}

	// Check for NLP operations
	if strings.Contains(procedure, "NLPService") {
		return t.config.NLPTimeout
	}

	// Check for external HTTP operations
	if strings.Contains(procedure, "IngestionService") {
		return t.config.ExternalHTTPTimeout
	}

	// Default timeout for other operations
	return t.config.DefaultTimeout
}

// TimeoutFromContext retrieves the timeout duration set by the timeout middleware.
// Returns 0 if no timeout was set.
func TimeoutFromContext(ctx context.Context) time.Duration {
	if v, ok := ctx.Value(timeoutKey{}).(time.Duration); ok {
		return v
	}
	return 0
}
