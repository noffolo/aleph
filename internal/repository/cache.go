package repository

import (
	"sync"
	"time"
)

const defaultToolCacheTTL = 5 * time.Minute
const defaultMaxSize = 500
const cleanupInterval = 5 * time.Minute

type cacheEntry struct {
	value     any
	expiresAt time.Time
	accessed  time.Time
}

type ToolCache struct {
	cache   *sync.Map
	ttl     time.Duration
	maxSize int
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func NewToolCache() *ToolCache {
	tc := &ToolCache{
		cache:   &sync.Map{},
		ttl:     defaultToolCacheTTL,
		maxSize: defaultMaxSize,
		stopCh:  make(chan struct{}),
	}
	tc.wg.Add(1)
	go tc.cleanupExpired()
	return tc
}

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
	entry.accessed = time.Now()
	tc.cache.Store(key, entry)
	return entry.value, true
}

func (tc *ToolCache) Set(key string, value any) {
	if tc.len() >= tc.maxSize {
		tc.evictLRU()
	}
	tc.cache.Store(key, cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(tc.ttl),
		accessed:  time.Now(),
	})
}

func (tc *ToolCache) Invalidate(key string) {
	tc.cache.Delete(key)
}

func (tc *ToolCache) SetMaxSize(size int) {
	if size > 0 {
		tc.maxSize = size
	}
	if tc.len() > tc.maxSize {
		tc.evictLRU()
	}
}

func (tc *ToolCache) Close() {
	close(tc.stopCh)
	tc.wg.Wait()
}

func (tc *ToolCache) len() int {
	count := 0
	tc.cache.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (tc *ToolCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	tc.cache.Range(func(key, val any) bool {
		entry, ok := val.(cacheEntry)
		if !ok {
			return true
		}
		if first || entry.accessed.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = entry.accessed
			first = false
		}
		return true
	})
	if oldestKey != "" {
		tc.cache.Delete(oldestKey)
	}
}

func (tc *ToolCache) cleanupExpired() {
	defer tc.wg.Done()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-tc.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			tc.cache.Range(func(key, val any) bool {
				entry, ok := val.(cacheEntry)
				if !ok || now.After(entry.expiresAt) {
					tc.cache.Delete(key)
				}
				return true
			})
		}
	}
}
