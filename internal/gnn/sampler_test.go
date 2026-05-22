package gnn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewNegativeSampler
// ============================================================================

func TestNewNegativeSampler_HappyPath(t *testing.T) {
	s := NewNegativeSampler(42, 1.0)
	assert.NotNil(t, s)
	assert.NotNil(t, s.rng)
	assert.Equal(t, NegSamplingUniform, s.strategy)
	assert.InDelta(t, 1.0, s.ratio, 1e-10)
}

func TestNewNegativeSampler_EdgeCase_ZeroRatio(t *testing.T) {
	s := NewNegativeSampler(42, 0.0)
	assert.NotNil(t, s)
	assert.InDelta(t, 0.0, s.ratio, 1e-10)
}

func TestNewNegativeSampler_ErrorPath_NegativeRatio(t *testing.T) {
	s := NewNegativeSampler(42, -1.0)
	assert.NotNil(t, s)
	// Negative ratio is accepted (no validation); Sample will handle it
	// by using len(edges) as numNeg when numNeg < 1
	assert.InDelta(t, -1.0, s.ratio, 1e-10)
}

// ============================================================================
// Sample
// ============================================================================

func TestSample_HappyPath(t *testing.T) {
	g := smallGraph()
	idx := g.BuildNodeIndex()
	s := NewNegativeSampler(42, 1.0)

	negatives := s.Sample(g.Edges, idx)

	assert.NotEmpty(t, negatives, "should generate at least one negative sample")

	// Verify no negative is a real edge
	realEdges := make(map[[2]int]bool)
	for _, e := range g.Edges {
		u, v := idx[e.Source], idx[e.Target]
		realEdges[[2]int{u, v}] = true
		realEdges[[2]int{v, u}] = true
	}
	for _, n := range negatives {
		assert.False(t, realEdges[n], "negative pair %v is a real edge", n)
		assert.NotEqual(t, n[0], n[1], "negative pair %v is a self-loop", n)
	}
}

func TestSample_EdgeCase_EmptyEdges(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "A", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "B", Type: "test"})
	idx := g.BuildNodeIndex()
	s := NewNegativeSampler(42, 1.0)

	// numNeg = 0, so it becomes len(edges)=0 — loop doesn't execute
	negatives := s.Sample(g.Edges, idx)
	assert.Empty(t, negatives, "empty edges should yield empty negatives")
}

func TestSample_ErrorPath_NilNodeIndexReturnsEmpty(t *testing.T) {
	g := smallGraph()
	s := NewNegativeSampler(42, 1.0)

	negatives := s.Sample(g.Edges, nil)
	assert.Empty(t, negatives)
}
