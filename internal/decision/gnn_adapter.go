package decision

import (
	"context"
	"fmt"
	"math"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

// LinkPredictor predicts links in a workspace knowledge graph.
// Implementations wrap GNN models and provide training + inference.
type LinkPredictor interface {
	// PredictLinks scores entityID against all nodes in the graph.
	// Returns a slice of scores (dot-product link predictions) the same
	// length as graph.NumNodes(), indexed by BuildNodeIndex order.
	PredictLinks(ctx context.Context, graph *gnn.Graph, entityID string) ([]float64, error)

	// TrainFromGraph builds node index, adjacency matrix, generates
	// negative samples, and trains the underlying GNN model.
	TrainFromGraph(ctx context.Context, graph *gnn.Graph, epochs int) error

	// IsTrained returns true if the predictor has been trained at least once.
	// The engine uses this to decide whether to blend GNN scores into confidence.
	IsTrained() bool
}

// compile-time interface check
var _ LinkPredictor = (*GNNLinkPredictor)(nil)

// GNNLinkPredictor wraps gnn.GNNModel + gnn.Trainer into the LinkPredictor
// interface. After training, it caches the node index and embeddings so
// PredictLinks can score an entity against all graph nodes.
type GNNLinkPredictor struct {
	model     *gnn.GNNModel
	trainer   *gnn.Trainer
	nodeIndex map[gnn.NodeID]int
	idByIndex []gnn.NodeID
	trained   bool
}

// NewGNNLinkPredictor creates a GNNLinkPredictor with randomly initialized
// embeddings of the given dimension and learning rate.
//
//   - numNodes: maximum number of nodes the model should support
//   - dim:      embedding dimension (e.g. 64)
//   - lr:       initial learning rate for BPR training
//
// The model can be resized automatically during TrainFromGraph.
func NewGNNLinkPredictor(numNodes, dim int, lr float64) *GNNLinkPredictor {
	model := gnn.NewGNNModel(numNodes, dim, 42)
	trainer := gnn.NewTrainer(model, lr)
	return &GNNLinkPredictor{
		model:   model,
		trainer: trainer,
	}
}

// TrainFromGraph implements LinkPredictor.
//
// It builds the node index from the graph, runs BuildAdjacency, creates
// positive-edge pairs and negative samples, trains for the given number
// of epochs, then caches the resulting embeddings for fast PredictLinks.
func (p *GNNLinkPredictor) TrainFromGraph(ctx context.Context, graph *gnn.Graph, epochs int) error {
	if graph.NumNodes() == 0 {
		return fmt.Errorf("gnn adapter: cannot train on empty graph")
	}

	// Recreate model if node count changed
	if p.model.NumNodes != graph.NumNodes() {
		p.model = gnn.NewGNNModel(graph.NumNodes(), p.model.Dim, 42)
		p.trainer = gnn.NewTrainer(p.model, p.trainer.LR)
	}

	idx := graph.BuildNodeIndex()
	p.model.BuildAdjacency(idx, graph.Edges)

	// Build positive edge pairs
	posEdges := make([][2]int, 0, graph.NumEdges())
	for _, e := range graph.Edges {
		u, okU := idx[e.Source]
		v, okV := idx[e.Target]
		if okU && okV {
			posEdges = append(posEdges, [2]int{u, v})
		}
	}

	if len(posEdges) == 0 {
		return fmt.Errorf("gnn adapter: no valid positive edges after building index")
	}

	// Generate negative samples (1:1 ratio)
	sampler := gnn.NewNegativeSampler(42, 1.0)
	negEdges := sampler.Sample(graph.Edges, idx)

	if len(negEdges) == 0 {
		return fmt.Errorf("gnn adapter: no negative samples generated")
	}

	// Train
	_ = p.trainer.Train(posEdges, negEdges, epochs)

	// Cache results
	p.nodeIndex = idx
	p.idByIndex = make([]gnn.NodeID, len(idx))
	for id, i := range idx {
		p.idByIndex[i] = id
	}
	p.trained = true

	return nil
}

// PredictLinks implements LinkPredictor.
//
// It runs a forward pass through the GNN (using the cached adjacency from
// training) and scores the given entity against every node in the graph
// via dot-product link prediction. Returns scores in BuildNodeIndex order.
func (p *GNNLinkPredictor) PredictLinks(ctx context.Context, graph *gnn.Graph, entityID string) ([]float64, error) {
	if !p.trained {
		return nil, fmt.Errorf("gnn adapter: not trained — call TrainFromGraph first")
	}

	if graph.NumNodes() != p.model.NumNodes {
		return nil, fmt.Errorf(
			"gnn adapter: graph size mismatch: model has %d nodes, graph has %d",
			p.model.NumNodes, graph.NumNodes(),
		)
	}

	emb := p.model.Forward()

	entityIdx, ok := p.nodeIndex[gnn.NodeID(entityID)]
	if !ok {
		return nil, fmt.Errorf("gnn adapter: entity %q not found in node index", entityID)
	}

	scores := make([]float64, len(emb))
	for i := range emb {
		scores[i] = gnn.PredictScore(emb, entityIdx, i)
	}

	return scores, nil
}

// IsTrained returns true if the predictor has been trained at least once.
func (p *GNNLinkPredictor) IsTrained() bool {
	return p.trained
}

// ConfidenceFromPredictions computes a confidence score (0.0–1.0) from
// predicted link scores using sigmoid normalization of the maximum score.
// This can be blended with keyword-matching confidence in the DecisionEngine.
func ConfidenceFromPredictions(scores []float64) float64 {
	if len(scores) == 0 {
		return 0.5
	}
	var maxScore float64
	for _, s := range scores {
		if s > maxScore {
			maxScore = s
		}
	}
	// Sigmoid of max score gives a 0.5–1.0 range when maxScore >= 0.
	return 1.0 / (1.0 + math.Exp(-maxScore))
}
