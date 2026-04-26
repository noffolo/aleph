package gnn

import (
	"math/rand"
)

// NegSamplingStrategy controls how negative edges are generated.
type NegSamplingStrategy int

const (
	// NegSamplingUniform picks random (u,v) pairs that are not in the edge set.
	NegSamplingUniform NegSamplingStrategy = iota
)

// NegativeSampler generates negative edge samples from a positive-only graph.
// For each positive edge, it produces a configurable number of negative
// (non-edge) pairs by replacing the target node with a uniformly random node.
type NegativeSampler struct {
	rng      *rand.Rand
	strategy NegSamplingStrategy
	ratio    float64 // number of negatives per positive (1.0 = 1:1)
}

// NewNegativeSampler creates a NegativeSampler.
//
//   - seed:  random seed for reproducibility
//   - ratio: number of negative samples per positive edge (1.0 = 1:1, 2.0 = 2:1)
func NewNegativeSampler(seed int64, ratio float64) *NegativeSampler {
	return &NegativeSampler{
		rng:      rand.New(rand.NewSource(seed)),
		strategy: NegSamplingUniform,
		ratio:    ratio,
	}
}

// Sample generates negative edges by corrupting the target node of each
// positive edge. It returns a slice of [2]int pairs corresponding to
// indices in the nodeIndex map.
//
// It guarantees:
//   - No returned pair is a true positive edge
//   - No self-loop (source == target)
//   - The number of negatives is approximately ceil(len(edges) * ratio)
func (s *NegativeSampler) Sample(edges []Edge, nodeIndex map[NodeID]int) [][2]int {
	// Build a set of existing edges for O(1) lookup.
	existing := make(map[[2]int]bool)
	for _, e := range edges {
		u, okU := nodeIndex[e.Source]
		v, okV := nodeIndex[e.Target]
		if okU && okV {
			existing[[2]int{u, v}] = true
			existing[[2]int{v, u}] = true // undirected
		}
	}

	numNodes := len(nodeIndex)
	numNeg := int(float64(len(edges)) * s.ratio)
	if numNeg < 1 {
		numNeg = len(edges)
	}

	// Collect all node indices for fast random access.
	nodeIDs := make([]int, 0, numNodes)
	for _, idx := range nodeIndex {
		nodeIDs = append(nodeIDs, idx)
	}

	negatives := make([][2]int, 0, numNeg)
	maxAttempts := numNeg * 10

	for attempts := 0; len(negatives) < numNeg && attempts < maxAttempts; attempts++ {
		pe := edges[s.rng.Intn(len(edges))]
		u, okU := nodeIndex[pe.Source]
		_, okV := nodeIndex[pe.Target]
		if !okU || !okV {
			continue
		}

		negV := nodeIDs[s.rng.Intn(len(nodeIDs))]
		if negV == u {
			continue
		}

		pair := [2]int{u, negV}
		if existing[pair] {
			continue
		}

		negatives = append(negatives, pair)

		existing[pair] = true
	}

	return negatives
}
