package suggestion

import (
	"fmt"
	"sync"
)

// usageTracker stores tool call counts in a concurrency-safe map.
// Keys follow the "category:name" convention used by tools.ToolRegistry.
type usageTracker struct {
	mu    sync.RWMutex
	stats map[string]int
}

func newUsageTracker() *usageTracker {
	return &usageTracker{
		stats: make(map[string]int),
	}
}

// TrackUsage increments the usage counter for the given tool.
// It is safe for concurrent use.
func (u *usageTracker) TrackUsage(category, name string) {
	key := fmt.Sprintf("%s:%s", category, name)
	u.mu.Lock()
	u.stats[key]++
	u.mu.Unlock()
}

// GetUsageStats returns a copy of the current usage statistics.
// The returned map is safe to read without synchronization.
func (u *usageTracker) GetUsageStats() map[string]int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	cp := make(map[string]int, len(u.stats))
	for k, v := range u.stats {
		cp[k] = v
	}
	return cp
}

// ResetStats clears all usage counters. Useful for testing.
func (u *usageTracker) ResetStats() {
	u.mu.Lock()
	u.stats = make(map[string]int)
	u.mu.Unlock()
}
