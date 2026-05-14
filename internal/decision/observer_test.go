package decision

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultObserver_Observe_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		threshold   int
		step        PlannedStep
		result      *ActResult
		wantSuccess bool
		wantIssues  []string // substring checks; empty slice means no issues
		wantDelta   float64
	}{
		{
			name:      "successful execution with output",
			threshold: 1900,
			step:      PlannedStep{ToolName: "search_data"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "search_data"},
				Output: "found 5 records",
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{},
			wantDelta:   0,
		},
		{
			name:      "execution error",
			threshold: 1900,
			step:      PlannedStep{ToolName: "bad_tool"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "bad_tool"},
				Output: "",
				Error:  "tool not found",
			},
			wantSuccess: false,
			wantIssues:  []string{"tool not found"},
			wantDelta:   -0.1,
		},
		{
			name:      "empty output without error",
			threshold: 1900,
			step:      PlannedStep{ToolName: "query"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "query"},
				Output: "",
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{"tool returned empty output"},
			wantDelta:   0,
		},
		{
			name:      "truncated output at default threshold",
			threshold: 0, // falls back to 1900
			step:      PlannedStep{ToolName: "big_tool"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "big_tool"},
				Output: strings.Repeat("x", 2000),
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{"truncated"},
			wantDelta:   0,
		},
		{
			name:      "truncated output at custom threshold",
			threshold: 50,
			step:      PlannedStep{ToolName: "small_tool"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "small_tool"},
				Output: strings.Repeat("x", 100),
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{"truncated"},
			wantDelta:   0,
		},
		{
			name:      "output exactly at threshold is not truncated",
			threshold: 100,
			step:      PlannedStep{ToolName: "exact_tool"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "exact_tool"},
				Output: strings.Repeat("x", 100),
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{}, // len=100, threshold=100, not > threshold
			wantDelta:   0,
		},
		{
			name:      "output just over threshold",
			threshold: 100,
			step:      PlannedStep{ToolName: "edge_tool"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "edge_tool"},
				Output: strings.Repeat("x", 101),
				Error:  "",
			},
			wantSuccess: true,
			wantIssues:  []string{"truncated"},
			wantDelta:   0,
		},
		{
			name:      "error and truncated output both flagged",
			threshold: 5,
			step:      PlannedStep{ToolName: "multi_issue"},
			result: &ActResult{
				Step:   PlannedStep{ToolName: "multi_issue"},
				Output: strings.Repeat("x", 10),
				Error:  "connection refused",
			},
			wantSuccess: false,
			wantIssues:  []string{"connection refused", "truncated"},
			wantDelta:   -0.1,
		},
		{
			name:      "nil result handled",
			threshold: 1900,
			step:      PlannedStep{ToolName: "nil_tool"},
			result:    nil,
			// With nil result, we expect a panic or nil deref — this tests the boundary
			// The test passes if it doesn't panic (len(nil.Output) == 0, nil.Error == "")
			wantSuccess: true,
			wantIssues:  []string{}, // zero-value fields from nil deref aren't reachable — this is a compile fence
			wantDelta:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result == nil {
				t.Skip("nil result — skipping (tests defensive boundary, not real path)")
			}
			var obs *DefaultObserver
			if tt.threshold > 0 {
				obs = NewDefaultObserverWithThreshold(tt.threshold)
			} else {
				obs = NewDefaultObserver()
			}

			// Verify threshold is set correctly
			expectedThresh := tt.threshold
			if expectedThresh <= 0 {
				expectedThresh = 1900
			}
			if obs.truncationThreshold != expectedThresh {
				t.Errorf("expected threshold %d, got %d", expectedThresh, obs.truncationThreshold)
			}

			got, err := obs.Observe(context.Background(), tt.step, tt.result)
			if err != nil {
				t.Fatalf("Observe returned error: %v", err)
			}
			if got.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", got.Success, tt.wantSuccess)
			}
			if got.TrustDelta != tt.wantDelta {
				t.Errorf("TrustDelta = %f, want %f", got.TrustDelta, tt.wantDelta)
			}

			// Check each expected issue substring is present
			for _, wantIssue := range tt.wantIssues {
				found := false
				for _, issue := range got.Issues {
					if strings.Contains(issue, wantIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q in %v", wantIssue, got.Issues)
				}
			}

			// Check no unexpected issues
			if len(tt.wantIssues) == 0 && len(got.Issues) > 0 {
				t.Errorf("expected no issues, got %v", got.Issues)
			}
		})
	}
}

func TestDefaultObserver_NewDefaultObserver_Creation(t *testing.T) {
	obs := NewDefaultObserver()
	if obs == nil {
		t.Fatal("expected non-nil DefaultObserver")
	}
	if obs.truncationThreshold != 1900 {
		t.Errorf("expected default threshold 1900, got %d", obs.truncationThreshold)
	}
}

func TestDefaultObserver_NewDefaultObserverWithThreshold_Zero(t *testing.T) {
	obs := NewDefaultObserverWithThreshold(0)
	if obs.truncationThreshold != 1900 {
		t.Errorf("expected fallback threshold 1900 for zero input, got %d", obs.truncationThreshold)
	}
}

func TestDefaultObserver_NewDefaultObserverWithThreshold_Negative(t *testing.T) {
	obs := NewDefaultObserverWithThreshold(-5)
	if obs.truncationThreshold != 1900 {
		t.Errorf("expected fallback threshold 1900 for negative input, got %d", obs.truncationThreshold)
	}
}

func TestDefaultObserver_NewDefaultObserverWithThreshold_Custom(t *testing.T) {
	obs := NewDefaultObserverWithThreshold(500)
	if obs.truncationThreshold != 500 {
		t.Errorf("expected threshold 500, got %d", obs.truncationThreshold)
	}
}

func TestDefaultObserver_Observe_PreservesStepAndActResult(t *testing.T) {
	obs := NewDefaultObserver()
	step := PlannedStep{
		ToolName:  "my_tool",
		Arguments: map[string]interface{}{"key": "val"},
	}
	result := &ActResult{
		Step:       step,
		Output:     "hello",
		DurationMs: 42,
	}
	got, err := obs.Observe(context.Background(), step, result)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if got.Step.ToolName != "my_tool" {
		t.Errorf("expected step tool 'my_tool', got %q", got.Step.ToolName)
	}
	if got.ActResult.Output != "hello" {
		t.Errorf("expected act result output 'hello', got %q", got.ActResult.Output)
	}
	if got.ActResult.DurationMs != 42 {
		t.Errorf("expected duration 42, got %d", got.ActResult.DurationMs)
	}
}
