package codeflow

import (
	"encoding/json"
	"fmt"
)

// GraphData represents a graph suitable for D3.js force-directed layout or GraphViz.
type GraphData struct {
	Nodes []GraphDataNode `json:"nodes"`
	Edges []GraphDataEdge `json:"edges"`
}

// GraphDataNode represents a node in the visualization graph.
type GraphDataNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Group  string `json:"group,omitempty"`  // Node category for coloring
	Weight int    `json:"weight,omitempty"` // Optional visual weight
}

// GraphDataEdge represents an edge in the visualization graph.
type GraphDataEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Weight int    `json:"weight"`
}

// ToD3JSON serializes the GraphData to D3.js force-directed graph JSON format.
func (gd *GraphData) ToD3JSON() string {
	data := map[string]interface{}{
		"nodes": gd.Nodes,
		"links": gd.Edges,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ToGraphViz serializes the GraphData to DOT format for GraphViz.
func (gd *GraphData) ToGraphViz() string {
	out := "digraph ToolGraph {\n"
	out += "  rankdir=LR;\n"
	out += "  node [shape=box, style=rounded];\n"

	for _, n := range gd.Nodes {
		label := n.Label
		if label == "" {
			label = n.ID
		}
		groupAttr := ""
		if n.Group != "" {
			groupAttr = fmt.Sprintf(", group=%q", n.Group)
		}
		out += fmt.Sprintf("  %q [label=%q%s];\n", n.ID, label, groupAttr)
	}

	for _, e := range gd.Edges {
		label := ""
		if e.Label != "" {
			label = fmt.Sprintf(" [label=%q]", e.Label)
		}
		out += fmt.Sprintf("  %q -> %q%s;\n", e.Source, e.Target, label)
	}

	out += "}\n"
	return out
}

// FromExecutionGraph converts a ToolExecutionGraph to a GraphData for visualization.
func FromExecutionGraph(eg *ToolExecutionGraph) *GraphData {
	if eg == nil {
		return &GraphData{Nodes: []GraphDataNode{}, Edges: []GraphDataEdge{}}
	}

	gd := &GraphData{
		Nodes: make([]GraphDataNode, 0, len(eg.Nodes)),
		Edges: make([]GraphDataEdge, 0, len(eg.Edges)),
	}

	for _, n := range eg.Nodes {
		gd.Nodes = append(gd.Nodes, GraphDataNode{
			ID:    n.ID,
			Label: n.Label,
			Group: string(n.NodeType),
		})
	}

	for _, e := range eg.Edges {
		gd.Edges = append(gd.Edges, GraphDataEdge{
			Source: e.Source,
			Target: e.Target,
			Label:  e.EdgeType,
			Weight: e.Weight,
		})
	}

	return gd
}
