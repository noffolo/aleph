package gnn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewTrainer
// ============================================================================

func TestNewTrainer_HappyPath(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)

	trainer := NewTrainer(model, 0.01)

	assert.NotNil(t, trainer)
	assert.Equal(t, model, trainer.Model)
	assert.InDelta(t, 0.01, trainer.LR, 1e-10)
	assert.InDelta(t, 1e-6, trainer.MinLR, 1e-10)
	assert.InDelta(t, 0.99, trainer.Decay, 1e-10)
	assert.Equal(t, 32, trainer.BatchSize)
	assert.NotNil(t, trainer.rng)
}

func TestNewTrainer_EdgeCase_ZeroLR(t *testing.T) {
	g := smallGraph()
	model := NewGNNModel(g.NumNodes(), 8, 42)

	trainer := NewTrainer(model, 0.0)
	assert.InDelta(t, 0.0, trainer.LR, 1e-10)
}

func TestNewTrainer_ErrorPath_NilModel(t *testing.T) {
	trainer := NewTrainer(nil, 0.01)
	assert.NotNil(t, trainer)
	assert.Nil(t, trainer.Model, "nil model should be stored as-is")
	// Training with nil model would panic, verifying the assignment is correct
}

// ============================================================================
// Train
// ============================================================================

func TestTrain_HappyPath(t *testing.T) {
	g := midGraph()
	model := gnnModelWithEmbeddings(g, 16, 42)

	posEdges := edgesToPairs(g)
	negEdges := generateNegPairs(g, 42, 2.0)

	trainer := NewTrainer(model, 0.02)
	trainer.BatchSize = 8

	result := trainer.Train(posEdges, negEdges, 20)

	assert.Equal(t, 20, result.EpochsRun)
	assert.Len(t, result.LossHistory, 20)

	// Loss should be positive and finite
	assert.Greater(t, result.FinalLoss, 0.0)
	assert.False(t, result.FinalLoss > 1e10, "loss should not overflow")

	// Loss should generally decrease (allow 0.01 tolerance for small models)
	initialLoss := result.LossHistory[0]
	assert.Less(t, result.FinalLoss, initialLoss+0.01,
		"loss should decrease: %.4f → %.4f", initialLoss, result.FinalLoss)
}

func TestTrain_EdgeCase_ZeroEpochs(t *testing.T) {
	g := smallGraph()
	model := gnnModelWithEmbeddings(g, 8, 42)

	posEdges := edgesToPairs(g)
	negEdges := generateNegPairs(g, 42, 1.0)

	trainer := NewTrainer(model, 0.01)

	// Train with 0 epochs panics: lossHistory[epochs-1] = lossHistory[-1]
	assert.Panics(t, func() {
		trainer.Train(posEdges, negEdges, 0)
	})
}

func TestTrain_ErrorPath_EmptyEdges(t *testing.T) {
	g := smallGraph()
	model := gnnModelWithEmbeddings(g, 8, 42)

	trainer := NewTrainer(model, 0.01)

	// Empty posEdges and negEdges — maxLen=0, no batches, loss=0
	result := trainer.Train([][2]int{}, [][2]int{}, 5)

	assert.Equal(t, 5, result.EpochsRun)
	// With no batches, LossHistory entries remain 0
	for i := 0; i < 5; i++ {
		assert.InDelta(t, 0.0, result.LossHistory[i], 1e-10,
			"loss at epoch %d should be 0 with no data", i)
	}
	assert.InDelta(t, 0.0, result.FinalLoss, 1e-10)
}

// ============================================================================
// step (unexported — accessible via package gnn)
// ============================================================================

func TestStep_HappyPath(t *testing.T) {
	g := midGraph()
	model := gnnModelWithEmbeddings(g, 8, 42)

	posEdges := edgesToPairs(g)
	negEdges := generateNegPairs(g, 42, 3.0)

	minLen := len(posEdges)
	if len(negEdges) < minLen {
		minLen = len(negEdges)
	}
	posBatch := posEdges[:minLen]
	negBatch := negEdges[:minLen]

	trainer := NewTrainer(model, 0.01)

	loss := trainer.step(posBatch, negBatch)

	assert.Greater(t, loss, 0.0, "BPR loss should be positive")
	assert.False(t, loss > 100.0, "loss should be reasonable, got %.2f", loss)
}

func TestStep_EdgeCase_SinglePair(t *testing.T) {
	g := NewGraph()
	g.AddNode(&WorkflowNode{ID: "A", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "B", Type: "test"})
	g.AddNode(&WorkflowNode{ID: "C", Type: "test"})
	g.AddEdge(Edge{Source: "A", Target: "B"})
	idx := g.BuildNodeIndex()

	model := NewGNNModel(3, 4, 42)
	model.BuildAdjacency(idx, g.Edges)

	trainer := NewTrainer(model, 0.01)

	// Single positive pair and single negative pair
	posBatch := [][2]int{{idx["A"], idx["B"]}}
	negBatch := [][2]int{{idx["A"], idx["C"]}} // A-C is negative

	loss := trainer.step(posBatch, negBatch)

	assert.Greater(t, loss, 0.0, "BPR loss should be positive for a single pair")
}

func TestStep_ErrorPath_EmptyBatch(t *testing.T) {
	g := smallGraph()
	model := gnnModelWithEmbeddings(g, 8, 42)

	trainer := NewTrainer(model, 0.01)

	loss := trainer.step([][2]int{}, [][2]int{})
	assert.True(t, loss != loss, "empty batch 0/0 division should produce NaN")
}
