package decision

import (
	"context"
	"errors"
	"testing"
)

// ─── validateToolName Tests ──────────────────────────────────────────────

func TestValidateToolName_BuiltinHappy(t *testing.T) {
	builtins := []string{"search_data", "analyze_sentiment", "get_trust_score"}
	for _, name := range builtins {
		if !validateToolName(context.Background(), name, nil) {
			t.Errorf("expected built-in tool %q to be valid even with nil registry", name)
		}
	}
}

func TestValidateToolName_NilRegistryNonBuiltin(t *testing.T) {
	if validateToolName(context.Background(), "custom_tool", nil) {
		t.Error("expected false for non-built-in tool with nil registry")
	}
}

func TestValidateToolName_RegistryError(t *testing.T) {
	// registry exists but returns nil/non-matched → edge case: exact match fails
	reg := &mockPluginRegistry{
		components: map[string]*ComponentMetadata{},
	}
	if validateToolName(context.Background(), "nonexistent_tool", reg) {
		t.Error("expected false for tool not in registry")
	}

	// Happy path via registry: tool registered by ID
	reg.components["registered_tool"] = &ComponentMetadata{ID: "registered_tool", Name: "Registered Tool"}
	if !validateToolName(context.Background(), "registered_tool", reg) {
		t.Error("expected true for tool registered by exact ID match")
	}
}

// ─── buildToolDefinitions Tests ──────────────────────────────────────────

type mockToolRepoForPlanner struct {
	tools []ToolDef
	err   error
}

func (m *mockToolRepoForPlanner) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	return nil
}

func (m *mockToolRepoForPlanner) GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	return nil, nil
}

func (m *mockToolRepoForPlanner) ListTools(ctx context.Context) ([]ToolDef, error) {
	return m.tools, m.err
}

func TestBuildToolDefinitions_HappyWithRegisteredTools(t *testing.T) {
	repo := &mockToolRepoForPlanner{
		tools: []ToolDef{
			{Name: "custom_search", Description: "Custom search tool"},
			{Name: "data_export", Description: "Export data tool"},
		},
	}
	defs, maps := buildToolDefinitions(context.Background(), repo)

	// Should return 3 builtins + 2 registered = 5 definitions
	if len(defs) != 5 {
		t.Fatalf("expected 5 definitions (3 builtin + 2 registered), got %d", len(defs))
	}
	if len(maps) != 5 {
		t.Fatalf("expected 5 map entries, got %d", len(maps))
	}

	// Check built-ins are first
	if defs[0].Function.Name != "search_data" {
		t.Errorf("expected first tool to be search_data, got %q", defs[0].Function.Name)
	}
	if defs[1].Function.Name != "analyze_sentiment" {
		t.Errorf("expected second tool to be analyze_sentiment, got %q", defs[1].Function.Name)
	}
	if defs[2].Function.Name != "get_trust_score" {
		t.Errorf("expected third tool to be get_trust_score, got %q", defs[2].Function.Name)
	}

	// Check registered tools follow
	if defs[3].Function.Name != "custom_search" {
		t.Errorf("expected fourth tool to be custom_search, got %q", defs[3].Function.Name)
	}
	if defs[4].Function.Name != "data_export" {
		t.Errorf("expected fifth tool to be data_export, got %q", defs[4].Function.Name)
	}

	// Verify map conversion preserves function names
	if maps[0]["function"].(map[string]any)["name"] != "search_data" {
		t.Error("map name mismatch for search_data")
	}

	// Verify parameter properties are in map
	fn := maps[2]["function"].(map[string]any)
	if fn["name"] != "get_trust_score" {
		t.Error("map name mismatch for get_trust_score")
	}
	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Error("expected parameters type 'object'")
	}
	if props, ok := params["properties"].(map[string]any); ok {
		if _, hasEntityID := props["entity_id"]; !hasEntityID {
			t.Error("expected entity_id property in get_trust_score map")
		}
	}
}

func TestBuildToolDefinitions_EdgeNilRepo(t *testing.T) {
	// nil metaRepo → only 3 builtins, no panic
	defs, maps := buildToolDefinitions(context.Background(), nil)

	if len(defs) != 3 {
		t.Fatalf("expected 3 built-in definitions with nil repo, got %d", len(defs))
	}
	if len(maps) != 3 {
		t.Fatalf("expected 3 map entries with nil repo, got %d", len(maps))
	}

	// Verify all three built-ins are present
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	for _, expected := range []string{"search_data", "analyze_sentiment", "get_trust_score"} {
		if !names[expected] {
			t.Errorf("missing built-in tool %q", expected)
		}
	}
}

func TestBuildToolDefinitions_ErrorListToolsFails(t *testing.T) {
	repo := &mockToolRepoForPlanner{
		err: errors.New("repository unavailable"),
	}
	defs, maps := buildToolDefinitions(context.Background(), repo)

	// Should gracefully degrade to 3 builtins
	if len(defs) != 3 {
		t.Fatalf("expected 3 built-in definitions when ListTools fails, got %d", len(defs))
	}
	if len(maps) != 3 {
		t.Fatalf("expected 3 map entries when ListTools fails, got %d", len(maps))
	}

	for _, d := range defs {
		if d.Type != "function" {
			t.Errorf("expected type 'function', got %q", d.Type)
		}
	}
}
