package suggestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUsageTracker(t *testing.T) {
	// Happy: creates a non-nil tracker with empty stats
	t.Run("creates empty tracker", func(t *testing.T) {
		ut := newUsageTracker()
		assert.NotNil(t, ut)
		assert.NotNil(t, ut.stats)
		assert.Len(t, ut.stats, 0)
	})

	// Edge: GetUsageStats on new tracker returns empty map
	t.Run("new tracker has empty stats", func(t *testing.T) {
		ut := newUsageTracker()
		stats := ut.GetUsageStats()
		assert.NotNil(t, stats)
		assert.Empty(t, stats)
	})

	// Edge: multiple new trackers are independent
	t.Run("independent instances", func(t *testing.T) {
		ut1 := newUsageTracker()
		ut2 := newUsageTracker()
		ut1.TrackUsage("cat", "tool")
		assert.Equal(t, 1, ut1.GetUsageStats()["cat:tool"])
		assert.Equal(t, 0, ut2.GetUsageStats()["cat:tool"])
	})
}

func TestUsageTracker_TrackUsage(t *testing.T) {
	// Happy: increments count correctly for single tool
	t.Run("single increment", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("osint", "shodan")
		assert.Equal(t, 1, ut.GetUsageStats()["osint:shodan"])
	})

	// Edge: multiple increments for same tool
	t.Run("multiple increments", func(t *testing.T) {
		ut := newUsageTracker()
		for i := 0; i < 5; i++ {
			ut.TrackUsage("finance", "prophet")
		}
		assert.Equal(t, 5, ut.GetUsageStats()["finance:prophet"])
	})

	// Edge: tracks different tools independently
	t.Run("independent tools", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("osint", "a")
		ut.TrackUsage("osint", "b")
		ut.TrackUsage("finance", "c")
		stats := ut.GetUsageStats()
		assert.Equal(t, 1, stats["osint:a"])
		assert.Equal(t, 1, stats["osint:b"])
		assert.Equal(t, 1, stats["finance:c"])
		assert.Len(t, stats, 3)
	})
}

func TestUsageTracker_GetUsageStats(t *testing.T) {
	// Happy: returns a copy, not the internal map
	t.Run("returns copy", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("cat", "tool")
		stats1 := ut.GetUsageStats()
		stats2 := ut.GetUsageStats()
		stats1["modified"] = 999

		// Original tracker should be unaffected by modification of returned copy
		assert.Equal(t, 0, ut.GetUsageStats()["modified"])
		// Both copies have different contents
		assert.Equal(t, 999, stats1["modified"])
		assert.Equal(t, 0, stats2["modified"])
	})

	// Edge: returns empty map for untouched tracker
	t.Run("empty tracker returns empty", func(t *testing.T) {
		ut := newUsageTracker()
		stats := ut.GetUsageStats()
		assert.Empty(t, stats)
	})

	// Edge: concurrent reading and writing does not race
	t.Run("concurrent get and track", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("cat", "tool")

		done := make(chan struct{})
		go func() {
			for i := 0; i < 100; i++ {
				ut.TrackUsage("cat", "tool")
			}
			close(done)
		}()
		for i := 0; i < 100; i++ {
			_ = ut.GetUsageStats()
		}
		<-done

		assert.Equal(t, 101, ut.GetUsageStats()["cat:tool"])
	})
}

func TestUsageTracker_ResetStats(t *testing.T) {
	// Happy: clears all tracked usage
	t.Run("clears all stats", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("osint", "shodan")
		ut.TrackUsage("finance", "prophet")
		ut.TrackUsage("synthesis", "tool")

		assert.Len(t, ut.GetUsageStats(), 3)
		ut.ResetStats()
		assert.Empty(t, ut.GetUsageStats())
	})

	// Edge: reset on already empty tracker is idempotent
	t.Run("reset empty is idempotent", func(t *testing.T) {
		ut := newUsageTracker()
		ut.ResetStats()
		assert.Empty(t, ut.GetUsageStats())
		ut.ResetStats()
		assert.Empty(t, ut.GetUsageStats())
	})

	// Edge: tracking after reset starts fresh
	t.Run("track after reset is fresh", func(t *testing.T) {
		ut := newUsageTracker()
		ut.TrackUsage("cat", "tool")
		ut.ResetStats()
		ut.TrackUsage("cat", "tool")
		assert.Equal(t, 1, ut.GetUsageStats()["cat:tool"])
	})
}
