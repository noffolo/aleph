package decision

import (
	"context"
	"errors"
	"testing"

	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/llm"
)

type mockProvider struct {
	completeFunc func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error)
}

func (m *mockProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &llm.CompletionResponse{}, nil
}

// mockToolRepository is a mock ToolRepository for testing.
type mockToolRepository struct {
	tools []ToolDef
}

func (m *mockToolRepository) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	return nil
}

func (m *mockToolRepository) GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	return nil, nil
}

func (m *mockToolRepository) ListTools(ctx context.Context) ([]ToolDef, error) {
	return m.tools, nil
}

// mockToolExecutor is a mock ToolExecutor for testing.
type mockToolExecutor struct {
	result string
	err    error
}

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]any, projectID string, agentID string) (string, bool, error) {
	return m.result, false, m.err
}

// mockPluginRegistry is a mock PluginRegistry for testing.
type mockPluginRegistry struct {
	components map[string]*ComponentMetadata
}

func (m *mockPluginRegistry) GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error) {
	if comp, ok := m.components[id]; ok {
		return comp, nil
	}
	return nil, errors.New("not found")
}

// mockLinkPredictor is a mock LinkPredictor for testing.
type mockLinkPredictor struct {
	trained    bool
	predErr    error
	linkScores []float64
}

func (m *mockLinkPredictor) PredictLinks(ctx context.Context, graph any, entityID string) ([]float64, error) {
	if m.predErr != nil {
		return nil, m.predErr
	}
	return m.linkScores, nil
}

func (m *mockLinkPredictor) TrainFromGraph(ctx context.Context, graph any, epochs int) error {
	m.trained = true
	return nil
}

func (m *mockLinkPredictor) IsTrained() bool {
	return m.trained
}

func TestEngine_Plan_DegradedMode(t *testing.T) {
	engine := NewEngine(EngineConfig{
		Provider: nil, // nil provider = degraded mode
	})
	ctx := context.Background()
	result, err := engine.Plan(ctx, "search for data", "proj1", "agent1", nil, nil)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if result.Reason != "degraded mode: heuristic planning (no LLM provider)" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
	if !result.CanProceed {
		t.Error("expected CanProceed=true")
	}
}

func TestEngine_PlanWithProvider_NilProvider(t *testing.T) {
	engine := NewEngine(EngineConfig{
		Provider: nil,
	})
	ctx := context.Background()
	result, err := engine.PlanWithProvider(ctx, "test query", "proj1", "agent1", nil, nil, nil)
	if err != nil {
		t.Fatalf("PlanWithProvider returned error: %v", err)
	}
	if result.Reason != "degraded mode: heuristic planning (no LLM provider)" {
		t.Errorf("expected degraded mode reason, got: %s", result.Reason)
	}
}

func TestEngine_PlanWithProvider_Success(t *testing.T) {
	mockProv := &mockProvider{
		completeFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
			return &llm.CompletionResponse{
				ToolCalls: []llm.ToolCall{
					{Name: "search_data", Arguments: map[string]any{"query": "test"}},
				},
			}, nil
		},
	}
	engine := NewEngine(EngineConfig{
		Provider: mockProv,
	})
	ctx := context.Background()
	agent := &alephv1.Agent{
		Model:  "test-model",
		ApiKey: "test-key",
	}
	result, err := engine.PlanWithProvider(ctx, "find data", "proj1", "agent1", nil, agent, mockProv)
	if err != nil {
		t.Fatalf("PlanWithProvider returned error: %v", err)
	}
	if !result.CanProceed {
		t.Error("expected CanProceed=true")
	}
	if result.Reason != "planned via LLM" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].ToolName != "search_data" {
		t.Errorf("expected tool search_data, got %s", result.Steps[0].ToolName)
	}
}

func TestEngine_Act_DelegatesToExecutor(t *testing.T) {
	executor := &mockToolExecutor{result: "success", err: nil}
	engine := NewEngine(EngineConfig{
		Executor: executor,
	})
	ctx := context.Background()
	step := PlannedStep{
		ToolName:  "test_tool",
		Arguments: map[string]any{"key": "value"},
	}
	result, err := engine.Act(ctx, step, "proj1")
	if err != nil {
		t.Fatalf("Act returned error: %v", err)
	}
	if result.Output != "success" {
		t.Errorf("expected output 'success', got %q", result.Output)
	}
	if result.Error != "" {
		t.Errorf("expected no error in result, got %s", result.Error)
	}
}

func TestEngine_Act_ExecutorError(t *testing.T) {
	executor := &mockToolExecutor{result: "", err: errors.New("tool failed")}
	engine := NewEngine(EngineConfig{
		Executor: executor,
	})
	ctx := context.Background()
	step := PlannedStep{
		ToolName:  "failing_tool",
		Arguments: map[string]any{},
	}
	result, err := engine.Act(ctx, step, "proj1")
	if err != nil {
		t.Fatalf("Act returned unexpected error: %v", err)
	}
	if result.Error != "tool failed" {
		t.Errorf("expected error 'tool failed', got %q", result.Error)
	}
	if result.Output != "" {
		t.Errorf("expected empty output, got %q", result.Output)
	}
}

func TestEngine_Observe_Success(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	ctx := context.Background()
	step := PlannedStep{ToolName: "test_tool"}
	result := &ActResult{
		Step:   step,
		Output: "some output",
		Error:  "",
	}
	obs, err := engine.Observe(ctx, step, result)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.Success {
		t.Error("expected Success=true")
	}
	if len(obs.Issues) != 0 {
		t.Errorf("expected no issues, got %v", obs.Issues)
	}
	// No history yet: TrustDelta = 0.05 (small positive for first unknown tool success)
	if obs.TrustDelta != 0.05 {
		t.Errorf("expected TrustDelta=0.05 for first successful execution, got %f", obs.TrustDelta)
	}

	// Pre-populate history so the next Observe computes real TrustDelta
	engine.toolHistory["test_tool"] = &toolExecutionHistory{successes: 2, failures: 0}

	// After history: rate = 2/2 = 1.0, delta = 1.0 - 0.5 = 0.5
	obs2, err := engine.Observe(ctx, step, result)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs2.TrustDelta != 0.5 {
		t.Errorf("expected TrustDelta=0.5 after 2/2 successes, got %f", obs2.TrustDelta)
	}
}

func TestEngine_Observe_Failure(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	ctx := context.Background()
	step := PlannedStep{ToolName: "test_tool"}
	result := &ActResult{
		Step:   step,
		Output: "",
		Error:  "execution failed",
	}
	obs, err := engine.Observe(ctx, step, result)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.Success {
		t.Error("expected Success=false")
	}
	if len(obs.Issues) != 1 || obs.Issues[0] != "execution failed" {
		t.Errorf("expected issue 'execution failed', got %v", obs.Issues)
	}
}

func TestEngine_Observe_EmptyOutput(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	ctx := context.Background()
	step := PlannedStep{ToolName: "test_tool"}
	result := &ActResult{
		Step:   step,
		Output: "",
		Error:  "", // no error but empty output
	}
	obs, err := engine.Observe(ctx, step, result)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	var foundEmptyIssue bool
	for _, issue := range obs.Issues {
		if issue == "tool returned empty output" {
			foundEmptyIssue = true
			break
		}
	}
	if !foundEmptyIssue {
		t.Error("expected 'tool returned empty output' in issues")
	}
}

func TestEngine_Reflect_AllSuccess(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	ctx := context.Background()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "test_tool"}},
		CanProceed: true,
	}
	observations := []Observation{
		{Success: true},
		{Success: true},
	}
	result, err := engine.Reflect(ctx, plan, observations)
	if err != nil {
		t.Fatalf("Reflect returned error: %v", err)
	}
	if !result.CanProceed {
		t.Error("expected CanProceed=true when all observations succeed")
	}
}

func TestEngine_Reflect_LastFailure(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	ctx := context.Background()
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "test_tool"}},
		CanProceed: true,
	}
	// Use a blocking/critical failure so DefaultReflector stops the plan
	observations := []Observation{
		{Success: true},
		{Success: false, Issues: []string{"unauthorized: access denied"}},
	}
	result, err := engine.Reflect(ctx, plan, observations)
	if err != nil {
		t.Fatalf("Reflect returned error: %v", err)
	}
	if result.CanProceed {
		t.Error("expected CanProceed=false when critical failure is detected")
	}
}

func TestEngine_Admit_EnoughResults(t *testing.T) {
	engine := NewEngine(EngineConfig{MaxAttempts: 3})
	ctx := context.Background()
	results := []*ActResult{
		{Output: "result1"},
		{Output: "result2"},
		{Output: "result3"},
	}
	admit, err := engine.Admit(ctx, results, 3)
	if err != nil {
		t.Fatalf("Admit returned error: %v", err)
	}
	if !admit {
		t.Error("expected Admit=true with len(results) >= maxAttempts")
	}
}

func TestEngine_Admit_LastError(t *testing.T) {
	engine := NewEngine(EngineConfig{MaxAttempts: 5})
	ctx := context.Background()
	results := []*ActResult{
		{Output: "result1"},
		{Output: "result2"},
		{Error: "some error"},
	}
	admit, err := engine.Admit(ctx, results, 5)
	if err != nil {
		t.Fatalf("Admit returned error: %v", err)
	}
	if !admit {
		t.Error("expected Admit=true when last result has error")
	}
}

func TestEngine_SortStepsByDependencies(t *testing.T) {
	tests := []struct {
		name  string
		steps []PlannedStep
		check func(t *testing.T, sorted []PlannedStep)
	}{
		{
			name:  "empty steps",
			steps: []PlannedStep{},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 0 {
					t.Errorf("expected 0 steps, got %d", len(sorted))
				}
			},
		},
		{
			name: "single step",
			steps: []PlannedStep{
				{ToolName: "search_data"},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 1 || sorted[0].ToolName != "search_data" {
					t.Error("single step should be preserved")
				}
			},
		},
		{
			name: "simple dependency order",
			steps: []PlannedStep{
				{ToolName: "step_b", Depends: []string{"step_a"}},
				{ToolName: "step_a"},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 2 {
					t.Fatalf("expected 2 steps, got %d", len(sorted))
				}
				if sorted[0].ToolName != "step_a" {
					t.Errorf("expected step_a first, got %s", sorted[0].ToolName)
				}
				if sorted[1].ToolName != "step_b" {
					t.Errorf("expected step_b second, got %s", sorted[1].ToolName)
				}
			},
		},
		{
			name: "chain of three dependencies",
			steps: []PlannedStep{
				{ToolName: "step_c", Depends: []string{"step_b"}},
				{ToolName: "step_b", Depends: []string{"step_a"}},
				{ToolName: "step_a"},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 3 {
					t.Fatalf("expected 3 steps, got %d", len(sorted))
				}
				if sorted[0].ToolName != "step_a" {
					t.Errorf("expected step_a first, got %s", sorted[0].ToolName)
				}
				if sorted[2].ToolName != "step_c" {
					t.Errorf("expected step_c last, got %s", sorted[2].ToolName)
				}
			},
		},
		{
			name: "unknown dependency tolerated",
			steps: []PlannedStep{
				{ToolName: "step_a", Depends: []string{"nonexistent"}},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 1 {
					t.Fatalf("expected 1 step, got %d", len(sorted))
				}
			},
		},
		{
			name: "circular dependency handled",
			steps: []PlannedStep{
				{ToolName: "step_a", Depends: []string{"step_b"}},
				{ToolName: "step_b", Depends: []string{"step_a"}},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				// Both steps should appear (cycle broken, no infinite loop)
				if len(sorted) != 2 {
					t.Fatalf("expected 2 steps, got %d", len(sorted))
				}
			},
		},
		{
			name: "no dependencies preserves order",
			steps: []PlannedStep{
				{ToolName: "step_a"},
				{ToolName: "step_b"},
				{ToolName: "step_c"},
			},
			check: func(t *testing.T, sorted []PlannedStep) {
				if len(sorted) != 3 {
					t.Fatalf("expected 3 steps, got %d", len(sorted))
				}
				if sorted[0].ToolName != "step_a" || sorted[1].ToolName != "step_b" || sorted[2].ToolName != "step_c" {
					t.Error("order should be preserved for independent steps")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortStepsByDependencies(tt.steps)
			tt.check(t, sorted)
		})
	}
}

func TestEngine_FailedStepNames(t *testing.T) {
	results := []*ActResult{
		{Step: PlannedStep{ToolName: "tool_a"}, Output: "success"},
		{Step: PlannedStep{ToolName: "tool_b"}, Error: "failed"},
		{Step: PlannedStep{ToolName: "tool_c"}, Output: "ok"},
		{Step: PlannedStep{ToolName: "tool_d"}, Error: "errored"},
	}
	failed := FailedStepNames(results)
	if !failed["tool_b"] {
		t.Error("expected tool_b in failed set")
	}
	if !failed["tool_d"] {
		t.Error("expected tool_d in failed set")
	}
	if failed["tool_a"] {
		t.Error("did not expect tool_a in failed set")
	}
	if failed["tool_c"] {
		t.Error("did not expect tool_c in failed set")
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed tools, got %d", len(failed))
	}
}

func TestEngine_ShouldSkipStep(t *testing.T) {
	failed := map[string]bool{"step_a": true}

	if !ShouldSkipStep(PlannedStep{ToolName: "step_b", Depends: []string{"step_a"}}, failed) {
		t.Error("should skip when dependency failed")
	}
	if ShouldSkipStep(PlannedStep{ToolName: "step_c", Depends: []string{"step_x"}}, failed) {
		t.Error("should NOT skip when dependency not in failed set")
	}
	if ShouldSkipStep(PlannedStep{ToolName: "step_d"}, failed) {
		t.Error("should NOT skip when no dependencies")
	}
}

func TestEngine_ShouldAutoSkip(t *testing.T) {
	tests := []struct {
		name       string
		threshold  float64
		step       PlannedStep
		expectSkip bool
	}{
		{
			name:       "threshold zero no skip",
			threshold:  0,
			step:       PlannedStep{ToolName: "t", RequiresConfirmation: true},
			expectSkip: false,
		},
		{
			name:       "threshold active with confirmation",
			threshold:  0.5,
			step:       PlannedStep{ToolName: "t", RequiresConfirmation: true},
			expectSkip: true,
		},
		{
			name:       "threshold active without confirmation",
			threshold:  0.5,
			step:       PlannedStep{ToolName: "t", RequiresConfirmation: false},
			expectSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(EngineConfig{ConfirmationThreshold: tt.threshold})
			got := e.ShouldAutoSkip(tt.step)
			if got != tt.expectSkip {
				t.Errorf("ShouldAutoSkip = %v, want %v", got, tt.expectSkip)
			}
		})
	}
}

func TestEngine_ConfirmationThresholdClamping(t *testing.T) {
	e := NewEngine(EngineConfig{ConfirmationThreshold: 2.5})
	if e.confirmationThreshold != 1.0 {
		t.Errorf("expected threshold clamped to 1.0, got %f", e.confirmationThreshold)
	}
	e2 := NewEngine(EngineConfig{ConfirmationThreshold: -1})
	if e2.confirmationThreshold != 0 {
		t.Errorf("expected threshold clamped to 0, got %f", e2.confirmationThreshold)
	}
}

func TestEngine_MultiStepPlan_WithDependencies(t *testing.T) {
	// Engine with a working executor that simulates multi-step execution
	stepResults := map[string]string{
		"search_data":       "found 10 records",
		"analyze_sentiment": "positive sentiment detected",
	}
	executor := &mockToolExecutor{
		result: "",
		err:    nil,
	}
	engine := NewEngine(EngineConfig{
		Executor: executor,
	})

	ctx := context.Background()
	plan := &PlanResult{
		Intent: Intent{PrimaryGoal: "analyze data"},
		Steps: []PlannedStep{
			{ToolName: "search_data"},
			{ToolName: "analyze_sentiment", Depends: []string{"search_data"}},
		},
		CanProceed: true,
	}

	// Simulate multi-step execution in dependency order
	sorted := SortStepsByDependencies(plan.Steps)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 sorted steps, got %d", len(sorted))
	}
	if sorted[0].ToolName != "search_data" {
		t.Errorf("expected search_data first, got %s", sorted[0].ToolName)
	}
	if sorted[1].ToolName != "analyze_sentiment" {
		t.Errorf("expected analyze_sentiment second, got %s", sorted[1].ToolName)
	}

	// Execute steps in order
	var results []*ActResult
	var failedDeps = make(map[string]bool)
	for _, step := range sorted {
		if ShouldSkipStep(step, failedDeps) {
			continue
		}
		// Set mock result for each step
		executor.result = stepResults[step.ToolName]
		executor.err = nil
		result, err := engine.Act(ctx, step, "proj1")
		if err != nil {
			t.Fatalf("Act failed: %v", err)
		}
		results = append(results, result)
		if result.Error != "" {
			failedDeps[step.ToolName] = true
		}
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Output != "found 10 records" {
		t.Errorf("expected 'found 10 records', got %q", results[0].Output)
	}
	if results[1].Output != "positive sentiment detected" {
		t.Errorf("expected 'positive sentiment detected', got %q", results[1].Output)
	}
}

func TestEngine_MultiStepPlan_FailedDependencySkips(t *testing.T) {
	executor := &mockToolExecutor{
		result: "",
		err:    nil,
	}
	engine := NewEngine(EngineConfig{
		Executor: executor,
	})

	ctx := context.Background()
	steps := []PlannedStep{
		{ToolName: "step_a"},
		{ToolName: "step_b", Depends: []string{"step_a"}},
		{ToolName: "step_c", Depends: []string{"step_b"}},
	}

	executor.result = ""
	executor.err = nil

	var results []*ActResult
	failedDeps := make(map[string]bool)

	for _, step := range steps {
		if ShouldSkipStep(step, failedDeps) {
			results = append(results, &ActResult{
				Step:   step,
				Output: "SKIPPED: dependency failed",
			})
			continue
		}
		result, _ := engine.Act(ctx, step, "proj1")
		results = append(results, result)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Now simulate: step_a fails. step_b should be skipped (depends on step_a).
	// step_c depends on step_b which was skipped (not failed), so step_c still runs.
	failedDeps["step_a"] = true
	var filteredResults []*ActResult
	for _, step := range steps {
		if ShouldSkipStep(step, failedDeps) {
			filteredResults = append(filteredResults, &ActResult{
				Step:   step,
				Output: "SKIPPED: dependency failed",
			})
			continue
		}
		executor.result = "ok"
		r, _ := engine.Act(ctx, step, "proj1")
		filteredResults = append(filteredResults, r)
	}

	if len(filteredResults) != 3 {
		t.Fatalf("expected 3 filtered results, got %d", len(filteredResults))
	}
	// step_a ran (but we failed it), step_b skipped (failed dep), step_c ran (dep step_b wasn't failed, it was skipped)
	if filteredResults[1].Output != "SKIPPED: dependency failed" {
		t.Errorf("step_b should be skipped, got %q", filteredResults[1].Output)
	}
	if filteredResults[2].Output != "ok" {
		t.Errorf("step_c should run (its dep step_b wasn't failed, just skipped), got %q", filteredResults[2].Output)
	}
}

func TestEngine_isKnownTool(t *testing.T) {
	// Engine with nil registry — only built-in tools are known
	engineNoReg := NewEngine(EngineConfig{})
	ctx := context.Background()

	tests := []struct {
		name     string
		engine   *Engine
		toolName string
		want     bool
	}{
		// Built-in tools always return true
		{name: "builtin search_data", engine: engineNoReg, toolName: "search_data", want: true},
		{name: "builtin analyze_sentiment", engine: engineNoReg, toolName: "analyze_sentiment", want: true},
		{name: "builtin get_trust_score", engine: engineNoReg, toolName: "get_trust_score", want: true},
		// Unknown with nil registry → false
		{name: "unknown tool nil registry", engine: engineNoReg, toolName: "unknown_tool", want: false},
		{name: "empty string nil registry", engine: engineNoReg, toolName: "", want: false},
	}

	// Test with nil registry
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.engine.isKnownTool(ctx, tt.toolName)
			if got != tt.want {
				t.Errorf("isKnownTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}

	// Test with registry containing a registered tool
	registeredTool := "my_registered_tool"
	reg := &mockPluginRegistry{
		components: map[string]*ComponentMetadata{
			registeredTool: {ID: registeredTool, Name: "My Registered Tool"},
		},
	}
	engineWithReg := NewEngine(EngineConfig{
		Registry: reg,
	})

	t.Run("registered tool returns true", func(t *testing.T) {
		got := engineWithReg.isKnownTool(ctx, registeredTool)
		if !got {
			t.Errorf("isKnownTool(%q) = false, want true (tool is registered)", registeredTool)
		}
	})

	t.Run("unregistered tool returns false", func(t *testing.T) {
		got := engineWithReg.isKnownTool(ctx, "unregistered_tool")
		if got {
			t.Error("isKnownTool(unregistered_tool) = true, want false")
		}
	})
}

func TestEngine_inferToolsFromMessage(t *testing.T) {
	engine := NewEngine(EngineConfig{})

	tests := []struct {
		name    string
		message string
		want    []string
	}{
		{"search keywords", "search for data about users", []string{"search_data"}},
		{"find keyword", "find the object", []string{"search_data"}},
		{"sentiment keyword", "analyze sentiment of this text", []string{"analyze_sentiment"}},
		{"trust keyword", "get trust score for prediction", []string{"get_trust_score"}},
		{"multiple keywords", "search data and analyze sentiment and get trust score", []string{"search_data", "analyze_sentiment", "get_trust_score"}},
		{"no matching keywords", "hello world", nil},
		{"empty message", "", nil},
		{"data keyword alone", "show me the data", []string{"search_data"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.inferToolsFromMessage(context.Background(), tt.message, nil)
			if len(got) != len(tt.want) {
				t.Errorf("inferToolsFromMessage(%q) = %v, want %v", tt.message, got, tt.want)
				return
			}
			for i, tool := range tt.want {
				if i >= len(got) || got[i] != tool {
					t.Errorf("inferToolsFromMessage(%q)[%d] = %q, want %q", tt.message, i, got[i], tool)
				}
			}
		})
	}
}

func TestEngine_BuildToolsMap(t *testing.T) {
	repo := &mockToolRepository{
		tools: []ToolDef{
			{Name: "custom_tool", Description: "A custom tool"},
		},
	}
	engine := NewEngine(EngineConfig{
		MetaRepo: repo,
	})
	ctx := context.Background()
	tools := engine.BuildToolsMap(ctx)
	if len(tools) < 3 {
		t.Errorf("expected at least 3 built-in tools, got %d", len(tools))
	}
	// Check that custom tool is included
	var foundCustom bool
	for _, tool := range tools {
		if fn, ok := tool["function"].(map[string]any); ok {
			if fn["name"] == "custom_tool" {
				foundCustom = true
				break
			}
		}
	}
	if !foundCustom {
		t.Error("expected custom_tool in BuildToolsMap result")
	}
}
