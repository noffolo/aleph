// Package synthesis provides a cross-document synthesis layer combining
// CodeFlow execution data, HumanEcosystems usage patterns, and OSINT security intelligence.
// Category: "synthesis", SourceType: "package"
package synthesis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ff3300/aleph-v2/internal/ethics"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
	he "github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
)

// UnifiedToolIntel combines data from CodeFlow, HumanEcosystems, and OSINT.
type UnifiedToolIntel struct {
	ToolID           string               `json:"tool_id"`
	Name             string               `json:"name"`
	Category         string               `json:"category"`
	HealthStatus     string               `json:"health_status"`
	ExecutionCount   int64                `json:"execution_count"`
	AvgDuration      time.Duration        `json:"avg_duration"`
	ErrorRate        float64              `json:"error_rate"`
	Anomalies        []codeflow.Anomaly   `json:"anomalies,omitempty"`
	UsageFrequency   int                  `json:"usage_frequency"`
	TopUsers         []string             `json:"top_users,omitempty"`
	RelatedTools     []he.Relation        `json:"related_tools,omitempty"`
	SecurityRiskScore float64             `json:"security_risk_score"`
	Warnings         []string             `json:"warnings,omitempty"`
	Recommendations  []string             `json:"recommendations,omitempty"`
}

// ToolRecommendation is a context-aware tool recommendation.
type ToolRecommendation struct {
	ToolID       string   `json:"tool_id"`
	Score        float64  `json:"score"` // 0-100 composite score
	Reason       string   `json:"reason"`
	Suggestions  []string `json:"suggestions,omitempty"`
}

// SynthesisEngine combines CodeFlow + HumanEcosystems + OSINT data sources.
type SynthesisEngine struct {
	codeFlow      *codeflow.CodeFlow
	usageTracker  *he.ToolUsageTracker
	shadowbroker  *osint.Shadowbroker
	toolIntel     *osint.ToolIntel
	logger        *slog.Logger

	// TimeDecayHalfLife controls how quickly past usage data is discounted
	// in the recommendation scoring. Default 0 = no decay. Set to e.g. 7*24*time.Hour
	// to halve the weight of data older than a week. This mitigates availability bias
	// (over-weighting of recent but potentially less representative patterns).
	TimeDecayHalfLife time.Duration

	// startupTime records when the engine was created for age-based decay.
	startupTime time.Time
}

// NewSynthesisEngine creates a new SynthesisEngine.
func NewSynthesisEngine(
	cf *codeflow.CodeFlow,
	ut *he.ToolUsageTracker,
	sb *osint.Shadowbroker,
	logger *slog.Logger,
) *SynthesisEngine {
	return &SynthesisEngine{
		codeFlow:     cf,
		usageTracker: ut,
		shadowbroker: sb,
		toolIntel:    osint.NewToolIntel(),
		logger:       logger.With("component", "synthesis"),
		startupTime:  time.Now(),
	}
}

// GetUnifiedToolIntel combines data from all three sources for a given tool.
func (se *SynthesisEngine) GetUnifiedToolIntel(ctx context.Context, toolID string) (*UnifiedToolIntel, error) {
	if toolID == "" {
		return nil, fmt.Errorf("toolID cannot be empty")
	}

	se.logger.Debug("building unified tool intel", "tool_id", toolID)

	// Gather data from each source in parallel using channels
	type result struct {
		metrics *codeflow.ExecutionMetrics
		records []codeflow.ExecutionRecord
		err     error
	}

	cfCh := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				se.logger.Error("synthesis goroutine panic", "goroutine", "codeflow", "recover", r)
				cfCh <- result{err: fmt.Errorf("codeflow goroutine panic: %v", r)}
			}
		}()
		metrics, err := se.codeFlow.GetMetrics(ctx, toolID)
		if err != nil {
			cfCh <- result{err: fmt.Errorf("get metrics: %w", err)}
			return
		}
		records, err := se.codeFlow.GetRecords(ctx, toolID)
		if err != nil {
			cfCh <- result{err: fmt.Errorf("get records: %w", err)}
			return
		}
		cfCh <- result{metrics: metrics, records: records}
	}()

	type usageResult struct {
		patterns []he.UsagePattern
		users    []string
		freq     int
		rels     map[string][]he.Relation
		err      error
	}

	heCh := make(chan usageResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				se.logger.Error("synthesis goroutine panic", "goroutine", "humanecosystems", "recover", r)
				heCh <- usageResult{err: fmt.Errorf("human ecosystems goroutine panic: %v", r)}
			}
		}()
		patterns, err := se.usageTracker.GetUsagePatterns(ctx, toolID)
		if err != nil {
			heCh <- usageResult{err: fmt.Errorf("get usage patterns: %w", err)}
			return
		}
		users, err := se.usageTracker.GetTopUsers(ctx, toolID, 5)
		if err != nil {
			heCh <- usageResult{err: fmt.Errorf("get top users: %w", err)}
			return
		}
		freq, err := se.usageTracker.GetToolFrequency(ctx, toolID)
		if err != nil {
			heCh <- usageResult{err: fmt.Errorf("get tool frequency: %w", err)}
			return
		}
		rels, err := se.usageTracker.GetRelationalContext(ctx, []string{toolID})
		if err != nil {
			heCh <- usageResult{err: fmt.Errorf("get relational context: %w", err)}
			return
		}
		heCh <- usageResult{
			patterns: patterns,
			users:    users,
			freq:     freq,
			rels:     rels,
		}
	}()

	type secResult struct {
		profile osint.ToolSecurityProfile
		err     error
	}

	secCh := make(chan secResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				se.logger.Error("synthesis goroutine panic", "goroutine", "shadowbroker", "recover", r)
				// Use select with default — channel may already be full from normal send
				select {
				case secCh <- secResult{err: fmt.Errorf("shadowbroker goroutine panic: %v", r)}:
				default:
				}
			}
		}()
		profile, err := se.shadowbroker.DiscoverToolSecurity(ctx, toolID)
		if err != nil {
			se.logger.Error("security discovery failed", "tool_id", toolID, "error", err)
			// Don't block on security failure — return empty profile
			secCh <- secResult{}
			return
		}
		secCh <- secResult{profile: profile}
	}()

	// Collect results
	cfResult := <-cfCh
	if cfResult.err != nil {
		return nil, fmt.Errorf("codeflow data: %w", cfResult.err)
	}

	heResult := <-heCh
	if heResult.err != nil {
		return nil, fmt.Errorf("human ecosystems data: %w", heResult.err)
	}

	secRes := <-secCh

	// Compute error rate
	var errorRate float64
	if cfResult.metrics.TotalCalls > 0 {
		errorRate = float64(cfResult.metrics.ErrorCount) / float64(cfResult.metrics.TotalCalls) * 100
	}

	// Detect anomalies from execution records
	allMetrics := make([]codeflow.ExecutionMetrics, len(cfResult.records))
	for i, r := range cfResult.records {
		allMetrics[i] = r.Metrics
	}
	analyzer := codeflow.NewToolAnalyzer(se.codeFlow)
	anomalies, _ := analyzer.DetectAnomalies(allMetrics)

	// Build warnings from all sources
	var warnings []string
	warnings = append(warnings, secRes.profile.Warnings...)
	for _, a := range anomalies {
		warnings = append(warnings, fmt.Sprintf("Anomaly: %s (%s) — %s", a.Type, a.Severity, a.Description))
	}

	// Build recommendations
	var recommendations []string
	recommendations = append(recommendations, secRes.profile.Recommendations...)
	if errorRate > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Error rate is %.1f%% — consider reviewing tool stability", errorRate))
	}
	if len(anomalies) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Found %d anomalies — investigate before critical impact", len(anomalies)))
	}

	intel := &UnifiedToolIntel{
		ToolID:            toolID,
		Name:              toolID,
		ExecutionCount:    cfResult.metrics.CallCount,
		AvgDuration:       cfResult.metrics.Duration,
		ErrorRate:         errorRate,
		Anomalies:         anomalies,
		UsageFrequency:    heResult.freq,
		TopUsers:          heResult.users,
		SecurityRiskScore: secRes.profile.RiskScore,
		Warnings:          warnings,
		Recommendations:   recommendations,
	}

	if rels, ok := heResult.rels[toolID]; ok {
		intel.RelatedTools = rels
	}

	return intel, nil
}

// GetCrossContextRecommendations generates context-aware tool recommendations
// based on usage patterns, security posture, and execution metrics.
func (se *SynthesisEngine) GetCrossContextRecommendations(ctx context.Context, userID string) ([]ToolRecommendation, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	se.logger.Debug("generating cross-context recommendations", "user_id", userID)

	// Collect tool IDs from usage patterns
	// We need all unique tool IDs from usage tracker
	// Since the tracker doesn't expose listing all tools, we iterate through patterns
	// by getting recommendations from relational context
	var recommendations []ToolRecommendation

	// Get all tools this user has used
	// Use execution records as source of known tools
	records, err := se.codeFlow.ListRecentExecutions(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("list recent executions: %w", err)
	}

	toolSet := make(map[string]bool)
	for _, r := range records {
		toolSet[r.ToolID] = true
	}

	for toolID := range toolSet {
		intel, err := se.GetUnifiedToolIntel(ctx, toolID)
		if err != nil {
			se.logger.Warn("skipping tool intel", "tool_id", toolID, "error", err)
			continue
		}

		// Calculate composite score (0-100)
		score := 50.0 // baseline

		// Penalize high error rate
		if intel.ErrorRate > 10 {
			score -= intel.ErrorRate * 2
		}

		// Penalize high security risk
		score -= intel.SecurityRiskScore * 0.3

		// Prefer popular tools
		if intel.UsageFrequency > 10 {
			score += 10
		}

		// Penalize anomalies
		score -= float64(len(intel.Anomalies)) * 15

		// Apply time-based decay to discount older data (availability bias
		// mitigation: older patterns get less weight than current behavior).
		if se.TimeDecayHalfLife > 0 {
			elapsed := time.Since(se.startupTime)
			score = ethics.DecayedScore(score, elapsed, se.TimeDecayHalfLife)
			if score > 100 {
				score = 100
			}
		}

		// Clamp
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		var reason string
		var suggestions []string

		if intel.ErrorRate > 10 {
			suggestions = append(suggestions,
				fmt.Sprintf("High error rate (%.1f%%) — consider fixes", intel.ErrorRate))
		}
		if intel.SecurityRiskScore > 50 {
			suggestions = append(suggestions, "Security review recommended")
		}
		if len(intel.Anomalies) > 0 {
			suggestions = append(suggestions, "Anomalies detected — investigate")
		}
		if score >= 70 {
			reason = "Well-performing tool with good metrics"
		} else if score >= 40 {
			reason = "Moderate metrics — monitor performance"
		} else {
			reason = "Below threshold — review before use"
		}

		recommendations = append(recommendations, ToolRecommendation{
			ToolID:     toolID,
			Score:      score,
			Reason:     reason,
			Suggestions: suggestions,
		})
	}

	return recommendations, nil
}
