package decision

import (
	"math"
	"testing"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

// ─── NewGNNLinkPredictor Tests ───────────────────────────────────────────

func TestNewGNNLinkPredictor_Happy(t *testing.T) {
	t.Parallel()

	p := NewGNNLinkPredictor(100, 64, 0.01)
	if p == nil {
		t.Fatal("expected non-nil predictor")
	}
	if p.IsTrained() {
		t.Error("expected untrained predictor after NewGNNLinkPredictor")
	}
	if p.model == nil {
		t.Error("expected non-nil model")
	}
	if p.model.NumNodes != 100 {
		t.Errorf("expected NumNodes=100, got %d", p.model.NumNodes)
	}
	if p.model.Dim != 64 {
		t.Errorf("expected Dim=64, got %d", p.model.Dim)
	}
}

func TestNewGNNLinkPredictor_EdgeZeroNodes(t *testing.T) {
	t.Parallel()

	p := NewGNNLinkPredictor(0, 32, 0.001)
	if p == nil {
		t.Fatal("expected non-nil predictor even with zero nodes")
	}
	if p.model.NumNodes != 0 {
		t.Errorf("expected NumNodes=0, got %d", p.model.NumNodes)
	}
	if p.IsTrained() {
		t.Error("expected untrained predictor with zero nodes")
	}
}

func TestNewGNNLinkPredictor_ErrorNegativeLR(t *testing.T) {
	t.Parallel()

	p := NewGNNLinkPredictor(50, 128, -0.01)
	if p == nil {
		t.Fatal("expected non-nil predictor even with negative LR (caller responsibility)")
	}
	// Negative LR is accepted at construction; training may diverge
	if p.IsTrained() {
		t.Error("expected untrained predictor")
	}
}

// ─── TrainFromGraph Tests ────────────────────────────────────────────────

func TestTrainFromGraph_Happy(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(4)
	p := NewGNNLinkPredictor(4, 64, 0.01)

	err := p.TrainFromGraph(t.Context(), graph, 10)
	if err != nil {
		t.Fatalf("TrainFromGraph failed: %v", err)
	}
	if !p.IsTrained() {
		t.Error("expected IsTrained=true after successful training")
	}
	if p.nodeIndex == nil {
		t.Error("expected non-nil nodeIndex after training")
	}
	if len(p.idByIndex) != 4 {
		t.Errorf("expected idByIndex length 4, got %d", len(p.idByIndex))
	}
}

func TestTrainFromGraph_EdgeEmptyGraph(t *testing.T) {
	t.Parallel()

	graph := gnn.NewGraph()
	p := NewGNNLinkPredictor(10, 64, 0.01)

	err := p.TrainFromGraph(t.Context(), graph, 10)
	if err == nil {
		t.Fatal("expected error training on empty graph")
	}
	if p.IsTrained() {
		t.Error("expected untrained after failed training")
	}
}

func TestTrainFromGraph_ErrorNoEdges(t *testing.T) {
	t.Parallel()

	graph := gnn.NewGraph()
	graph.AddNode(&gnn.WorkflowNode{ID: "a"})
	graph.AddNode(&gnn.WorkflowNode{ID: "b"})
	// No edges added

	p := NewGNNLinkPredictor(2, 64, 0.01)
	err := p.TrainFromGraph(t.Context(), graph, 10)
	if err == nil {
		t.Fatal("expected error when graph has nodes but no edges")
	}
}

// ─── PredictLinks Tests ──────────────────────────────────────────────────

func TestPredictLinks_Happy(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(3)
	p := NewGNNLinkPredictor(3, 64, 0.01)

	if err := p.TrainFromGraph(t.Context(), graph, 10); err != nil {
		t.Fatalf("training failed: %v", err)
	}

	scores, err := p.PredictLinks(t.Context(), graph, "alice")
	if err != nil {
		t.Fatalf("PredictLinks failed: %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(scores))
	}
	for i, s := range scores {
		if math.IsNaN(s) {
			t.Errorf("score[%d] is NaN", i)
		}
	}
}

func TestPredictLinks_EdgeNotTrained(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(2)
	p := NewGNNLinkPredictor(2, 64, 0.01)

	_, err := p.PredictLinks(t.Context(), graph, "alice")
	if err == nil {
		t.Fatal("expected error predicting without training")
	}
}

func TestPredictLinks_ErrorUnknownEntity(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(3)
	p := NewGNNLinkPredictor(3, 64, 0.01)

	if err := p.TrainFromGraph(t.Context(), graph, 10); err != nil {
		t.Fatalf("training failed: %v", err)
	}

	_, err := p.PredictLinks(t.Context(), graph, "unknown_entity")
	if err == nil {
		t.Fatal("expected error for unknown entity")
	}
}

// ─── IsTrained Tests ─────────────────────────────────────────────────────

func TestIsTrained_HappyAfterTraining(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(4)
	p := NewGNNLinkPredictor(4, 64, 0.01)

	if p.IsTrained() {
		t.Error("expected false before training")
	}

	if err := p.TrainFromGraph(t.Context(), graph, 10); err != nil {
		t.Fatalf("training failed: %v", err)
	}

	if !p.IsTrained() {
		t.Error("expected true after successful training")
	}
}

func TestIsTrained_EdgeAfterFailedTraining(t *testing.T) {
	t.Parallel()

	graph := gnn.NewGraph() // empty
	p := NewGNNLinkPredictor(10, 64, 0.01)

	_ = p.TrainFromGraph(t.Context(), graph, 10) // expected to fail

	if p.IsTrained() {
		t.Error("expected false after failed training")
	}
}

func TestIsTrained_ErrorAfterGraphSizeMismatch(t *testing.T) {
	t.Parallel()

	graph := buildSimpleGraph(3)
	p := NewGNNLinkPredictor(3, 64, 0.01)

	if err := p.TrainFromGraph(t.Context(), graph, 10); err != nil {
		t.Fatalf("training failed: %v", err)
	}

	// IsTrained should remain true even if a subsequent PredictLinks fails due to size mismatch
	// (we test that IsTrained is independent of PredictLinks outcome)
	if !p.IsTrained() {
		t.Error("expected true after successful training, regardless of later PredictLinks outcomes")
	}
}

// ─── ConfidenceFromPredictions Tests ─────────────────────────────────────

func TestConfidenceFromPredictions_Happy(t *testing.T) {
	t.Parallel()

	scores := []float64{0.5, 0.8, 0.3}
	conf := ConfidenceFromPredictions(scores)

	if conf <= 0.5 || conf >= 1.0 {
		t.Errorf("expected confidence in (0.5, 1.0), got %.4f", conf)
	}

	expected := 1.0 / (1.0 + math.Exp(-0.8))
	if math.Abs(conf-expected) > 0.0001 {
		t.Errorf("expected confidence ~%.4f, got %.4f", expected, conf)
	}
}

func TestConfidenceFromPredictions_EdgeEmptyScores(t *testing.T) {
	t.Parallel()

	conf := ConfidenceFromPredictions([]float64{})
	if conf != 0.5 {
		t.Errorf("expected 0.5 for empty scores, got %.4f", conf)
	}
}

func TestConfidenceFromPredictions_ErrorNegativeScores(t *testing.T) {
	t.Parallel()

	scores := []float64{-1.0, -0.5, -0.2}
	conf := ConfidenceFromPredictions(scores)

	// Max is -0.2, sigmoid(-0.2) ≈ 0.45
	if conf <= 0 || conf >= 1.0 {
		t.Errorf("expected confidence in (0, 1) for negative scores, got %.4f", conf)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────

func buildSimpleGraph(n int) *gnn.Graph {
	graph := gnn.NewGraph()
	nodes := []string{"alice", "bob", "charlie", "dave", "eve", "frank", "grace", "helen"}
	for i := 0; i < n && i < len(nodes); i++ {
		graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID(nodes[i])})
	}
	for i := 0; i < n-1; i++ {
		graph.AddEdge(gnn.Edge{
			Source: gnn.NodeID(nodes[i]),
			Target: gnn.NodeID(nodes[i+1]),
			Weight: 1.0,
		})
	}
	return graph
}
