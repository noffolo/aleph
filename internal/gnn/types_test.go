package gnn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewGraph
// ============================================================================

func TestNewGraph_HappyPath(t *testing.T) {
	g := NewGraph()
	assert.NotNil(t, g)
	assert.NotNil(t, g.Nodes)
	assert.Len(t, g.Nodes, 0)
	assert.Len(t, g.Edges, 0)
}

func TestNewGraph_EdgeCase_MultipleGraphs(t *testing.T) {
	g1 := NewGraph()
	g2 := NewGraph()
	assert.NotSame(t, g1, g2, "each NewGraph should return a distinct graph")

	g1.AddNode(&WorkflowNode{ID: "X", Type: "test"})
	assert.Len(t, g1.Nodes, 1)
	assert.Len(t, g2.Nodes, 0, "g2 should not be affected by g1 changes")
}

func TestNewGraph_ErrorPath_IsValid(t *testing.T) {
	// Contract: NewGraph always returns usable graph with initialized maps
	g := NewGraph()
	assert.Equal(t, 0, g.NumNodes())
	assert.Equal(t, 0, g.NumEdges())
	assert.Empty(t, g.NodeIDs())
	idx := g.BuildNodeIndex()
	assert.Empty(t, idx)
}

// ============================================================================
// AddNode
// ============================================================================

func TestAddNode_HappyPath(t *testing.T) {
	g := NewGraph()
	n := &WorkflowNode{ID: "node1", Type: "file", InitialEmbedding: []float64{1.0, 2.0}}
	g.AddNode(n)

	assert.Len(t, g.Nodes, 1)
	stored, ok := g.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, n, stored)
	assert.Equal(t, "file", stored.Type)
	assert.Equal(t, []float64{1.0, 2.0}, stored.InitialEmbedding)
}

func TestAddNode_EdgeCase_OverwriteExisting(t *testing.T) {
	g := NewGraph()
	n1 := &WorkflowNode{ID: "dup", Type: "file"}
	n2 := &WorkflowNode{ID: "dup", Type: "function", InitialEmbedding: []float64{9.0}}

	g.AddNode(n1)
	g.AddNode(n2)

	assert.Len(t, g.Nodes, 1, "overwritten node should not increase count")
	stored := g.Nodes["dup"]
	assert.Equal(t, "function", stored.Type, "type should be from second node")
	assert.Equal(t, []float64{9.0}, stored.InitialEmbedding)
}

func TestAddNode_ErrorPath_NilNode(t *testing.T) {
	g := NewGraph()
	assert.Panics(t, func() {
		g.AddNode(nil)
	}, "adding nil node should panic")
}

// ============================================================================
// AddEdge
// ============================================================================

func TestAddEdge_HappyPath(t *testing.T) {
	g := NewGraph()
	g.AddEdge(Edge{Source: "A", Target: "B", Weight: 2.5})

	assert.Len(t, g.Edges, 1)
	assert.Equal(t, NodeID("A"), g.Edges[0].Source)
	assert.Equal(t, NodeID("B"), g.Edges[0].Target)
	assert.InDelta(t, 2.5, g.Edges[0].Weight, 1e-10)
}

func TestAddEdge_EdgeCase_ZeroWeightDefaultsToOne(t *testing.T) {
	g := NewGraph()
	g.AddEdge(Edge{Source: "X", Target: "Y", Weight: 0.0})

	assert.Len(t, g.Edges, 1)
	assert.InDelta(t, 1.0, g.Edges[0].Weight, 1e-10,
		"zero weight should default to 1.0")
}

func TestAddEdge_ErrorPath_EmptySourceAndTarget(t *testing.T) {
	g := NewGraph()
	g.AddEdge(Edge{Source: "", Target: "", Weight: 1.0})

	// Empty strings are valid NodeIDs — edge is appended as-is
	assert.Len(t, g.Edges, 1)
	assert.Equal(t, NodeID(""), g.Edges[0].Source)
	assert.Equal(t, NodeID(""), g.Edges[0].Target)
}

// ============================================================================
// NumNodes / NumEdges
// ============================================================================

func TestNumNodes_HappyPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "a", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "b", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "c", Type: "test"})
	assert.Equal(t, 3, g.NumNodes())
}

func TestNumNodes_EdgeCase_EmptyGraph(t *testing.T) {
	g := NewGraph()
	assert.Equal(t, 0, g.NumNodes())
}

func TestNumNodes_ErrorPath_OverwriteDoesNotIncrease(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "x", Type: "a"})
	g.AddNode(&WorkflowNode{ID: "x", Type: "b"})
	assert.Equal(t, 1, g.NumNodes(),
		"overwriting a node should not increase count")
}

func TestNumEdges_HappyPath(t *testing.T) {
	g := NewGraph()
	g.AddEdge(Edge{Source: "A", Target: "B"})
	g.AddEdge(Edge{Source: "B", Target: "C"})
	g.AddEdge(Edge{Source: "C", Target: "D"})
	assert.Equal(t, 3, g.NumEdges())
}

func TestNumEdges_EdgeCase_EmptyGraph(t *testing.T) {
	g := NewGraph()
	assert.Equal(t, 0, g.NumEdges())
}

func TestNumEdges_ErrorPath_DuplicatesNotFiltered(t *testing.T) {
	g := NewGraph()
	g.AddEdge(Edge{Source: "A", Target: "B"})
	g.AddEdge(Edge{Source: "A", Target: "B"})
	assert.Equal(t, 2, g.NumEdges(),
		"duplicate edges are not filtered — raw edge count is returned")
}

// ============================================================================
// NodeIDs
// ============================================================================

func TestNodeIDs_HappyPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "alpha", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "beta", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "gamma", Type: "test"})

	ids := g.NodeIDs()
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, NodeID("alpha"))
	assert.Contains(t, ids, NodeID("beta"))
	assert.Contains(t, ids, NodeID("gamma"))
}

func TestNodeIDs_EdgeCase_EmptyGraph(t *testing.T) {
	g := NewGraph()
	ids := g.NodeIDs()
	assert.Empty(t, ids)
	assert.NotNil(t, ids, "NodeIDs should return nil or empty slice, not nil")
}

func TestNodeIDs_ErrorPath_AfterOverwrite(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "unique", Type: "first"})
	g.AddNode(&WorkflowNode{ID: "unique", Type: "second"})

	ids := g.NodeIDs()
	assert.Len(t, ids, 1, "overwrite should keep exactly one ID")
	assert.Equal(t, NodeID("unique"), ids[0])
}

// ============================================================================
// BuildNodeIndex
// ============================================================================

func TestBuildNodeIndex_HappyPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "C", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "A", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "B", Type: "test"})

	idx := g.BuildNodeIndex()
	assert.Len(t, idx, 3)

	// Deterministic: sorted alphabetically
	assert.Equal(t, 0, idx["A"])
	assert.Equal(t, 1, idx["B"])
	assert.Equal(t, 2, idx["C"])
}

func TestBuildNodeIndex_EdgeCase_EmptyGraph(t *testing.T) {
	g := NewGraph()
	idx := g.BuildNodeIndex()
	assert.Empty(t, idx)
	assert.NotNil(t, idx)
}

func TestBuildNodeIndex_ErrorPath_DeterministicAcrossCalls(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "Z", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "M", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "A", Type: "test"})

	idx1 := g.BuildNodeIndex()
	idx2 := g.BuildNodeIndex()

	for id, v1 := range idx1 {
		v2, ok := idx2[id]
		assert.True(t, ok, "node %s missing in second index", id)
		assert.Equal(t, v1, v2, "index mismatch for node %s: %d vs %d", id, v1, v2)
	}
}
