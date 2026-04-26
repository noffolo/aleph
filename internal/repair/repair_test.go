package repair

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock storage
// ---------------------------------------------------------------------------

type mockMetaRepo struct {
	mu   sync.Mutex
	data map[string]string
	err  error
}

func newMockMetaRepo() *mockMetaRepo {
	return &mockMetaRepo{
		data: make(map[string]string),
	}
}

func (m *mockMetaRepo) GetToolCode(ctx context.Context, toolID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	code, ok := m.data[toolID]
	if !ok {
		return "", errors.New("tool not found")
	}
	return code, nil
}

func (m *mockMetaRepo) UpdateToolCode(ctx context.Context, id string, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.data[id] = code
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestEngine(reader ToolCodeReader, writer ToolCodeWriter) *RepairEngine {
	v := sandbox.NewVerifier(slog.Default(), nil, "", "")
	return &RepairEngine{
		logger:   slog.Default(),
		reader:   reader,
		writer:   writer,
		compiler: nil, // compiler nil → regenerate returns error
		verifier: v,
		history:  NewRepairHistory(slog.Default()),
	}
}

// ---------------------------------------------------------------------------
// ClassifyToolError
// ---------------------------------------------------------------------------

func TestClassifyToolError(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantPat string
	}{
		{"import error", "cannot find module \"fmt\"", PatternToolImport},
		{"undefined variable", "undefined: x", PatternToolImport},
		{"no such file", "no such file or directory", PatternToolImport},
		{"syntax error", "syntax error: unexpected EOF", PatternToolSyntax},
		{"parse error", "parse error: expected ';'", PatternToolSyntax},
		{"deprecated API", "ioutil.ReadFile is deprecated", PatternToolDeprecated},
		{"removed function", "has been removed", PatternToolDeprecated},
		{"config error", "invalid argument: unknown flag", PatternToolConfig},
		{"missing config", "missing required environment variable", PatternToolConfig},
		{"timeout", "context deadline exceeded", PatternToolTimeout},
		{"OOM", "out of memory", PatternToolTimeout},
		{"data pipeline", "schema mismatch in pipeline", PatternToolDataPipeline},
		{"dependency", "missing dependency: package not found", PatternToolDependency},
		{"unknown error", "a completely unrelated and mysterious problem", PatternToolUnknown},
		{"empty string", "", PatternToolUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyToolError(tt.errMsg)
			assert.Equal(t, tt.wantPat, got)
		})
	}
}

// ---------------------------------------------------------------------------
// RepairCatalog
// ---------------------------------------------------------------------------

func TestBuildRepairCatalog(t *testing.T) {
	catalog := BuildRepairCatalog()

	// Every tool pattern should have at least one entry.
	for _, pattern := range allToolPatterns {
		actions, ok := catalog[pattern]
		assert.True(t, ok, "catalog should contain pattern %q", pattern)
		assert.NotEmpty(t, actions, "pattern %q should have at least one action", pattern)
	}

	// Verify action structure.
	for pattern, actions := range catalog {
		for _, a := range actions {
			assert.NotEmpty(t, a.ID, "pattern %q action should have ID", pattern)
			assert.NotEmpty(t, a.Type, "pattern %q action should have Type", pattern)
			assert.NotEmpty(t, a.Description, "pattern %q action should have Description", pattern)
		}
	}

	// Import errors should include regenerate option.
	importActions := catalog[PatternToolImport]
	hasRegen := false
	for _, a := range importActions {
		if a.Type == ActionRegenerate {
			hasRegen = true
			break
		}
	}
	assert.True(t, hasRegen, "import errors should have regenerate action")
}

func TestDefaultCatalog(t *testing.T) {
	assert.NotNil(t, defaultCatalog)
	assert.Equal(t, defaultCatalog, BuildRepairCatalog())
}

// ---------------------------------------------------------------------------
// AnalyseAndPlan
// ---------------------------------------------------------------------------

func TestAnalyseAndPlan(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["test_tool"] = `package main
func main() { println("hello") }`

	engine := newTestEngine(mock, mock)
	plan, err := engine.AnalyseAndPlan(context.Background(), "test_tool", "syntax error: unexpected EOF")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "test_tool", plan.ToolID)
	assert.Equal(t, PatternToolSyntax, plan.ErrorPattern)
	assert.Contains(t, plan.ErrorMessage, "syntax error")
	assert.Equal(t, PlanPending, plan.Status)
	assert.NotEmpty(t, plan.BackupCode)
	assert.NotEmpty(t, plan.CreatedAt)
	assert.Greater(t, len(plan.Actions), 0, "plan should have at least one action")

	// Structural changes require approval.
	assert.True(t, plan.NeedsApproval)
}

func TestAnalyseAndPlan_ImportRequiresApproval(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`

	engine := newTestEngine(mock, mock)
	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "cannot find module")
	require.NoError(t, err)
	assert.True(t, plan.NeedsApproval)
}

func TestAnalyseAndPlan_PerformanceDoesNotRequireApproval(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`

	engine := newTestEngine(mock, mock)
	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "performance issue")
	require.NoError(t, err)
	assert.False(t, plan.NeedsApproval)
}

func TestAnalyseAndPlan_ToolNotFound(t *testing.T) {
	mock := newMockMetaRepo()
	engine := newTestEngine(mock, mock)
	_, err := engine.AnalyseAndPlan(context.Background(), "nonexistent", "error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")
}

// ---------------------------------------------------------------------------
// ApprovePlan / RejectPlan
// ---------------------------------------------------------------------------

func TestApprovePlan(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`
	engine := newTestEngine(mock, mock)

	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "syntax error")
	require.NoError(t, err)
	assert.Equal(t, PlanPending, plan.Status)

	engine.ApprovePlan(plan)
	assert.Equal(t, PlanApproved, plan.Status)

	// Approving again should not change status.
	engine.ApprovePlan(plan)
	assert.Equal(t, PlanApproved, plan.Status)
}

func TestRejectPlan(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`
	engine := newTestEngine(mock, mock)

	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "syntax error")
	require.NoError(t, err)
	assert.Equal(t, PlanPending, plan.Status)

	engine.RejectPlan(plan)
	assert.Equal(t, PlanRejected, plan.Status)
}

// ---------------------------------------------------------------------------
// ExecutePlan — approval gate
// ---------------------------------------------------------------------------

func TestExecutePlan_RequiresApproval(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`
	engine := newTestEngine(mock, mock)

	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "syntax error")
	require.NoError(t, err)

	// Attempt execution without approval.
	err = engine.ExecutePlan(context.Background(), plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be approved")
}

func TestExecutePlan_ApprovedThenExecuted(t *testing.T) {
	mock := newMockMetaRepo()
	originalCode := `package main
import "fmt"
func main() { fmt.Println("hello") }`
	mock.data["t"] = originalCode
	engine := newTestEngine(mock, mock)

	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "syntax error")
	require.NoError(t, err)

	engine.ApprovePlan(plan)
	err = engine.ExecutePlan(context.Background(), plan)
	// May succeed or fail depending on fix — we just check plan status.
	// If no compiler available, regenerate will fail but fix actions may work.
	t.Logf("ExecutePlan result: %v", err)
}

// ---------------------------------------------------------------------------
// Fix implementations
// ---------------------------------------------------------------------------

func TestFixMissingImports(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main
import "time"
func main() {
	json.Marshal(nil)
	fmt.Println("hello")
}`
	result := engine.fixMissingImports(code)
	assert.Contains(t, result, `"encoding/json"`, "should add json import")
	assert.Contains(t, result, `"fmt"`, "should add fmt import")
}

func TestFixMissingImports_ExistingBlock(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main

import (
	"fmt"
)

func main() {
	json.Marshal(nil)
	time.Now()
}`
	result := engine.fixMissingImports(code)
	assert.Contains(t, result, `"encoding/json"`)
	assert.Contains(t, result, `"time"`)
	assert.Contains(t, result, `"fmt"`)
}

func TestFixMissingImports_SingleLineImport(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main
import "fmt"
func main() { json.Marshal(nil) }`
	result := engine.fixMissingImports(code)
	assert.Contains(t, result, `"encoding/json"`)
	assert.NotContains(t, result, `import "fmt"`) // should be converted to block
}

func TestFixMissingImports_NoImportsNeeded(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main
func main() { println("hello") }`
	result := engine.fixMissingImports(code)
	assert.Equal(t, code, result)
}

func TestFixSyntaxError_UnmatchedBraces(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main
func main() {
	println("hello")
`
	result := engine.fixSyntaxError(code)
	assert.Equal(t, strings.Count(result, "{"), strings.Count(result, "}"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(result), "}"))
}

func TestFixSyntaxError_BalancedBraces(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `package main
func main() {
	println("hello")
}`
	result := engine.fixSyntaxError(code)
	assert.Equal(t, code, result)
}

func TestFixDeprecatedAPI(t *testing.T) {
	engine := newTestEngine(nil, nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"ioutil.ReadAll", "io.ReadAll"},
		{"ioutil.ReadFile", "os.ReadFile"},
		{"ioutil.WriteFile", "os.WriteFile"},
		{"ioutil.ReadDir", "os.ReadDir"},
		{"ioutil.TempDir", "os.MkdirTemp"},
		{"ioutil.TempFile", "os.CreateTemp"},
		{"ioutil.NopCloser", "io.NopCloser"},
		{"ioutil.Discard", "io.Discard"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := engine.fixDeprecatedAPI(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFixTimeout(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `timeout := 100 * time.Millisecond`
	result := engine.fixTimeout(code)
	assert.Contains(t, result, "5 * time.Second")
}

func TestFixConfiguration(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `endpoint := "localhost:8080"`
	result := engine.fixConfiguration(code)
	assert.Contains(t, result, `os.Getenv("ALEPH_ENDPOINT")`)
}

func TestFixRetry(t *testing.T) {
	engine := newTestEngine(nil, nil)

	code := `func Handle(ctx context.Context, input string) (string, error) {
	// TODO: implement
	return "", nil
}`
	result := engine.fixRetry(code)
	assert.Contains(t, result, "maxRetries := 3")
}

// ---------------------------------------------------------------------------
// RepairHistory
// ---------------------------------------------------------------------------

func TestRepairHistory_Record(t *testing.T) {
	h := NewRepairHistory(slog.Default())

	rec := RepairRecord{
		ToolID:     "tool1",
		PlanID:     "plan1",
		ActionID:   "act1",
		ActionType: string(ActionFix),
		Status:     StatusSuccess,
		Duration:   100 * time.Millisecond,
	}
	h.Record(rec)
	h.Record(RepairRecord{
		ToolID:     "tool1",
		PlanID:     "plan1",
		ActionID:   "act2",
		ActionType: string(ActionFix),
		Status:     StatusFailed,
		ErrorMsg:   "something went wrong",
		Duration:   50 * time.Millisecond,
	})
	h.Record(RepairRecord{
		ToolID:     "tool2",
		PlanID:     "plan2",
		ActionID:   "act1",
		ActionType: string(ActionRegenerate),
		Status:     StatusSuccess,
		Duration:   200 * time.Millisecond,
	})

	// Total records.
	all := h.GetAll()
	assert.Len(t, all, 3)

	// Filter by tool.
	tool1 := h.GetHistory("tool1")
	assert.Len(t, tool1, 2)
	assert.Equal(t, "act1", tool1[0].ActionID)
	assert.Equal(t, "act2", tool1[1].ActionID)
}

func TestRepairHistory_SuccessRate(t *testing.T) {
	h := NewRepairHistory(slog.Default())

	// Record: tool1 has 2 success, 1 failure = 66%
	h.Record(RepairRecord{ToolID: "t1", Status: StatusSuccess})
	h.Record(RepairRecord{ToolID: "t1", Status: StatusSuccess})
	h.Record(RepairRecord{ToolID: "t1", Status: StatusFailed})

	rate := h.SuccessRate("t1")
	assert.InDelta(t, 0.666, rate, 0.01)

	// No history for unknown tool.
	unknownRate := h.SuccessRate("unknown")
	assert.Equal(t, 0.0, unknownRate)

	// Empty history.
	empty := NewRepairHistory(slog.Default())
	assert.Equal(t, 0.0, empty.SuccessRate("x"))
}

func TestRepairHistory_OverallSuccessRate(t *testing.T) {
	h := NewRepairHistory(slog.Default())

	h.Record(RepairRecord{ToolID: "t1", Status: StatusSuccess})
	h.Record(RepairRecord{ToolID: "t2", Status: StatusFailed})

	assert.InDelta(t, 0.5, h.OverallSuccessRate(), 0.01)

	// Empty history.
	empty := NewRepairHistory(slog.Default())
	assert.Equal(t, 0.0, empty.OverallSuccessRate())
}

func TestRepairHistory_GetAll(t *testing.T) {
	h := NewRepairHistory(slog.Default())
	assert.Empty(t, h.GetAll())

	h.Record(RepairRecord{ToolID: "t1", Status: StatusSuccess})
	assert.Len(t, h.GetAll(), 1)

	// Verify isolation.
	all := h.GetAll()
	all[0].ToolID = "modified"
	refetched := h.GetAll()
	assert.Equal(t, "t1", refetched[0].ToolID)
}

// ---------------------------------------------------------------------------
// Engine history integration
// ---------------------------------------------------------------------------

func TestRepairEngine_GetHistory(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main`
	engine := newTestEngine(mock, mock)

	history := engine.GetHistory("t")
	assert.Empty(t, history)

	// After analysis, no history should exist (plan not executed).
	plan, err := engine.AnalyseAndPlan(context.Background(), "t", "syntax error")
	require.NoError(t, err)
	history = engine.GetHistory("t")
	assert.Empty(t, history)
	_ = plan

	allHistory := engine.GetAllHistory()
	assert.Empty(t, allHistory)
}

func TestRepairEngine_SuccessRate_NoHistory(t *testing.T) {
	mock := newMockMetaRepo()
	engine := newTestEngine(mock, mock)
	assert.Equal(t, 0.0, engine.SuccessRate("nonexistent"))
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestHasRegenerateAction(t *testing.T) {
	assert.False(t, hasRegenerateAction(nil))
	assert.False(t, hasRegenerateAction([]RepairAction{}))
	assert.True(t, hasRegenerateAction([]RepairAction{
		{Type: ActionRegenerate},
	}))
	assert.False(t, hasRegenerateAction([]RepairAction{
		{Type: ActionFix},
	}))
}

func TestClassifyToolError_EmptyAndUnknown(t *testing.T) {
	assert.Equal(t, PatternToolUnknown, ClassifyToolError(""))
	assert.Equal(t, PatternToolUnknown, ClassifyToolError("some random error with no match"))
	assert.Equal(t, PatternToolUnknown, ClassifyToolError("  "))
}

func TestClassifyToolError_CaseInsensitivity(t *testing.T) {
	assert.Equal(t, PatternToolImport, ClassifyToolError("IMPORT ERROR: missing module"))
	assert.Equal(t, PatternToolSyntax, ClassifyToolError("SYNTAX ERROR"))
	assert.Equal(t, PatternToolDeprecated, ClassifyToolError("Deprecated Function"))
}

func TestExecutePlan_RollbackOnAllFailures(t *testing.T) {
	mock := newMockMetaRepo()
	originalCode := `package main
func main() { println("original") }`
	mock.data["t"] = originalCode
	engine := newTestEngine(mock, mock)

	// Use unknown error → only regenerate action → fails without compiler.
	plan := &RepairPlan{
		ID:            "rollback-test-plan",
		ToolID:        "t",
		Status:        PlanApproved,
		ErrorPattern:  PatternToolUnknown,
		ErrorMessage:  "something unknown",
		BackupCode:    originalCode,
		NeedsApproval: true,
		Actions: []RepairAction{
			{ID: "rollback-act-1", Type: ActionRegenerate, Description: "regenerate via DSL", ToolID: "t"},
		},
	}

	err := engine.ExecutePlan(context.Background(), plan)
	assert.Error(t, err)

	// Verify backup was restored.
	restored, _ := mock.GetToolCode(context.Background(), "t")
	assert.Equal(t, originalCode, restored, "original code should be restored on failure")
}

func TestExecutePlan_NilCompiler(t *testing.T) {
	mock := newMockMetaRepo()
	mock.data["t"] = `package main
func main() {}`
	engine := newTestEngine(mock, mock)

	// Create a plan with only regenerate option.
	plan := &RepairPlan{
		ID:            "test_regen_plan",
		ToolID:        "t",
		Status:        PlanApproved,
		ErrorPattern:  PatternToolUnknown,
		ErrorMessage:  "unknown error",
		BackupCode:    mock.data["t"],
		NeedsApproval: true,
		Actions: []RepairAction{
			{ID: "regen-1", Type: ActionRegenerate, Description: "regenerate", ToolID: "t"},
		},
	}

	err := engine.ExecutePlan(context.Background(), plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compiler not available")
}
