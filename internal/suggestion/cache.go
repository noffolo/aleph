package suggestion

import "sync"

const defaultEmbedCacheSize = 100

// embedCache provides a simple LRU cache for tool description embeddings.
// It maps "category:name" keys to fixed-dimension float32 vectors.
type embedCache struct {
	mu      sync.RWMutex
	entries map[string][]float32
	order   []string
	maxSize int
}

func newEmbedCache(maxSize int) *embedCache {
	if maxSize <= 0 {
		maxSize = defaultEmbedCacheSize
	}
	return &embedCache{
		entries: make(map[string][]float32),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

func (c *embedCache) get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	emb, ok := c.entries[key]
	return emb, ok
}

func (c *embedCache) put(key string, emb []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		c.entries[key] = emb
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
		c.order = append(c.order, key)
		return
	}

	for len(c.entries) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[key] = emb
	c.order = append(c.order, key)
}

func (c *embedCache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
