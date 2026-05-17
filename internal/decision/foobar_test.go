package decision

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/stretchr/testify/assert"
)

// mockToolRepository implements ToolRepository with no-op methods and custom ListTools.
type mockToolRepo2 struct {
	tools     []ToolDef
	listError error
}

func (m *mockToolRepo2) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	return nil
}
func (m *mockToolRepo2) GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockToolRepo2) ListTools(ctx context.Context) ([]ToolDef, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.tools, nil
}

func TestEngine_Plan_WithProvider(t *testing.T) {
	mockProv := &mockProvider{
		completeFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
			return &llm.CompletionResponse{
				ToolCalls: []llm.ToolCall{
					{Name: "search_data", Arguments: map[string]interface{}{"object_name": "users"}},
				},
			}, nil
		},
	}
	engine := NewEngine(EngineConfig{
		Provider: mockProv,
	})
	ctx := context.Background()
	result, err := engine.Plan(ctx, "find user data", "proj1", "agent1", nil, nil)
	assert.NoError(t, err)
	assert.True(t, result.CanProceed)
	assert.Greater(t, result.Intent.Confidence, 0.5)
	assert.NotEmpty(t, result.Steps)
}

func TestEngine_Plan_WithProviderAndMetaRepo(t *testing.T) {
	repo := &mockToolRepo2{tools: []ToolDef{{Name: "custom_search", Description: "custom"}}}
	mockProv := &mockProvider{
		completeFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
			return &llm.CompletionResponse{}, nil
		},
	}
	engine := NewEngine(EngineConfig{
		Provider: mockProv,
		MetaRepo: repo,
	})
	ctx := context.Background()
	result, err := engine.Plan(ctx, "search for custom data", "proj1", "agent1", nil, nil)
	assert.NoError(t, err)
	assert.True(t, result.CanProceed)
}

func TestEngine_Plan_WithProviderAndOntology(t *testing.T) {
	mockProv := &mockProvider{
		completeFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
			return &llm.CompletionResponse{}, nil
		},
	}
	engine := NewEngine(EngineConfig{Provider: mockProv})
	ctx := context.Background()
	ontContent := []byte(`object User { name string }`)
	result, err := engine.Plan(ctx, "find User data", "proj1", "agent1", ontContent, nil)
	assert.NoError(t, err)
	assert.True(t, result.CanProceed)
	assert.Contains(t, result.Intent.TargetObjects, "User")
}

func TestEngine_Observe_FailureWithNoHistory(t *testing.T) {
	e := NewEngine(EngineConfig{})
	step := PlannedStep{ToolName: "new_tool"}
	result := &ActResult{Step: step, Error: "execution error", Output: ""}
	obs, err := e.Observe(context.Background(), step, result)
	assert.NoError(t, err)
	assert.False(t, obs.Success)
	assert.Equal(t, -0.1, obs.TrustDelta)
}

func TestEngine_Reflect_UnexpectedOnly(t *testing.T) {
	e := NewEngine(EngineConfig{})
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "tool1", Fallback: "fallback_tool"}},
		CanProceed: true,
	}
	observations := []Observation{
		{Success: true, Issues: []string{"soft issue"}},
	}
	result, err := e.Reflect(context.Background(), plan, observations)
	assert.NoError(t, err)
	assert.True(t, result.CanProceed)
	assert.Equal(t, ReplanPartial, result.ReplanType)
}

func TestEngine_Reflect_AllExpected(t *testing.T) {
	e := NewEngine(EngineConfig{})
	plan := &PlanResult{
		Intent:     Intent{PrimaryGoal: "test"},
		Steps:      []PlannedStep{{ToolName: "tool1"}},
		CanProceed: true,
	}
	observations := []Observation{
		{Success: true},
		{Success: true},
	}
	result, err := e.Reflect(context.Background(), plan, observations)
	assert.NoError(t, err)
	assert.True(t, result.CanProceed)
	assert.Equal(t, ReplanNone, result.ReplanType)
}

func TestEngine_Observe_LongOutputTruncated(t *testing.T) {
	longOutput := ""
	for i := 0; i < 2000; i++ {
		longOutput += "x"
	}
	e := NewEngine(EngineConfig{TruncationThreshold: 1900})
	step := PlannedStep{ToolName: "big_output_tool"}
	result := &ActResult{Step: step, Output: longOutput, Error: ""}
	obs, err := e.Observe(context.Background(), step, result)
	assert.NoError(t, err)
	hasTrunc := false
	for _, issue := range obs.Issues {
		if len(issue) > 0 {
			hasTrunc = true
		}
	}
	assert.True(t, hasTrunc)
}
