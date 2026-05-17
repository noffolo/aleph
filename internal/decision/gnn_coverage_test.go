package decision

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGNNLinkPredictor_Creation(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(10, 64, 0.01)
	assert.NotNil(t, p)
	assert.False(t, p.IsTrained())
	assert.NotNil(t, p.model)
	assert.NotNil(t, p.trainer)
}

func TestGNNLinkPredictor_TrainFromGraph_EmptyGraph(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(10, 64, 0.01)

	graph := gnn.NewGraph()
	err := p.TrainFromGraph(context.Background(), graph, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot train on empty graph")
	assert.False(t, p.IsTrained())
}

func TestGNNLinkPredictor_TrainFromGraph(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(4, 16, 0.01)

	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("A")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("B")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("C")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("D")})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("A"), Target: gnn.NodeID("B"), Weight: 1.0})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("B"), Target: gnn.NodeID("C"), Weight: 0.8})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("C"), Target: gnn.NodeID("D"), Weight: 0.6})

	err := p.TrainFromGraph(context.Background(), graph, 3)
	require.NoError(t, err)
	assert.True(t, p.IsTrained())
}

func TestGNNLinkPredictor_PredictLinks_NotTrained(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(4, 16, 0.01)
	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("A")})

	_, err := p.PredictLinks(context.Background(), graph, "A")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not trained")
}

func TestGNNLinkPredictor_PredictLinks_GraphSizeMismatch(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(4, 16, 0.01)

	// Train on a graph with 4 nodes
	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("A")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("B")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("C")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("D")})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("A"), Target: gnn.NodeID("B"), Weight: 1.0})
	err := p.TrainFromGraph(context.Background(), graph, 3)
	require.NoError(t, err)

	// Try to predict on a graph with different number of nodes
	smallGraph := gnn.NewGraph()
	smallGraph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("X")})
	smallGraph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("Y")})

	_, err = p.PredictLinks(context.Background(), smallGraph, "X")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "graph size mismatch")
}

func TestGNNLinkPredictor_PredictLinks_EntityNotFound(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(3, 8, 0.01)

	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("A")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("B")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("C")})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("A"), Target: gnn.NodeID("B"), Weight: 1.0})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("B"), Target: gnn.NodeID("C"), Weight: 0.5})

	err := p.TrainFromGraph(context.Background(), graph, 3)
	require.NoError(t, err)

	_, err = p.PredictLinks(context.Background(), graph, "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in node index")
}

func TestGNNLinkPredictor_PredictLinks_Success(t *testing.T) {
	t.Parallel()
	p := NewGNNLinkPredictor(3, 8, 0.01)

	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("A")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("B")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("C")})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("A"), Target: gnn.NodeID("B"), Weight: 1.0})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("B"), Target: gnn.NodeID("C"), Weight: 0.5})

	err := p.TrainFromGraph(context.Background(), graph, 3)
	require.NoError(t, err)

	scores, err := p.PredictLinks(context.Background(), graph, "A")
	require.NoError(t, err)
	assert.Len(t, scores, 3) // one score per node

	// All scores should be finite numbers
	for i := range scores {
		assert.Greater(t, scores[i], -99999.0, "score[%d] should be finite", i)
		assert.Less(t, scores[i], 99999.0, "score[%d] should be finite", i)
	}

	// After training, IsTrained should be true
	assert.True(t, p.IsTrained())
}

func TestConfidenceFromPredictions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		scores []float64
	}{
		{"empty scores", []float64{}},
		{"single zero", []float64{0.0}},
		{"single positive", []float64{0.8}},
		{"single negative", []float64{-1.0}},
		{"multiple positive", []float64{0.3, 0.7, 0.5}},
		{"mixed scores", []float64{-0.5, 0.2, 0.9, -1.0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf := ConfidenceFromPredictions(tc.scores)
			// Confidence should always be in [0, 1]
			assert.GreaterOrEqual(t, conf, 0.0)
			assert.LessOrEqual(t, conf, 1.0)
		})
	}

	// Empty returns 0.5
	assert.Equal(t, 0.5, ConfidenceFromPredictions([]float64{}))

	// Higher max score = higher confidence (monotonically increasing after sigmoid)
	assert.Greater(t, ConfidenceFromPredictions([]float64{2.0}), ConfidenceFromPredictions([]float64{0.0}))
	assert.Greater(t, ConfidenceFromPredictions([]float64{1.0}), ConfidenceFromPredictions([]float64{-1.0}))
}

// ─── IsTrained via interface check ──────────────────────────────────────────

func TestGNNLinkPredictor_SatisfiesInterface(t *testing.T) {
	t.Parallel()
	var _ LinkPredictor = (*GNNLinkPredictor)(nil)
	p := NewGNNLinkPredictor(3, 8, 0.01)
	assert.False(t, p.IsTrained())

	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("X")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("Y")})
	graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("Z")})
	graph.AddEdge(gnn.Edge{Source: gnn.NodeID("X"), Target: gnn.NodeID("Y"), Weight: 1.0})

	err := p.TrainFromGraph(context.Background(), graph, 2)
	require.NoError(t, err)
	assert.True(t, p.IsTrained())
}
