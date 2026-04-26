package codeflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGraphData(t *testing.T) {
	gd := &GraphData{}
	assert.Empty(t, gd.Nodes)
	assert.Empty(t, gd.Edges)
}

func TestFromExecutionGraph_NilInput(t *testing.T) {
	gd := FromExecutionGraph(nil)
	assert.NotNil(t, gd)
	assert.Empty(t, gd.Nodes)
	assert.Empty(t, gd.Edges)
}

func TestFromExecutionGraph_WithNodesAndEdges(t *testing.T) {
	eg := &ToolExecutionGraph{
		ToolID: "tool_a",
		Nodes: []GraphNode{
			{ID: "tool_a", Label: "Tool A", NodeType: NodeTypeTool},
			{ID: "dep_1", Label: "Dep 1", NodeType: NodeTypeDependency},
		},
		Edges: []GraphEdge{
			{Source: "tool_a", Target: "dep_1", EdgeType: "dataflow", Weight: 3},
		},
	}

	gd := FromExecutionGraph(eg)
	assert.Len(t, gd.Nodes, 2)
	assert.Len(t, gd.Edges, 1)

	assert.Equal(t, "tool_a", gd.Nodes[0].ID)
	assert.Equal(t, "Tool A", gd.Nodes[0].Label)
	assert.Equal(t, "tool", gd.Nodes[0].Group)

	assert.Equal(t, "dep_1", gd.Nodes[1].ID)

	assert.Equal(t, "tool_a", gd.Edges[0].Source)
	assert.Equal(t, "dep_1", gd.Edges[0].Target)
	assert.Equal(t, "dataflow", gd.Edges[0].Label)
	assert.Equal(t, 3, gd.Edges[0].Weight)
}

func TestGraphData_ToD3JSON(t *testing.T) {
	gd := &GraphData{
		Nodes: []GraphDataNode{
			{ID: "n1", Label: "Node 1", Group: "tool"},
			{ID: "n2", Label: "Node 2", Group: "data"},
		},
		Edges: []GraphDataEdge{
			{Source: "n1", Target: "n2", Label: "calls", Weight: 1},
		},
	}

	json := gd.ToD3JSON()
	assert.Contains(t, json, `"nodes"`)
	assert.Contains(t, json, `"links"`)
	assert.Contains(t, json, `"n1"`)
	assert.Contains(t, json, `"n2"`)
}

func TestGraphData_ToD3JSON_Empty(t *testing.T) {
	gd := &GraphData{}
	json := gd.ToD3JSON()
	assert.Contains(t, json, `"nodes"`)
	assert.Contains(t, json, `"links"`)
}

func TestGraphData_ToGraphViz(t *testing.T) {
	gd := &GraphData{
		Nodes: []GraphDataNode{
			{ID: "tool_a", Label: "Tool A", Group: "tool"},
		},
		Edges: []GraphDataEdge{
			{Source: "tool_a", Target: "dep_1", Label: "dep", Weight: 1},
		},
	}

	dot := gd.ToGraphViz()
	assert.Contains(t, dot, "digraph ToolGraph")
	assert.Contains(t, dot, "rankdir=LR")
	assert.Contains(t, dot, `"tool_a"`)
	assert.Contains(t, dot, `"dep_1"`)
	assert.Contains(t, dot, `[label="dep"]`)
}

func TestGraphData_ToGraphViz_NoLabel(t *testing.T) {
	gd := &GraphData{
		Nodes: []GraphDataNode{
			{ID: "tool_a"}, // no Label or Group
		},
		Edges: []GraphDataEdge{
			{Source: "tool_a", Target: "dep_1", Weight: 1}, // no Label
		},
	}

	dot := gd.ToGraphViz()
	assert.Contains(t, dot, `"tool_a"`)
	// ID used as fallback label
	assert.Contains(t, dot, `[label="tool_a"`)
}

func TestGraphData_ToGraphViz_WithGroup(t *testing.T) {
	gd := &GraphData{
		Nodes: []GraphDataNode{
			{ID: "n1", Label: "Node 1", Group: "tools"},
		},
	}

	dot := gd.ToGraphViz()
	assert.Contains(t, dot, `group="tools"`)
}

func TestGraphDataEdgeWeight(t *testing.T) {
	e := GraphDataEdge{Source: "a", Target: "b", Weight: 42}
	assert.Equal(t, 42, e.Weight)
}

func TestGraphDataNodeWeight(t *testing.T) {
	n := GraphDataNode{ID: "x", Label: "X", Weight: 7}
	assert.Equal(t, 7, n.Weight)
}
