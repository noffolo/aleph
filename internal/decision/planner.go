package decision

import (
	"context"
)

// validateToolName checks if the tool name corresponds to a known tool.
// Built-in tools (search_data, analyze_sentiment, get_trust_score) always return true.
// For all other tools, checks the registry for an exact ID match.
// Unlike the previous implementation, this does NOT use prefix/substring matching,
// ensuring only explicitly registered tools are considered valid.
func validateToolName(ctx context.Context, name string, registry PluginRegistry) bool {
	switch name {
	case "search_data", "analyze_sentiment", "get_trust_score":
		return true
	}

	if registry == nil {
		return false
	}

	// Exact match only: try direct lookup by tool name as component ID
	comp, err := registry.GetComponentByID(ctx, name)
	if err == nil && comp != nil {
		return true
	}

	return false
}

// buildToolDefinitions delegates to engine.go's canonical buildToolDefinitions.
// This function is kept as a package-level accessor for tests and callers
// that don't have an Engine instance.
func buildToolDefinitions(ctx context.Context, metaRepo ToolRepository) ([]ToolDefinition, []map[string]any) {
	// Create a temporary engine to get the canonical tool definitions
	e := &Engine{metaRepo: metaRepo}
	defs := e.buildToolDefinitions(ctx)
	maps := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		fnMap := map[string]any{
			"name":        d.Function.Name,
			"description": d.Function.Description,
		}
		if d.Function.Parameters != nil {
			props := make(map[string]any)
			for k, v := range d.Function.Parameters.Properties {
				p := map[string]any{"type": v.Type}
				if v.Description != "" {
					p["description"] = v.Description
				}
				if v.Default != nil {
					p["default"] = v.Default
				}
				props[k] = p
			}
			fnMap["parameters"] = map[string]any{
				"type":       "object",
				"properties": props,
				"required":   d.Function.Parameters.Required,
			}
		}
		maps = append(maps, map[string]any{
			"type":     "function",
			"function": fnMap,
		})
	}
	return defs, maps
}
