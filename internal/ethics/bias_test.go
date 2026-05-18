package ethics

import (
	"math"
	"testing"
	"time"
)

// ─── CheckDataBalance tests ───────────────────────────────────────────

func TestCheckDataBalance_Uniform(t *testing.T) {
	counts := map[string]int{
		"type_a": 100,
		"type_b": 100,
		"type_c": 100,
	}
	result := CheckDataBalance(counts, 0.70)
	if !result.Passed {
		t.Errorf("uniform distribution should pass: score=%.3f msg=%s", result.Score, result.Message)
	}
	if result.Score < 0.99 {
		t.Errorf("uniform distribution should have score near 1.0, got %.3f", result.Score)
	}
}

func TestCheckDataBalance_Skewed(t *testing.T) {
	counts := map[string]int{
		"type_a": 900,
		"type_b": 90,
		"type_c": 10,
	}
	result := CheckDataBalance(counts, 0.70)
	if result.Passed {
		t.Errorf("skewed distribution should fail: score=%.3f msg=%s", result.Score, result.Message)
	}
	if result.Score >= 0.70 {
		t.Errorf("skewed distribution should have score < 0.70, got %.3f", result.Score)
	}
}

func TestCheckDataBalance_Empty(t *testing.T) {
	result := CheckDataBalance(map[string]int{}, 0.70)
	if result.Passed {
		t.Error("empty input should not pass")
	}
	if result.Score != 0 {
		t.Errorf("empty input should have score 0, got %.3f", result.Score)
	}
}

func TestCheckDataBalance_SingleGroup(t *testing.T) {
	counts := map[string]int{
		"type_a": 500,
	}
	result := CheckDataBalance(counts, 0.70)
	// single group has maxEntropy=0, normalized to 1.0
	if !result.Passed {
		t.Errorf("single group should pass (score=%.3f, msg=%s)", result.Score, result.Message)
	}
}

func TestCheckDataBalance_AllZero(t *testing.T) {
	counts := map[string]int{
		"type_a": 0,
		"type_b": 0,
	}
	result := CheckDataBalance(counts, 0.70)
	if result.Passed {
		t.Error("all-zero counts should not pass")
	}
}

func TestEdgeTypeBalance_Proxy(t *testing.T) {
	result := EdgeTypeBalance(map[string]int{"train": 200, "eval": 50})
	if result.Kind != BiasDataBias {
		t.Errorf("expected data bias kind, got %s", result.Kind)
	}
}

// ─── ComputeDemographicParity tests ───────────────────────────────────

func TestComputeDemographicParity_Equal(t *testing.T) {
	// Two groups with identical embeddings → perfect parity.
	emb := make([][]float64, 4)
	for i := range emb {
		emb[i] = []float64{1.0, 0.0}
	}
	groups := map[string][]int{
		"group_a": {0, 1},
		"group_b": {2, 3},
	}
	result := ComputeDemographicParity(emb, groups, 0.10)
	if !result.Passed {
		t.Errorf("identical groups should pass: score=%.3f msg=%s", result.Score, result.Message)
	}
}

func TestComputeDemographicParity_Unequal(t *testing.T) {
	// Two groups with very different embeddings.
	emb := make([][]float64, 4)
	emb[0] = []float64{10.0, 10.0}
	emb[1] = []float64{10.0, 10.0}
	emb[2] = []float64{0.0, 1.0}
	emb[3] = []float64{0.0, 1.0}
	groups := map[string][]int{
		"group_a": {0, 1},
		"group_b": {2, 3},
	}
	result := ComputeDemographicParity(emb, groups, 0.10)
	if result.Passed {
		t.Errorf("unequal groups should fail: score=%.3f msg=%s", result.Score, result.Message)
	}
}

func TestComputeDemographicParity_EmptyEmbeddings(t *testing.T) {
	result := ComputeDemographicParity([][]float64{}, map[string][]int{"a": {0}}, 0.10)
	if result.Passed {
		t.Error("empty embeddings should not pass")
	}
}

func TestComputeDemographicParity_NoGroups(t *testing.T) {
	emb := [][]float64{{1.0, 0.0}, {0.0, 1.0}}
	result := ComputeDemographicParity(emb, map[string][]int{}, 0.10)
	if !result.Passed {
		t.Error("no groups should pass (skip)")
	}
}

func TestComputeDemographicParity_SingleGroup(t *testing.T) {
	emb := [][]float64{{1.0, 0.0}, {0.0, 1.0}}
	groups := map[string][]int{"only": {0, 1}}
	result := ComputeDemographicParity(emb, groups, 0.10)
	if !result.Passed {
		t.Error("single group should pass (skip)")
	}
}

// ─── ComputeDiversityScore tests ──────────────────────────────────────

func TestComputeDiversityScore_HighDiversity(t *testing.T) {
	// All pairs have zero similarity → perfectly diverse.
	n := 5
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		for j := range matrix[i] {
			if i != j {
				matrix[i][j] = 0.0
			} else {
				matrix[i][j] = 1.0
			}
		}
	}
	result := ComputeDiversityScore(n, matrix, 0.35)
	if !result.Passed {
		t.Errorf("high diversity should pass: score=%.3f msg=%s", result.Score, result.Message)
	}
	if result.Score < 0.99 {
		t.Errorf("diversity score should be near 1.0, got %.3f", result.Score)
	}
}

func TestComputeDiversityScore_LowDiversity(t *testing.T) {
	// All pairs have high similarity.
	n := 5
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		for j := range matrix[i] {
			matrix[i][j] = 0.9 // very similar
		}
	}
	result := ComputeDiversityScore(n, matrix, 0.35)
	if result.Passed {
		t.Errorf("low diversity should fail: score=%.3f msg=%s", result.Score, result.Message)
	}
}

func TestComputeDiversityScore_SingleItem(t *testing.T) {
	result := ComputeDiversityScore(1, [][]float64{{1.0}}, 0.35)
	if !result.Passed {
		t.Error("single item should pass (skip)")
	}
}

func TestComputeDiversityScore_EmptyMatrix(t *testing.T) {
	result := ComputeDiversityScore(5, [][]float64{}, 0.35)
	if !result.Passed {
		t.Error("empty matrix should pass (skip)")
	}
}

// ─── DecayWeight tests ────────────────────────────────────────────────

func TestDecayWeight_ZeroElapsed(t *testing.T) {
	w := DecayWeight(0, 24*time.Hour)
	if w != 1.0 {
		t.Errorf("zero elapsed should return 1.0, got %.3f", w)
	}
}

func TestDecayWeight_HalfLife(t *testing.T) {
	w := DecayWeight(24*time.Hour, 24*time.Hour)
	expected := 0.5
	if math.Abs(w-expected) > 1e-10 {
		t.Errorf("elapsed=halfLife should return 0.5, got %.3f", w)
	}
}

func TestDecayWeight_TwoHalfLives(t *testing.T) {
	w := DecayWeight(48*time.Hour, 24*time.Hour)
	expected := 0.25
	if math.Abs(w-expected) > 1e-10 {
		t.Errorf("elapsed=2*halfLife should return 0.25, got %.3f", w)
	}
}

func TestDecayWeight_ZeroHalfLife(t *testing.T) {
	w := DecayWeight(24*time.Hour, 0)
	if w != 1.0 {
		t.Errorf("zero halfLife should return 1.0, got %.3f", w)
	}
}

func TestDecayWeight_NegativeHalfLife(t *testing.T) {
	w := DecayWeight(24*time.Hour, -1*time.Hour)
	if w != 1.0 {
		t.Errorf("negative halfLife should return 1.0, got %.3f", w)
	}
}

func TestDecayedScore_Integration(t *testing.T) {
	raw := 100.0
	decayed := DecayedScore(raw, 24*time.Hour, 24*time.Hour)
	if math.Abs(decayed-50.0) > 1e-10 {
		t.Errorf("half-life decayed score should be 50, got %.3f", decayed)
	}
}

// ─── BiasReport tests ─────────────────────────────────────────────────

func TestBiasReport_AllPassed(t *testing.T) {
	report := BiasReport{
		Checks: []BiasCheckResult{
			{Passed: true},
			{Passed: true},
		},
	}
	if !report.AllPassed() {
		t.Error("all-passed report should return true")
	}
}

func TestBiasReport_SomeFailed(t *testing.T) {
	report := BiasReport{
		Checks: []BiasCheckResult{
			{Passed: true},
			{Passed: false},
		},
	}
	if report.AllPassed() {
		t.Error("report with failures should return false")
	}
}

func TestBiasReport_Summary(t *testing.T) {
	report := BiasReport{
		Checks: []BiasCheckResult{
			{Passed: true, Kind: BiasDataBias},
			{Passed: false, Kind: BiasAlgorithmicBias},
			{Passed: true, Kind: BiasConfirmationBias},
		},
	}
	s := report.Summary()
	expected := "2/3 bias checks passed"
	if s != expected {
		t.Errorf("summary mismatch:\n  got:  %s\n  want: %s", s, expected)
	}
}

func TestBiasReport_Empty(t *testing.T) {
	report := BiasReport{}
	// Empty report is vacuously true — no checks means none failed.
	if !report.AllPassed() {
		t.Error("empty report should vacuously pass")
	}
	s := report.Summary()
	if s != "no bias checks performed" {
		t.Errorf("empty summary mismatch: %s", s)
	}
}

// ─── RunAllChecks tests ───────────────────────────────────────────────

func TestRunAllChecks_Full(t *testing.T) {
	counts := map[string]int{"a": 100, "b": 100}
	emb := [][]float64{{1.0, 0.0}, {0.0, 1.0}, {1.0, 0.0}, {0.0, 1.0}}
	groups := map[string][]int{"g1": {0, 1}, "g2": {2, 3}}
	matrix := [][]float64{{1.0, 0.0}, {0.0, 1.0}}

	report := RunAllChecks(counts, emb, groups, matrix, 2)
	if len(report.Checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(report.Checks))
	}
}

func TestRunAllChecks_Partial(t *testing.T) {
	// Only data balance, skip others.
	report := RunAllChecks(map[string]int{"a": 1}, nil, nil, nil, 0)
	if len(report.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(report.Checks))
	}
	if report.Checks[0].Kind != BiasDataBias {
		t.Errorf("expected data bias check, got %s", report.Checks[0].Kind)
	}
}

func TestRunAllChecks_None(t *testing.T) {
	report := RunAllChecks(nil, nil, nil, nil, 0)
	if len(report.Checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(report.Checks))
	}
}

// ─── Integration: using ethics alongside gnn ──────────────────────────

func TestIntegration_DataBalanceWithGNN(t *testing.T) {
	// Simulate checking the balance of edge types in a graph.
	edgeTypeCounts := map[string]int{
		"train": 400,
		"eval":  100,
	}
	result := CheckDataBalance(edgeTypeCounts, 0.50)
	// 400/500 vs 100/500 → entropy normalized (2 types, max=1.0)
	// p_train=0.8, p_eval=0.2
	// entropy = -(0.8*log2(0.8) + 0.2*log2(0.2))
	t.Logf("Edge type balance: score=%.3f passed=%v msg=%s", result.Score, result.Passed, result.Message)
}

func TestIntegration_DemographicParityWithEmbeddings(t *testing.T) {
	// Simulate checking fairness of node embeddings.
	nodes := 8
	dim := 4
	emb := make([][]float64, nodes)
	for i := range emb {
		emb[i] = make([]float64, dim)
		for j := range emb[i] {
			emb[i][j] = 0.5 // all same → perfect parity
		}
	}
	groups := map[string][]int{
		"users": {0, 1, 2, 3},
		"tools": {4, 5, 6, 7},
	}
	result := ComputeDemographicParity(emb, groups, 0.10)
	if !result.Passed {
		t.Errorf("all-identical embeddings should have parity: score=%.3f", result.Score)
	}
	if result.Score < 0.99 {
		t.Errorf("parity score should be near 1.0, got %.3f", result.Score)
	}
}

func TestIntegration_RunAllChecksThenReport(t *testing.T) {
	counts := map[string]int{"x": 300, "y": 300, "z": 300}
	emb := [][]float64{{1, 0}, {1, 0}, {0, 1}, {0, 1}}
	groups := map[string][]int{"a": {0, 1}, "b": {2, 3}}
	matrix := [][]float64{{1, 0.1, 0.2}, {0.1, 1, 0.1}, {0.2, 0.1, 1}}

	report := RunAllChecks(counts, emb, groups, matrix, 3)
	t.Logf("Bias report: %s", report.Summary())
	for _, c := range report.Checks {
		t.Logf("  [%s] passed=%v score=%.3f msg=%s", c.Kind, c.Passed, c.Score, c.Message)
	}
}
