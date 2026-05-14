package decision

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrPlanNil is returned when Reflect receives a nil plan.
var ErrPlanNil = errors.New("plan is nil")

// GapType classifies the nature of an observation's gap.
type GapType string

const (
	// GapExpected means the observation succeeded as predicted.
	GapExpected GapType = "expected"
	// GapUnexpected means the observation failed but the failure was recoverable.
	GapUnexpected GapType = "unexpected"
	// GapCritical means the observation failed in a way that blocks further progress.
	GapCritical GapType = "critical"
)

// ObservationAnalysis captures the semantic result of analyzing one observation.
type ObservationAnalysis struct {
	GapType    GapType // classification: expected, unexpected, or critical
	Details    string  // human-readable explanation of what happened
	Impact     string  // how this affects the plan going forward
	Confidence float64 // 0.0-1.0 confidence in this analysis
	StepTool   string  // tool name from the observed step
}

// ReflectionResult enrichs a PlanResult with semantic analysis.
type ReflectionResult struct {
	Plan            *PlanResult
	Analyses        []ObservationAnalysis
	Summary         string // what worked overall
	FailurePatterns []string
	CriticalCount   int
	UnexpectedCount int
	ExpectedCount   int
}

// Reflector is the interface for the Reflect phase.
type Reflector interface {
	Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error)
}

// DefaultReflector implements the Reflect phase of the decision loop.
// It analyzes observations from completed actions and determines:
//   - Whether to continue or stop
//   - Which tools to call next
//   - Whether the goal has been achieved
type DefaultReflector struct{}

// NewDefaultReflector creates a new DefaultReflector.
func NewDefaultReflector() *DefaultReflector {
	return &DefaultReflector{}
}

// Reflect analyzes the observations and produces an updated plan.
// It iterates ALL observations (not just the last one), classifies each
// as expected/unexpected/critical, and produces a structured reflection
// with actionable insights.
//
// On failures, Reflect generates CorrectionSteps and sets ReplanType:
//   - ReplanPartial: individual failed steps get fallback-based correction steps
//   - ReplanFull: the entire plan must be recreated (critical or cascading failure)
//   - ReplanNone: everything succeeded
func (r *DefaultReflector) Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error) {
	if plan == nil {
		return nil, ErrPlanNil
	}

	// If the plan already says we can't proceed, don't change it
	if !plan.CanProceed {
		return plan, nil
	}

	// No observations yet — nothing to reflect on
	if len(observations) == 0 {
		return plan, nil
	}

	// Analyze every observation
	analyses := make([]ObservationAnalysis, 0, len(observations))
	criticalCount := 0
	unexpectedCount := 0
	expectedCount := 0
	failurePatterns := make([]string, 0)
	succeededTools := make([]string, 0)

	for i, obs := range observations {
		analysis := classifyObservation(obs, i, len(observations))

		switch analysis.GapType {
		case GapCritical:
			criticalCount++
			failurePatterns = append(failurePatterns, fmt.Sprintf("%s: %s", analysis.StepTool, analysis.Details))
		case GapUnexpected:
			unexpectedCount++
			failurePatterns = append(failurePatterns, fmt.Sprintf("%s: %s", analysis.StepTool, analysis.Details))
		case GapExpected:
			expectedCount++
			if obs.Success {
				succeededTools = append(succeededTools, analysis.StepTool)
			}
		}

		analyses = append(analyses, analysis)
	}

	// Build summary
	summary := buildReflectionSummary(expectedCount, unexpectedCount, criticalCount, succeededTools)

	// Copy steps
	newSteps := make([]PlannedStep, len(plan.Steps))
	copy(newSteps, plan.Steps)

	// Determine whether we can proceed
	if criticalCount > 0 {
		// Any critical failure triggers a FULL replan
		lastCritical := ""
		for _, a := range analyses {
			if a.GapType == GapCritical {
				lastCritical = a.Details
			}
		}
		corrections := buildCorrectionSteps(observations, plan)
		return &PlanResult{
			Intent:          plan.Intent,
			Steps:           newSteps,
			CanProceed:      false,
			Reason:          fmt.Sprintf("critical failure detected: %s; %s", lastCritical, summary),
			CorrectionSteps: corrections,
			ReplanType:      ReplanFull,
		}, nil
	}

	// Only unexpected (recoverable) failures — PARTIAL replan
	if unexpectedCount > 0 {
		corrections := buildCorrectionSteps(observations, plan)
		return &PlanResult{
			Intent:          plan.Intent,
			Steps:           newSteps,
			CanProceed:      true,
			Reason:          fmt.Sprintf("completed with %d recoverable issue(s); %s", unexpectedCount, summary),
			CorrectionSteps: corrections,
			ReplanType:      ReplanPartial,
		}, nil
	}

	// All expected — everything went according to plan
	return &PlanResult{
		Intent:          plan.Intent,
		Steps:           newSteps,
		CanProceed:      true,
		Reason:          summary,
		CorrectionSteps: nil,
		ReplanType:      ReplanNone,
	}, nil
}

// buildCorrectionSteps generates alternate PlannedSteps for failed observations.
// For each failed step, it uses the step's Fallback field if populated,
// otherwise constructs a diagnostic fallback (query_dispatch with error details).
// This provides the Run() loop with actionable corrections for partial replanning.
func buildCorrectionSteps(observations []Observation, plan *PlanResult) []PlannedStep {
	var corrections []PlannedStep
	for _, obs := range observations {
		if obs.Success {
			continue
		}
		step := obs.Step

		// Use explicit fallback if the step defined one
		if step.Fallback != "" {
			corrections = append(corrections, PlannedStep{
				ToolName:  step.Fallback,
				Arguments: map[string]interface{}{"original_tool": step.ToolName, "error": strings.Join(obs.Issues, "; ")},
				Rationale: fmt.Sprintf("fallback for %s after failure: %s", step.ToolName, strings.Join(obs.Issues, "; ")),
			})
			continue
		}

		// No fallback — inject a diagnostic query_dispatch as a safe default
		corrections = append(corrections, PlannedStep{
			ToolName: "query_dispatch",
			Arguments: map[string]interface{}{
				"query": fmt.Sprintf("diagnose failure of %s: %s", step.ToolName, strings.Join(obs.Issues, "; ")),
			},
			Rationale: fmt.Sprintf("diagnostic fallback for failed tool %s", step.ToolName),
		})
	}
	return corrections
}

// classifyObservation assigns a GapType to a single observation based on
// its success/failure, issue patterns, and position in the sequence.
func classifyObservation(obs Observation, index int, total int) ObservationAnalysis {
	toolName := obs.Step.ToolName
	if toolName == "" {
		toolName = "unknown"
	}

	if obs.Success {
		// Successful execution — but might have soft issues (empty output, truncation)
		if len(obs.Issues) > 0 {
			return ObservationAnalysis{
				GapType:    GapUnexpected,
				Details:    fmt.Sprintf("tool %s succeeded but has issues: %s", toolName, strings.Join(obs.Issues, "; ")),
				Impact:     "soft issues may degrade downstream quality",
				Confidence: 0.8,
				StepTool:   toolName,
			}
		}
		return ObservationAnalysis{
			GapType:    GapExpected,
			Details:    fmt.Sprintf("tool %s completed successfully", toolName),
			Impact:     "no impact; plan proceeds as expected",
			Confidence: 0.95,
			StepTool:   toolName,
		}
	}

	// Failure — determine if critical or unexpected (recoverable)
	issueText := strings.Join(obs.Issues, "; ")

	// Heuristics for critical failures:
	// - Multiple issues suggest cascading failure
	// - Very low trust delta suggests fundamental problem
	// - Issues containing "not found", "unauthorized", "permission" suggest blocking errors
	if isBlockingFailure(issueText) || len(obs.Issues) > 2 || obs.TrustDelta < -0.3 {
		return ObservationAnalysis{
			GapType:    GapCritical,
			Details:    fmt.Sprintf("tool %s failed critically: %s", toolName, issueText),
			Impact:     "blocks further progress; requires intervention or replanning",
			Confidence: 0.85,
			StepTool:   toolName,
		}
	}

	// Recoverable failure
	return ObservationAnalysis{
		GapType:    GapUnexpected,
		Details:    fmt.Sprintf("tool %s failed but may be recoverable: %s", toolName, issueText),
		Impact:     "partial progress possible; consider retrying with modified approach",
		Confidence: 0.7,
		StepTool:   toolName,
	}
}

// isBlockingFailure checks if an issue string contains indicators of
// irrecoverable errors that block further progress.
func isBlockingFailure(issueText string) bool {
	blocking := []string{"not found", "unauthorized", "permission denied", "forbidden", "invalid", "does not exist"}
	lower := strings.ToLower(issueText)
	for _, b := range blocking {
		if strings.Contains(lower, b) {
			return true
		}
	}
	return false
}

// buildReflectionSummary creates a human-readable summary of the reflection.
func buildReflectionSummary(expectedCount, unexpectedCount, criticalCount int, succeededTools []string) string {
	parts := make([]string, 0, 4)

	if expectedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d steps completed as expected", expectedCount))
	}
	if len(succeededTools) > 0 && len(succeededTools) <= 5 {
		parts = append(parts, fmt.Sprintf("successful tools: %s", strings.Join(succeededTools, ", ")))
	}
	if unexpectedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d recoverable issues detected", unexpectedCount))
	}
	if criticalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d critical failures", criticalCount))
	}

	if len(parts) == 0 {
		return "reflection complete: no actionable observations"
	}

	return "reflection: " + strings.Join(parts, "; ")
}