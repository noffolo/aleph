package repository

import (
	"sync"
	"time"
)

// defaultToolCacheTTL is the default time-to-live for cached tool entries.
const defaultToolCacheTTL = 5 * time.Minute

// cacheEntry holds a cached value with its expiration time.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// ToolCache is a simple in-memory cache for ToolRecord lists
// using sync.Map with configurable TTL.
type ToolCache struct {
	cache *sync.Map
	ttl   time.Duration
}

// NewToolCache creates a new ToolCache with a 5-minute TTL.
func NewToolCache() *ToolCache {
	return &ToolCache{
		cache: &sync.Map{},
		ttl:   defaultToolCacheTTL,
	}
}

// Get retrieves a cached value by key. Returns false if the key
// does not exist or the entry has expired.
func (tc *ToolCache) Get(key string) (any, bool) {
	v, ok := tc.cache.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := v.(cacheEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		tc.cache.Delete(key)
		return nil, false
	}
	return entry.value, true
}

// Set stores a value in the cache under the given key with the
// configured TTL.
func (tc *ToolCache) Set(key string, value any) {
	tc.cache.Store(key, cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(tc.ttl),
	})
}

// Invalidate removes a key from the cache. Used when the underlying
// data changes (e.g. tool creation, update, or deletion).
func (tc *ToolCache) Invalidate(key string) {
	tc.cache.Delete(key)
}
