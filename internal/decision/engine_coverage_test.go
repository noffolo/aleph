package decision

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_ExtractObjectReferences(t *testing.T) {
	t.Parallel()
	e := NewEngine(EngineConfig{})

	cases := []struct {
		msg      string
		wantRefs []string
	}{
		{"Analyze Company and Market data", []string{"Analyze", "Company", "Market"}},
		{"simple lowercase query", nil},
		{"", nil},
		{"GDP Forecast for Q3", []string{"GDP", "Forecast", "Q3"}},
		{"TEST Capital Asset Model", []string{"TEST", "Capital", "Asset", "Model"}},
	}
	for _, tc := range cases {
		t.Run(tc.msg, func(t *testing.T) {
			refs := e.extractObjectReferences(tc.msg)
			for _, want := range tc.wantRefs {
				assert.Contains(t, refs, want)
			}
		})
	}

	t.Run("lowercase_only", func(t *testing.T) {
		refs := e.extractObjectReferences("simple lowercase query")
		assert.Empty(t, refs)
	})

	t.Run("empty_string", func(t *testing.T) {
		assert.Empty(t, e.extractObjectReferences(""))
	})
}

func TestSortStepsByDependencies(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		assert.Nil(t, SortStepsByDependencies(nil))
	})

	t.Run("single_step", func(t *testing.T) {
		steps := []PlannedStep{{ToolName: "A"}}
		result := SortStepsByDependencies(steps)
		assert.Equal(t, "A", result[0].ToolName)
	})

	t.Run("two_independent", func(t *testing.T) {
		steps := []PlannedStep{{ToolName: "A"}, {ToolName: "B"}}
		result := SortStepsByDependencies(steps)
		assert.Len(t, result, 2)
	})

	t.Run("dependency_chain", func(t *testing.T) {
		steps := []PlannedStep{
			{ToolName: "B", Depends: []string{"A"}},
			{ToolName: "A"},
		}
		result := SortStepsByDependencies(steps)
		assert.Equal(t, "A", result[0].ToolName)
		assert.Equal(t, "B", result[1].ToolName)
	})

	t.Run("unknown_dependency_tolerated", func(t *testing.T) {
		steps := []PlannedStep{
			{ToolName: "A", Depends: []string{"NonExistent"}},
			{ToolName: "B"},
		}
		result := SortStepsByDependencies(steps)
		assert.Len(t, result, 2)
	})

	t.Run("cycle_detected", func(t *testing.T) {
		steps := []PlannedStep{
			{ToolName: "A", Depends: []string{"B"}},
			{ToolName: "B", Depends: []string{"A"}},
		}
		result := SortStepsByDependencies(steps)
		assert.Len(t, result, 2)
		found := make(map[string]bool)
		for _, s := range result {
			found[s.ToolName] = true
		}
		assert.True(t, found["A"])
		assert.True(t, found["B"])
	})
}

func TestFailedStepNames(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, FailedStepNames(nil))
	})

	t.Run("no_failures", func(t *testing.T) {
		results := []*ActResult{
			{Step: PlannedStep{ToolName: "step1"}, Error: ""},
			{Step: PlannedStep{ToolName: "step2"}, Error: ""},
		}
		assert.Empty(t, FailedStepNames(results))
	})

	t.Run("some_failures", func(t *testing.T) {
		results := []*ActResult{
			{Step: PlannedStep{ToolName: "step1"}, Error: "fail"},
			{Step: PlannedStep{ToolName: "step2"}, Error: ""},
			{Step: PlannedStep{ToolName: "step3"}, Error: "fail too"},
		}
		failed := FailedStepNames(results)
		assert.True(t, failed["step1"])
		assert.False(t, failed["step2"])
		assert.True(t, failed["step3"])
		assert.Len(t, failed, 2)
	})
}

func TestShouldSkipStep(t *testing.T) {
	t.Parallel()

	failedDeps := map[string]bool{"depA": true, "depB": false}

	t.Run("no_deps", func(t *testing.T) {
		assert.False(t, ShouldSkipStep(PlannedStep{ToolName: "X"}, failedDeps))
	})

	t.Run("failed_dep", func(t *testing.T) {
		assert.True(t, ShouldSkipStep(PlannedStep{ToolName: "X", Depends: []string{"depA"}}, failedDeps))
	})

	t.Run("not_failed_dep", func(t *testing.T) {
		assert.False(t, ShouldSkipStep(PlannedStep{ToolName: "X", Depends: []string{"depB"}}, failedDeps))
	})

	t.Run("unknown_dep", func(t *testing.T) {
		assert.False(t, ShouldSkipStep(PlannedStep{ToolName: "X", Depends: []string{"unknown"}}, failedDeps))
	})
}

func TestEngine_IsKnownTool(t *testing.T) {
	t.Parallel()
	e := NewEngine(EngineConfig{})

	assert.True(t, e.isKnownTool(context.Background(), "search_data"))
	assert.True(t, e.isKnownTool(context.Background(), "analyze_sentiment"))
	assert.True(t, e.isKnownTool(context.Background(), "get_trust_score"))
	assert.False(t, e.isKnownTool(context.Background(), "unknown_tool"))
}

func TestEngine_ExtractObjectReferencesWithOntology(t *testing.T) {
	t.Parallel()
	e := NewEngine(EngineConfig{})

	t.Run("empty_ont_falls_back_to_heuristic", func(t *testing.T) {
		refs := e.extractObjectReferencesWithOntology("Capital Asset Model", nil)
		assert.NotEmpty(t, refs)
	})

	t.Run("invalid_dsl_falls_back", func(t *testing.T) {
		refs := e.extractObjectReferencesWithOntology("Capital Asset Model", []byte("invalid {{ dsl"))
		assert.NotEmpty(t, refs)
	})

	t.Run("valid_ontology_matches", func(t *testing.T) {
		ontContent := []byte(`
			object CapitalAsset {
				field id: string
				field name: string
			}
			object MarketData {
				field ticker: string
				field price: number
			}
		`)
		refs := e.extractObjectReferencesWithOntology("show me CapitalAsset data and MarketData prices", ontContent)
		assert.Contains(t, refs, "CapitalAsset")
		assert.Contains(t, refs, "MarketData")
	})

	t.Run("valid_ontology_no_match", func(t *testing.T) {
		ontContent := []byte(`
			object CapitalAsset {
				field id: string
			}
		`)
		refs := e.extractObjectReferencesWithOntology("show me weather data", ontContent)
		assert.Empty(t, refs)
	})
}

func TestEngine_Admit(t *testing.T) {
	t.Parallel()
	e := NewEngine(EngineConfig{MaxAttempts: 3})

	t.Run("empty_results", func(t *testing.T) {
		done, err := e.Admit(context.Background(), nil, 0)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("single_success", func(t *testing.T) {
		results := []*ActResult{{Error: ""}}
		done, err := e.Admit(context.Background(), results, 0)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("single_error", func(t *testing.T) {
		results := []*ActResult{{Error: "error"}}
		done, err := e.Admit(context.Background(), results, 0)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("max_attempts_reached", func(t *testing.T) {
		results := make([]*ActResult, 3)
		for i := range results {
			results[i] = &ActResult{Error: ""}
		}
		done, err := e.Admit(context.Background(), results, 3)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("max_attempts_default", func(t *testing.T) {
		results := make([]*ActResult, 5)
		for i := range results {
			results[i] = &ActResult{Error: ""}
		}
		done, err := e.Admit(context.Background(), results, 0)
		require.NoError(t, err)
		assert.True(t, done)
	})
}

func TestNewEngine_Defaults(t *testing.T) {
	t.Parallel()

	t.Run("zero_max_attempts", func(t *testing.T) {
		e := NewEngine(EngineConfig{MaxAttempts: 0})
		assert.Equal(t, 5, e.maxAttempts)
	})

	t.Run("negative_max_attempts", func(t *testing.T) {
		e := NewEngine(EngineConfig{MaxAttempts: -1})
		assert.Equal(t, 5, e.maxAttempts)
	})

	t.Run("negative_conf_threshold", func(t *testing.T) {
		e := NewEngine(EngineConfig{ConfirmationThreshold: -0.5})
		assert.Equal(t, 0.0, e.confirmationThreshold)
	})

	t.Run("conf_threshold_above_1", func(t *testing.T) {
		e := NewEngine(EngineConfig{ConfirmationThreshold: 1.5})
		assert.Equal(t, 1.0, e.confirmationThreshold)
	})

	t.Run("zero_truncation", func(t *testing.T) {
		e := NewEngine(EngineConfig{TruncationThreshold: 0})
		assert.Equal(t, 1900, e.truncationThreshold)
	})

	t.Run("nil_reflector", func(t *testing.T) {
		e := NewEngine(EngineConfig{Reflector: nil})
		assert.NotNil(t, e.reflector)
	})
}

func TestToolExecutionHistory(t *testing.T) {
	t.Parallel()

	h := &toolExecutionHistory{}
	assert.Equal(t, 0, h.total())

	h.successes = 3
	h.failures = 2
	assert.Equal(t, 5, h.total())
}

func TestEngine_TrainLinkModel(t *testing.T) {
	t.Parallel()

	t.Run("no_link_predictor_noop", func(t *testing.T) {
		e := NewEngine(EngineConfig{})
		err := e.TrainLinkModel(context.Background(), nil, 5)
		assert.NoError(t, err)
	})

	t.Run("nil_graph_with_predictor", func(t *testing.T) {
		e := NewEngine(EngineConfig{
			LinkPredictor: NewGNNLinkPredictor(4, 8, 0.01),
		})
		err := e.TrainLinkModel(context.Background(), nil, 5)
		assert.Error(t, err)
	})

	t.Run("empty_graph_with_predictor", func(t *testing.T) {
		e := NewEngine(EngineConfig{
			LinkPredictor: NewGNNLinkPredictor(4, 8, 0.01),
		})
		graph := gnn.NewGraph()
		err := e.TrainLinkModel(context.Background(), graph, 5)
		assert.Error(t, err)
	})

	t.Run("valid_graph_with_predictor", func(t *testing.T) {
		e := NewEngine(EngineConfig{
			LinkPredictor: NewGNNLinkPredictor(3, 8, 0.01),
		})
		graph := gnn.NewGraph()
		graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("N1")})
		graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("N2")})
		graph.AddNode(&gnn.WorkflowNode{ID: gnn.NodeID("N3")})
		graph.AddEdge(gnn.Edge{Source: gnn.NodeID("N1"), Target: gnn.NodeID("N2"), Weight: 0.5})
		err := e.TrainLinkModel(context.Background(), graph, 2)
		assert.NoError(t, err)
	})
}

func TestDefaultAdmitter(t *testing.T) {
	t.Parallel()
	a := NewDefaultAdmitter()

	t.Run("empty_results", func(t *testing.T) {
		done, err := a.Admit(context.Background(), nil, 5)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("max_attempts_reached", func(t *testing.T) {
		results := []*ActResult{{}, {}}
		done, err := a.Admit(context.Background(), results, 2)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("last_error", func(t *testing.T) {
		results := []*ActResult{{Error: ""}, {Error: "failed"}}
		done, err := a.Admit(context.Background(), results, 10)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("continue_on_success", func(t *testing.T) {
		results := []*ActResult{{Error: ""}}
		done, err := a.Admit(context.Background(), results, 10)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("zero_max_ignored", func(t *testing.T) {
		results := []*ActResult{{}, {}, {}}
		done, err := a.Admit(context.Background(), results, 0)
		require.NoError(t, err)
		assert.False(t, done)
	})
}
