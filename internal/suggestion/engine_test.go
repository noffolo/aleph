package suggestion

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	toolreg "github.com/ff3300/aleph-v2/internal/tools"
)

// errorEmbedder always returns an error from Embed.
type errorEmbedder struct{}

func (e *errorEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("embedder is down")
}

// zeroVecEmbedder returns a zero-length embedding.
type zeroVecEmbedder struct{}

func (z *zeroVecEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{}, nil
}

func TestEngine_Suggest_EmbedderError(t *testing.T) {
	// Happy: embedder error is propagated
	t.Run("propagated error", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "tool1", category: "test", description: "test tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		eng := NewEngine(reg, &errorEmbedder{})

		_, err := eng.Suggest(context.Background(), "hello", 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embed user message")
	})

	// Edge: no tools in registry returns nil (even with valid embedder)
	t.Run("empty registry", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "hello", 5)
		assert.NoError(t, err)
		assert.Nil(t, suggestions)
	})

	// Error: nil embedder returns nil
	t.Run("nil embedder", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		eng := NewEngine(reg, nil)

		suggestions, err := eng.Suggest(context.Background(), "hello", 5)
		assert.NoError(t, err)
		assert.Nil(t, suggestions)
	})
}

func TestEngine_Suggest_WithUsage(t *testing.T) {
	// Happy: usage stats affect rankings
	t.Run("usage boosts ranking", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "a", category: "cat", description: "tool a"},
			{name: "b", category: "cat", description: "tool b"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		// Track usage for tool 'b' heavily
		for i := 0; i < 100; i++ {
			eng.TrackUsage("cat", "b")
		}

		suggestions, err := eng.Suggest(context.Background(), "some query", 5)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 2)
		// Tool 'b' should have non-zero usage score
		assert.Greater(t, suggestions[0].UsageScore+suggestions[1].UsageScore, 0.0)
	})

	// Edge: zero usage across all tools
	t.Run("zero usage", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "x", category: "cat", description: "tool x"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "query", 5)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 1)
		assert.Equal(t, 0.0, suggestions[0].UsageScore)
	})

	// Edge: single tool registry
	t.Run("single tool", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "solo", category: "cat", description: "only tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "query", 5)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 1)
		assert.Equal(t, "solo", suggestions[0].ToolName)
	})
}

func TestEngine_CollectAllTools(t *testing.T) {
	// Happy: collects tools from multiple categories
	t.Run("multiple categories", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "t1", category: "osint", description: "osint tool"},
			{name: "t2", category: "finance", description: "finance tool"},
			{name: "t3", category: "osint", description: "another osint tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		all := eng.collectAllTools()
		assert.Len(t, all, 3)
	})

	// Edge: empty registry returns empty slice
	t.Run("empty registry", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		all := eng.collectAllTools()
		assert.Len(t, all, 0)
	})

	// Edge: single category
	t.Run("single category", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "a", category: "osint", description: "d"},
			{name: "b", category: "osint", description: "d"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		all := eng.collectAllTools()
		assert.Len(t, all, 2)
	})
}

func TestEngine_GetOrComputeEmbedding(t *testing.T) {
	// Happy: cache miss computes and stores
	t.Run("cache miss computes", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		emb, err := eng.getOrComputeEmbedding(context.Background(), "key1", "text1")
		assert.NoError(t, err)
		assert.NotNil(t, emb)
		assert.Len(t, emb, 768)
	})

	// Edge: cache hit returns cached value
	t.Run("cache hit returns cached", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		// First call: compute
		emb1, _ := eng.getOrComputeEmbedding(context.Background(), "key2", "text2")
		// Second call: should be cached
		emb2, err := eng.getOrComputeEmbedding(context.Background(), "key2", "same text")
		assert.NoError(t, err)
		assert.Equal(t, emb1, emb2, "cached embedding should match computed")
	})

	// Error: embedder error propagates and not cached
	t.Run("embedder error", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		eng := NewEngine(reg, &errorEmbedder{})

		_, err := eng.getOrComputeEmbedding(context.Background(), "key3", "text3")
		assert.Error(t, err)
		assert.Equal(t, 0, eng.cache.size(), "failed embed should not be cached")
	})
}

func TestEngine_Suggest_ZeroVectorEmbedding(t *testing.T) {
	// Happy: zero-length embedding works correctly
	t.Run("zero vector user message", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "t", category: "cat", description: "tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		eng := NewEngine(reg, &zeroVecEmbedder{})

		suggestions, err := eng.Suggest(context.Background(), "query", 5)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 1)
		assert.Equal(t, 0.0, suggestions[0].Similarity)
	})

	// Edge: zero vector with usage
	t.Run("zero vector with usage", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "t", category: "cat", description: "tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		eng := NewEngine(reg, &zeroVecEmbedder{})
		eng.TrackUsage("cat", "t")

		suggestions, err := eng.Suggest(context.Background(), "query", 5)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 1)
		// Usage part of composite should make score > 0
		assert.Greater(t, suggestions[0].Score, 0.0)
	})

	// Error: empty tool embedding vs zero vector user
	t.Run("zero vector tool also", func(t *testing.T) {
		// Both user embedding and tool embedding are zero-length
		input := []ToolEmbed{
			{Name: "t", Category: "cat", Embedding: []float32{}},
		}
		result := Rank(input, []float32{}, nil, toolKeyStr)
		assert.Len(t, result, 1)
		assert.Equal(t, 0.0, result[0].Similarity)
	})
}

func TestEngine_Suggest_ManyCategories(t *testing.T) {
	// Happy: multiple tools across many categories
	t.Run("many categories", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "a", category: "osint", description: "osint tool"},
			{name: "b", category: "finance", description: "finance tool"},
			{name: "c", category: "synthesis", description: "synth tool"},
			{name: "d", category: "adaptation", description: "adapt tool"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "query", 10)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 4)
	})

	// Edge: topN greater than available tools
	t.Run("topN exceeds tools", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "a", category: "cat", description: "d"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "query", 100)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 1)
	})

	// Edge: topN zero returns all
	t.Run("topN zero returns all", func(t *testing.T) {
		mockTools := []mockTool{
			{name: "a", category: "cat", description: "d1"},
			{name: "b", category: "cat", description: "d2"},
		}
		reg := buildMockRegistry(t, mockTools)
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		suggestions, err := eng.Suggest(context.Background(), "query", 0)
		assert.NoError(t, err)
		assert.Len(t, suggestions, 2)
	})
}

func TestEngine_TrackUsage_MultipleCategories(t *testing.T) {
	// Happy: track across categories
	t.Run("multiple categories", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		eng.TrackUsage("osint", "shodan")
		eng.TrackUsage("finance", "prophet")
		eng.TrackUsage("osint", "thehound")

		stats := eng.GetUsageStats()
		assert.Equal(t, 1, stats["osint:shodan"])
		assert.Equal(t, 1, stats["finance:prophet"])
		assert.Equal(t, 1, stats["osint:thehound"])
	})

	// Edge: track same tool many times
	t.Run("high frequency", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		for i := 0; i < 1000; i++ {
			eng.TrackUsage("cat", "tool")
		}
		assert.Equal(t, 1000, eng.GetUsageStats()["cat:tool"])
	})

	// Edge: track zero times
	t.Run("never tracked", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		assert.Equal(t, 0, eng.GetUsageStats()["cat:tool"])
	})
}

func TestEngine_ResetStats_EmptyAlready(t *testing.T) {
	// Happy: reset empty is no-op
	t.Run("reset empty", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		eng.ResetStats()
		assert.Empty(t, eng.GetUsageStats())
	})

	// Edge: reset after tracking
	t.Run("reset after tracking", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		eng.TrackUsage("cat", "tool")
		eng.ResetStats()
		assert.Empty(t, eng.GetUsageStats())
	})

	// Edge: double reset
	t.Run("double reset", func(t *testing.T) {
		reg := toolreg.NewToolRegistry()
		mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
		eng := NewEngine(reg, mock)

		eng.TrackUsage("cat", "tool")
		eng.ResetStats()
		eng.ResetStats()
		assert.Empty(t, eng.GetUsageStats())
	})
}
