package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"golang.org/x/sync/semaphore"
)

// BulkheadConfig defines semaphore pools for different operation domains.
type BulkheadConfig struct {
	// QueryPoolSize limits concurrent query operations.
	QueryPoolSize int64
	// IngestionPoolSize limits concurrent ingestion operations.
	IngestionPoolSize int64
	// ChatPoolSize limits concurrent chat/LLM operations.
	ChatPoolSize int64
}

// DefaultBulkheadConfig provides sensible defaults for bulkhead pools.
var DefaultBulkheadConfig = BulkheadConfig{
	QueryPoolSize:     20,
	IngestionPoolSize: 5,
	ChatPoolSize:      10,
}

// bulkheadKey is a private context key for bulkhead middleware.
type bulkheadKey struct{}

// BulkheadInterceptor implements Connect RPC interceptors with bulkhead pattern.
type BulkheadInterceptor struct {
	config         BulkheadConfig
	queryPool      *semaphore.Weighted
	ingestionPool  *semaphore.Weighted
	chatPool       *semaphore.Weighted
	poolsByDomain  map[string]*semaphore.Weighted
	mu             sync.RWMutex
}

// NewBulkheadInterceptor creates a new bulkhead interceptor with the given config.
// If config is nil, DefaultBulkheadConfig will be used.
func NewBulkheadInterceptor(config *BulkheadConfig) *BulkheadInterceptor {
	if config == nil {
		cfg := DefaultBulkheadConfig
		config = &cfg
	}

	interceptor := &BulkheadInterceptor{
		config:        *config,
		queryPool:     semaphore.NewWeighted(config.QueryPoolSize),
		ingestionPool: semaphore.NewWeighted(config.IngestionPoolSize),
		chatPool:      semaphore.NewWeighted(config.ChatPoolSize),
		poolsByDomain: make(map[string]*semaphore.Weighted),
	}

	// Initialize domain mapping
	interceptor.poolsByDomain["query"] = interceptor.queryPool
	interceptor.poolsByDomain["ingestion"] = interceptor.ingestionPool
	interceptor.poolsByDomain["chat"] = interceptor.chatPool

	return interceptor
}

// WrapUnary adds bulkhead protection to unary RPC calls.
func (b *BulkheadInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		pool := b.poolForProcedure(req.Spec().Procedure)
		if pool == nil {
			// No bulkhead for this domain
			return next(ctx, req)
		}

		if !pool.TryAcquire(1) {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("service overloaded: too many concurrent %s operations", b.domainForProcedure(req.Spec().Procedure)))
		}
		defer pool.Release(1)

		ctx = context.WithValue(ctx, bulkheadKey{}, b.domainForProcedure(req.Spec().Procedure))
		return next(ctx, req)
	}
}

// WrapStreamingHandler adds bulkhead protection to streaming handler calls.
func (b *BulkheadInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		pool := b.poolForProcedure(conn.Spec().Procedure)
		if pool == nil {
			// No bulkhead for this domain
			return next(ctx, conn)
		}

		if !pool.TryAcquire(1) {
			return connect.NewError(connect.CodeUnavailable, fmt.Errorf("service overloaded: too many concurrent %s operations", b.domainForProcedure(conn.Spec().Procedure)))
		}
		defer pool.Release(1)

		ctx = context.WithValue(ctx, bulkheadKey{}, b.domainForProcedure(conn.Spec().Procedure))
		return next(ctx, conn)
	}
}

// WrapStreamingClient is a no-op for client-side streaming.
func (b *BulkheadInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// poolForProcedure determines which semaphore pool to use based on the procedure name.
func (b *BulkheadInterceptor) poolForProcedure(procedure string) *semaphore.Weighted {
	domain := b.domainForProcedure(procedure)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.poolsByDomain[domain]
}

// domainForProcedure maps procedure names to bulkhead domains.
func (b *BulkheadInterceptor) domainForProcedure(procedure string) string {
	// Query operations
	if strings.Contains(procedure, "QueryService") ||
		strings.Contains(procedure, "ProjectService") ||
		strings.Contains(procedure, "RegistryService") ||
		strings.Contains(procedure, "LibraryService") {
		return "query"
	}

	// Ingestion operations
	if strings.Contains(procedure, "IngestionService") {
		return "ingestion"
	}

	// Chat/LLM operations
	if strings.Contains(procedure, "AgentService") ||
		strings.Contains(procedure, "SkillService") ||
		strings.Contains(procedure, "ToolService") ||
		strings.Contains(procedure, "NLPService") {
		return "chat"
	}

	// Default: no bulkhead
	return ""
}

// BulkheadDomainFromContext retrieves the bulkhead domain set by the bulkhead middleware.
// Returns empty string if no bulkhead was applied.
func BulkheadDomainFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(bulkheadKey{}).(string); ok {
		return v
	}
	return ""
}

// Stats returns current usage statistics for all bulkhead pools.
func (b *BulkheadInterceptor) Stats() map[string]int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make(map[string]int64)
	for domain := range b.poolsByDomain {
		// semaphore.Weighted doesn't expose current count, so we can only report capacity
		// In a real implementation, we'd track usage separately
		stats[domain] = b.configForDomain(domain)
	}
	return stats
}

// configForDomain returns the pool size for a given domain.
func (b *BulkheadInterceptor) configForDomain(domain string) int64 {
	switch domain {
	case "query":
		return b.config.QueryPoolSize
	case "ingestion":
		return b.config.IngestionPoolSize
	case "chat":
		return b.config.ChatPoolSize
	default:
		return 0
	}
}