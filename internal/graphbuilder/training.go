package graphbuilder

import (
	"fmt"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

type TrainResult struct {
	Model       *gnn.GNNModel
	LossHistory []float64 `json:"loss_history"`
	FinalLoss   float64   `json:"final_loss"`
	AUC         float64   `json:"auc"`
	MRR         float64   `json:"mrr"`
	EpochsRun   int       `json:"epochs_run"`
}

func (b *PoliticalGraphBuilder) TrainGNN(embeddingDim, epochs int) (*TrainResult, error) {
	numNodes := b.Graph.NumNodes()
	if numNodes < 2 {
		return nil, fmt.Errorf("need at least 2 nodes to train GNN, got %d", numNodes)
	}

	model := gnn.NewGNNModel(numNodes, embeddingDim, b.Seed)
	nodeIndex := b.Graph.BuildNodeIndex()
	model.BuildAdjacency(nodeIndex, b.Graph.Edges)

	sampler := gnn.NewNegativeSampler(b.Seed, 1.0)
	negEdges := sampler.Sample(b.Graph.Edges, nodeIndex)

	posEdges := make([][2]int, 0, b.Graph.NumEdges())
	for _, e := range b.Graph.Edges {
		u, okU := nodeIndex[e.Source]
		v, okV := nodeIndex[e.Target]
		if okU && okV {
			posEdges = append(posEdges, [2]int{u, v})
		}
	}

	if len(posEdges) == 0 {
		return nil, fmt.Errorf("no valid positive edges found for training")
	}

	trainer := gnn.NewTrainer(model, 0.01)
	trainResult := trainer.Train(posEdges, negEdges, epochs)

	embeddings := model.Forward()
	evaluator := gnn.NewEvaluator()

	auc := evaluator.AUC(embeddings, posEdges, negEdges)
	mrr := evaluator.MRR(embeddings, posEdges)

	return &TrainResult{
		Model:       model,
		LossHistory: trainResult.LossHistory,
		FinalLoss:   trainResult.FinalLoss,
		AUC:         auc,
		MRR:         mrr,
		EpochsRun:   trainResult.EpochsRun,
	}, nil
}
