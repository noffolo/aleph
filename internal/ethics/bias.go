// Package ethics provides bias detection and remediation functions for the
// aleph-v2 system. It implements mitigations for the bias categories identified
// in docs/development-bias-checklist.md.
//
// Mitigations implemented:
//   - Data bias:   CheckDataBalance — measures label/type distribution via entropy
//   - Algorithmic bias: ComputeDemographicParity — fairness metric on embeddings
//   - Confirmation bias: ComputeDiversityScore — Jaccard-based recommendation diversity
//   - Availability bias: DecayWeight — exponential time decay for scoring
package ethics

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// BiasKind identifies the type of bias being checked.
type BiasKind string

const (
	BiasDataBias         BiasKind = "data_bias"
	BiasAlgorithmicBias  BiasKind = "algorithmic_bias"
	BiasConfirmationBias BiasKind = "confirmation_bias"
	BiasAvailabilityBias BiasKind = "availability_bias"
)

// Default thresholds for bias checks.
const (
	DefaultDataBalanceThreshold     = 0.70 // minimum normalized entropy to pass
	DefaultParityThreshold          = 0.10 // maximum group score difference to pass
	DefaultDiversityThreshold       = 0.35 // maximum average Jaccard similarity to pass
)

// BiasCheckResult captures the outcome of a single bias check.
type BiasCheckResult struct {
	Kind      BiasKind `json:"kind"`
	Passed    bool     `json:"passed"`
	Score     float64  `json:"score"`     // 0.0 (worst) to 1.0 (best)
	Threshold float64  `json:"threshold"` // the threshold used for this check
	Message   string   `json:"message"`   // human-readable outcome
}

// BiasReport aggregates all bias checks.
type BiasReport struct {
	Checks []BiasCheckResult `json:"checks"`
}

// AllPassed returns true only when every individual check passed.
func (r *BiasReport) AllPassed() bool {
	for _, c := range r.Checks {
		if !c.Passed {
			return false
		}
	}
	return true
}

// Summary returns a one-line status string for the report.
func (r *BiasReport) Summary() string {
	var passed, total int
	for _, c := range r.Checks {
		total++
		if c.Passed {
			passed++
		}
	}
	if total == 0 {
		return "no bias checks performed"
	}
	return fmt.Sprintf("%d/%d bias checks passed", passed, total)
}

// ────────────────────────────────────────────────────────────────────────────
// Data Bias: Distribution Balance (entropy-based)
// ────────────────────────────────────────────────────────────────────────────

// CheckDataBalance verifies that the distribution of items across categories
// is not overly skewed. It computes the normalized entropy of the distribution
// and compares it against a threshold.
//
//   - score = 1.0: perfectly uniform distribution (all groups equal size)
//   - score = 0.0: all items belong to a single group
//   - passed: score >= threshold
//
// This directly mitigates **data bias** by detecting training-data skew
// before it propagates into model predictions.
func CheckDataBalance(categoryCounts map[string]int, threshold float64) BiasCheckResult {
	if len(categoryCounts) == 0 {
		return BiasCheckResult{
			Kind:      BiasDataBias,
			Passed:    false,
			Score:     0,
			Threshold: threshold,
			Message:   "no category data to evaluate",
		}
	}

	total := 0
	for _, c := range categoryCounts {
		total += c
	}
	if total == 0 {
		return BiasCheckResult{
			Kind:      BiasDataBias,
			Passed:    false,
			Score:     0,
			Threshold: threshold,
			Message:   "all category counts are zero",
		}
	}

	k := len(categoryCounts)
	maxEntropy := math.Log2(float64(k))

	var observedEntropy float64
	for _, c := range categoryCounts {
		if c > 0 {
			p := float64(c) / float64(total)
			observedEntropy -= p * math.Log2(p)
		}
	}

	normalized := 1.0
	if maxEntropy > 0 {
		normalized = observedEntropy / maxEntropy
	}

	passed := normalized >= threshold

	var msg string
	if passed {
		msg = fmt.Sprintf("distribution is balanced (normalized entropy=%.3f, threshold=%.2f)", normalized, threshold)
	} else {
		msg = fmt.Sprintf("distribution is skewed (normalized entropy=%.3f, threshold=%.2f) — consider stratified sampling or re-weighting", normalized, threshold)
	}

	return BiasCheckResult{
		Kind:      BiasDataBias,
		Passed:    passed,
		Score:     normalized,
		Threshold: threshold,
		Message:   msg,
	}
}

// EdgeTypeBalance is a convenience wrapper that checks balance across edge
// types in a graph. It accepts counts of edges per type (e.g. "train" vs
// "eval") or per node type (e.g. "user" vs "tool").
func EdgeTypeBalance(typeCounts map[string]int) BiasCheckResult {
	return CheckDataBalance(typeCounts, DefaultDataBalanceThreshold)
}

// ────────────────────────────────────────────────────────────────────────────
// Algorithmic Bias: Demographic Parity (fairness metric)
// ────────────────────────────────────────────────────────────────────────────

// ComputeDemographicParity checks whether prediction scores are independent of
// group membership. It compares the mean embedding magnitude (or centroid)
// across demographic groups defined by groupIndices.
//
//   - score = 1.0 - maxPairwiseDifference: 1.0 means all groups score equally
//   - passed: 1.0 - score <= threshold (i.e. max difference is small)
//
// This directly mitigates **algorithmic bias** by surfacing cases where the
// model systematically favors or disfavors certain groups.
func ComputeDemographicParity(embeddings [][]float64, groupIndices map[string][]int, threshold float64) BiasCheckResult {
	if len(embeddings) == 0 {
		return BiasCheckResult{
			Kind:      BiasAlgorithmicBias,
			Passed:    false,
			Score:     0,
			Threshold: threshold,
			Message:   "no embeddings to evaluate",
		}
	}
	if len(groupIndices) == 0 {
		return BiasCheckResult{
			Kind:      BiasAlgorithmicBias,
			Passed:    true,
			Score:     1.0,
			Threshold: threshold,
			Message:   "no groups defined — parity check skipped",
		}
	}
	if len(groupIndices) < 2 {
		return BiasCheckResult{
			Kind:      BiasAlgorithmicBias,
			Passed:    true,
			Score:     1.0,
			Threshold: threshold,
			Message:   "fewer than 2 groups — parity check skipped",
		}
	}

	// Compute mean score per group (average embedding magnitude).
	groupMeans := make(map[string]float64, len(groupIndices))
	for group, indices := range groupIndices {
		if len(indices) == 0 {
			groupMeans[group] = 0
			continue
		}
		var sum float64
		for _, idx := range indices {
			if idx < 0 || idx >= len(embeddings) {
				continue
			}
			sum += vectorMagnitude(embeddings[idx])
		}
		groupMeans[group] = sum / float64(len(indices))
	}

	// Find max pairwise difference.
	var maxDiff float64
	groups := make([]string, 0, len(groupMeans))
	for g := range groupMeans {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			diff := math.Abs(groupMeans[groups[i]] - groupMeans[groups[j]])
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}

	// Score: 1.0 = perfect parity, 0.0 = max separation.
	score := 1.0 - maxDiff
	if score < 0 {
		score = 0
	}
	passed := maxDiff <= threshold

	var msg string
	if passed {
		msg = fmt.Sprintf("demographic parity holds (max group diff=%.4f, threshold=%.2f)", maxDiff, threshold)
	} else {
		msg = fmt.Sprintf("demographic parity violation detected (max group diff=%.4f, threshold=%.2f) — groups are not scoring equally", maxDiff, threshold)
	}

	return BiasCheckResult{
		Kind:      BiasAlgorithmicBias,
		Passed:    passed,
		Score:     score,
		Threshold: threshold,
		Message:   msg,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Confirmation Bias: Diversity Score
// ────────────────────────────────────────────────────────────────────────────

// ComputeDiversityScore measures how diverse a set of recommendations is.
// It computes the average pairwise Jaccard similarity between items and
// returns a diversity score: 1 - avgSimilarity.
//
//   - score = 1.0: every recommendation is maximally different (avg Jaccard = 0)
//   - score = 0.0: all recommendations are identical (avg Jaccard = 1)
//   - passed: 1 - score <= threshold  (i.e. avg similarity is low enough)
//
// The jaccardMatrix[i][j] should be pre-computed pairwise Jaccard similarity
// between recommendation i and j (diagonal = 1.0).
//
// This directly mitigates **confirmation bias** by detecting when the system
// keeps recommending the same type of items instead of surfacing diverse options.
func ComputeDiversityScore(recommendations int, jaccardMatrix [][]float64, threshold float64) BiasCheckResult {
	if recommendations <= 1 {
		return BiasCheckResult{
			Kind:      BiasConfirmationBias,
			Passed:    true,
			Score:     1.0,
			Threshold: threshold,
			Message:   "fewer than 2 recommendations — diversity check skipped",
		}
	}

	if len(jaccardMatrix) < 2 {
		return BiasCheckResult{
			Kind:      BiasConfirmationBias,
			Passed:    true,
			Score:     1.0,
			Threshold: threshold,
			Message:   "similarity matrix too small — diversity check skipped",
		}
	}

	var totalSim float64
	var pairs int
	for i := 0; i < recommendations && i < len(jaccardMatrix); i++ {
		for j := i + 1; j < recommendations && j < len(jaccardMatrix[i]); j++ {
			totalSim += jaccardMatrix[i][j]
			pairs++
		}
	}

	if pairs == 0 {
		return BiasCheckResult{
			Kind:      BiasConfirmationBias,
			Passed:    true,
			Score:     1.0,
			Threshold: threshold,
			Message:   "no pairwise comparisons available — diversity check skipped",
		}
	}

	avgSimilarity := totalSim / float64(pairs)
	diversityScore := 1.0 - avgSimilarity
	passed := avgSimilarity <= threshold

	var msg string
	if passed {
		msg = fmt.Sprintf("recommendations are diverse (avg pairwise similarity=%.3f, threshold=%.2f)", avgSimilarity, threshold)
	} else {
		msg = fmt.Sprintf("recommendations lack diversity (avg pairwise similarity=%.3f, threshold=%.2f) — consider diversifying candidates", avgSimilarity, threshold)
	}

	return BiasCheckResult{
		Kind:      BiasConfirmationBias,
		Passed:    passed,
		Score:     diversityScore,
		Threshold: threshold,
		Message:   msg,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Availability Bias: Time-based Decay Weight
// ────────────────────────────────────────────────────────────────────────────

// DecayWeight computes an exponential decay factor based on elapsed time.
// The formula is: 0.5^(elapsed / halfLife).
//
//   - elapsed = 0 returns 1.0 (full weight for current data)
//   - elapsed = halfLife returns 0.5
//   - elapsed = 2 * halfLife returns 0.25
//
// Use in scoring and recommendation pipelines to discount older data,
// directly mitigating **availability bias** (over-weighting of recent but
// potentially less representative patterns) by applying controlled decay.
func DecayWeight(elapsed, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return 1.0 // no decay when halfLife is zero or negative
	}
	if elapsed <= 0 {
		return 1.0 // current data gets full weight
	}
	return math.Pow(0.5, float64(elapsed)/float64(halfLife))
}

// DecayedScore applies DecayWeight to a raw score and returns the
// time-discounted value.
func DecayedScore(rawScore float64, elapsed, halfLife time.Duration) float64 {
	return rawScore * DecayWeight(elapsed, halfLife)
}

// ────────────────────────────────────────────────────────────────────────────
// Composite check
// ────────────────────────────────────────────────────────────────────────────

// RunAllChecks runs every available bias check with default thresholds.
// Pass nil for optional arguments to use default thresholds.
//
// Parameters:
//   - categoryCounts: for data balance (may be nil to skip)
//   - embeddings: for demographic parity (may be nil to skip)
//   - groupIndices: for demographic parity (may be nil to skip)
//   - jaccardSim: for diversity (may be nil to skip)
//   - numRecs: number of recommendations for diversity check
func RunAllChecks(
	categoryCounts map[string]int,
	embeddings [][]float64,
	groupIndices map[string][]int,
	jaccardSim [][]float64,
	numRecs int,
) BiasReport {
	report := BiasReport{}

	// Data bias
	if categoryCounts != nil {
		report.Checks = append(report.Checks,
			CheckDataBalance(categoryCounts, DefaultDataBalanceThreshold))
	}

	// Algorithmic bias
	if embeddings != nil && groupIndices != nil {
		report.Checks = append(report.Checks,
			ComputeDemographicParity(embeddings, groupIndices, DefaultParityThreshold))
	}

	// Confirmation bias
	if jaccardSim != nil && numRecs > 1 {
		report.Checks = append(report.Checks,
			ComputeDiversityScore(numRecs, jaccardSim, DefaultDiversityThreshold))
	}

	return report
}

// ─── helpers ────────────────────────────────────────────────────────────────

// vectorMagnitude returns the Euclidean norm of a float64 vector.
func vectorMagnitude(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}
