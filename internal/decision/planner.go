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

// buildToolDefinitions is the standalone version used by engine.go.
// The Engine's buildToolDefinitions method delegates to this logic.
func buildToolDefinitions(ctx context.Context, metaRepo ToolRepository) ([]ToolDefinition, []map[string]interface{}) {
	defs := buildHardcodedDefs()
	maps := buildHardcodedMaps()

	if metaRepo != nil {
		registeredTools, err := metaRepo.ListTools(ctx)
		if err == nil {
			for _, t := range registeredTools {
				fn := FunctionDef{
					Name:        t.Name,
					Description: t.Description,
				}
				defs = append(defs, ToolDefinition{
					Type:     "function",
					Function: fn,
				})
				fnMap := map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
				}
				maps = append(maps, map[string]interface{}{
					"type":     "function",
					"function": fnMap,
				})
			}
		}
	}

	return defs, maps
}

func buildHardcodedDefs() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "search_data",
				Description: "Search records from a specific business object defined in the ontology.",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"object_name": {Type: "string"},
						"limit":       {Type: "integer", Default: float64(10)},
					},
					Required: []string{"object_name"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "analyze_sentiment",
				Description: "Analyze the sentiment of text data. Returns a score from -1.0 (negative) to 1.0 (positive) and a label (positive/negative/neutral).",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"text": {Type: "string", Description: "The text to analyze"},
					},
					Required: []string{"text"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_trust_score",
				Description: "Get the trust score for a prediction entity. Returns the Brier score (0.0 = perfect, 1.0 = worst) and trust level.",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"entity_id": {Type: "string", Description: "The entity ID to check trust for"},
					},
					Required: []string{"entity_id"},
				},
			},
		},
	}
}

func buildHardcodedMaps() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "search_data",
				"description": "Search records from a specific business object defined in the ontology.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"object_name": map[string]interface{}{"type": "string"},
						"limit":       map[string]interface{}{"type": "integer", "default": 10},
					},
					"required": []string{"object_name"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "analyze_sentiment",
				"description": "Analyze the sentiment of text data. Returns a score from -1.0 (negative) to 1.0 (positive) and a label (positive/negative/neutral).",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{"type": "string", "description": "The text to analyze"},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_trust_score",
				"description": "Get the trust score for a prediction entity. Returns the Brier score (0.0 = perfect, 1.0 = worst) and trust level.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{"type": "string", "description": "The entity ID to check trust for"},
					},
					"required": []string{"entity_id"},
				},
			},
		},
	}
}
