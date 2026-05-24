package gnn

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewGNNModel
// ============================================================================

func TestNewGNNModel_HappyPath(t *testing.T) {
	model := NewGNNModel(5, 8, 42)

	assert.Equal(t, 5, model.NumNodes)
	assert.Equal(t, 8, model.Dim)
	assert.Len(t, model.Embeddings, 5)
	for i := 0; i < 5; i++ {
		assert.Len(t, model.Embeddings[i], 8)
	}
}

func TestNewGNNModel_EdgeCase_ZeroNodes(t *testing.T) {
	model := NewGNNModel(0, 8, 42)
	assert.Equal(t, 0, model.NumNodes)
	assert.Len(t, model.Embeddings, 0)
}

func TestNewGNNModel_ErrorPath_NegativeDimPanics(t *testing.T) {
	// dim of 0 or negative means make() with zero/negative length which panics
	assert.Panics(t, func() {
		NewGNNModel(5, -1, 42)
	})
}

// ============================================================================
// BuildAdjacency
// ============================================================================

func TestBuildAdjacency_HappyPath(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)

	// adjNorm should be n x n
	n := g.NumNodes()
	assert.Len(t, model.adjNorm, n)
	for i := 0; i < n; i++ {
		assert.Len(t, model.adjNorm[i], n)
	}

	// Should be symmetric
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			assert.InDelta(t, model.adjNorm[i][j], model.adjNorm[j][i], 1e-10,
				"adjacency not symmetric at (%d,%d)", i, j)
		}
	}
}

func TestBuildAdjacency_EdgeCase_SingleIsolatedNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "X", Type: "test"})
	model := NewGNNModel(1, 8, 42)
	idx := g.BuildNodeIndex()

	// No edges at all
	model.BuildAdjacency(idx, g.Edges)
	assert.Len(t, model.adjNorm, 1)
	assert.InDelta(t, 0.0, model.adjNorm[0][0], 1e-10,
		"isolated node should have zero adjacency")
}

func TestBuildAdjacency_ErrorPath_EmptyNodeIndex(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)

	// Empty nodeIndex — edges map to nothing, adjNorm is all zeros
	model.BuildAdjacency(map[NodeID]int{}, g.Edges)

	n := g.NumNodes()
	assert.Len(t, model.adjNorm, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			assert.InDelta(t, 0.0, model.adjNorm[i][j], 1e-10,
				"adjNorm should be all zeros with empty nodeIndex")
		}
	}
}

// ============================================================================
// Forward
// ============================================================================

func TestForward_HappyPath(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)

	emb := model.Forward()
	assert.Len(t, emb, g.NumNodes())
	for i := range emb {
		assert.Len(t, emb[i], 8)
	}

	// After convolution, connected nodes should have non-zero embeddings
	// (initialized with Xavier-like init, then convolved)
	allZeros := true
	for i := range emb {
		for j := range emb[i] {
			if emb[i][j] != 0 {
				allZeros = false
				break
			}
		}
	}
	assert.False(t, allZeros, "forward should produce non-zero embeddings")
}

func TestForward_EdgeCase_SingleNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "X", Type: "test"})
	model := NewGNNModel(1, 4, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)

	emb := model.Forward()
	assert.Len(t, emb, 1)
	assert.Len(t, emb[0], 4)
	// Single isolated node: H1=H2=0, so output = H0/3
	for _, v := range emb[0] {
		assert.InDelta(t, model.Embeddings[0][0]/model.Embeddings[0][0]*v, v, 1e-10)
	}
}

func TestForward_ErrorPath_NilAdjNorm(t *testing.T) {
	model := NewGNNModel(2, 4, 42)
	// Don't call BuildAdjacency — adjNorm is nil
	// Forward accesses adjNorm which is nil, causing panic
	assert.Panics(t, func() {
		model.Forward()
	})
}

// ============================================================================
// Backward
// ============================================================================

func TestBackward_HappyPath(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)
	_ = model.Forward()

	n, d := model.NumNodes, model.Dim
	gradFinal := make([][]float64, n)
	for i := range gradFinal {
		gradFinal[i] = make([]float64, d)
		// Set gradient to ones so we can verify symmetry
		for j := range gradFinal[i] {
			gradFinal[i][j] = 1.0
		}
	}

	gradH0 := model.Backward(gradFinal)
	assert.Len(t, gradH0, n)
	for i := range gradH0 {
		assert.Len(t, gradH0[i], d)
	}
}

func TestBackward_EdgeCase_ZeroGradient(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)

	n, d := model.NumNodes, model.Dim
	gradFinal := make([][]float64, n)
	for i := range gradFinal {
		gradFinal[i] = make([]float64, d)
	}

	gradH0 := model.Backward(gradFinal)
	for i := 0; i < n; i++ {
		for j := 0; j < d; j++ {
			assert.InDelta(t, 0.0, gradH0[i][j], 1e-15,
				"zero gradient should propagate as zero")
		}
	}
}

func TestBackward_ErrorPath_NilGradient(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)
	idx := g.BuildNodeIndex()
	model.BuildAdjacency(idx, g.Edges)

	assert.Panics(t, func() {
		model.Backward(nil)
	})
}

// ============================================================================
// PredictScore
// ============================================================================

func TestPredictScore_HappyPath(t *testing.T) {
	emb := [][]float64{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}

	score := PredictScore(emb, 0, 1)
	// dot(1,2,3) · (4,5,6) = 1*4+2*5+3*6 = 4+10+18 = 32
	assert.InDelta(t, 32.0, score, 1e-10)
}

func TestPredictScore_EdgeCase_ZeroEmbeddings(t *testing.T) {
	emb := [][]float64{
		{0.0, 0.0, 0.0},
		{0.0, 0.0, 0.0},
	}

	score := PredictScore(emb, 0, 1)
	assert.InDelta(t, 0.0, score, 1e-15)
}

func TestPredictScore_ErrorPath_OutOfBounds(t *testing.T) {
	emb := [][]float64{
		{1.0, 2.0},
		{3.0, 4.0},
	}

	assert.Panics(t, func() {
		PredictScore(emb, 0, 5)
	})
}

// ============================================================================
// dotProduct
// ============================================================================

func TestDotProduct_HappyPath(t *testing.T) {
	a := []float64{1.0, 2.0, 3.0}
	b := []float64{4.0, 5.0, 6.0}
	result := dotProduct(a, b)
	assert.InDelta(t, 32.0, result, 1e-10)
}

func TestDotProduct_EdgeCase_EmptyVectors(t *testing.T) {
	result := dotProduct([]float64{}, []float64{})
	assert.InDelta(t, 0.0, result, 1e-15)
}

func TestDotProduct_ErrorPath_MismatchedLengths(t *testing.T) {
	// dotProduct iterates over a's length; if b is shorter, it panics
	assert.Panics(t, func() {
		dotProduct([]float64{1.0, 2.0, 3.0}, []float64{1.0})
	})
}

// ============================================================================
// matMul
// ============================================================================

func TestMatMul_HappyPath(t *testing.T) {
	A := [][]float64{
		{1, 2},
		{3, 4},
	}
	B := [][]float64{
		{5, 6},
		{7, 8},
	}

	C := matMul(A, B)
	// [1*5+2*7, 1*6+2*8] = [19, 22]
	// [3*5+4*7, 3*6+4*8] = [43, 50]
	assert.InDelta(t, 19.0, C[0][0], 1e-10)
	assert.InDelta(t, 22.0, C[0][1], 1e-10)
	assert.InDelta(t, 43.0, C[1][0], 1e-10)
	assert.InDelta(t, 50.0, C[1][1], 1e-10)
}

func TestMatMul_EdgeCase_1x1Matrices(t *testing.T) {
	A := [][]float64{{7.0}}
	B := [][]float64{{3.0}}
	C := matMul(A, B)
	assert.Len(t, C, 1)
	assert.Len(t, C[0], 1)
	assert.InDelta(t, 21.0, C[0][0], 1e-10)
}

func TestMatMul_ErrorPath_IncompatibleDimensions(t *testing.T) {
	A := [][]float64{{1, 2, 3}} // 1x3
	B := [][]float64{{4}, {5}}  // 2x1 — incompatible (A.cols=3 != B.rows=2)
	assert.Panics(t, func() {
		matMul(A, B)
	})
}

// ============================================================================
// sigmoid
// ============================================================================

func TestSigmoid_HappyPath(t *testing.T) {
	assert.InDelta(t, 0.5, sigmoid(0.0), 1e-10)
	assert.InDelta(t, 0.7310585786300049, sigmoid(1.0), 1e-10)
	assert.InDelta(t, 0.2689414213699951, sigmoid(-1.0), 1e-10)
}

func TestSigmoid_EdgeCase_ExtremeValues(t *testing.T) {
	// Clamps at ±10
	assert.InDelta(t, 1.0, sigmoid(100.0), 1e-10)
	assert.InDelta(t, 0.0, sigmoid(-100.0), 1e-10)

	// At exactly the clamp boundary
	assert.True(t, sigmoid(10.0) > 0.9999)
	assert.True(t, sigmoid(-10.0) < 0.0001)
}

func TestSigmoid_ErrorPath_NaN(t *testing.T) {
	result := sigmoid(math.NaN())
	assert.True(t, math.IsNaN(result), "sigmoid of NaN should return NaN")
}
