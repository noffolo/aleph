package decision

import (
	"context"
	"strings"
)

// validateToolName checks if the tool name corresponds to a known tool.
// Built-in tools (search_data, analyze_sentiment, get_trust_score) always return true.
// For all other tools, checks the registry for a matching component.
func validateToolName(ctx context.Context, name string, registry PluginRegistry) bool {
	switch name {
	case "search_data", "analyze_sentiment", "get_trust_score":
		return true
	}

	if registry == nil {
		return false
	}

	// For custom/registered tools, check if the name matches any known component
	// Components are stored by ID, not name, so we try a direct lookup
	comp, err := registry.GetComponentByID(ctx, name)
	if err == nil && comp != nil {
		return true
	}

	// Case-insensitive match against common patterns
	lower := strings.ToLower(name)
	knownToolPatterns := []string{
		"execute", "run", "call", "invoke",
		"transform", "process", "compute",
		"send", "fetch", "load", "save",
	}

	for _, pattern := range knownToolPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// buildToolDefinitions delegates to engine.go's canonical buildToolDefinitions.
// This function is kept as a package-level accessor for tests and callers
// that don't have an Engine instance.
func buildToolDefinitions(ctx context.Context, metaRepo ToolRepository) ([]ToolDefinition, []map[string]interface{}) {
	// Create a temporary engine to get the canonical tool definitions
	e := &Engine{metaRepo: metaRepo}
	defs := e.buildToolDefinitions(ctx)
	maps := make([]map[string]interface{}, 0, len(defs))
	for _, d := range defs {
		fnMap := map[string]interface{}{
			"name":        d.Function.Name,
			"description": d.Function.Description,
		}
		if d.Function.Parameters != nil {
			props := make(map[string]interface{})
			for k, v := range d.Function.Parameters.Properties {
				p := map[string]interface{}{"type": v.Type}
				if v.Description != "" {
					p["description"] = v.Description
				}
				if v.Default != nil {
					p["default"] = v.Default
				}
				props[k] = p
			}
			fnMap["parameters"] = map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   d.Function.Parameters.Required,
			}
		}
		maps = append(maps, map[string]interface{}{
			"type":     "function",
			"function": fnMap,
		})
	}
	return defs, maps
}
