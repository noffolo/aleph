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

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID string, agentID string) (string, bool, error) {
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

func (m *mockLinkPredictor) PredictLinks(ctx context.Context, graph interface{}, entityID string) ([]float64, error) {
	if m.predErr != nil {
		return nil, m.predErr
	}
	return m.linkScores, nil
}

func (m *mockLinkPredictor) TrainFromGraph(ctx context.Context, graph interface{}, epochs int) error {
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
					{Name: "search_data", Arguments: map[string]interface{}{"query": "test"}},
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
		Arguments: map[string]interface{}{"key": "value"},
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
		Arguments: map[string]interface{}{},
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
	observations := []Observation{
		{Success: true},
		{Success: false, Issues: []string{"tool error"}},
	}
	result, err := engine.Reflect(ctx, plan, observations)
	if err != nil {
		t.Fatalf("Reflect returned error: %v", err)
	}
	if result.CanProceed {
		t.Error("expected CanProceed=false when last observation failed")
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
		if fn, ok := tool["function"].(map[string]interface{}); ok {
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