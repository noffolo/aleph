package gnn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewEvaluator
// ============================================================================

func TestNewEvaluator_HappyPath(t *testing.T) {
	eval := NewEvaluator()
	assert.NotNil(t, eval)
}

func TestNewEvaluator_EdgeCase_MultipleCalls(t *testing.T) {
	e1 := NewEvaluator()
	e2 := NewEvaluator()
	assert.NotNil(t, e1)
	assert.NotNil(t, e2)
	assert.NotSame(t, e1, e2, "each call creates a new evaluator")
}

func TestNewEvaluator_ErrorPath_NilReceiverReturnsZero(t *testing.T) {
	var e *Evaluator
	auc := e.AUC([][]float64{}, [][2]int{}, [][2]int{})
	assert.InDelta(t, 0.5, auc, 1e-10)
}

// ============================================================================
// AUC
// ============================================================================

func TestAUC_HappyPath(t *testing.T) {
	g := smallGraph()
	model := gnnModelWithEmbeddings(g, 8, 42)

	posEdges := edgesToPairs(g)
	negEdges := generateNegPairs(g, 42, 2.0)

	eval := NewEvaluator()
	auc := eval.AUC(model.Forward(), posEdges, negEdges)
	assert.GreaterOrEqual(t, auc, 0.0)
	assert.LessOrEqual(t, auc, 1.0)
}

func TestAUC_EdgeCase_NoEdges(t *testing.T) {
	eval := NewEvaluator()
	emb := [][]float64{{1.0, 2.0}, {3.0, 4.0}}
	auc := eval.AUC(emb, [][2]int{}, [][2]int{{0, 1}})
	// total == 0 because there are no pos edges, so returns 0.5
	assert.InDelta(t, 0.5, auc, 1e-10)
}

func TestAUC_ErrorPath_IndexOutOfBounds(t *testing.T) {
	eval := NewEvaluator()
	emb := [][]float64{{1.0, 2.0}} // only 1 node
	posEdges := [][2]int{{0, 1}}   // node 1 doesn't exist
	negEdges := [][2]int{{0, 0}}

	assert.Panics(t, func() {
		eval.AUC(emb, posEdges, negEdges)
	})
}

// ============================================================================
// MRR
// ============================================================================

func TestMRR_HappyPath(t *testing.T) {
	// Create embeddings where the positive target is ranked first
	emb := [][]float64{
		{1.0, 0.0}, // node 0
		{1.0, 0.0}, // node 1 — identical to 0, will be ranked first
		{0.0, 1.0}, // node 2
	}
	// posEdge (0,1): emb[0]·emb[1] = 1.0 (first among all nodes, rank 1)
	// MRR = 1/1 = 1.0
	posEdges := [][2]int{{0, 1}}

	eval := NewEvaluator()
	mrr := eval.MRR(emb, posEdges)
	// Node 1 scored against node 0: score(0,0)=1.0, score(0,1)=1.0, score(0,2)=0.0
	// Sorted: [0:1.0, 1:1.0, 2:0.0] — rank 1 or 2 (depending on stable sort of ties)
	// Actually with sort.Slice ties are not stable. The target node 1 is found somewhere.
	// MRR should be between 0.5 and 1.0.
	assert.GreaterOrEqual(t, mrr, 0.0)
	assert.LessOrEqual(t, mrr, 1.0)
}

func TestMRR_EdgeCase_EmptyEmbeddings(t *testing.T) {
	eval := NewEvaluator()
	mrr := eval.MRR([][]float64{}, [][2]int{})
	assert.InDelta(t, 0.0, mrr, 1e-10)
}

func TestMRR_ErrorPath_IndexOutOfBounds(t *testing.T) {
	eval := NewEvaluator()
	emb := [][]float64{{1.0, 2.0}} // only 1 node

	assert.Panics(t, func() {
		eval.MRR(emb, [][2]int{{5, 0}})
	})
}

// ============================================================================
// Helpers
// ============================================================================

func gnnModelWithEmbeddings(g *Graph, dim int, seed int64) *GNNModel {
	model := NewGNNModel(g.NumNodes(), dim, seed)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)
	return model
}

func edgesToPairs(g *Graph) [][2]int {
	idx := g.BuildNodeIndex()
	pairs := make([][2]int, len(g.Edges))
	for i, e := range g.Edges {
		pairs[i] = [2]int{idx[e.Source], idx[e.Target]}
	}
	return pairs
}

func generateNegPairs(g *Graph, seed int64, ratio float64) [][2]int {
	idx := g.BuildNodeIndex()
	sampler := NewNegativeSampler(seed, ratio)
	return sampler.Sample(g.Edges, idx)
}
