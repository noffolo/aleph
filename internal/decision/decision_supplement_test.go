package decision

import (
	"context"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	t.Run("builtin search_data", func(t *testing.T) {
		if !validateToolName(context.Background(), "search_data", nil) {
			t.Error("search_data should be valid")
		}
	})
	t.Run("builtin analyze_sentiment", func(t *testing.T) {
		if !validateToolName(context.Background(), "analyze_sentiment", nil) {
			t.Error("analyze_sentiment should be valid")
		}
	})
	t.Run("builtin get_trust_score", func(t *testing.T) {
		if !validateToolName(context.Background(), "get_trust_score", nil) {
			t.Error("get_trust_score should be valid")
		}
	})
	t.Run("nil registry returns false", func(t *testing.T) {
		if validateToolName(context.Background(), "unknown_tool", nil) {
			t.Error("unknown tool with nil registry should be invalid")
		}
	})
	t.Run("registered tool", func(t *testing.T) {
		reg := &mockPluginRegistry{
			components: map[string]*ComponentMetadata{
				"my_tool": {ID: "my_tool", Name: "My Tool"},
			},
		}
		if !validateToolName(context.Background(), "my_tool", reg) {
			t.Error("my_tool should be valid via registry")
		}
	})
	t.Run("unregistered tool", func(t *testing.T) {
		reg := &mockPluginRegistry{components: map[string]*ComponentMetadata{}}
		if validateToolName(context.Background(), "missing_tool", reg) {
			t.Error("missing_tool should not be valid")
		}
	})
}

func TestBuildToolDefinitions(t *testing.T) {
	t.Run("always returns items", func(t *testing.T) {
		repo := &mockToolRepository{tools: []ToolDef{}}
		defs, maps := buildToolDefinitions(context.Background(), repo)
		if len(defs) == 0 || len(maps) == 0 {
			t.Errorf("expected non-empty, got %d+%d", len(defs), len(maps))
		}
	})
	t.Run("includes custom tool", func(t *testing.T) {
		repo := &mockToolRepository{tools: []ToolDef{{Name: "custom_search", Description: "custom"}}}
		defs, _ := buildToolDefinitions(context.Background(), repo)
		found := false
		for _, d := range defs {
			if d.Function.Name == "custom_search" {
				found = true
			}
		}
		if !found {
			t.Error("expected custom_search in definitions")
		}
	})
}
