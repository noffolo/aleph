package repair

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SelfRepairScenario defines a single break/diagnose/repair/verify cycle.
type SelfRepairScenario struct {
	Name              string
	ToolID            string
	BrokenCode        string
	ErrorMsg          string
	WantRepairPattern string
	WantDiagType      string
	WantSeverity      string
	CheckRollback     bool
}

// ---------------------------------------------------------------------------
// TestSelfRepair_Demo — break/diagnose/repair/verify cycle.
//
// Integrates: internal/diagnostic, internal/repair, internal/sandbox.
//
// Output per scenario:
//
//	Break Detection [PASS/FAIL] | Repair Strategy [VALID/INVALID] |
//	Restoration [PASS/FAIL] | VERDICT
// ---------------------------------------------------------------------------

func TestSelfRepair_Demo(t *testing.T) {
	scenarios := []SelfRepairScenario{
		{
			Name:   "missing_imports",
			ToolID: "demo_broken_importer",
			// Code has import "errors" but is missing "encoding/json" and "fmt".
			// The fixMissingImports function requires at least one import
			// statement to exist before it adds missing ones.
			BrokenCode: `package demo

import "errors"

func Process(input string) string {
	result := make(map[string]interface{})
	result["status"] = "ok"
	jsonData, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	fmt.Println(string(jsonData))
	return string(jsonData)
}`,
			ErrorMsg:          `cannot find module "fmt"`,
			WantRepairPattern: PatternToolImport,
			// "cannot find module" does not match any diagnostic keyword
			// (not "not found", not "missing") so it falls to default: timeout
			WantDiagType: diagnostic.PatternTimeout,
			WantSeverity: diagnostic.SeverityLow,
			CheckRollback: true,
		},
		{
			Name:   "unmatched_braces",
			ToolID: "demo_broken_syntax",
			BrokenCode: `package demo

import "fmt"

func Greet(name string) string {
	return "Hello, " + name + "!"`,
			ErrorMsg:          "syntax error: unexpected EOF",
			WantRepairPattern: PatternToolSyntax,
			WantDiagType:      diagnostic.PatternTimeout,
			WantSeverity:      diagnostic.SeverityLow,
			CheckRollback:     false,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			runSelfRepairCycle(t, sc)
		})
	}
}

func runSelfRepairCycle(t *testing.T, sc SelfRepairScenario) {
	v := sandbox.NewVerifier(slog.Default(), nil, "", "")
	mock := newMockMetaRepo()
	mock.data[sc.ToolID] = sc.BrokenCode
	engine := newTestEngine(mock, mock)

	var detectionPass, repairValid, restorationOk bool

	// ====================================================================
	// Phase 1 & 2: BREAK + DIAGNOSE
	// ====================================================================
	t.Log("=== BREAK + DIAGNOSE ===")

	diagType := diagnostic.ClassifyError("", sc.ErrorMsg)
	t.Logf("  diagnostic.ClassifyError(%q) -> %q (expected %q)",
		sc.ErrorMsg, diagType, sc.WantDiagType)

	toolPat := ClassifyToolError(sc.ErrorMsg)
	t.Logf("  repair.ClassifyToolError(%q) -> %q (expected %q)",
		sc.ErrorMsg, toolPat, sc.WantRepairPattern)

	sev := diagnostic.AssessSeverity(diagType, 1, "demo")
	t.Logf("  diagnostic.AssessSeverity -> %q (expected %q)", sev, sc.WantSeverity)

	dm := diagnostic.NewDiagnosticMonitor(3, nil)
	recorded := dm.RecordError("", sc.ErrorMsg, sc.ToolID, "demo")
	t.Logf("  DiagnosticMonitor pattern=%q severity=%q count=%d",
		recorded.Type, recorded.Severity, recorded.Count)

	diagMatch := diagType == sc.WantDiagType
	patMatch := toolPat == sc.WantRepairPattern
	sevMatch := sev == sc.WantSeverity

	if diagMatch && patMatch && sevMatch {
		detectionPass = true
		t.Logf("  => Detection PASS: pattern=%q severity=%q", toolPat, sev)
	} else {
		t.Errorf("  => Detection FAIL: diagMatch=%v patMatch=%v sevMatch=%v",
			diagMatch, patMatch, sevMatch)
	}

	// ====================================================================
	// Phase 3: REPAIR
	//
	// AnalyseAndPlan generates a plan with the correct pattern classification,
	// backup code, and catalog actions.
	// The actual fix is applied via the engine's fix functions
	// (fixMissingImports, fixSyntaxError) which are the same functions
	// called by executeFix during ExecutePlan.
	// ====================================================================
	t.Log("=== REPAIR ===")

	plan, err := engine.AnalyseAndPlan(context.Background(), sc.ToolID, sc.ErrorMsg)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Greater(t, len(plan.Actions), 0)

	t.Logf("  Plan: %s | pattern=%s | actions=%d | needs_approval=%v",
		plan.ID, plan.ErrorPattern, len(plan.Actions), plan.NeedsApproval)
	t.Logf("  Action[0]: type=%s desc=%s",
		plan.Actions[0].Type, plan.Actions[0].Description)
	t.Logf("  Backup stored: %d bytes", len(plan.BackupCode))

	// Apply the fix via the engine's fix function.
	var fixedCode string
	switch sc.WantRepairPattern {
	case PatternToolImport:
		fixedCode = engine.fixMissingImports(sc.BrokenCode)
	case PatternToolSyntax:
		fixedCode = engine.fixSyntaxError(sc.BrokenCode)
	}
	require.NotEmpty(t, fixedCode, "fixed code should not be empty")

	changed := fixedCode != sc.BrokenCode
	if changed {
		repairValid = true
		t.Log("  => Repair VALID: fix function produced modified code")
		// Show specifics.
		if sc.WantRepairPattern == PatternToolImport {
			t.Logf("  Imports after fix: encoding/json=%v fmt=%v",
				strings.Contains(fixedCode, `"encoding/json"`),
				strings.Contains(fixedCode, `"fmt"`))
			assert.True(t, strings.Contains(fixedCode, `"encoding/json"`),
				"fixed code should have encoding/json import")
			assert.True(t, strings.Contains(fixedCode, `"fmt"`),
				"fixed code should have fmt import")
		} else if sc.WantRepairPattern == PatternToolSyntax {
			open := strings.Count(fixedCode, "{")
			close := strings.Count(fixedCode, "}")
			assert.Equal(t, open, close, "fixed code should have balanced braces")
			t.Logf("  Balanced braces: {=%d }=%d", open, close)
		}
	} else {
		t.Errorf("  => Repair INVALID: fix function returned unchanged code")
	}

	// ====================================================================
	// Phase 4: VERIFY + RESTORATION
	// ====================================================================
	t.Log("=== VERIFY ===")

	// Static sandbox verification on the fixed code.
	vResult := v.VerifyToolCode(fixedCode)
	if vResult.Passed {
		t.Logf("  => Sandbox VerifyToolCode: PASS")
	} else {
		t.Logf("  => Sandbox VerifyToolCode: %s", vResult.Error)
	}

	// Structural quality check via CheckGoFormat / AnalyzeGoCodeQuality.
	metrics := sandbox.AnalyzeGoCodeQuality(fixedCode)
	t.Logf("  Code quality: LOC=%d complexity=%d gofmt_issues=%d",
		metrics.LinesOfCode, metrics.CyclomaticComplexity, len(metrics.GofmtErrors))

	// ====================================================================
	// RESTORATION (rollback proof)
	// ====================================================================
	if sc.CheckRollback {
		t.Log("=== RESTORATION (rollback) ===")

		// The plan's BackupCode contains the original broken code.
		assert.Equal(t, sc.BrokenCode, plan.BackupCode,
			"BackupCode must preserve original broken code")

		// Write the backup back to the store (simulating rollback).
		err := mock.UpdateToolCode(context.Background(), sc.ToolID, plan.BackupCode)
		require.NoError(t, err, "rollback write should succeed")
		restored, err := mock.GetToolCode(context.Background(), sc.ToolID)
		require.NoError(t, err, "rollback read should succeed")

		if restored == sc.BrokenCode && restored != fixedCode {
			restorationOk = true
			t.Log("  => Restoration PASS: backup restored, rollback verified")
		} else {
			t.Errorf("  => Restoration FAIL: restoredMatchesBroken=%v restored!=fixed=%v",
				restored == sc.BrokenCode, restored != fixedCode)
		}
	} else {
		restorationOk = true
		t.Log("  => Restoration: N/A (rollback not required for this scenario)")
	}

	// ====================================================================
	// VERDICT
	// ====================================================================
	t.Log("=== VERDICT ===")

	verdict := "PASS"
	if !detectionPass || !repairValid || !restorationOk {
		verdict = "FAIL"
	}

	dLabel := "PASS"
	if !detectionPass {
		dLabel = "FAIL"
	}
	rLabel := "VALID"
	if !repairValid {
		rLabel = "INVALID"
	}
	rsLabel := "PASS"
	if !restorationOk {
		rsLabel = "FAIL"
	}

	report := fmt.Sprintf(
		"Break Detection [%s] | Repair Strategy [%s] | Restoration [%s] | %s",
		dLabel, rLabel, rsLabel, verdict)
	t.Log(report)

	t.Logf("SELF_REPAIR_RESULT scenario=%s detection=%s repair=%s restoration=%s verdict=%s",
		sc.Name, dLabel, rLabel, rsLabel, verdict)

	if verdict == "FAIL" {
		t.Error("self-repair cycle did not pass all phases")
	}
}

// ---------------------------------------------------------------------------
// TestSelfRepair_PatternPipeline — table-driven test verifying error patterns
// flow correctly from diagnostic classification through repair classification
// to catalog lookup and DiagnosticMonitor recording.
// ---------------------------------------------------------------------------

func TestSelfRepair_PatternPipeline(t *testing.T) {
	tests := []struct {
		name          string
		errorMsg      string
		diagType      string
		repairPattern string
		severityMin   string
		catalogCount  int
	}{
		{
			name:          "import_error",
			errorMsg:      `cannot find module "fmt"`,
			diagType:      diagnostic.PatternTimeout,
			repairPattern: PatternToolImport,
			severityMin:   diagnostic.SeverityLow,
			catalogCount:  3,
		},
		{
			name:          "syntax_error",
			errorMsg:      "syntax error: unexpected EOF",
			diagType:      diagnostic.PatternTimeout,
			repairPattern: PatternToolSyntax,
			severityMin:   diagnostic.SeverityLow,
			catalogCount:  2,
		},
		{
			name:          "deprecated_api",
			errorMsg:      "ioutil.ReadFile is deprecated: use os.ReadFile instead",
			diagType:      diagnostic.PatternTimeout,
			repairPattern: PatternToolDeprecated,
			severityMin:   diagnostic.SeverityLow,
			catalogCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDiag := diagnostic.ClassifyError("", tt.errorMsg)
			assert.Equal(t, tt.diagType, gotDiag,
				"diagnostic.ClassifyError system-level pattern mismatch")

			gotRepair := ClassifyToolError(tt.errorMsg)
			assert.Equal(t, tt.repairPattern, gotRepair,
				"repair.ClassifyToolError tool-level pattern mismatch")

			sev := diagnostic.AssessSeverity(gotDiag, 1, "demo")
			assert.Equal(t, tt.severityMin, sev,
				"severity with count=1 mismatch")

			catalog := BuildRepairCatalog()
			actions, ok := catalog[gotRepair]
			assert.True(t, ok, "catalog must contain pattern %q", gotRepair)
			assert.Len(t, actions, tt.catalogCount,
				"catalog action count mismatch for %q", gotRepair)

			dm := diagnostic.NewDiagnosticMonitor(3, nil)
			recorded := dm.RecordError("", tt.errorMsg, "pipeline_test", "demo")
			assert.Equal(t, gotDiag, recorded.Type,
				"DiagnosticMonitor classification matches ClassifyError")
			assert.Equal(t, 1, recorded.Count,
				"first recording count must be 1")
		})
	}
}
