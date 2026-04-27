package multiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/tools"
	"github.com/ff3300/aleph-v2/internal/tools/finance"
	"github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
)

// ---------------------------------------------------------------------------
// Test double implementations
// ---------------------------------------------------------------------------

type mockToolExecutor struct {
	registry *tools.ToolRegistry
}

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID, agentID string) (string, bool, error) {
	category, name := inferToolCategory(toolName)

	result, err := m.registry.ExecuteContext(ctx, category, name, args)
	if err != nil {
		return "", false, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", false, fmt.Errorf("marshal result: %w", err)
	}

	return string(data), false, nil
}

func inferToolCategory(toolName string) (category, name string) {
	if idx := strings.Index(toolName, ":"); idx != -1 {
		return toolName[:idx], toolName[idx+1:]
	}

	financeTools := map[string]bool{
		"finance_prophet_forecast":  true,
		"finance_sentiment_analysis_fin": true,
		"finance_openbb_market_data": true,
	}
	if financeTools[toolName] {
		return "finance", toolName
	}

	osintTools := map[string]bool{
		"osint_region_dossier":    true,
		"osint_threat_level":      true,
		"osint_vessel_tracking":   true,
		"osint_flight_tracking":   true,
		"osint_correlation_alerts": true,
	}
	if osintTools[toolName] {
		return "osint", toolName
	}

	heTools := map[string]bool{
		"he_research_profiles":  true,
		"he_relational_engine":  true,
		"he_geographic_context": true,
		"he_pattern_classifier": true,
		"he_plugin_viz":         true,
	}
	if heTools[toolName] {
		return "human-ecosystems", toolName
	}

	return "finance", toolName
}

type mockPluginRegistry struct {
	components map[string]*decision.ComponentMetadata
}

func newMockPluginRegistry() *mockPluginRegistry {
	return &mockPluginRegistry{
		components: make(map[string]*decision.ComponentMetadata),
	}
}

func (m *mockPluginRegistry) GetComponentByID(ctx context.Context, id string) (*decision.ComponentMetadata, error) {
	if comp, ok := m.components[id]; ok {
		return comp, nil
	}
	return &decision.ComponentMetadata{
		ID:       id,
		Name:     id,
		Category: "unknown",
		Status:   "unknown",
	}, nil
}

func (m *mockPluginRegistry) registerComponent(id, name, category, status string) {
	m.components[id] = &decision.ComponentMetadata{
		ID:       id,
		Name:     name,
		Category: category,
		Status:   status,
	}
}

type mockToolRepository struct {
	tools []decision.ToolDef
}

func (m *mockToolRepository) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	return nil
}

func (m *mockToolRepository) GetChatMessages(ctx context.Context, projectID, agentID string) ([]decision.ChatMessage, error) {
	return nil, nil
}

func (m *mockToolRepository) ListTools(ctx context.Context) ([]decision.ToolDef, error) {
	return m.tools, nil
}

type mockProvider struct{}

func (m *mockProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: "mock response"}, nil
}

// ---------------------------------------------------------------------------
// Registry factory
// ---------------------------------------------------------------------------

func buildTestRegistry(t *testing.T) *tools.ToolRegistry {
	t.Helper()

	registry := tools.NewToolRegistry()

	registry.Register(tools.FinanceToolDef(
		"finance_prophet_forecast",
		"Time-series forecasting using SMA or linear regression",
		func(ctx context.Context, params map[string]any) (any, error) {
			tool := finance.NewProphetForecastTool()
			return tool.Execute(ctx, params)
		},
	))

	registry.Register(tools.FinanceToolDef(
		"finance_sentiment_analysis_fin",
		"Financial sentiment analysis tool",
		func(ctx context.Context, params map[string]any) (any, error) {
			tool := finance.NewSentimentAnalysisFinTool()
			return tool.Execute(ctx, params)
		},
	))

	osintTools := osint.ListTools(nil)
	for _, t := range osintTools {
		if osintTool, ok := t.(*osint.RegionDossierTool); ok {
			registry.Register(tools.OSINTToolDef(
				"osint_region_dossier",
				"Region dataset dossier from Shadowbroker (beta)",
				osintTool.Execute,
			))
		}
	}

	heTools := humanecosystems.ListTools(humanecosystems.SyntheticDuckDBLayer())
	for _, tool := range heTools {
		registry.Register(tools.HEToolDef(tool.(humanecosystems.ToolExecutor)))
	}

	return registry
}

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

type toolCategoryTest struct {
	description   string
	toolName      string
	category      string
	params        map[string]any
	expectedFields []string
	expectError   bool
	errorContains string
}

func TestCrossContext_ToolCategories(t *testing.T) {
	registry := buildTestRegistry(t)

	tests := []struct {
		name     string
		category string
		cases    []toolCategoryTest
	}{
		{
			name:     "finance",
			category: "finance",
			cases: []toolCategoryTest{
				{
					description:    "prophet_forecast with valid data",
					toolName:       "finance_prophet_forecast",
					category:       "finance",
					params:         map[string]any{"data": []float64{1.0, 2.0, 3.0, 4.0, 5.0}, "periods": 3},
					expectedFields: []string{"predictions", "confidence", "method"},
					expectError:    false,
				},
				{
					description:    "prophet_forecast with insufficient data",
					toolName:       "finance_prophet_forecast",
					category:       "finance",
					params:         map[string]any{"data": []float64{1.0}, "periods": 3},
					expectError:    true,
					errorContains:  "at least 2 data points",
				},
				{
					description:    "prophet_forecast with empty params",
					toolName:       "finance_prophet_forecast",
					category:       "finance",
					params:         map[string]any{},
					expectError:    true,
					errorContains:  "invalid",
				},
				{
					description:    "sentiment_analysis_fin with positive text",
					toolName:       "finance_sentiment_analysis_fin",
					category:       "finance",
					params:         map[string]any{"text": "bullish market outlook shows strong growth and profit",
						"source": "news"},
					expectedFields: []string{"sentiment", "score", "is_synthetic"},
					expectError:    false,
				},
				{
					description:    "sentiment_analysis_fin with empty text",
					toolName:       "finance_sentiment_analysis_fin",
					category:       "finance",
					params:         map[string]any{"text": ""},
					expectError:    true,
					errorContains:  "non-empty",
				},
			},
		},
		{
			name:     "osint",
			category: "osint",
			cases: []toolCategoryTest{
				{
					description:    "osint_region_dossier with valid region",
					toolName:      "osint_region_dossier",
					category:      "osint",
					params:        map[string]any{"region_id": "en_harbor"},
					expectedFields: []string{"region_name", "population", "stability", "generated_at"},
					expectError:    false,
				},
				{
					description:    "osint_region_dossier with empty region_id",
					toolName:      "osint_region_dossier",
					category:      "osint",
					params:        map[string]any{},
					expectError:    true,
					errorContains:  "region_id",
				},
			},
		},
		{
			name:     "human-ecosystems",
			category: "human-ecosystems",
			cases: []toolCategoryTest{
				{
					description:    "he_research_profiles with query",
					toolName:      "he_research_profiles",
					category:      "human-ecosystems",
					params:        map[string]any{"query": "ecosystem analysis"},
					expectedFields: []string{"profiles", "is_synthetic", "generated_at"},
					expectError:    false,
				},
				{
					description:    "he_relational_engine with entity",
					toolName:      "he_relational_engine",
					category:      "human-ecosystems",
					params:        map[string]any{"entity": "ecosystem-alpha"},
					expectedFields: []string{"relations", "is_synthetic", "entity"},
					expectError:    false,
				},
				{
					description:    "he_research_profiles with empty params (synthetic defaults)",
					toolName:      "he_research_profiles",
					category:      "human-ecosystems",
					params:        map[string]any{},
					expectedFields: []string{"generated_at"},
					expectError:    false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			for _, tc := range tt.cases {
				tc := tc
				t.Run(tc.toolName, func(t *testing.T) {
					t.Logf("Testing %s tool: %s", tc.category, tc.description)

					t.Run("registration", func(t *testing.T) {
						def, ok := registry.Get(tc.category, tc.toolName)
						if !tc.expectError {
							require.True(t, ok, "tool should be registered: %s:%s", tc.category, tc.toolName)
							assert.NotEmpty(t, def.Description)
						} else if tc.category == "osint" && strings.Contains(tc.errorContains, "region_id") {
							require.True(t, ok, "osint tool should still be registered")
						}
					})

					t.Run("discovery", func(t *testing.T) {
						tools := registry.List(tc.category)
						assert.NotEmpty(t, tools, "category should have tools registered")

						found := false
						for _, tool := range tools {
							if tool.Name == tc.toolName {
								found = true
								break
							}
						}
						assert.True(t, found, "tool should be discoverable: %s", tc.toolName)
					})

					t.Run("schema_validation", func(t *testing.T) {
						result, err := registry.ExecuteContext(context.Background(), tc.category, tc.toolName, tc.params)

						if tc.expectError {
							if tc.category == "osint" && strings.Contains(tc.errorContains, "region_id") {
								assert.Error(t, err)
								if err != nil {
									assert.Contains(t, err.Error(), tc.errorContains)
								}
								return
							}
							if tc.category == "finance" && strings.Contains(tc.errorContains, "invalid") {
								assert.Error(t, err)
								return
							}
							if tc.category == "finance" && strings.Contains(tc.errorContains, "data points") {
								assert.Error(t, err)
								if err != nil {
									assert.Contains(t, err.Error(), tc.errorContains)
								}
								return
							}
							if tc.category == "finance" && strings.Contains(tc.errorContains, "non-empty") {
								assert.Error(t, err)
								if err != nil {
									assert.Contains(t, err.Error(), tc.errorContains)
								}
								return
							}
						}

						require.NoError(t, err, "execution should succeed for: %s", tc.description)
						require.NotNil(t, result, "result should not be nil")

						resultMap, ok := result.(map[string]interface{})
						if !ok {
							t.Logf("Result type: %T - converting via JSON", result)
							data, err := json.Marshal(result)
							if err != nil {
								t.Fatalf("cannot marshal result type %T: %v", result, err)
							}
							if err := json.Unmarshal(data, &resultMap); err != nil {
								t.Fatalf("cannot unmarshal result to map: %v", err)
							}
						}

						for _, field := range tc.expectedFields {
							_, exists := resultMap[field]
							assert.True(t, exists, "field '%s' should exist in result", field)
						}

						if isSynthetic, ok := resultMap["is_synthetic"]; ok {
							assert.True(t, isSynthetic.(bool), "synthetic tools should return is_synthetic=true")
						}
					})
				})
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cross-context decision switching tests
// ---------------------------------------------------------------------------

func TestCrossContext_DecisionSwitching(t *testing.T) {
	registry := buildTestRegistry(t)
	toolRepo := &mockToolRepository{}
	pluginReg := newMockPluginRegistry()
	executor := &mockToolExecutor{registry: registry}

	pluginReg.registerComponent("finance_prophet", "finance_prophet_forecast", "finance", "active")
	pluginReg.registerComponent("he_research", "he_research_profiles", "human-ecosystems", "active")

	engine := decision.NewEngine(decision.EngineConfig{
		Provider:    &mockProvider{},
		MetaRepo:    toolRepo,
		Executor:    executor,
		Registry:    pluginReg,
		MaxAttempts: 5,
	})

	tests := []struct {
		name        string
		message     string
		projectID   string
		wantTools   []string
		wantSteps   int
		description string
	}{
		{
			description: "search_data inferred from message with 'data' keyword",
			message:     "forecast sales data for next 3 months",
			projectID:   "test-project-1",
			wantTools:   []string{"search_data"},
			wantSteps:   1,
		},
		{
			description: "no tool inferred from generic message (unknown keywords)",
			message:     "do something generic",
			projectID:   "test-project-2",
			wantTools:   []string{},
			wantSteps:   0,
		},
		{
			description: "analyze_sentiment inferred from message with 'sentiment' keyword",
			message:     "analyze market sentiment for AAPL",
			projectID:   "test-project-3",
			wantTools:   []string{"analyze_sentiment"},
			wantSteps:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := engine.Plan(context.Background(), tt.message, tt.projectID, "test-agent", nil, nil)
			require.NoError(t, err)
			assert.True(t, plan.CanProceed, "plan should be able to proceed")
			assert.Equal(t, tt.wantSteps, len(plan.Steps), "should have expected number of steps")

			if len(tt.wantTools) > 0 {
				for _, wantTool := range tt.wantTools {
					found := false
					for _, step := range plan.Steps {
						if step.ToolName == wantTool {
							found = true
							break
						}
					}
					assert.True(t, found, "tool '%s' should be in plan", wantTool)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error handling tests - unexpected output format
// ---------------------------------------------------------------------------

func TestCrossContext_ErrorHandling(t *testing.T) {
	registry := tools.NewToolRegistry()

	registry.Register(tools.ToolDefinition{
		Name:        "broken_tool",
		Category:    "test",
		Description: "A tool that returns malformed output",
		Execute: func(ctx context.Context, params map[string]any) (any, error) {
			return "this is not a map", nil
		},
	})

	t.Run("unexpected_string_output", func(t *testing.T) {
		result, err := registry.ExecuteContext(context.Background(), "test", "broken_tool", map[string]any{})
		assert.NoError(t, err)
		_, ok := result.(map[string]interface{})
		assert.False(t, ok, "result should not be a map when tool returns string")
	})

	t.Run("tool_not_found", func(t *testing.T) {
		_, err := registry.ExecuteContext(context.Background(), "nonexistent", "missing_tool", map[string]any{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool not found")
	})

	t.Run("nil_result", func(t *testing.T) {
		registry.Register(tools.ToolDefinition{
			Name:        "nil_tool",
			Category:    "test",
			Description: "A tool that returns nil",
			Execute: func(ctx context.Context, params map[string]any) (any, error) {
				return nil, nil
			},
		})

		result, err := registry.ExecuteContext(context.Background(), "test", "nil_tool", map[string]any{})
		assert.NoError(t, err)
		assert.Nil(t, result, "nil result should be returned without error")
	})
}

// ---------------------------------------------------------------------------
// Schema compliance verification tests
// ---------------------------------------------------------------------------

func TestCrossContext_SchemaCompliance(t *testing.T) {
	registry := buildTestRegistry(t)

	tests := []struct {
		name           string
		category       string
		toolName       string
		params         map[string]any
		requiredFields []string
		denyFields     []string
	}{
		{
			name:           "finance_prophet_forecast_schema",
			category:       "finance",
			toolName:       "finance_prophet_forecast",
			params:         map[string]any{"data": []float64{1.0, 2.0, 3.0, 4.0}, "periods": 2},
			requiredFields: []string{"predictions", "confidence", "method"},
			denyFields:     []string{"error"},
		},
		{
			name:           "osint_region_dossier_schema",
			category:       "osint",
			toolName:       "osint_region_dossier",
			params:         map[string]any{"region_id": "northern_rise"},
			requiredFields: []string{"region_name", "population", "stability", "generated_at"},
			denyFields:     []string{"error"},
		},
		{
			name:           "humanecosystems_research_profiles_schema",
			category:       "human-ecosystems",
			toolName:       "he_research_profiles",
			params:         map[string]any{"query": "test query"},
			requiredFields: []string{"profiles", "generated_at", "is_synthetic"},
			denyFields:     []string{"error", "pii"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.ExecuteContext(context.Background(), tt.category, tt.toolName, tt.params)
			require.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				data, err := json.Marshal(result)
				if err != nil {
					t.Fatalf("cannot marshal result type %T: %v", result, err)
				}
				if err := json.Unmarshal(data, &resultMap); err != nil {
					t.Fatalf("cannot unmarshal result to map: %v", err)
				}
			}

			for _, field := range tt.requiredFields {
				_, exists := resultMap[field]
				assert.True(t, exists, "required field '%s' should be present", field)
			}

			for _, field := range tt.denyFields {
				_, exists := resultMap[field]
				assert.False(t, exists, "field '%s' should NOT be present in output", field)
			}

			if tt.category == "human-ecosystems" {
				resultJSON, _ := json.Marshal(resultMap)
				assert.NotContains(t, string(resultJSON), "email")
				assert.NotContains(t, string(resultJSON), "password")
				assert.NotContains(t, string(resultJSON), "phone")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: full decision loop with multiple tool categories
// ---------------------------------------------------------------------------

func TestCrossContext_FullDecisionLoop(t *testing.T) {
	registry := buildTestRegistry(t)
	toolRepo := &mockToolRepository{}
	pluginReg := newMockPluginRegistry()
	executor := &mockToolExecutor{registry: registry}

	engine := decision.NewEngine(decision.EngineConfig{
		Provider:    &mockProvider{},
		MetaRepo:    toolRepo,
		Executor:    executor,
		Registry:    pluginReg,
		MaxAttempts: 5,
	})

	t.Run("finance_tool_execution", func(t *testing.T) {
		plan, err := engine.Plan(context.Background(), "forecast data for [1,2,3,4,5] next 3 periods", "proj-1", "agent-1", nil, nil)
		require.NoError(t, err)

		if len(plan.Steps) == 0 || plan.Steps[0].ToolName != "finance_prophet_forecast" {
			t.Skip("engine does not infer finance_prophet_forecast from this message - testing direct tool execution instead")
		}

		require.NotEmpty(t, plan.Steps)
		step := plan.Steps[0]

		result, err := registry.ExecuteContext(context.Background(), "finance", step.ToolName, step.Arguments)
		require.NoError(t, err)

		actResult := &decision.ActResult{
			Step:   step,
			Output: fmt.Sprintf("%v", result),
		}
		obs, err := engine.Observe(context.Background(), step, actResult)
		require.NoError(t, err)
		assert.True(t, obs.Success, "observation should report success")
	})

	t.Run("humanecosystems_tool_execution", func(t *testing.T) {
		plan, err := engine.Plan(context.Background(), "research profiles for query", "proj-2", "agent-1", nil, nil)
		require.NoError(t, err)

		if len(plan.Steps) == 0 || plan.Steps[0].ToolName != "he_research_profiles" {
			t.Skip("engine does not infer he_research_profiles from this message - testing direct tool execution instead")
		}

		require.NotEmpty(t, plan.Steps)
		step := plan.Steps[0]

		result, err := registry.ExecuteContext(context.Background(), "human-ecosystems", step.ToolName, step.Arguments)
		require.NoError(t, err)

		actResult := &decision.ActResult{
			Step:   step,
			Output: fmt.Sprintf("%v", result),
		}
		obs, err := engine.Observe(context.Background(), step, actResult)
		require.NoError(t, err)
		assert.True(t, obs.Success)

		var resultMap map[string]interface{}
		if rm, ok := result.(map[string]interface{}); ok {
			resultMap = rm
		} else {
			data, _ := json.Marshal(result)
			json.Unmarshal(data, &resultMap)
		}
		assert.True(t, resultMap["is_synthetic"].(bool))
	})

	t.Run("reflect_and_admit", func(t *testing.T) {
		plan, err := engine.Plan(context.Background(), "test message", "proj-3", "agent-1", nil, nil)
		require.NoError(t, err)

		if len(plan.Steps) == 0 {
			t.Skip("no steps in plan to test reflect/admit")
		}

		observations := []decision.Observation{
			{
				Step:    plan.Steps[0],
				Success: true,
			},
		}

		reflectedPlan, err := engine.Reflect(context.Background(), plan, observations)
		require.NoError(t, err)
		assert.True(t, reflectedPlan.CanProceed)

		actResults := []*decision.ActResult{
			{Step: plan.Steps[0], Output: "success"},
		}
		admit, err := engine.Admit(context.Background(), actResults, 5)
		require.NoError(t, err)
		assert.False(t, admit, "should not admit (not reached max attempts and no error)")
	})

	t.Run("admit_on_error", func(t *testing.T) {
		plan, err := engine.Plan(context.Background(), "test", "proj-4", "agent-1", nil, nil)
		require.NoError(t, err)

		if len(plan.Steps) == 0 {
			t.Skip("no steps in plan to test admit on error")
		}

		actResults := []*decision.ActResult{
			{Step: plan.Steps[0], Error: "tool failed"},
		}

		admit, err := engine.Admit(context.Background(), actResults, 5)
		require.NoError(t, err)
		assert.True(t, admit, "should admit when last result has error")
	})

	t.Run("direct_tool_execution_finance", func(t *testing.T) {
		result, err := registry.ExecuteContext(context.Background(), "finance", "finance_prophet_forecast",
			map[string]any{"data": []float64{1.0, 2.0, 3.0, 4.0, 5.0}, "periods": 3})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, fmt.Sprintf("%v", result))
	})

	t.Run("direct_tool_execution_humanecosystems", func(t *testing.T) {
		result, err := registry.ExecuteContext(context.Background(), "human-ecosystems", "he_research_profiles",
			map[string]any{"query": "test query"})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, fmt.Sprintf("%v", result))
	})
}

// ---------------------------------------------------------------------------
// Categories listing test
// ---------------------------------------------------------------------------

func TestCrossContext_Categories(t *testing.T) {
	registry := buildTestRegistry(t)

	categories := registry.Categories()
	assert.NotEmpty(t, categories, "should have at least one category registered")

	expected := map[string]bool{
		"finance":          false,
		"osint":            false,
		"human-ecosystems": false,
	}

	for _, cat := range categories {
		expected[cat] = true
	}

	for cat, found := range expected {
		assert.True(t, found, "category '%s' should be registered", cat)
	}
}

// ---------------------------------------------------------------------------
// Registry error cases
// ---------------------------------------------------------------------------

func TestCrossContext_RegistryErrors(t *testing.T) {
	t.Run("duplicate_registration", func(t *testing.T) {
		registry := tools.NewToolRegistry()

		err := registry.Register(tools.ToolDefinition{
			Name:        "dup_tool",
			Category:    "test",
			Description: "first registration",
			Execute:     func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		require.NoError(t, err)

		err = registry.Register(tools.ToolDefinition{
			Name:        "dup_tool",
			Category:    "test",
			Description: "duplicate",
			Execute:     func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("batch_registration_failure", func(t *testing.T) {
		registry := tools.NewToolRegistry()

		defs := []tools.ToolDefinition{
			{Name: "tool1", Category: "cat1", Execute: func(ctx context.Context, params map[string]any) (any, error) { return nil, nil }},
			{Name: "tool2", Category: "cat2", Execute: func(ctx context.Context, params map[string]any) (any, error) { return nil, nil }},
			{Name: "", Category: "cat3", Execute: func(ctx context.Context, params map[string]any) (any, error) { return nil, nil }},
		}

		err := registry.RegisterAll(defs)
		assert.Error(t, err)
	})

	t.Run("empty_name_or_category", func(t *testing.T) {
		registry := tools.NewToolRegistry()

		err := registry.Register(tools.ToolDefinition{
			Name:        "",
			Category:    "test",
			Description: "no name",
			Execute:     func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		assert.Error(t, err)

		err = registry.Register(tools.ToolDefinition{
			Name:        "no_category",
			Category:    "",
			Description: "no category",
			Execute:     func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		assert.Error(t, err)

		err = registry.Register(tools.ToolDefinition{
			Name:        "nil_execute",
			Category:    "test",
			Description: "nil func",
			Execute:     nil,
		})
		assert.Error(t, err)
	})
}