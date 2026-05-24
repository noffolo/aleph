package ingestion

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterBasic(t *testing.T) {
	rl := NewTokenBucketRateLimiter(2, 5) // 2 req/s, burst 5

	// First 5 calls should be immediate (burst)
	start := time.Now()
	for i := 0; i < 5; i++ {
		err := rl.Wait()
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 100*time.Millisecond, "burst calls should be fast")

	// 6th call should be rate-limited (~500ms wait at 2 rps)
	start = time.Now()
	err := rl.Wait()
	require.NoError(t, err)
	elapsed = time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond, "should wait ~500ms for next token")
}

func TestRateLimiterMultipleWaits(t *testing.T) {
	rl := NewTokenBucketRateLimiter(5, 1) // 5 req/s, minimal burst

	var totalWait time.Duration
	for i := 0; i < 6; i++ {
		start := time.Now()
		err := rl.Wait()
		require.NoError(t, err)
		totalWait += time.Since(start)
	}
	// 6 requests at 5 rps should take ~1 second total
	assert.GreaterOrEqual(t, totalWait, 800*time.Millisecond)
	assert.Less(t, totalWait, 2*time.Second)
}

func TestRateLimiterConcurrent(t *testing.T) {
	// Use a low rate so wall-clock time is meaningful even with concurrent callers.
	// 5 req/s, burst 2 → tokens every 200ms
	rl := NewTokenBucketRateLimiter(5, 2)
	const goroutines = 3
	const callsPerGoroutine = 10

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalWait time.Duration
	start := time.Now()

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < callsPerGoroutine; i++ {
				callStart := time.Now()
				err := rl.Wait()
				mu.Lock()
				totalWait += time.Since(callStart)
				mu.Unlock()
				if err != nil {
					t.Errorf("Wait() returned error: %v", err)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// 30 calls total with burst 2 → 28 rate-limited calls at 200ms each
	// Serialized wait: 28 × 200ms = 5.6s minimum
	// With 3 concurrent goroutines: wall-clock ~1.9s
	t.Logf("30 concurrent calls at 5 rps: wall=%v, totalSerialWait=%v", elapsed, totalWait)
	assert.GreaterOrEqual(t, elapsed, 800*time.Millisecond, "should be rate-limited under concurrency")
	// Total serialized wait must always be >= rate-limited minimum (28*200ms = 5.6s)
	assert.GreaterOrEqual(t, totalWait, 5600*time.Millisecond, "total serialized wait must respect rate limit")
}
