package gnn

import (
	"math"
	"math/rand" // #nosec G404 — safe: Xavier-like weight initialization for ML model, not security-sensitive
)

// DefaultEmbeddingDim is the default embedding dimension (64).
const DefaultEmbeddingDim = 64

// GNNModel implements a 2-layer LightGCN-style graph neural network for
// link prediction. The model has no learnable weight matrices — convolution
// is simply normalized adjacency multiplication. The only learnable
// parameters are the initial node embeddings.
//
// Forward pass:
//
//	H_0  = base embeddings
//	H_1  = A_norm @ H_0
//	H_2  = A_norm @ H_1
//	H    = (H_0 + H_1 + H_2) / 3
//
// Link prediction score: s(u,v) = dot(H[u], H[v])
type GNNModel struct {
	NumNodes   int
	Dim        int
	Embeddings [][]float64 // learnable base embeddings [numNodes][dim]
	adjNorm    [][]float64 // row-normalized adjacency [numNodes][numNodes]
}

// NewGNNModel creates a new GNN model with randomly initialized embeddings.
func NewGNNModel(numNodes, dim int, seed int64) *GNNModel {
	rng := rand.New(rand.NewSource(seed))
	emb := make([][]float64, numNodes)
	scale := math.Sqrt(1.0 / float64(dim)) // Xavier-like init
	for i := range emb {
		emb[i] = make([]float64, dim)
		for j := range emb[i] {
			emb[i][j] = rng.Float64()*2*scale - scale
		}
	}
	return &GNNModel{
		NumNodes:   numNodes,
		Dim:        dim,
		Embeddings: emb,
	}
}

// BuildAdjacency computes the symmetrically-normalized adjacency matrix
// from the graph: A_norm = D^{-1/2} @ A @ D^{-1/2}.
// Isolated nodes remain as zero rows.
func (m *GNNModel) BuildAdjacency(nodeIndex map[NodeID]int, edges []Edge) {
	n := m.NumNodes
	adj := make([][]float64, n)
	for i := range adj {
		adj[i] = make([]float64, n)
	}

	for _, e := range edges {
		u, okU := nodeIndex[e.Source]
		v, okV := nodeIndex[e.Target]
		if okU && okV {
			w := e.Weight
			if w == 0 {
				w = 1.0
			}
			adj[u][v] += w
			adj[v][u] += w
		}
	}

	degrees := make([]float64, n)
	for i := range adj {
		for j := range adj[i] {
			degrees[i] += adj[i][j]
		}
	}

	m.adjNorm = make([][]float64, n)
	for i := range m.adjNorm {
		m.adjNorm[i] = make([]float64, n)
		for j := range m.adjNorm[i] {
			if degrees[i] > 0 && degrees[j] > 0 {
				invSqrtDeg := 1.0 / math.Sqrt(degrees[i]*degrees[j])
				m.adjNorm[i][j] = adj[i][j] * invSqrtDeg
			}
		}
	}
}

// Forward computes LightGCN embeddings from the current base embeddings
// and cached normalized adjacency.
func (m *GNNModel) Forward() [][]float64 {
	n := m.NumNodes
	d := m.Dim

	H1 := matMul(m.adjNorm, m.Embeddings)
	H2 := matMul(m.adjNorm, H1)

	result := make([][]float64, n)
	for i := range result {
		result[i] = make([]float64, d)
		for j := 0; j < d; j++ {
			result[i][j] = (m.Embeddings[i][j] + H1[i][j] + H2[i][j]) / 3.0
		}
	}
	return result
}

// Backward computes the gradient w.r.t. base embeddings given the gradient
// w.r.t. final (combined) embeddings.
//
//	dL/dH_0 = (I + A_norm + A_norm^2)^T / 3  @  dL/dH_final
//
// Since A_norm is symmetric, (I + A_norm + A_norm^2) is also symmetric.
func (m *GNNModel) Backward(gradFinal [][]float64) [][]float64 {
	n := m.NumNodes
	d := m.Dim

	aG := matMul(m.adjNorm, gradFinal)
	aaG := matMul(m.adjNorm, aG)

	gradH0 := make([][]float64, n)
	for i := range gradH0 {
		gradH0[i] = make([]float64, d)
		inv3 := 1.0 / 3.0
		for j := 0; j < d; j++ {
			gradH0[i][j] = (gradFinal[i][j] + aG[i][j] + aaG[i][j]) * inv3
		}
	}
	return gradH0
}

// PredictScore computes the link prediction score between two nodes.
func PredictScore(embeddings [][]float64, u, v int) float64 {
	return dotProduct(embeddings[u], embeddings[v])
}

// ─── Linear algebra utilities (dense, single-threaded) ────────────────────

// dotProduct computes the dot product of two vectors.
func dotProduct(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// matMul multiplies a [m×k] matrix A by a [k×n] matrix B.
// Both are row-major [][]float64.
func matMul(A, B [][]float64) [][]float64 {
	m := len(A)
	k := len(A[0])
	n := len(B[0])

	C := make([][]float64, m)
	for i := range C {
		C[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			var sum float64
			for t := 0; t < k; t++ {
				sum += A[i][t] * B[t][j]
			}
			C[i][j] = sum
		}
	}
	return C
}

// sigmoid computes the logistic function 1/(1+exp(-x)).
// Clamps x to [-10, 10] to avoid floating-point overflow.
func sigmoid(x float64) float64 {
	if x > 10 {
		return 1.0
	}
	if x < -10 {
		return 0.0
	}
	return 1.0 / (1.0 + math.Exp(-x))
}
