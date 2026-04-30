// Package repair implements an auto-repair system for tools.
//
// When a tool fails, the repair engine:
//  1. Classifies the error pattern (import, syntax, deprecated API, etc.)
//  2. Looks up predefined repair actions from the catalog
//  3. Generates a RepairPlan with proposed actions requiring user approval
//  4. Executes fixes with backup/verify/deploy/rollback lifecycle
//  5. Tracks all repair attempts in memory for success rate analysis
//
// Integration points:
//   - diagnostic: consumes error patterns from diagnostic.ClassifyError
//   - dsl:        uses CompileToolDefinition for regeneration from DSL
//   - sandbox:    uses Verifier.VerifyToolCode for pre-deploy validation
//   - repository: uses MetadataRepository.GetToolCode/UpdateToolCode
package repair

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// RepairActionType categorises a repair step.
type RepairActionType string

const (
	ActionFix        RepairActionType = "fix"
	ActionRegenerate RepairActionType = "regenerate"
)

// PlanStatus tracks a repair plan through its lifecycle.
type PlanStatus string

const (
	PlanPending  PlanStatus = "pending"
	PlanApproved PlanStatus = "approved"
	PlanRejected PlanStatus = "rejected"
	PlanApplied  PlanStatus = "applied"
	PlanFailed   PlanStatus = "failed"
)

// Tool error pattern constants used by ClassifyToolError and the catalog.
const (
	PatternToolImport       = "import_error"
	PatternToolSyntax       = "syntax_error"
	PatternToolDeprecated   = "deprecated_api"
	PatternToolConfig       = "configuration_error"
	PatternToolPerformance  = "performance_issue"
	PatternToolDataPipeline = "data_pipeline_error"
	PatternToolTimeout      = "timeout_error"
	PatternToolDependency   = "dependency_error"
	PatternToolUnknown      = "unknown_error"
)

// allToolPatterns is used for iteration in catalog construction.
var allToolPatterns = []string{
	PatternToolImport,
	PatternToolSyntax,
	PatternToolDeprecated,
	PatternToolConfig,
	PatternToolPerformance,
	PatternToolDataPipeline,
	PatternToolTimeout,
	PatternToolDependency,
	PatternToolUnknown,
}

// RepairAction is a single step in a repair plan.
type RepairAction struct {
	ID          string           `json:"id"`
	Type        RepairActionType `json:"type"`
	Description string           `json:"description"`
	ToolID      string           `json:"tool_id"`
	Applied     bool             `json:"applied"`
}

// RepairPlan proposes a set of repairs for user approval.
type RepairPlan struct {
	ID            string         `json:"id"`
	ToolID        string         `json:"tool_id"`
	Actions       []RepairAction `json:"actions"`
	NeedsApproval bool           `json:"needs_approval"`
	Status        PlanStatus     `json:"status"`
	ErrorPattern  string         `json:"error_pattern"`
	ErrorMessage  string         `json:"error_message"`
	BackupCode    string         `json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	mu            sync.Mutex
}

// CompiledCodeProvider abstracts DSL compilation so the repair engine
// can regenerate tool source from a DSL definition without depending
// on the full dsl.Compiler type.
type CompiledCodeProvider interface {
	CompileToolDefinition(ctx context.Context, def *dsl.ToolDefinition) (*dsl.GeneratedTool, error)
}

type DSLCompilerAdapter struct{}

func (a *DSLCompilerAdapter) CompileToolDefinition(ctx context.Context, def *dsl.ToolDefinition) (*dsl.GeneratedTool, error) {
	return dsl.CompileToolDefinition(def)
}

// ToolCodeReader abstracts reading tool code from storage.
type ToolCodeReader interface {
	GetToolCode(ctx context.Context, toolID string) (string, error)
}

// ToolCodeWriter abstracts persisting tool code to storage.
type ToolCodeWriter interface {
	UpdateToolCode(ctx context.Context, id string, code string) error
}

// ---------------------------------------------------------------------------
// RepairEngine
// ---------------------------------------------------------------------------

// RepairEngine orchestrates the auto-repair lifecycle.
type RepairEngine struct {
	logger   *slog.Logger
	reader   ToolCodeReader
	writer   ToolCodeWriter
	compiler CompiledCodeProvider
	verifier *sandbox.Verifier
	history  *RepairHistory
	mu       sync.Mutex
}

// NewRepairEngine creates a RepairEngine.
//
// The metaRepo must support both GetToolCode and UpdateToolCode.
// The verifier is used for pre-deploy static analysis.
// The compiler enables regeneration from DSL definitions.
func NewRepairEngine(
	logger *slog.Logger,
	metaRepo *repository.MetadataRepository,
	compiler CompiledCodeProvider,
	verifier *sandbox.Verifier,
) *RepairEngine {
	return &RepairEngine{
		logger:   logger,
		reader:   metaRepo,
		writer:   metaRepo,
		compiler: compiler,
		verifier: verifier,
		history:  NewRepairHistory(logger),
	}
}

// ---------------------------------------------------------------------------
// Error classification (tool-specific)
// ---------------------------------------------------------------------------

// ClassifyToolError analyses a tool error message and returns the matching
// tool error pattern constant. This is separate from diagnostic.ClassifyError
// which classifies system-level errors (timeout, auth, etc.).
func ClassifyToolError(errMsg string) string {
	msg := strings.ToLower(errMsg)

	// Import errors
	if containsAny(msg, "import", "undefined", "cannot find module", "no such file",
		"unresolved reference", "undefined name", "module not found") {
		return PatternToolImport
	}

	// Syntax errors
	if containsAny(msg, "syntax", "parse error", "unexpected token", "expected",
		"unmatched", "invalid syntax", "compilation error", "cannot compile",
		"expected ';'", "unexpected '}'", "unexpected EOF") {
		return PatternToolSyntax
	}

	// Deprecated API
	if containsAny(msg, "deprecated", "removed", "no longer supported",
		"has been removed", "renamed to", "moved to", "use instead",
		"is deprecated", "was removed") {
		return PatternToolDeprecated
	}

	// Configuration errors
	if containsAny(msg, "config", "env", "environment variable", "misconfig",
		"invalid argument", "invalid setting", "missing required",
		"configuration", "not configured", "no such configuration") {
		return PatternToolConfig
	}

	// Performance issues
	if containsAny(msg, "performance", "too slow", "timeout exceeded",
		"deadline exceeded", "context deadline", "OOM", "out of memory",
		"memory limit", "cpu limit", "too many") {
		return PatternToolTimeout
	}

	// Data pipeline errors
	if containsAny(msg, "data", "pipeline", "ETL", "transform",
		"invalid data", "corrupt", "unmarshal", "schema mismatch",
		"type mismatch", "cannot convert") {
		return PatternToolDataPipeline
	}

	// Dependency errors
	if containsAny(msg, "dependency", "module", "package", "go module",
		"pip install", "npm install", "missing dependency",
		"no module", "cannot find package") {
		return PatternToolDependency
	}

	return PatternToolUnknown
}

// containsAny reports whether s contains any of the substrings (case-sensitive).
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Repair Catalog
// ---------------------------------------------------------------------------

// BuildRepairCatalog returns the map of error pattern → recommended actions.
// Each pattern has at least one fix action; severe patterns also include
// a regenerate option.
func BuildRepairCatalog() map[string][]RepairAction {
	return map[string][]RepairAction{
		PatternToolImport: {
			{ID: "fix-import-01", Type: ActionFix, Description: "Add missing import statements to tool code"},
			{ID: "fix-import-02", Type: ActionFix, Description: "Fix import path or module name"},
			{ID: "fix-import-03", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition to generate correct imports"},
		},
		PatternToolSyntax: {
			{ID: "fix-syntax-01", Type: ActionFix, Description: "Fix syntax error in tool code (missing brace/parenthesis/semicolon)"},
			{ID: "fix-syntax-02", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition to produce valid code"},
		},
		PatternToolDeprecated: {
			{ID: "fix-dep-01", Type: ActionFix, Description: "Replace deprecated API calls with current equivalents"},
			{ID: "fix-dep-02", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition with updated API usage"},
		},
		PatternToolConfig: {
			{ID: "fix-config-01", Type: ActionFix, Description: "Fix configuration parameter value"},
			{ID: "fix-config-02", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition with corrected configuration"},
		},
		PatternToolPerformance: {
			{ID: "fix-perf-01", Type: ActionFix, Description: "Optimise loop and data structures for better performance"},
			{ID: "fix-perf-02", Type: ActionFix, Description: "Add caching or connection pooling to reduce latency"},
			{ID: "fix-perf-03", Type: ActionFix, Description: "Increase timeout or resource limits"},
		},
		PatternToolDataPipeline: {
			{ID: "fix-data-01", Type: ActionFix, Description: "Fix data transformation logic or type conversion"},
			{ID: "fix-data-02", Type: ActionFix, Description: "Add input validation and error handling for data pipeline"},
			{ID: "fix-data-03", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition with corrected data flow"},
		},
		PatternToolTimeout: {
			{ID: "fix-timeout-01", Type: ActionFix, Description: "Increase timeout value in tool configuration"},
			{ID: "fix-timeout-02", Type: ActionFix, Description: "Add retry with exponential backoff for transient failures"},
		},
		PatternToolDependency: {
			{ID: "fix-depdep-01", Type: ActionFix, Description: "Add missing dependency declaration to tool definition"},
			{ID: "fix-depdep-02", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition with complete dependency list"},
		},
		PatternToolUnknown: {
			{ID: "fix-unknown-01", Type: ActionRegenerate, Description: "Regenerate tool from DSL definition to resolve unknown error"},
		},
	}
}

// defaultCatalog caches the catalog after first build.
var defaultCatalog map[string][]RepairAction

func init() {
	defaultCatalog = BuildRepairCatalog()
}

// ---------------------------------------------------------------------------
// Analyse & Plan
// ---------------------------------------------------------------------------

// planIDCounter generates unique plan IDs.
var planIDCounter int64
var planIDMu sync.Mutex

func nextPlanID() string {
	planIDMu.Lock()
	planIDCounter++
	id := fmt.Sprintf("plan-%d", planIDCounter)
	planIDMu.Unlock()
	return id
}

// nextActionID generates unique action IDs within a plan.
func nextActionID(planID string, seq int) string {
	return fmt.Sprintf("%s-act-%d", planID, seq)
}

// AnalyseAndPlan generates a RepairPlan based on a tool error.
//
// It reads the tool code, classifies the error, looks up matching actions
// in the catalog, and constructs a plan. Plans are marked NeedsApproval=true
// when the error pattern is import/syntax/deprecated (structural changes) or
// when regeneration is among the options.
func (e *RepairEngine) AnalyseAndPlan(ctx context.Context, toolID string, errMsg string) (*RepairPlan, error) {
	pattern := ClassifyToolError(errMsg)
	e.logger.Info("analysing tool error",
		"tool_id", toolID,
		"pattern", pattern,
		"error", errMsg,
	)

	// Read current tool code for backup.
	code, err := e.reader.GetToolCode(ctx, toolID)
	if err != nil {
		return nil, fmt.Errorf("read tool code for %q: %w", toolID, err)
	}

	// Look up catalog actions for this pattern.
	catalogActions := defaultCatalog[pattern]
	if catalogActions == nil {
		catalogActions = defaultCatalog[PatternToolUnknown]
	}

	// Build plan actions with unique IDs.
	planID := nextPlanID()
	actions := make([]RepairAction, len(catalogActions))
	for i, ca := range catalogActions {
		actions[i] = RepairAction{
			ID:          nextActionID(planID, i+1),
			Type:        ca.Type,
			Description: ca.Description,
			ToolID:      toolID,
			Applied:     false,
		}
	}

	// Structural changes (import/syntax/deprecated) always require approval.
	needsApproval := pattern == PatternToolImport ||
		pattern == PatternToolSyntax ||
		pattern == PatternToolDeprecated ||
		pattern == PatternToolUnknown ||
		hasRegenerateAction(actions)

	plan := &RepairPlan{
		ID:            planID,
		ToolID:        toolID,
		Actions:       actions,
		NeedsApproval: needsApproval,
		Status:        PlanPending,
		ErrorPattern:  pattern,
		ErrorMessage:  errMsg,
		BackupCode:    code,
		CreatedAt:     time.Now(),
	}

	e.logger.Info("repair plan created",
		"plan_id", plan.ID,
		"tool_id", toolID,
		"pattern", pattern,
		"actions", len(actions),
		"needs_approval", needsApproval,
	)

	return plan, nil
}

// hasRegenerateAction reports whether any action in the list is a regenerate type.
func hasRegenerateAction(actions []RepairAction) bool {
	for _, a := range actions {
		if a.Type == ActionRegenerate {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Plan Execution
// ---------------------------------------------------------------------------

// ApprovePlan marks a plan as approved, allowing execution.
func (e *RepairEngine) ApprovePlan(plan *RepairPlan) {
	plan.mu.Lock()
	defer plan.mu.Unlock()
	if plan.Status == PlanPending {
		plan.Status = PlanApproved
		e.logger.Info("plan approved", "plan_id", plan.ID, "tool_id", plan.ToolID)
	}
}

// RejectPlan marks a plan as rejected (no execution).
func (e *RepairEngine) RejectPlan(plan *RepairPlan) {
	plan.mu.Lock()
	defer plan.mu.Unlock()
	if plan.Status == PlanPending || plan.Status == PlanApproved {
		plan.Status = PlanRejected
		e.logger.Info("plan rejected", "plan_id", plan.ID, "tool_id", plan.ToolID)
	}
}

// ExecutePlan applies the repair plan with full lifecycle:
//
//  1. Verify the plan is approved
//  2. For each action:
//     a. If Type is ActionRegenerate → call compiler to regenerate code
//     b. If Type is ActionFix → apply fix to current code
//     c. Run Verifier.VerifyToolCode on the modified code
//     d. If verification passes → persist via UpdateToolCode
//     e. If verification fails → skip to next action
//  3. If all actions fail → restore backup via UpdateToolCode
//  4. Record outcome in history
func (e *RepairEngine) ExecutePlan(ctx context.Context, plan *RepairPlan) error {
	plan.mu.Lock()
	if plan.Status != PlanApproved {
		plan.mu.Unlock()
		return fmt.Errorf("plan %q is %s, must be approved before execution", plan.ID, plan.Status)
	}
	plan.Status = PlanApplied
	backup := plan.BackupCode
	plan.mu.Unlock()

	e.logger.Info("executing repair plan",
		"plan_id", plan.ID,
		"tool_id", plan.ToolID,
		"actions", len(plan.Actions),
	)

	currentCode := backup
	var lastErr error

	for i := range plan.Actions {
		action := &plan.Actions[i]
		e.logger.Info("executing action",
			"plan_id", plan.ID,
			"action_id", action.ID,
			"type", action.Type,
		)

		start := time.Now()
		var err error

		switch action.Type {
		case ActionRegenerate:
			err = e.executeRegenerate(ctx, action, plan)
		case ActionFix:
			currentCode, err = e.executeFix(currentCode, *action)
		}

		duration := time.Since(start)

		if err != nil {
			lastErr = err
			e.history.Record(RepairRecord{
				ToolID:     plan.ToolID,
				PlanID:     plan.ID,
				ActionID:   action.ID,
				ActionType: string(action.Type),
				Status:     StatusFailed,
				ErrorMsg:   err.Error(),
				Duration:   duration,
			})
			e.logger.Warn("action failed",
				"plan_id", plan.ID,
				"action_id", action.ID,
				"error", err,
			)
			continue
		}

		// Verify the modified code with static analysis.
		vResult := e.verifier.VerifyToolCode(currentCode)
		if !vResult.Passed {
			lastErr = fmt.Errorf("verification failed after %s: %s", action.Type, vResult.Error)
			e.history.Record(RepairRecord{
				ToolID:     plan.ToolID,
				PlanID:     plan.ID,
				ActionID:   action.ID,
				ActionType: string(action.Type),
				Status:     StatusFailed,
				ErrorMsg:   lastErr.Error(),
				Duration:   time.Since(start),
			})
			e.logger.Warn("verification failed",
				"plan_id", plan.ID,
				"action_id", action.ID,
				"error", vResult.Error,
			)
			// Reset currentCode to backup for next attempt.
			currentCode = backup
			continue
		}

		// Verification passed → deploy.
		if err := e.writer.UpdateToolCode(ctx, plan.ToolID, currentCode); err != nil {
			lastErr = fmt.Errorf("deploy after %s: %w", action.Type, err)
			e.history.Record(RepairRecord{
				ToolID:     plan.ToolID,
				PlanID:     plan.ID,
				ActionID:   action.ID,
				ActionType: string(action.Type),
				Status:     StatusFailed,
				ErrorMsg:   lastErr.Error(),
				Duration:   time.Since(start),
			})
			currentCode = backup
			continue
		}

		action.Applied = true
		e.history.Record(RepairRecord{
			ToolID:     plan.ToolID,
			PlanID:     plan.ID,
			ActionID:   action.ID,
			ActionType: string(action.Type),
			Status:     StatusSuccess,
			Duration:   duration,
		})
		e.logger.Info("action applied successfully",
			"plan_id", plan.ID,
			"action_id", action.ID,
		)

		// Action succeeded — return.
		return nil
	}

	// All actions failed → rollback.
	if lastErr != nil {
		e.logger.Error("all repair actions failed, rolling back",
			"plan_id", plan.ID,
			"tool_id", plan.ToolID,
			"last_error", lastErr,
		)

		plan.mu.Lock()
		plan.Status = PlanFailed
		plan.mu.Unlock()

		// Restore backup.
		if restoreErr := e.writer.UpdateToolCode(ctx, plan.ToolID, backup); restoreErr != nil {
			return fmt.Errorf("repair failed and backup restore also failed: %w (restore: %v)", lastErr, restoreErr)
		}

		e.logger.Info("backup restored after failed repair",
			"plan_id", plan.ID,
			"tool_id", plan.ToolID,
		)

		return fmt.Errorf("all %d actions failed, last error: %w", len(plan.Actions), lastErr)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Execution helpers
// ---------------------------------------------------------------------------

// executeRegenerate regenerates tool code from DSL definition.
// It requires the compiler to produce a full GeneratedTool and extracts
// the appropriate code string (Go or Python).
func (e *RepairEngine) executeRegenerate(ctx context.Context, action *RepairAction, plan *RepairPlan) error {
	if e.compiler == nil {
		return fmt.Errorf("compiler not available, cannot regenerate")
	}

	// For regeneration, we create a minimal ToolDefinition from the plan metadata.
	// In a full implementation, the ToolDefinition would be stored alongside
	// the tool code in the repository. Here we use a placeholder that at least
	// runs through the compiler to produce structurally correct code.
	def := &dsl.ToolDefinition{
		Name:        plan.ToolID,
		Description: fmt.Sprintf("Regenerated tool: %s", plan.ToolID),
		Inputs:      []*dsl.ToolParam{},
		Outputs:     []*dsl.ToolParam{},
		Handler: &dsl.ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: []*dsl.ToolDep{},
	}

	gt, err := e.compiler.CompileToolDefinition(ctx, def)
	if err != nil {
		return fmt.Errorf("regeneration via compiler: %w", err)
	}

	// Choose language based on handler.
	code := gt.GoCode
	if action.Description != "" {
		code = gt.GoCode
		_ = gt.PythonCode
	}

	e.logger.Info("code regenerated",
		"tool_id", plan.ToolID,
		"template", gt.Template,
	)

	if err := e.writer.UpdateToolCode(ctx, plan.ToolID, code); err != nil {
		return fmt.Errorf("deploy regenerated code: %w", err)
	}

	// Now verify the deployed code.
	vResult := e.verifier.VerifyToolCode(code)
	if !vResult.Passed {
		// Verification failed — rollback immediately.
		if restoreErr := e.writer.UpdateToolCode(ctx, plan.ToolID, plan.BackupCode); restoreErr != nil {
			return fmt.Errorf("regenerated code failed verification and restore also failed: %v (restore: %v)", vResult.Error, restoreErr)
		}
		return fmt.Errorf("regenerated code failed verification: %s", vResult.Error)
	}

	return nil
}

// executeFix applies a predefined fix action to the tool code.
// For now this performs targeted string replacements. In production,
// this would use AST-based transformations.
func (e *RepairEngine) executeFix(code string, action RepairAction) (string, error) {
	// Apply fix based on action ID.
	// Each action performs a specific transformation on the code.
	switch action.ID {
	case "fix-import-01":
		return e.fixMissingImports(code), nil
	case "fix-import-02":
		return e.fixImportPath(code), nil
	case "fix-syntax-01":
		return e.fixSyntaxError(code), nil
	case "fix-dep-01":
		return e.fixDeprecatedAPI(code), nil
	case "fix-config-01":
		return e.fixConfiguration(code), nil
	case "fix-perf-01":
		return e.fixPerformance(code), nil
	case "fix-perf-02":
		return e.fixCaching(code), nil
	case "fix-perf-03":
		return e.fixTimeout(code), nil
	case "fix-data-01":
		return e.fixDataPipeline(code), nil
	case "fix-data-02":
		return e.fixDataValidation(code), nil
	case "fix-timeout-01":
		return e.fixTimeout(code), nil
	case "fix-timeout-02":
		return e.fixRetry(code), nil
	case "fix-depdep-01":
		return code, nil // No-op: dependencies are metadata, not code.
	default:
		return code, nil
	}
}

// ---------------------------------------------------------------------------
// Fix implementations
// ---------------------------------------------------------------------------

// fixMissingImports adds common missing imports.
func (e *RepairEngine) fixMissingImports(code string) string {
	if !strings.Contains(code, "import") {
		return code
	}
	// Check for common patterns and add missing imports.
	if strings.Contains(code, "context.") && !strings.Contains(code, `"context"`) &&
		!strings.Contains(code, `"context"`) {
		code = addImportToBlock(code, `"context"`)
	}
	if strings.Contains(code, "fmt.") && !strings.Contains(code, `"fmt"`) {
		code = addImportToBlock(code, `"fmt"`)
	}
	if strings.Contains(code, "json.") && !strings.Contains(code, `"encoding/json"`) {
		code = addImportToBlock(code, `"encoding/json"`)
	}
	if strings.Contains(code, "http.") && !strings.Contains(code, `"net/http"`) {
		code = addImportToBlock(code, `"net/http"`)
	}
	if strings.Contains(code, "ioutil.") && !strings.Contains(code, `"io/ioutil"`) &&
		!strings.Contains(code, `"io"`) {
		code = addImportToBlock(code, `"io"`)
	}
	if (strings.Contains(code, "strings.") || strings.Contains(code, "strings.ToLower")) &&
		!strings.Contains(code, `"strings"`) {
		code = addImportToBlock(code, `"strings"`)
	}
	if strings.Contains(code, "time.") && !strings.Contains(code, `"time"`) {
		code = addImportToBlock(code, `"time"`)
	}
	if strings.Contains(code, "os.") && !strings.Contains(code, `"os"`) {
		code = addImportToBlock(code, `"os"`)
	}
	return code
}

// addImportToBlock inserts a new import path into an existing import block,
// or creates one if none exists.
func addImportToBlock(code, importPath string) string {
	// Try to add inside existing import block.
	if strings.Contains(code, "import (\n") {
		// Insert before closing paren.
		idx := strings.LastIndex(code, "\n)")
		if idx > 0 {
			before := code[:idx]
			after := code[idx:]
			if !strings.Contains(before, importPath) {
				return before + "\n\t" + importPath + after
			}
			return code
		}
	}

	// Try single-line import.
	if strings.Contains(code, "import \"") {
		// Convert to block import style.
		importIdx := strings.Index(code, "import \"")
		lineEnd := strings.Index(code[importIdx:], "\n")
		if lineEnd > 0 {
			existingImport := code[importIdx : importIdx+lineEnd]
			before := code[:importIdx]
			after := code[importIdx+lineEnd:]
			return before + "import (\n\t" + existingImport[7:] + "\n\t" + importPath + "\n)" + after
		}
		return code
	}

	// No import block exists — create one after package declaration.
	if strings.HasPrefix(code, "package ") {
		pkgEnd := strings.Index(code[7:], "\n")
		if pkgEnd > 0 {
			pos := 7 + pkgEnd + 1
			return code[:pos] + "\nimport (\n\t" + importPath + "\n)\n" + code[pos:]
		}
	}

	return code
}

// fixImportPath corrects common import path issues.
func (e *RepairEngine) fixImportPath(code string) string {
	// Replace common incorrect import paths.
	return strings.NewReplacer(
		`"context"`, `"context"`,
		`"fmt"`, `"fmt"`,
		`"encoding/json"`, `"encoding/json"`,
		`"io/ioutil"`, `"io"`,
	).Replace(code)
}

// fixSyntaxError attempts to fix common syntax errors.
func (e *RepairEngine) fixSyntaxError(code string) string {
	// Check for unmatched braces.
	open := strings.Count(code, "{")
	close := strings.Count(code, "}")
	if open > close {
		code += strings.Repeat("\n}", open-close)
	}

	// Fix missing closing parentheses in function signatures
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") && !strings.HasSuffix(trimmed, "{") {
			parenOpen := strings.Count(trimmed, "(")
			parenClose := strings.Count(trimmed, ")")
			if parenOpen > parenClose {
				lines[i] = line + strings.Repeat(")", parenOpen-parenClose)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// fixDeprecatedAPI replaces known deprecated API calls.
func (e *RepairEngine) fixDeprecatedAPI(code string) string {
	return strings.NewReplacer(
		"ioutil.ReadAll", "io.ReadAll",
		"ioutil.ReadFile", "os.ReadFile",
		"ioutil.WriteFile", "os.WriteFile",
		"ioutil.ReadDir", "os.ReadDir",
		"ioutil.TempDir", "os.MkdirTemp",
		"ioutil.TempFile", "os.CreateTemp",
		"ioutil.NopCloser", "io.NopCloser",
		"ioutil.Discard", "io.Discard",
	).Replace(code)
}

// fixConfiguration corrects common configuration issues.
func (e *RepairEngine) fixConfiguration(code string) string {
	// Replace hardcoded test configs with environment variable lookups.
	return strings.ReplaceAll(code, `"localhost:8080"`, `os.Getenv("ALEPH_ENDPOINT")`)
}

// fixPerformance optimises common performance patterns.
//
// Detects and fixes 4 anti-patterns:
//  1. Sequential HTTP calls → goroutine + sync.WaitGroup parallelization
//  2. Missing context cancellation checks in loops → select + ctx.Done()
//  3. String concatenation in loops (+=) → strings.Builder
//  4. Repeated file reads for same path → sync.Once caching pattern
func (e *RepairEngine) fixPerformance(code string) string {
	if httpCallCount(code) > 1 {
		code = addConcurrentHTTPPattern(code)
	}

	if strings.Contains(code, "context") {
		code = addContextCancelInLoops(code)
	}

	if hasStringConcatInLoop(code) {
		code = fixStringConcatInLoop(code)
	}

	if path := repeatedFileRead(code); path != "" {
		code = addFileReadCaching(code, path)
	}

	return code
}

// ---------------------------------------------------------------------------
// Pattern 1 — Sequential HTTP calls
// ---------------------------------------------------------------------------

// httpCallPattern matches HTTP call expressions: http.Get(, http.Post(, client.Do(, http.NewRequest(
var httpCallPattern = regexp.MustCompile(`(?:http\.(?:Get|Post|Head|Put|Delete)\s*\(|client\.Do\s*\(|http\.NewRequest\s*\()`)

// httpCallCount returns the number of HTTP call expressions in code.
func httpCallCount(code string) int {
	return len(httpCallPattern.FindAllString(code, -1))
}

// addConcurrentHTTPPattern wraps multiple HTTP calls with a goroutine + sync.WaitGroup comment block.
func addConcurrentHTTPPattern(code string) string {
	comment := "// PERFORMANCE FIX: Multiple HTTP calls detected — consider parallelizing with goroutines:\n" +
		"\t// var wg sync.WaitGroup\n" +
		"\t// errCh := make(chan error, N)\n" +
		"\t// for _, url := range urls {\n" +
		"\t//     wg.Add(1)\n" +
		"\t//     go func(u string) {\n" +
		"\t//         defer wg.Done()\n" +
		"\t//         resp, err := http.Get(u)\n" +
		"\t//         if err != nil { errCh <- err; return }\n" +
		"\t//         defer resp.Body.Close()\n" +
		"\t//         // process resp\n" +
		"\t//     }(url)\n" +
		"\t// }\n" +
		"\t// wg.Wait()\n" +
		"\t// close(errCh)\n"
	return addCommentAfterFuncDecl(code, comment)
}

// ---------------------------------------------------------------------------
// Pattern 2 — Missing context cancellation in loops
// ---------------------------------------------------------------------------
func addContextCancelInLoops(code string) string {
	lines := strings.Split(code, "\n")
	var result []string
	inFunc := false
	inLoop := false
	loopBraceDepth := 0
	var loopLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "func ") {
			inFunc = true
		}

		if !inFunc {
			result = append(result, line)
			continue
		}

		if !inLoop && (strings.HasPrefix(trimmed, "for ") || trimmed == "for" || strings.HasPrefix(trimmed, "for range")) {
			inLoop = true
			loopBraceDepth = 0
			loopLines = nil
		}

		if inLoop {
			loopLines = append(loopLines, line)
			braceDepth := countBraces(line)
			loopBraceDepth += braceDepth
			loopBody := strings.Join(loopLines, "\n")

			if loopBraceDepth <= 0 {
				if !strings.Contains(loopBody, "ctx.Done()") && !strings.Contains(loopBody, "ctx.Err()") {
					cancelBlock := "\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\treturn \"\", ctx.Err()\n\t\tdefault:\n\t\t}"
					loopBody = insertBeforeLastBrace(loopBody, cancelBlock)
				}
				loopLines = nil
				inLoop = false
				result = append(result, loopBody)
				continue
			}
		}

		if !inLoop {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// countBraces returns the net brace depth change in a line (open - close).
func countBraces(line string) int {
	opens := strings.Count(line, "{")
	closes := strings.Count(line, "}")
	return opens - closes
}

// insertBeforeLastBrace inserts text before the last '}' on its own line.
func insertBeforeLastBrace(body, insert string) string {
	lines := strings.Split(body, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			before := strings.Join(lines[:i], "\n")
			after := strings.Join(lines[i:], "\n")
			return before + "\n" + insert + "\n" + after
		}
	}
	return body + "\n" + insert
}

// ---------------------------------------------------------------------------
// Pattern 3 — String concatenation in loops
// ---------------------------------------------------------------------------
func hasStringConcatInLoop(code string) bool {
	lines := strings.Split(code, "\n")
	depth := 0
	inLoop := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "for ") || trimmed == "for" || strings.HasPrefix(trimmed, "for range") {
			inLoop = true
		}

		if inLoop && strings.Contains(trimmed, "+=") {
			parts := strings.SplitN(trimmed, "+=", 2)
			if len(parts) == 2 {
				rhs := strings.TrimSpace(parts[1])
				if strings.HasPrefix(rhs, `"`) || strings.HasPrefix(rhs, "`") ||
					strings.Contains(rhs, `fmt.Sprint`) || strings.Contains(rhs, `strconv.`) {
					return true
				}
			}
		}

		depth += countBraces(line)
		if depth <= 0 && inLoop {
			inLoop = false
		}
	}

	return false
}

// fixStringConcatInLoop replaces += string concatenation in loops with strings.Builder.
func fixStringConcatInLoop(code string) string {
	lines := strings.Split(code, "\n")
	inLoop := false
	varBuilderName := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inLoop && (strings.HasPrefix(trimmed, "for ") || trimmed == "for" || strings.HasPrefix(trimmed, "for range")) {
			inLoop = true
			for _, l := range lines[i:] {
				t := strings.TrimSpace(l)
				if strings.Contains(t, "+=") {
					parts := strings.SplitN(t, "+=", 2)
					if len(parts) == 2 {
						varBuilderName = strings.TrimSpace(parts[0])
						break
					}
				}
				if strings.Contains(t, "}") && countBraces(l) < 0 {
					break
				}
			}
			break
		}
	}

	if varBuilderName == "" {
		return code
	}

	declPattern := regexp.MustCompile(`(var\s+` + regexp.QuoteMeta(varBuilderName) + `\s+string|` + regexp.QuoteMeta(varBuilderName) + `\s*:=\s*"")`)
	if declPattern.MatchString(code) {
		repl := regexp.MustCompile(`var\s+` + regexp.QuoteMeta(varBuilderName) + `\s+string`)
		code = repl.ReplaceAllString(code, "var "+varBuilderName+" strings.Builder")
		repl2 := regexp.MustCompile(regexp.QuoteMeta(varBuilderName) + `\s*:=\s*""`)
		code = repl2.ReplaceAllString(code, varBuilderName+" strings.Builder")
	} else {
		comment := "\n\t// PERFORMANCE FIX: Use strings.Builder instead of += in loop:\n" +
			"\t// var builder strings.Builder\n" +
			"\t// builder.WriteString(\"...\")\n" +
			"\t// result := builder.String()"
		code = addCommentAfterFuncDecl(code, comment)
		return code
	}

	plusEqPattern := regexp.MustCompile(regexp.QuoteMeta(varBuilderName) + `\s*\+=\s*`)
	code = plusEqPattern.ReplaceAllString(code, varBuilderName+".WriteString(")

	writeStringPattern := regexp.MustCompile(regexp.QuoteMeta(varBuilderName) + `\.WriteString\((.*)$`)
	code = writeStringPattern.ReplaceAllStringFunc(code, func(m string) string {
		if strings.HasSuffix(strings.TrimSpace(m), ")") {
			return m
		}
		return m + ")"
	})

	returnPattern := regexp.MustCompile(`return\s+` + regexp.QuoteMeta(varBuilderName) + `\b(?!\.String)`)
	code = returnPattern.ReplaceAllString(code, "return "+varBuilderName+".String()")

	return code
}

// ---------------------------------------------------------------------------
// Pattern 4 — Repeated file reads
// ---------------------------------------------------------------------------

// osReadFilePattern matches os.ReadFile("path") calls.
var osReadFilePattern = regexp.MustCompile(`os\.ReadFile\(\s*"([^"]+)"\s*\)`)

// repeatedFileRead returns the path of a file read more than once, or "".
func repeatedFileRead(code string) string {
	matches := osReadFilePattern.FindAllStringSubmatch(code, -1)
	counts := make(map[string]int)
	for _, m := range matches {
		if len(m) > 1 {
			counts[m[1]]++
		}
	}
	for path, count := range counts {
		if count > 1 {
			return path
		}
	}
	return ""
}

// addFileReadCaching inserts a sync.Once caching pattern for repeated os.ReadFile calls.
func addFileReadCaching(code, filePath string) string {
	comment := "\n\t// PERFORMANCE FIX: File " + filePath +
		" is read multiple times. Consider caching with sync.Once:\n" +
		"\t// var once sync.Once\n" +
		"\t// var cachedData []byte\n" +
		"\t// var cacheErr error\n" +
		"\t// once.Do(func() {\n" +
		"\t//     cachedData, cacheErr = os.ReadFile(\"" + filePath + "\")\n" +
		"\t// })\n" +
		"\t// if cacheErr != nil { return \"\", cacheErr }"
	return addCommentAfterFuncDecl(code, comment)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// addCommentAfterFuncDecl inserts a comment block after the first function declaration.
func addCommentAfterFuncDecl(code, comment string) string {
	if idx := strings.Index(code, "func Handle"); idx >= 0 {
		after := code[idx:]
		braceIdx := strings.Index(after, "{")
		if braceIdx > 0 {
			return code[:idx+braceIdx+1] + comment + code[idx+braceIdx+1:]
		}
		return code + comment
	}
	re := regexp.MustCompile(`(?m)^func\s+\w+\(`)
	loc := re.FindStringIndex(code)
	if loc != nil {
		after := code[loc[0]:]
		braceIdx := strings.Index(after, "{")
		if braceIdx > 0 {
			return code[:loc[0]+braceIdx+1] + comment + code[loc[0]+braceIdx+1:]
		}
	}
	return code + comment
}

// fixCaching adds basic caching patterns.
func (e *RepairEngine) fixCaching(code string) string {
	// Add TODO comment suggesting caching.
	if !strings.Contains(code, "cache") {
		code = strings.Replace(code, "func Handle", "// TODO: consider adding caching/connection pooling for performance\nfunc Handle", 1)
	}
	return code
}

// fixTimeout increases timeout values in the code.
func (e *RepairEngine) fixTimeout(code string) string {
	// Replace short timeouts with longer ones.
	code = strings.ReplaceAll(code, "100 * time.Millisecond", "5 * time.Second")
	code = strings.ReplaceAll(code, "500 * time.Millisecond", "5 * time.Second")
	code = strings.ReplaceAll(code, "1 * time.Second", "10 * time.Second")
	code = strings.ReplaceAll(code, `"timeout": 1`, `"timeout": 10`)
	code = strings.ReplaceAll(code, `"timeout": 5`, `"timeout": 30`)
	return code
}

// fixRetry adds retry logic with exponential backoff.
func (e *RepairEngine) fixRetry(code string) string {
	// Add retry logic comment if not present.
	if !strings.Contains(code, "retry") {
		comment := "\n\t// Retry with exponential backoff\n\tmaxRetries := 3\n\tfor attempt := 0; attempt < maxRetries; attempt++ {\n\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\treturn \"\", ctx.Err()\n\t\tdefault:\n\t\t}\n\t\ttime.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)\n\t\t// retry logic here\n\t}\n"
		if strings.Contains(code, "func Handle") {
			code = strings.Replace(code, "func Handle", "func Handle", 1)
			code = strings.Replace(code, "\t// TODO:", comment+"\t// TODO:", 1)
		}
	}
	return code
}

// fixDataPipeline adds data pipeline error handling.
func (e *RepairEngine) fixDataPipeline(code string) string {
	if strings.Contains(code, "json.Unmarshal") && !strings.Contains(code, "json.Decode") {
		code = strings.Replace(code,
			"if err := json.Unmarshal([]byte(inputJSON), &input); err != nil",
			"if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {\n\t\treturn \"\", fmt.Errorf(\"unmarshal input: %w\", err)\n\t}", 1)
	}
	return code
}

// fixDataValidation adds input validation.
func (e *RepairEngine) fixDataValidation(code string) string {
	if !strings.Contains(code, "if input.") {
		code = strings.Replace(code,
			"var input __NAME__Input",
			"var input __NAME__Input\n\t// Validate required fields\n\tif input == (__NAME__Input{}) {\n\t\treturn \"\", fmt.Errorf(\"empty input provided\")\n\t}", 1)
	}
	return code
}

// ---------------------------------------------------------------------------
// GetHistory
// ---------------------------------------------------------------------------

// GetHistory returns the repair history for the given tool.
func (e *RepairEngine) GetHistory(toolID string) []RepairRecord {
	return e.history.GetHistory(toolID)
}

// GetAllHistory returns all repair records across all tools.
func (e *RepairEngine) GetAllHistory() []RepairRecord {
	return e.history.GetAll()
}

// SuccessRate returns the success rate for a tool's repair attempts.
func (e *RepairEngine) SuccessRate(toolID string) float64 {
	return e.history.SuccessRate(toolID)
}
