package gnn

import (
	"sort"
)

// Evaluator computes link prediction quality metrics (AUC, MRR).
type Evaluator struct{}

// NewEvaluator creates an Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// AUC computes the Area Under the ROC Curve for link prediction.
// It compares every positive edge score against every negative edge score.
// AUC = (#correctly ordered pairs + 0.5 * #ties) / #total pairs.
func (e *Evaluator) AUC(embeddings [][]float64, posEdges, negEdges [][2]int) float64 {
	var correct float64
	var total int

	for _, pos := range posEdges {
		posScore := dotProduct(embeddings[pos[0]], embeddings[pos[1]])
		for _, neg := range negEdges {
			negScore := dotProduct(embeddings[neg[0]], embeddings[neg[1]])
			if posScore > negScore {
				correct++
			} else if posScore == negScore {
				correct += 0.5
			}
			total++
		}
	}

	if total == 0 {
		return 0.5
	}
	return correct / float64(total)
}

// MRR computes the Mean Reciprocal Rank for link prediction.
// For each positive edge (u, v), it ranks all target candidates by their
// score with u, and computes 1/rank(v).
func (e *Evaluator) MRR(embeddings [][]float64, posEdges [][2]int) float64 {
	numNodes := len(embeddings)
	if numNodes == 0 || len(posEdges) == 0 {
		return 0
	}

	var totalRR float64

	for _, pos := range posEdges {
		u := pos[0]

		// Score all possible targets from u.
		type scored struct {
			idx   int
			score float64
		}
		scores := make([]scored, numNodes)
		for v := 0; v < numNodes; v++ {
			scores[v] = scored{idx: v, score: dotProduct(embeddings[u], embeddings[v])}
		}

		// Sort descending by score.
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].score > scores[j].score
		})

		// Find rank of the true positive target (1-based).
		for rank, s := range scores {
			if s.idx == pos[1] {
				totalRR += 1.0 / float64(rank+1)
				break
			}
		}
	}

	return totalRR / float64(len(posEdges))
}
