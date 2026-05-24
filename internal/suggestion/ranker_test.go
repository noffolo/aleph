package suggestion

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRank(t *testing.T) {
	// Happy: ranks by similarity when no usage stats
	t.Run("ranks by similarity", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 2, 3}},
			{Name: "b", Category: "cat", Embedding: []float32{3, 2, 1}},
		}
		result := Rank(input, []float32{1, 2, 3}, nil, toolKeyStr)
		assert.Len(t, result, 2)
		assert.Equal(t, "a", result[0].ToolName)
		assert.Greater(t, result[0].Score, result[1].Score)
	})

	// Edge: empty toolDefs returns nil
	t.Run("empty input", func(t *testing.T) {
		result := Rank(nil, []float32{1, 2, 3}, nil, toolKeyStr)
		assert.Nil(t, result)
	})

	// Edge: single tool returns single normalized result
	t.Run("single tool", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "solo", Category: "cat", Embedding: []float32{1, 0}},
		}
		result := Rank(input, []float32{1, 0}, nil, toolKeyStr)
		assert.Len(t, result, 1)
		inDelta(t, 1.0, result[0].Score)
		inDelta(t, 1.0, result[0].Similarity)
	})
}

func TestRank_WithUsage(t *testing.T) {
	// Happy: usage affects composite score
	t.Run("usage weighted", func(t *testing.T) {
		usageStats := map[string]int{"cat:a": 0, "cat:b": 10}
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
			{Name: "b", Category: "cat", Embedding: []float32{0, 1}},
		}
		result := Rank(input, []float32{1, 0}, usageStats, toolKeyStr)
		assert.Len(t, result, 2)
		assert.Equal(t, "a", result[0].ToolName, "similarity should dominate over usage when similarity is high")
		assert.Greater(t, result[1].UsageScore, result[0].UsageScore, "b should have higher usage score")
	})

	// Edge: all usage zero — usage component is zero
	t.Run("all usage zero", func(t *testing.T) {
		usageStats := map[string]int{"cat:a": 0, "cat:b": 0}
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
			{Name: "b", Category: "cat", Embedding: []float32{0, 1}},
		}
		result := Rank(input, []float32{1, 0}, usageStats, toolKeyStr)
		assert.Len(t, result, 2)
		assert.Equal(t, 0.0, result[0].UsageScore)
		assert.Equal(t, 0.0, result[1].UsageScore)
	})

	// Edge: high usage can boost a dissimilar tool higher
	t.Run("usage boosts dissimilar", func(t *testing.T) {
		usageStats := map[string]int{"cat:a": 0, "cat:b": 1000000}
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
			{Name: "b", Category: "cat", Embedding: []float32{0, 1}},
		}
		result := Rank(input, []float32{1, 0}, usageStats, toolKeyStr)
		assert.Len(t, result, 2)
		assert.Equal(t, "a", result[0].ToolName, "high similarity should still dominate 0.7 weight")
		inDelta(t, 1.0, result[1].UsageScore)
	})
}

func TestMaxUsage(t *testing.T) {
	// Happy: returns maximum from stats
	t.Run("max value", func(t *testing.T) {
		stats := map[string]int{"a": 3, "b": 5, "c": 1}
		assert.Equal(t, 5, maxUsage(stats))
	})

	// Edge: empty stats returns 0
	t.Run("empty stats", func(t *testing.T) {
		assert.Equal(t, 0, maxUsage(nil))
		assert.Equal(t, 0, maxUsage(map[string]int{}))
	})

	// Edge: single entry
	t.Run("single entry", func(t *testing.T) {
		stats := map[string]int{"a": 42}
		assert.Equal(t, 42, maxUsage(stats))
	})
}

func TestNormalizeScores_Extended(t *testing.T) {
	// Happy: normalizes a range
	t.Run("normalizes range", func(t *testing.T) {
		input := []Suggestion{
			{Score: 0.0}, {Score: 0.5}, {Score: 1.0},
		}
		result := make([]Suggestion, len(input))
		copy(result, input)
		normalizeScores(result)
		inDelta(t, 0.0, result[0].Score)
		inDelta(t, 0.5, result[1].Score)
		inDelta(t, 1.0, result[2].Score)
	})

	// Edge: empty slice is no-op
	t.Run("empty slice", func(t *testing.T) {
		var s []Suggestion
		normalizeScores(s)
		assert.Nil(t, s)
	})

	// Edge: all identical scores become 1.0
	t.Run("all identical", func(t *testing.T) {
		input := []Suggestion{
			{Score: 0.3}, {Score: 0.3}, {Score: 0.3},
		}
		result := make([]Suggestion, len(input))
		copy(result, input)
		normalizeScores(result)
		for _, s := range result {
			inDelta(t, 1.0, s.Score)
		}
	})
}

func TestBuildReason(t *testing.T) {
	// Happy: both high
	t.Run("both high", func(t *testing.T) {
		assert.Equal(t, "Similar to tools you've used before", buildReason(0.6, 0.6))
	})

	// Edge: exactly at boundary values — both just above 0.5
	t.Run("boundary both high", func(t *testing.T) {
		assert.Equal(t, "Similar to tools you've used before", buildReason(0.500001, 0.500001))
	})

	// Edge: strong semantic only (>= 0.7) but usage low
	t.Run("boundary strong semantic", func(t *testing.T) {
		assert.Equal(t, "Strong semantic match with your request", buildReason(0.700001, 0.1))
	})
}

func TestCosineSimilarity_AllZero(t *testing.T) {
	// Happy: identical zero vectors (norm=0 falls through to return 0)
	t.Run("all zero returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, cosineSimilarity([]float32{0, 0}, []float32{0, 0}))
	})

	// Edge: one is zero, one is non-zero
	t.Run("one zero other non-zero", func(t *testing.T) {
		assert.Equal(t, 0.0, cosineSimilarity([]float32{0, 0}, []float32{1, 2}))
	})

	// Edge: negative similarity clamped to 0
	t.Run("negative clamped", func(t *testing.T) {
		assert.Equal(t, 0.0, cosineSimilarity([]float32{1, 0}, []float32{-2, 0}))
	})
}

func TestRank_SortedDescending(t *testing.T) {
	// Happy: results sorted in descending score order
	t.Run("descending order", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "low", Category: "cat", Embedding: []float32{0, 1}},
			{Name: "mid", Category: "cat", Embedding: []float32{1, 1}},
			{Name: "high", Category: "cat", Embedding: []float32{1, 0.1}},
		}
		result := Rank(input, []float32{1, 0}, nil, toolKeyStr)
		assert.Len(t, result, 3)
		for i := 1; i < len(result); i++ {
			assert.GreaterOrEqual(t, result[i-1].Score, result[i].Score)
		}
	})

	// Edge: same score preserves relative order (stable sort)
	t.Run("tie preservation", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
			{Name: "b", Category: "cat", Embedding: []float32{1, 0}},
		}
		result := Rank(input, []float32{1, 0}, nil, toolKeyStr)
		assert.Len(t, result, 2)
		inDelta(t, result[0].Score, result[1].Score)
	})

	// Error: zero embedding vectors for all tools
	t.Run("zero vectors", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "x", Category: "cat", Embedding: []float32{0, 0}},
			{Name: "y", Category: "cat", Embedding: []float32{0, 0}},
		}
		result := Rank(input, []float32{1, 0}, nil, toolKeyStr)
		assert.Len(t, result, 2)
		// All similarities are 0, so scores should be uniform after normalize
		inDelta(t, 1.0, result[0].Score)
		inDelta(t, 1.0, result[1].Score)
	})
}

func TestRank_ScoreComponents(t *testing.T) {
	// Happy: verify all fields are populated
	t.Run("all fields populated", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "tool", Category: "cat", Description: "desc", Embedding: []float32{1, 2}},
		}
		result := Rank(input, []float32{1, 2}, nil, toolKeyStr)
		assert.Len(t, result, 1)
		assert.Equal(t, "tool", result[0].ToolName)
		assert.Equal(t, "cat", result[0].Category)
		assert.Equal(t, "desc", result[0].Description)
		inDelta(t, 1.0, result[0].Similarity)
		assert.GreaterOrEqual(t, result[0].Score, 0.0)
		assert.NotEmpty(t, result[0].Reason)
	})

	// Edge: usage score for untracked tool is 0
	t.Run("untracked usage zero", func(t *testing.T) {
		usageStats := map[string]int{"cat:other": 100}
		input := []ToolEmbed{
			{Name: "tool", Category: "cat", Embedding: []float32{1, 0}},
		}
		result := Rank(input, []float32{1, 0}, usageStats, toolKeyStr)
		assert.Equal(t, 0.0, result[0].UsageScore)
	})

	// Edge: score is in [0, 1] range
	t.Run("score in range", func(t *testing.T) {
		input := []ToolEmbed{
			{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
			{Name: "b", Category: "cat", Embedding: []float32{0, 1}},
		}
		result := Rank(input, []float32{0.5, 0.5}, nil, toolKeyStr)
		for _, s := range result {
			assert.GreaterOrEqual(t, s.Score, 0.0)
			assert.LessOrEqual(t, s.Score, 1.0+1e-9)
		}
	})
}

// inDelta is a helper that asserts two float64 values are within epsilon of each other.
func inDelta(t *testing.T, expected, actual float64) {
	t.Helper()
	if math.Abs(expected-actual) > 1e-6 {
		t.Errorf("expected %v, got %v (delta=%v)", expected, actual, math.Abs(expected-actual))
	}
}
