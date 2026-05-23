package ingestion

import (
	"errors"
	"fmt"
	"sync"
)

var ErrSourceTypeNotFound = errors.New("source type not found in registry")

type Fetcher interface {
	SourceType() string
	Validate() error
}

type FetcherFactory func(sourceType string) Fetcher

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]FetcherFactory
}

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

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		keys = append(keys, k)
	}
	return keys
}

// RegisteredSourceTypes returns all source types in the global registry.
// Used by the frontend to list available source types for the dropdown.
func RegisteredSourceTypes() []string {
	return GlobalRegistry.List()
}
