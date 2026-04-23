package handler

import (
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/dsl"
)

const (
	defaultMaxEntries = 64
	defaultTTL        = 30 * time.Minute
)

type cacheEntry struct {
	program   *dsl.Program
	createdAt time.Time
}

type programCache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
}

func newProgramCache() *programCache {
	return &programCache{
		entries:    make(map[string]*cacheEntry),
		maxEntries: defaultMaxEntries,
		ttl:        defaultTTL,
	}
}

func (c *programCache) Get(key string) *dsl.Program {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if time.Since(entry.createdAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil
	}

	return entry.program
}

func (c *programCache) Set(key string, program *dsl.Program) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If over max entries, evict the oldest entry
	if len(c.entries) >= c.maxEntries {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldestKey == "" || v.createdAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.createdAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &cacheEntry{
		program:   program,
		createdAt: time.Now(),
	}
}
