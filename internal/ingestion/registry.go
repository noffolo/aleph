package ingestion

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrSourceTypeNotFound is returned when a source type is not registered in the registry.
var ErrSourceTypeNotFound = errors.New("source type not found in registry")

// Fetcher represents a data source that can be ingested. Each Fetcher identifies
// itself via SourceType and can validate its configuration via Validate.
// Validate should check that the source configuration is consistent and actionable
// (e.g., URLs are reachable, credentials are present, required fields are set).
type Fetcher interface {
	SourceType() string
	Validate() error
}

// FetcherFactory creates a new Fetcher for the given source type string.
// The sourceType parameter is the string key that was used to register the factory.
type FetcherFactory func(sourceType string) Fetcher

// Registry manages a collection of FetcherFactory handlers keyed by source type.
// It is safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]FetcherFactory
}

// GlobalRegistry is the default registry instance used by the ingestion engine
// for registry-based source type dispatch.
var GlobalRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]FetcherFactory)}
}

func (r *Registry) Register(sourceType string, factory FetcherFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[sourceType] = factory
}

func (r *Registry) Create(sourceType string) (Fetcher, error) {
	r.mu.RLock()
	factory, ok := r.handlers[sourceType]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSourceTypeNotFound, sourceType)
	}
	return factory(sourceType), nil
}

// Reset clears all registered handlers. Used for test isolation.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = make(map[string]FetcherFactory)
}

// List returns all registered source type names in sorted order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RegisteredSourceTypes returns all source types in the global registry.
// Used by the frontend to list available source types for the dropdown.
func RegisteredSourceTypes() []string {
	return GlobalRegistry.List()
}
