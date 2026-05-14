package telemetry

import (
	"testing"
)

// ── Metrics recording functions ────────────────────────────────────────────

func TestRecordNLPRequest(t *testing.T) {
	// RecordNLPRequest is a thin wrapper around Prometheus CounterVec.Inc().
	// It should not panic with valid labels.
	RecordNLPRequest("analyze_sentiment", "success")
	RecordNLPRequest("analyze_sentiment", "error")
	RecordNLPRequest("analyze_market", "success")
}

func TestRecordDBQuery(t *testing.T) {
	// RecordDBQuery wraps HistogramVec.Observe().
	RecordDBQuery("select", 0.042)
	RecordDBQuery("insert", 0.001)
	RecordDBQuery("update", 0.157)
	RecordDBQuery("select", 0.0)
}

func TestRecordPAORACycle(t *testing.T) {
	// RecordPAORACycle wraps CounterVec.Inc() for decision engine phases.
	RecordPAORACycle("plan", "success")
	RecordPAORACycle("act", "failure")
	RecordPAORACycle("observe", "success")
	RecordPAORACycle("reflect", "success")
	RecordPAORACycle("admit", "failure")
}

func TestSetDBConnections(t *testing.T) {
	// SetDBConnections is a gauge setter.
	SetDBConnections(0)
	SetDBConnections(5)
	SetDBConnections(42)
	SetDBConnections(0)
}

func TestRecordMetrics_AllCombinations(t *testing.T) {
	// Smoke test: call all 4 functions with various inputs.
	methods := []string{"analyze_sentiment", "simulate_market", "generate_ensemble"}
	for _, m := range methods {
		RecordNLPRequest(m, "success")
	}

	operations := []string{"select", "insert", "update", "delete"}
	for _, op := range operations {
		RecordDBQuery(op, 0.01)
	}

	phases := []string{"plan", "act", "observe", "reflect", "admit"}
	outcomes := []string{"success", "failure"}
	for _, p := range phases {
		for _, o := range outcomes {
			RecordPAORACycle(p, o)
		}
	}

	for _, n := range []float64{0, 1, 10, 100} {
		SetDBConnections(n)
	}
}
