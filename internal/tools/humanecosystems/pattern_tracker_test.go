package humanecosystems

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolUsageTracker_RecordUsage(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	t.Run("records usage successfully", func(t *testing.T) {
		err := tut.RecordUsage(ctx, "user1", "tool1", "data analysis")
		require.NoError(t, err)

		patterns, err := tut.GetUsagePatterns(ctx, "tool1")
		require.NoError(t, err)
		assert.Len(t, patterns, 1)
		assert.Equal(t, "user1", patterns[0].UserID)
		assert.Equal(t, "tool1", patterns[0].ToolID)
		assert.Equal(t, 1, patterns[0].Frequency)
	})

	t.Run("increments frequency on repeated usage", func(t *testing.T) {
		err := tut.RecordUsage(ctx, "user1", "tool1", "data analysis")
		require.NoError(t, err)

		patterns, err := tut.GetUsagePatterns(ctx, "tool1")
		require.NoError(t, err)
		assert.Len(t, patterns, 1)
		assert.Equal(t, 2, patterns[0].Frequency)
	})

	t.Run("rejects empty userID", func(t *testing.T) {
		err := tut.RecordUsage(ctx, "", "tool1", "test")
		assert.Error(t, err)
	})

	t.Run("rejects empty toolID", func(t *testing.T) {
		err := tut.RecordUsage(ctx, "user1", "", "test")
		assert.Error(t, err)
	})

	t.Run("tracks different users separately", func(t *testing.T) {
		err := tut.RecordUsage(ctx, "user2", "tool1", "forecast")
		require.NoError(t, err)

		patterns, err := tut.GetUsagePatterns(ctx, "tool1")
		require.NoError(t, err)
		assert.Len(t, patterns, 2)
	})
}

func TestToolUsageTracker_GetUsagePatterns(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	t.Run("returns empty slice for unused tool", func(t *testing.T) {
		patterns, err := tut.GetUsagePatterns(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, patterns)
	})

	t.Run("returns only patterns for requested tool", func(t *testing.T) {
		tut.RecordUsage(ctx, "user1", "tool_a", "analysis")
		tut.RecordUsage(ctx, "user2", "tool_b", "viz")
		tut.RecordUsage(ctx, "user3", "tool_a", "report")

		patterns, err := tut.GetUsagePatterns(ctx, "tool_a")
		require.NoError(t, err)
		for _, p := range patterns {
			assert.Equal(t, "tool_a", p.ToolID)
		}
	})

	t.Run("rejects empty toolID", func(t *testing.T) {
		_, err := tut.GetUsagePatterns(ctx, "")
		assert.Error(t, err)
	})
}

func TestToolUsageTracker_GetRelationalContext(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	t.Run("returns empty relations for single tool co-used with nothing", func(t *testing.T) {
		tut.RecordUsage(ctx, "user1", "tool_isolated", "test")
		rels, err := tut.GetRelationalContext(ctx, []string{"tool_isolated"})
		require.NoError(t, err)
		assert.Empty(t, rels["tool_isolated"])
	})

	t.Run("detects co-used tools", func(t *testing.T) {
		tut.RecordUsage(ctx, "user_alpha", "tool_x", "analysis")
		tut.RecordUsage(ctx, "user_alpha", "tool_y", "viz")

		rels, err := tut.GetRelationalContext(ctx, []string{"tool_x"})
		require.NoError(t, err)
		assert.NotEmpty(t, rels["tool_x"])
	})

	t.Run("rejects empty toolIDs", func(t *testing.T) {
		_, err := tut.GetRelationalContext(ctx, []string{})
		assert.Error(t, err)
	})
}

func TestToolUsageTracker_GetTopUsers(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tut.RecordUsage(ctx, "heavy_user", "tool_popular", "frequent")
	}

	tut.RecordUsage(ctx, "light_user", "tool_popular", "once")

	t.Run("returns top users sorted by frequency", func(t *testing.T) {
		users, err := tut.GetTopUsers(ctx, "tool_popular", 2)
		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, "heavy_user", users[0])
	})

	t.Run("respects limit", func(t *testing.T) {
		users, err := tut.GetTopUsers(ctx, "tool_popular", 1)
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})
}

func TestToolUsageTracker_GetToolFrequency(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	tut.RecordUsage(ctx, "u1", "tool_freq", "a")
	tut.RecordUsage(ctx, "u1", "tool_freq", "b")
	tut.RecordUsage(ctx, "u2", "tool_freq", "c")

	freq, err := tut.GetToolFrequency(ctx, "tool_freq")
	require.NoError(t, err)
	assert.Equal(t, 3, freq)

	freq, err = tut.GetToolFrequency(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, freq)
}

func TestToolUsageTracker_MaxEntries(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	// Fill beyond max
	for i := 0; i < 11000; i++ {
		err := tut.RecordUsage(ctx, "user", "tool", "test")
		require.NoError(t, err)
	}

	// Should not panic and entries should be bounded
	freq, err := tut.GetToolFrequency(ctx, "tool")
	require.NoError(t, err)
	assert.Greater(t, freq, 0)

	// Give time for timestamp differentiation
	time.Sleep(time.Millisecond)
}

func TestTimeOfDay(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{3, "night"},
		{6, "morning"},
		{12, "afternoon"},
		{18, "evening"},
		{23, "night"},
	}

	for _, tc := range tests {
		ts := time.Date(2024, 1, 1, tc.hour, 0, 0, 0, time.UTC)
		got := timeOfDay(ts)
		assert.Equal(t, tc.want, got, "hour=%d", tc.hour)
	}
}
