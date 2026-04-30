package decision

import (
	"context"
	"strings"
	"testing"
)

func TestReflect_NilPlan(t *testing.T) {
	r := NewDefaultReflector()
	_, err := r.Reflect(context.Background(), nil, nil)
	if err != ErrPlanNil {
		t.Errorf("expected ErrPlanNil, got %v", err)
	}
}

func TestReflect_CannotProceedUnchanged(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		CanProceed: false,
		Reason:     "already blocked",
	}
	result, err := r.Reflect(context.Background(), plan, []Observation{{Success: true}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CanProceed {
		t.Error("expected CanProceed=false when plan already blocked")
	}
	if result.Reason != "already blocked" {
		t.Errorf("expected reason unchanged, got %q", result.Reason)
	}
}

func TestReflect_EmptyObservations(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		CanProceed: true,
	}
	result, err := r.Reflect(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.CanProceed {
		t.Error("expected CanProceed=true with no observations")
	}
}

func TestReflectWithMultipleObservations(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "analyze market"},
		Steps:      []PlannedStep{{ToolName: "fetch_data"}, {ToolName: "analyze"}, {ToolName: "report"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:    PlannedStep{ToolName: "fetch_data"},
			Success: true,
		},
		{
			Step:    PlannedStep{ToolName: "analyze"},
			Success: true,
		},
		{
			Step:    PlannedStep{ToolName: "report"},
			Success: true,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.CanProceed {
		t.Error("expected CanProceed=true when all observations succeed")
	}

	if !strings.Contains(result.Reason, "3 steps completed as expected") {
		t.Errorf("expected summary to mention 3 completed steps, got %q", result.Reason)
	}

	if !strings.Contains(result.Reason, "fetch_data") || !strings.Contains(result.Reason, "analyze") || !strings.Contains(result.Reason, "report") {
		t.Errorf("expected summary to list successful tools, got %q", result.Reason)
	}

	if len(result.Steps) != 3 {
		t.Errorf("expected 3 steps preserved, got %d", len(result.Steps))
	}
}

func TestReflectWithMixedResults(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "gather intelligence"},
		Steps:      []PlannedStep{{ToolName: "search_web"}, {ToolName: "enrich"}, {ToolName: "summarize"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:       PlannedStep{ToolName: "search_web"},
			Success:    true,
			Issues:     []string{"output was truncated due to context limits"},
			TrustDelta: 0,
		},
		{
			Step:       PlannedStep{ToolName: "enrich"},
			Success:    false,
			Issues:     []string{"timeout retrieving source"},
			TrustDelta: -0.1,
		},
		{
			Step:       PlannedStep{ToolName: "summarize"},
			Success:    true,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.CanProceed {
		t.Error("expected CanProceed=true with only recoverable failures")
	}

	if !strings.Contains(result.Reason, "recoverable issue") {
		t.Errorf("expected reason to mention recoverable issues, got %q", result.Reason)
	}

	if !strings.Contains(result.Reason, "1 steps completed as expected") {
		t.Errorf("expected reason to mention 1 expected step (summarize), got %q", result.Reason)
	}
}

func TestReflectWithAllFailures(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "fetch remote data"},
		Steps:      []PlannedStep{{ToolName: "api_call"}, {ToolName: "parse"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:       PlannedStep{ToolName: "api_call"},
			Success:    false,
			Issues:     []string{"unauthorized: API key invalid"},
			TrustDelta: -0.5,
		},
		{
			Step:       PlannedStep{ToolName: "parse"},
			Success:    false,
			Issues:     []string{"no input data available"},
			TrustDelta: -0.2,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed=false when critical failures present")
	}

	if !strings.Contains(result.Reason, "critical failure") {
		t.Errorf("expected reason to mention critical failure, got %q", result.Reason)
	}
}

func TestReflect_SuccessWithSoftIssues(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "query_db"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:       PlannedStep{ToolName: "query_db"},
			Success:    true,
			Issues:     []string{"tool returned empty output"},
			TrustDelta: 0,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.CanProceed {
		t.Error("expected CanProceed=true with soft issues on successful execution")
	}

	if !strings.Contains(result.Reason, "recoverable issue") {
		t.Errorf("expected reason to mention recoverable issue for success+issues, got %q", result.Reason)
	}
}

func TestReflect_TrustDeltaCriticalThreshold(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "external_api"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:       PlannedStep{ToolName: "external_api"},
			Success:    false,
			Issues:     []string{"connection refused"},
			TrustDelta: -0.35,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed=false when trust delta < -0.3")
	}
}

func TestReflect_MultipleIssuesCritical(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "pipeline"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:       PlannedStep{ToolName: "pipeline"},
			Success:    false,
			Issues:     []string{"error step 1", "error step 2", "error step 3"},
			TrustDelta: -0.1,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed=false when >2 issues (cascading failure)")
	}
}

func TestReflect_MixedCriticalAndRecoverable(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "multi-step analysis"},
		Steps:      []PlannedStep{{ToolName: "search"}, {ToolName: "enrich"}, {ToolName: "export"}},
		CanProceed: true,
	}

	observations := []Observation{
		{
			Step:    PlannedStep{ToolName: "search"},
			Success: true,
		},
		{
			Step:       PlannedStep{ToolName: "enrich"},
			Success:    false,
			Issues:     []string{"rate limit exceeded"},
			TrustDelta: -0.05,
		},
		{
			Step:       PlannedStep{ToolName: "export"},
			Success:    false,
			Issues:     []string{"permission denied: write access forbidden"},
			TrustDelta: -0.4,
		},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed=false due to critical failure (permission denied)")
	}

	if !strings.Contains(result.Reason, "critical failure") {
		t.Errorf("expected critical failure in reason, got %q", result.Reason)
	}
}

func TestReflect_BackwardsCompatibility_LastFailureStopsPlan(t *testing.T) {
	r := NewDefaultReflector()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "tool_a"}},
		CanProceed: true,
	}

	observations := []Observation{
		{Success: true},
		{Success: false, Issues: []string{"tool not found"}, TrustDelta: -0.5},
	}

	result, err := r.Reflect(context.Background(), plan, observations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed=false when last observation is a blocking failure")
	}
}

func TestClassifyObservation_UnknownTool(t *testing.T) {
	obs := Observation{
		Step:    PlannedStep{ToolName: ""},
		Success: true,
	}
	analysis := classifyObservation(obs, 0, 1)
	if analysis.StepTool != "unknown" {
		t.Errorf("expected StepTool='unknown' for empty tool name, got %q", analysis.StepTool)
	}
	if analysis.GapType != GapExpected {
		t.Errorf("expected GapExpected for successful observation, got %s", analysis.GapType)
	}
}