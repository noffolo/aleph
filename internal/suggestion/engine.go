package suggestion

import (
	"context"
	"fmt"
	"sort"

	"github.com/ff3300/aleph-v2/internal/tools"
)

// NewEngine creates a suggestion engine backed by the given tool registry and embedder.
func NewEngine(registry *tools.ToolRegistry, embedder Embedder) *Engine {
	return &Engine{
		registry: registry,
		embedder: embedder,
		usage:    newUsageTracker(),
		cache:    newEmbedCache(defaultEmbedCacheSize),
	}
}

// Suggest returns up to topN tool suggestions for the given user message.
// Each suggestion includes a composite score (0.0-1.0) based on embedding
// similarity and usage frequency, along with a human-readable reason.
func (e *Engine) Suggest(ctx context.Context, userMessage string, topN int) ([]Suggestion, error) {
	if userMessage == "" || e.embedder == nil {
		return nil, nil
	}

	userEmbedding, err := e.embedder.Embed(ctx, userMessage)
	if err != nil {
		return nil, fmt.Errorf("embed user message: %w", err)
	}

	allTools := e.collectAllTools()
	if len(allTools) == 0 {
		return nil, nil
	}

	toolEmbeds := make([]ToolEmbed, 0, len(allTools))
	for _, t := range allTools {
		key := toolKeyStr(t.Category, t.Name)
		text := t.Name + ": " + t.Description

		embedding, err := e.getOrComputeEmbedding(ctx, key, text)
		if err != nil {
			continue
		}
		toolEmbeds = append(toolEmbeds, ToolEmbed{
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			Embedding:   embedding,
		})
	}

	if len(toolEmbeds) == 0 {
		return nil, nil
	}

	usageStats := e.usage.GetUsageStats()
	suggestions := Rank(toolEmbeds, userEmbedding, usageStats, toolKeyStr)

	if topN > 0 && len(suggestions) > topN {
		suggestions = suggestions[:topN]
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})
	return suggestions, nil
}

// TrackUsage records a tool usage event so future suggestions reflect
// the user's frequency patterns.
func (e *Engine) TrackUsage(category, name string) {
	e.usage.TrackUsage(category, name)
}

// GetUsageStats returns a copy of the current usage statistics.
func (e *Engine) GetUsageStats() map[string]int {
	return e.usage.GetUsageStats()
}

// ResetStats clears all usage counters (primarily for testing).
func (e *Engine) ResetStats() {
	e.usage.ResetStats()
}

func (e *Engine) collectAllTools() []tools.ToolDefinition {
	var all []tools.ToolDefinition
	for _, cat := range e.registry.Categories() {
		all = append(all, e.registry.List(cat)...)
	}
	return all
}

func (e *Engine) getOrComputeEmbedding(ctx context.Context, key, text string) ([]float32, error) {
	if cached, ok := e.cache.get(key); ok {
		return cached, nil
	}
	embedding, err := e.embedder.Embed(ctx, text)
	if err != nil {
		return nil, err
	}
	e.cache.put(key, embedding)
	return embedding, nil
}

func toolKeyStr(category, name string) string {
	return category + ":" + name
}
