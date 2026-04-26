package humanecosystems

import (
	"context"
	"fmt"
	"time"
)

// PluginViz generates structured JSON output for frontend visualisation of
// human ecosystem data. Designed to be consumed by web-based visualisation
// libraries (D3.js, vis.js, etc.).
type PluginViz struct {
	db *DuckDBLayer
}

// Name returns the tool name.
func (p *PluginViz) Name() string {
	return "he_plugin_viz"
}

// Description returns the tool description.
func (p *PluginViz) Description() string {
	return "Generates visualizations and diagrams for human ecosystem structures (beta) | is_synthetic=true | privacy-preserving"
}

// NewPluginViz creates a PluginViz backed by the given DuckDB layer.
func NewPluginViz(db *DuckDBLayer) *PluginViz {
	return &PluginViz{db: db}
}

// Execute runs the visualization plugin. Expects args with optional
// "viz_type" string (graph, heatmap, timeline). Returns structured JSON
// suitable for frontend rendering.
func (p *PluginViz) Execute(ctx context.Context, args map[string]any) (any, error) {
	vizType, _ := args["viz_type"].(string)
	if vizType == "" {
		vizType = "graph"
	}

	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "all"
	}

	if p.db.IsAvailable() {
		return p.queryViz(ctx, vizType, scope)
	}
	return p.syntheticViz(vizType, scope), nil
}

func (p *PluginViz) queryViz(ctx context.Context, vizType, scope string) (any, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, name, category FROM system_tools WHERE source_type = 'package' LIMIT 20`)
	if err != nil {
		return p.syntheticViz(vizType, scope), nil
	}
	defer rows.Close()

	nodes := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name, category string
		if err := rows.Scan(&id, &name, &category); err != nil {
			continue
		}
		nodes = append(nodes, map[string]interface{}{
			"id":          sha256Hash("viz:" + id),
			"label":       name,
			"group":       category,
			"is_synthetic": false,
		})
	}

	return buildVizOutput(vizType, scope, nodes, false), nil
}

func (p *PluginViz) syntheticViz(vizType, scope string) map[string]interface{} {
	nodes := []map[string]interface{}{
		{"id": sha256Hash("viz:ecosystem"), "label": "Human Ecosystem", "group": "root", "is_synthetic": true},
		{"id": sha256Hash("viz:research"), "label": "Research", "group": "activity", "is_synthetic": true},
		{"id": sha256Hash("viz:relations"), "label": "Relations", "group": "activity", "is_synthetic": true},
		{"id": sha256Hash("viz:geography"), "label": "Geography", "group": "context", "is_synthetic": true},
		{"id": sha256Hash("viz:patterns"), "label": "Patterns", "group": "analysis", "is_synthetic": true},
	}

	return buildVizOutput(vizType, scope, nodes, true)
}

func buildVizOutput(vizType, scope string, nodes []map[string]interface{}, isSynthetic bool) map[string]interface{} {
	output := map[string]interface{}{
		"viz_type":     vizType,
		"scope":        scope,
		"is_synthetic": isSynthetic,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}

	switch vizType {
	case "graph":
		output["nodes"] = nodes
		output["edges"] = buildSyntheticEdges(nodes, isSynthetic)
	case "heatmap":
		output["matrix"] = map[string]interface{}{
			"rows":    nodes,
			"columns": []string{"density", "activity", "connectivity"},
			"values":  fmt.Sprintf("%dx%d matrix", len(nodes), 3),
		}
	case "timeline":
		output["events"] = []map[string]interface{}{
			{"timestamp": time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339), "event": "analysis_start", "is_synthetic": isSynthetic},
			{"timestamp": time.Now().UTC().Format(time.RFC3339), "event": "current_state", "is_synthetic": isSynthetic},
		}
	default:
		output["nodes"] = nodes
		output["edges"] = buildSyntheticEdges(nodes, isSynthetic)
	}

	return output
}

func buildSyntheticEdges(nodes []map[string]interface{}, isSynthetic bool) []map[string]interface{} {
	edges := make([]map[string]interface{}, 0)
	for i := 1; i < len(nodes); i++ {
		if idA, ok := nodes[i-1]["id"].(string); ok {
			if idB, ok := nodes[i]["id"].(string); ok {
				edges = append(edges, map[string]interface{}{
					"from":         idA,
					"to":           idB,
					"label":        "connects",
					"is_synthetic": isSynthetic,
				})
			}
		}
	}
	return edges
}
