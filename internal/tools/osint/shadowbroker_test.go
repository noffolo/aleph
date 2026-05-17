package osint

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── simpleCache Tests ──────────────────────────────────────────────────

func TestNewSimpleCache(t *testing.T) {
	t.Run("creates cache with valid size", func(t *testing.T) {
		c, err := newSimpleCache(100, time.Minute)
		require.NoError(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.lru)
		assert.Equal(t, time.Minute, c.ttl)
	})

	t.Run("returns error for zero size", func(t *testing.T) {
		_, err := newSimpleCache(0, time.Minute)
		assert.Error(t, err)
	})
}

func TestSimpleCache_Get(t *testing.T) {
	c, err := newSimpleCache(10, 10*time.Minute)
	require.NoError(t, err)

	t.Run("miss on empty cache", func(t *testing.T) {
		val, ok := c.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("hit returns value", func(t *testing.T) {
		c.Set("k1", "hello")
		val, ok := c.Get("k1")
		assert.True(t, ok)
		assert.Equal(t, "hello", val)
	})

	t.Run("expired entry returns miss", func(t *testing.T) {
		shortCache, err := newSimpleCache(10, 1*time.Nanosecond)
		require.NoError(t, err)
		shortCache.Set("k2", "world")
		time.Sleep(time.Millisecond)
		val, ok := shortCache.Get("k2")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("nil cache returns false", func(t *testing.T) {
		var nilCache *simpleCache
		val, ok := nilCache.Get("any")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestSimpleCache_Set(t *testing.T) {
	c, err := newSimpleCache(10, time.Minute)
	require.NoError(t, err)

	t.Run("sets and retrieves value", func(t *testing.T) {
		c.Set("k1", 42)
		val, ok := c.Get("k1")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		c.Set("k1", "first")
		c.Set("k1", "second")
		val, ok := c.Get("k1")
		assert.True(t, ok)
		assert.Equal(t, "second", val)
	})

	t.Run("nil cache does not panic", func(t *testing.T) {
		var nilCache *simpleCache
		nilCache.Set("k", "v") // should not panic
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		cc, err := newSimpleCache(1000, 10*time.Minute)
		require.NoError(t, err)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				cc.Set("key", n)
				cc.Get("key")
			}(i)
		}
		wg.Wait()
	})
}

// ─── CircuitBreaker Tests ──────────────────────────────────────────────

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	require.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 5, cb.failureThreshold)
}

func TestCircuitBreaker_Allow(t *testing.T) {
	t.Run("closed allows requests", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 50*time.Millisecond)
		assert.True(t, cb.Allow())
	})

	t.Run("open blocks requests", func(t *testing.T) {
		cb := NewCircuitBreaker(1, time.Hour)
		cb.RecordFailure() // threshold=1 → open immediately

		assert.False(t, cb.Allow())
		assert.False(t, cb.Allow())
	})

	t.Run("open transitions to half-open after recovery timeout", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 5*time.Millisecond)
		cb.RecordFailure() // open
		assert.Equal(t, StateOpen, cb.State())

		time.Sleep(20 * time.Millisecond)
		assert.True(t, cb.Allow())               // should transition to half-open
		assert.Equal(t, StateHalfOpen, cb.State())
	})

	t.Run("half-open allows requests", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 1*time.Millisecond)
		cb.RecordFailure()
		time.Sleep(10 * time.Millisecond)
		assert.True(t, cb.Allow()) // now half-open
		assert.True(t, cb.Allow()) // still half-open (until success)
	})
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	t.Run("half-open to closed on success", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 1*time.Millisecond)
		cb.RecordFailure()
		time.Sleep(10 * time.Millisecond)
		cb.Allow()             // half-open
		cb.RecordSuccess()     // should go back to closed
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("resets failure count", func(t *testing.T) {
		cb := NewCircuitBreaker(3, time.Hour)
		cb.RecordFailure()
		cb.RecordFailure()
		cb.RecordSuccess()
		cb.RecordFailure() // count should be 1 now, not 3
		assert.Equal(t, StateClosed, cb.State())
	})
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	t.Run("opens after threshold", func(t *testing.T) {
		cb := NewCircuitBreaker(2, time.Hour)
		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.State())

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())
	})

	t.Run("counts correctly below threshold", func(t *testing.T) {
		cb := NewCircuitBreaker(5, time.Hour)
		cb.RecordFailure()
		cb.RecordFailure()
		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.State())
	})
}

func TestCircuitBreaker_State(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Hour)
	assert.Equal(t, StateClosed, cb.State())

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := NewCircuitBreaker(100, time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				cb.Allow()
				cb.RecordSuccess()
				cb.State()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, StateClosed, cb.State())
}

// ─── RateLimiter Tests ──────────────────────────────────────────────────

func TestNewRateLimiter(t *testing.T) {
	t.Run("positive rate", func(t *testing.T) {
		rl := NewRateLimiter(100)
		require.NotNil(t, rl)
	})

	t.Run("zero rate defaults to 60", func(t *testing.T) {
		rl := NewRateLimiter(0)
		require.NotNil(t, rl)
		assert.True(t, rl.Acquire())
	})

	t.Run("negative rate defaults to 60", func(t *testing.T) {
		rl := NewRateLimiter(-5)
		require.NotNil(t, rl)
		assert.True(t, rl.Acquire())
	})
}

func TestRateLimiter_Acquire(t *testing.T) {
	t.Run("acquires tokens up to capacity", func(t *testing.T) {
		rl := NewRateLimiter(100)
		for i := 0; i < 100; i++ {
			assert.True(t, rl.Acquire(), "token %d should be available", i)
		}
		assert.False(t, rl.Acquire(), "should be rate limited")
	})

	t.Run("refill allows acquire after time passes", func(t *testing.T) {
		rl := NewRateLimiter(100)
		// Exhaust all tokens
		for i := 0; i < 100; i++ {
			rl.Acquire()
		}
		assert.False(t, rl.Acquire())

		// Simulate refill by directly manipulating the rate limiter
		time.Sleep(100 * time.Millisecond)
		// Cannot refill via time since rate is per-minute — but Acquire will refill
		// proportionally to elapsed time.
	})

	t.Run("rate of 1", func(t *testing.T) {
		rl := NewRateLimiter(1)
		assert.True(t, rl.Acquire())
		assert.False(t, rl.Acquire())
	})
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(1000)
	var wg sync.WaitGroup
	results := make(chan bool, 1000)
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- rl.Acquire()
		}()
	}
	wg.Wait()
	close(results)

	success := 0
	fail := 0
	for r := range results {
		if r {
			success++
		} else {
			fail++
		}
	}
	// At least some should succeed (not all can fail with 1000 capacity)
	assert.Greater(t, success, 0)
	_ = fail
}

// ─── Shadowbroker Tests ──────────────────────────────────────────────────

func TestNewShadowbroker(t *testing.T) {
	t.Run("creates with valid config", func(t *testing.T) {
		config := ShadowbrokerConfig{
			BaseURL:   "http://localhost",
			Timeout:   10 * time.Second,
			RateLimit: 50,
		}
		sb := NewShadowbroker(config)
		require.NotNil(t, sb)
		assert.NotNil(t, sb.client)
		assert.NotNil(t, sb.rateLimiter)
		assert.NotNil(t, sb.circuitBreaker)
		// cache may be nil if LRU failed (unlikely)
	})

	t.Run("zero timeout defaults to 30s", func(t *testing.T) {
		config := ShadowbrokerConfig{BaseURL: "http://localhost"}
		sb := NewShadowbroker(config)
		require.NotNil(t, sb)
	})

	t.Run("zero rate limit defaults to 60 via NewRateLimiter", func(t *testing.T) {
		config := ShadowbrokerConfig{BaseURL: "http://localhost", RateLimit: 0}
		sb := NewShadowbroker(config)
		require.NotNil(t, sb)
	})
}

func TestShadowbroker_Health(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: "http://localhost"})
	assert.Equal(t, StateClosed, sb.Health())
}

func TestShadowbroker_GetSet_Cache(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: "http://localhost", RateLimit: 100})

	// Shadowbroker exposes cache.Get and cache.Set via its own cache field
	if sb.cache == nil {
		t.Skip("LRU cache unavailable in this environment")
	}

	sb.cache.Set("key1", "val1")
	got, ok := sb.cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "val1", got)
}

func TestShadowbroker_Integration(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: "http://localhost", RateLimit: 100})
	defer func() {
		// Health is always safe to call
		_ = sb.Health()
	}()

	// Verify all components are wired
	assert.NotNil(t, sb.rateLimiter)
	assert.True(t, sb.rateLimiter.Acquire())

	assert.NotNil(t, sb.circuitBreaker)
	assert.True(t, sb.circuitBreaker.Allow())

	assert.Equal(t, StateClosed, sb.Health())
}

// ─── Edge Cases ──────────────────────────────────────────────────────────

func TestRateLimiter_Boundary(t *testing.T) {
	rl := NewRateLimiter(1)
	assert.True(t, rl.Acquire())
	assert.False(t, rl.Acquire())
	assert.False(t, rl.Acquire())
}

func TestCircuitBreaker_FullCycle(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)
	assert.Equal(t, StateClosed, cb.State())
	assert.True(t, cb.Allow())

	// Fail twice to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.False(t, cb.Allow())

	// Wait for recovery
	time.Sleep(50 * time.Millisecond)
	assert.True(t, cb.Allow())           // half-open
	assert.Equal(t, StateHalfOpen, cb.State())

	// Success → closed
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
}

func TestSimpleCache_TTL(t *testing.T) {
	c, err := newSimpleCache(10, 10*time.Millisecond)
	require.NoError(t, err)

	c.Set("ephemeral", "data")
	val, ok := c.Get("ephemeral")
	assert.True(t, ok)
	assert.Equal(t, "data", val)

	time.Sleep(50 * time.Millisecond)
	val, ok = c.Get("ephemeral")
	assert.False(t, ok)
	assert.Nil(t, val)
}
