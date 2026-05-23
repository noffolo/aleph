package ingestion

import (
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
