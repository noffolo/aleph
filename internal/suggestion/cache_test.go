package suggestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEmbedCache(t *testing.T) {
	// Happy: explicit custom size
	t.Run("custom size", func(t *testing.T) {
		c := newEmbedCache(7)
		assert.NotNil(t, c)
		assert.Equal(t, 7, c.maxSize)
		assert.Equal(t, 0, c.size())
		assert.NotNil(t, c.entries)
	})

	// Edge: zero maxSize defaults to defaultEmbedCacheSize
	t.Run("zero defaults", func(t *testing.T) {
		c := newEmbedCache(0)
		assert.Equal(t, defaultEmbedCacheSize, c.maxSize)
	})

	// Edge: negative maxSize defaults to defaultEmbedCacheSize
	t.Run("negative defaults", func(t *testing.T) {
		c := newEmbedCache(-5)
		assert.Equal(t, defaultEmbedCacheSize, c.maxSize)
	})
}

func TestEmbedCache_Get(t *testing.T) {
	// Happy: cache hit returns correct embedding
	t.Run("hit", func(t *testing.T) {
		c := newEmbedCache(10)
		c.put("osint:shodan", []float32{0.1, 0.2, 0.3})
		v, ok := c.get("osint:shodan")
		assert.True(t, ok)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, v)
	})

	// Edge: cache miss returns false
	t.Run("miss", func(t *testing.T) {
		c := newEmbedCache(10)
		_, ok := c.get("nonexistent")
		assert.False(t, ok)
	})

	// Edge: evicted key is a miss
	t.Run("after eviction", func(t *testing.T) {
		c := newEmbedCache(2)
		c.put("a", []float32{1})
		c.put("b", []float32{2})
		c.put("c", []float32{3}) // evicts 'a' (oldest LRU)
		_, ok := c.get("a")
		assert.False(t, ok, "evicted key should no longer exist")
		_, ok = c.get("b")
		assert.True(t, ok, "non-evicted key should still exist")
	})
}

func TestEmbedCache_Put(t *testing.T) {
	// Happy: insert new entry
	t.Run("new entry", func(t *testing.T) {
		c := newEmbedCache(10)
		c.put("finance:prophet", []float32{0.5, 0.6})
		assert.Equal(t, 1, c.size())
		v, ok := c.get("finance:prophet")
		assert.True(t, ok)
		assert.Equal(t, []float32{0.5, 0.6}, v)
	})

	// Edge: update existing key refreshes LRU position
	t.Run("update existing", func(t *testing.T) {
		c := newEmbedCache(3)
		c.put("a", []float32{1})
		c.put("b", []float32{2})
		c.put("c", []float32{3})
		c.put("a", []float32{99}) // update 'a' - value changes, moves to end
		v, _ := c.get("a")
		assert.Equal(t, []float32{99}, v, "value should be updated")
		assert.Equal(t, 3, c.size(), "size should not change on update")
		// 'a' should be last in LRU order (most recently used)
		assert.Equal(t, "a", c.order[len(c.order)-1])
	})

	// Edge: exceeding capacity evicts oldest (LRU)
	t.Run("eviction", func(t *testing.T) {
		c := newEmbedCache(2)
		c.put("first", []float32{1})
		c.put("second", []float32{2})
		c.put("third", []float32{3})
		assert.Equal(t, 2, c.size(), "size capped at max")
		_, ok := c.get("first")
		assert.False(t, ok, "oldest should be evicted")
		_, ok = c.get("second")
		assert.True(t, ok)
		_, ok = c.get("third")
		assert.True(t, ok)
	})
}

func TestEmbedCache_Size(t *testing.T) {
	// Happy: empty cache
	t.Run("empty", func(t *testing.T) {
		c := newEmbedCache(10)
		assert.Equal(t, 0, c.size())
	})

	// Happy: populated cache
	t.Run("populated", func(t *testing.T) {
		c := newEmbedCache(10)
		c.put("a", []float32{1})
		c.put("b", []float32{2})
		c.put("c", []float32{3})
		assert.Equal(t, 3, c.size())
	})

	// Edge: size stable after eviction round
	t.Run("after eviction", func(t *testing.T) {
		c := newEmbedCache(3)
		c.put("a", []float32{1})
		c.put("b", []float32{2})
		c.put("c", []float32{3})
		c.put("d", []float32{4}) // triggers eviction
		c.put("e", []float32{5}) // triggers another eviction
		assert.Equal(t, 3, c.size(), "size should remain at capacity")
	})
}
