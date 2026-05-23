package graphbuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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

// StoreEmbeddings persists GNN node embeddings into a DuckDB memory_store table.
// Each node's final embedding (from model.Forward()) is stored with its key,
// node type, embedding vector, and metadata JSON.
func StoreEmbeddings(db *sql.DB, graph *gnn.Graph, model *gnn.GNNModel) error {
	if db == nil {
		return fmt.Errorf("StoreEmbeddings: db is nil")
	}
	if graph == nil {
		return fmt.Errorf("StoreEmbeddings: graph is nil")
	}
	if model == nil {
		return fmt.Errorf("StoreEmbeddings: model is nil")
	}

	dim := model.Dim
	_, err := db.ExecContext(context.Background(), "CREATE TABLE IF NOT EXISTS memory_store ("+
		"key VARCHAR PRIMARY KEY, "+
		"node_type VARCHAR, "+
		"metadata JSON, "+
		"embedding FLOAT["+fmt.Sprintf("%d", dim)+"], "+
		"stored_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"+
		")")
	if err != nil {
		return fmt.Errorf("StoreEmbeddings: create table: %w", err)
	}

	if model.NumNodes == 0 {
		return nil
	}

	embeddings := model.Forward()
	if len(embeddings) == 0 {
		return nil
	}

	nodeIndex := graph.BuildNodeIndex()

	for nodeID, idx := range nodeIndex {
		emb := embeddings[idx]
		embLiteral := floatsToLiteral(emb)

		node := graph.Nodes[nodeID]
		nodeType := ""
		if node != nil {
			nodeType = node.Type
		}
		meta := map[string]interface{}{
			"node_type": nodeType,
			"node_id":   string(nodeID),
		}
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("StoreEmbeddings: marshal metadata for %s: %w", nodeID, err)
		}

		_, err = db.ExecContext(context.Background(),
			"DELETE FROM memory_store WHERE key = ?", string(nodeID))
		if err != nil {
			return fmt.Errorf("StoreEmbeddings: delete old row for %s: %w", nodeID, err)
		}

		_, err = db.ExecContext(context.Background(),
			"INSERT INTO memory_store (key, node_type, metadata, embedding) VALUES (?, ?, ?, "+embLiteral+")",
			string(nodeID), nodeType, metaBytes)
		if err != nil {
			return fmt.Errorf("StoreEmbeddings: insert row for %s: %w", nodeID, err)
		}
	}

	return nil
}

// floatsToLiteral formats a []float64 as a DuckDB array literal, e.g. "[0.1,0.2,0.3]::FLOAT[3]".
func floatsToLiteral(nums []float64) string {
	if len(nums) == 0 {
		return "[]::FLOAT[0]"
	}
	parts := make([]string, len(nums))
	for i, v := range nums {
		parts[i] = fmt.Sprintf("%.16g", v)
	}
	return fmt.Sprintf("[%s]::FLOAT[%d]", strings.Join(parts, ","), len(nums))
}
