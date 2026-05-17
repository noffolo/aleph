package suggestion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	toolreg "github.com/ff3300/aleph-v2/internal/tools"
)

func TestEngine_Suggest_EmptyMessage(t *testing.T) {
	mockTools := []mockTool{
		{name: "tool1", category: "test", description: "test tool"},
	}
	reg := buildMockRegistry(t, mockTools)
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	suggestions, err := eng.Suggest(context.Background(), "", 5)
	assert.NoError(t, err)
	assert.Nil(t, suggestions, "empty message should return nil")
}

func TestEngine_Suggest_NilEmbedder(t *testing.T) {
	mockTools := []mockTool{
		{name: "tool1", category: "test", description: "test tool"},
	}
	reg := buildMockRegistry(t, mockTools)
	eng := NewEngine(reg, nil)

	suggestions, err := eng.Suggest(context.Background(), "hello", 5)
	assert.NoError(t, err)
	assert.Nil(t, suggestions, "nil embedder should return nil")
}

func TestEngine_TrackUsage(t *testing.T) {
	reg := toolreg.NewToolRegistry()
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	eng.TrackUsage("finance", "prophet")
	eng.TrackUsage("finance", "prophet")
	eng.TrackUsage("osint", "shodan")

	stats := eng.GetUsageStats()
	assert.Equal(t, 2, stats["finance:prophet"])
	assert.Equal(t, 1, stats["osint:shodan"])
	assert.Equal(t, 0, stats["nonexistent:tool"])
}

func TestEngine_ResetStats(t *testing.T) {
	reg := toolreg.NewToolRegistry()
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	eng.TrackUsage("cat", "tool")
	assert.Equal(t, 1, eng.GetUsageStats()["cat:tool"])

	eng.ResetStats()
	assert.Empty(t, eng.GetUsageStats())
}

func TestEngine_Suggest_TopNLimits(t *testing.T) {
	mockTools := []mockTool{
		{name: "a", category: "cat", description: "first tool"},
		{name: "b", category: "cat", description: "second tool"},
		{name: "c", category: "cat", description: "third tool"},
	}
	reg := buildMockRegistry(t, mockTools)
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	suggestions, err := eng.Suggest(context.Background(), "test query", 2)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(suggestions), 2)
}

func TestEngine_Suggest_TopNZero_ReturnsAll(t *testing.T) {
	mockTools := []mockTool{
		{name: "a", category: "cat", description: "first tool"},
		{name: "b", category: "cat", description: "second tool"},
	}
	reg := buildMockRegistry(t, mockTools)
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	suggestions, err := eng.Suggest(context.Background(), "test query", 0)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 2)
}

func TestEngine_Suggest_EmptyRegistry(t *testing.T) {
	reg := toolreg.NewToolRegistry()
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	suggestions, err := eng.Suggest(context.Background(), "hello world", 5)
	assert.NoError(t, err)
	assert.Nil(t, suggestions)
}

func TestToolKeyStr(t *testing.T) {
	assert.Equal(t, "finance:prophet", toolKeyStr("finance", "prophet"))
	assert.Equal(t, "osint:shodan", toolKeyStr("osint", "shodan"))
	assert.Equal(t, ":empty", toolKeyStr("", "empty"))
	assert.Equal(t, "cat:", toolKeyStr("cat", ""))
}

func TestEngine_Suggest_EmbeddingError_Continues(t *testing.T) {
	mockTools := []mockTool{
		{name: "a", category: "cat", description: "tool a"},
	}
	reg := buildMockRegistry(t, mockTools)
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	suggestions, err := eng.Suggest(context.Background(), "query", 5)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 1)
}

func TestNewEngine(t *testing.T) {
	reg := toolreg.NewToolRegistry()
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)
	assert.NotNil(t, eng)
	assert.NotNil(t, eng.registry)
	assert.NotNil(t, eng.embedder)
	assert.NotNil(t, eng.usage)
	assert.NotNil(t, eng.cache)
}
