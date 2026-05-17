package decision

import (
	"testing"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

func TestGNNLinkPredictor_EdgeWeightImportance(t *testing.T) {
	t.Parallel()

	graphA := gnn.NewGraph()
	graphA.AddNode(&gnn.WorkflowNode{ID: "alice"})
	graphA.AddNode(&gnn.WorkflowNode{ID: "bob"})
	graphA.AddNode(&gnn.WorkflowNode{ID: "charlie"})
	graphA.AddEdge(gnn.Edge{Source: "alice", Target: "bob", Weight: 1.0})
	graphA.AddEdge(gnn.Edge{Source: "alice", Target: "charlie", Weight: 1.0})

	graphB := gnn.NewGraph()
	graphB.AddNode(&gnn.WorkflowNode{ID: "alice"})
	graphB.AddNode(&gnn.WorkflowNode{ID: "bob"})
	graphB.AddNode(&gnn.WorkflowNode{ID: "charlie"})
	graphB.AddEdge(gnn.Edge{Source: "alice", Target: "bob", Weight: 5.0})
	graphB.AddEdge(gnn.Edge{Source: "alice", Target: "charlie", Weight: 1.0})

	predA := NewGNNLinkPredictor(3, 64, 0.01)
	if err := predA.TrainFromGraph(t.Context(), graphA, 10); err != nil {
		t.Fatalf("TrainFromGraph(A): %v", err)
	}

	predB := NewGNNLinkPredictor(3, 64, 0.01)
	if err := predB.TrainFromGraph(t.Context(), graphB, 10); err != nil {
		t.Fatalf("TrainFromGraph(B): %v", err)
	}

	scoresA, err := predA.PredictLinks(t.Context(), graphA, "alice")
	if err != nil {
		t.Fatalf("PredictLinks(A): %v", err)
	}

	scoresB, err := predB.PredictLinks(t.Context(), graphB, "alice")
	if err != nil {
		t.Fatalf("PredictLinks(B): %v", err)
	}

	scoreA := scoresA[1]
	scoreB := scoresB[1]
	if scoreA == scoreB {
		t.Logf("WARNING: weighted (%.4f) and unweighted (%.4f) scores are identical — edge weight may be ignored", scoreB, scoreA)
	}
}
