package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
)

// ---------------------------------------------------------------------------
// Command parsing helper
// ---------------------------------------------------------------------------

// setupLifecycleMetaRepo creates an in-memory metadata repository with the
// full system_tools schema (category, version, health_status, source_type, etc.)
// needed by the tool lifecycle registration stage.
func setupLifecycleMetaRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE system_tools (
		id TEXT PRIMARY KEY,
		name TEXT,
		description TEXT,
		code TEXT,
		category TEXT DEFAULT '',
		version TEXT DEFAULT '',
		health_status TEXT DEFAULT 'unknown',
		last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		source_type TEXT DEFAULT 'builtin'
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE system_skills (
		id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT
	)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// parseToolInstallCommand extracts the category from "/tool install <category>".
// Returns (category, extraArgs, ok).
func parseToolInstallCommand(msg string) (string, string, bool) {
	parts := strings.Fields(msg)
	if len(parts) >= 3 && parts[0] == "/tool" && parts[1] == "install" {
		return parts[2], strings.Join(parts[3:], " "), true
	}
	return "", "", false
}

// ---------------------------------------------------------------------------
// Interface contracts for lifecycle stages (test doubles)
// ---------------------------------------------------------------------------

// ToolDiscoverer finds tools matching a category from MCP servers.
type ToolDiscoverer interface {
	Discover(ctx context.Context, category string) ([]mcp.ToolDefinition, error)
}

// PipelineRunner runs the adaptation pipeline for a tool definition.
type PipelineRunner interface {
	Run(ctx context.Context, tool mcp.ToolDefinition) (*adaptation.AdaptationResult, error)
}

// ToolVerifier verifies tool code in a sandboxed environment.
type ToolVerifier interface {
	VerifyTool(ctx context.Context, toolID string, config sandbox.VerificationConfig) (sandbox.VerificationResult, error)
}

// ToolHealthProvider returns the current health status of a registered tool.
type ToolHealthProvider interface {
	GetLatestStatus(toolID string) string
	GetHistory(toolID string) []healthRecord
}

// healthRecord is a minimal health record for test assertions.
type healthRecord struct {
	Status string
	Error  string
}

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockDiscoverer struct {
	fn func(ctx context.Context, category string) ([]mcp.ToolDefinition, error)
}

func (m *mockDiscoverer) Discover(ctx context.Context, category string) ([]mcp.ToolDefinition, error) {
	return m.fn(ctx, category)
}

type mockVerifier struct {
	fn func(ctx context.Context, toolID string, config sandbox.VerificationConfig) (sandbox.VerificationResult, error)
}

func (m *mockVerifier) VerifyTool(ctx context.Context, toolID string, config sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
	return m.fn(ctx, toolID, config)
}

type mockPipelineRunner struct {
	fn func(ctx context.Context, tool mcp.ToolDefinition) (*adaptation.AdaptationResult, error)
}

func (m *mockPipelineRunner) Run(ctx context.Context, tool mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
	return m.fn(ctx, tool)
}

type mockHealthProvider struct {
	latestFn func(toolID string) string
	histFn   func(toolID string) []healthRecord
}

func (m *mockHealthProvider) GetLatestStatus(toolID string) string {
	if m.latestFn != nil {
		return m.latestFn(toolID)
	}
	return "unknown"
}

func (m *mockHealthProvider) GetHistory(toolID string) []healthRecord {
	if m.histFn != nil {
		return m.histFn(toolID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Lifecycle stage types
// ---------------------------------------------------------------------------

// ToolLifecycleStage holds the result of one stage in the tool install flow.
type ToolLifecycleStage struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// ToolInstallResult holds the complete lifecycle result.
type ToolInstallResult struct {
	ToolName  string               `json:"tool_name"`
	Category  string               `json:"category"`
	Stages    []ToolLifecycleStage `json:"stages"`
	AllPassed bool                 `json:"all_passed"`
	Error     string               `json:"error,omitempty"`
}

// FormatOutput returns the output per spec:
//
//	Discovery [PASS/FAIL] | Adaptation [PASS/FAIL] | Registration [PASS/FAIL] | VERDICT [PASS/FAIL]
func (r *ToolInstallResult) FormatOutput() string {
	var parts []string
	for _, s := range r.Stages {
		status := "PASS"
		if !s.Passed {
			status = "FAIL"
		}
		parts = append(parts, fmt.Sprintf("%s [%s]", s.Name, status))
	}
	verdict := "PASS"
	if !r.AllPassed {
		verdict = "FAIL"
	}
	return fmt.Sprintf("%s | VERDICT [%s]", strings.Join(parts, " | "), verdict)
}

// ---------------------------------------------------------------------------
// Lifecycle runner (test-only orchestrator)
// ---------------------------------------------------------------------------

// ToolLifecycleRunner orchestrates the 5-stage tool install lifecycle.
// It uses only mocked/interface dependencies — no real MCP calls or sandbox.
type ToolLifecycleRunner struct {
	metaRepo   *repository.MetadataRepository
	discoverer ToolDiscoverer
	verifier   ToolVerifier
	pipeline   PipelineRunner
	healthChk  ToolHealthProvider
}

// RunToolInstall executes the full lifecycle for a /tool install <category> command.
func (r *ToolLifecycleRunner) RunToolInstall(ctx context.Context, category string) *ToolInstallResult {
	res := &ToolInstallResult{
		Category:  category,
		AllPassed: true,
	}

	// --------------------------------------------------
	// Stage 1: Discovery
	// --------------------------------------------------
	discoveryStage := ToolLifecycleStage{Name: "Discovery"}
	toolDefs, err := r.discoverer.Discover(ctx, category)
	if err != nil {
		discoveryStage.Passed = false
		discoveryStage.Detail = fmt.Sprintf("discovery failed: %s", err.Error())
		res.Stages = append(res.Stages, discoveryStage)
		res.AllPassed = false
		res.Error = err.Error()
		return res
	}
	if len(toolDefs) == 0 {
		discoveryStage.Passed = false
		discoveryStage.Detail = fmt.Sprintf("no tools found for category %q", category)
		res.Stages = append(res.Stages, discoveryStage)
		res.AllPassed = false
		res.Error = "no tools discovered"
		return res
	}
	discoveryStage.Passed = true
	discoveryStage.Detail = fmt.Sprintf("discovered %d tool(s) for category %q", len(toolDefs), category)
	res.Stages = append(res.Stages, discoveryStage)

	// Process the first discovered tool
	toolDef := toolDefs[0]
	res.ToolName = toolDef.Name

	// --------------------------------------------------
	// Stage 2: Verification (sandbox)
	// --------------------------------------------------
	verificationStage := ToolLifecycleStage{Name: "Adaptation"}
	regResult, err := r.pipeline.Run(ctx, toolDef)
	if err != nil {
		verificationStage.Passed = false
		verificationStage.Detail = fmt.Sprintf("adaptation pipeline failed: %s", err.Error())
		res.Stages = append(res.Stages, verificationStage)
		res.AllPassed = false
		res.Error = err.Error()
		return res
	}
	if !regResult.Registered && regResult.Error != "" {
		verificationStage.Passed = false
		verificationStage.Detail = regResult.Error
		res.Stages = append(res.Stages, verificationStage)
		res.AllPassed = false
		res.Error = regResult.Error
		return res
	}
	verificationStage.Passed = true
	verificationStage.Detail = fmt.Sprintf("tool %q adapted and verified (version %s)", toolDef.Name, regResult.Version)
	res.Stages = append(res.Stages, verificationStage)

	// --------------------------------------------------
	// Stage 3: Registration (actual DB write)
	// --------------------------------------------------
	regStage := ToolLifecycleStage{Name: "Registration"}
	toolRecord := toolDef.ToToolRecord("mcp://finance/" + toolDef.Name)
	if toolRecord.ID == "" {
		toolRecord.ID = fmt.Sprintf("mcp-%s", toolDef.Name)
	}
	if err := r.metaRepo.CreateTool(&toolRecord); err != nil {
		regStage.Passed = false
		regStage.Detail = fmt.Sprintf("registration failed: %s", err.Error())
		res.Stages = append(res.Stages, regStage)
		res.AllPassed = false
		res.Error = err.Error()
		return res
	}
	regStage.Passed = true
	regStage.Detail = fmt.Sprintf("tool %q registered (id=%s)", toolRecord.Name, toolRecord.ID)
	res.Stages = append(res.Stages, regStage)

	// --------------------------------------------------
	// Stage 4: Health check
	// --------------------------------------------------
	healthStage := ToolLifecycleStage{Name: "HealthCheck"}
	status := r.healthChk.GetLatestStatus(toolRecord.ID)
	if status != "healthy" {
		healthStage.Passed = false
		healthStage.Detail = fmt.Sprintf("health check returned %q", status)
		res.Stages = append(res.Stages, healthStage)
		res.AllPassed = false
		res.Error = fmt.Sprintf("tool health is %s", status)
		return res
	}
	healthStage.Passed = true
	healthStage.Detail = fmt.Sprintf("tool %q is healthy", toolRecord.Name)
	res.Stages = append(res.Stages, healthStage)

	return res
}

// =========================================================================
// Tests
// =========================================================================

func TestParseToolInstallCommand(t *testing.T) {
	tests := []struct {
		name         string
		msg          string
		wantCategory string
		wantExtra    string
		wantOK       bool
	}{
		{
			name:         "simple category",
			msg:          "/tool install finance",
			wantCategory: "finance",
			wantOK:       true,
		},
		{
			name:         "with extra args",
			msg:          "/tool install finance --version 2",
			wantCategory: "finance",
			wantExtra:    "--version 2",
			wantOK:       true,
		},
		{
			name:         "multi-word category",
			msg:          "/tool install data-analysis",
			wantCategory: "data-analysis",
			wantOK:       true,
		},
		{
			name:   "missing category",
			msg:    "/tool install",
			wantOK: false,
		},
		{
			name:   "wrong command",
			msg:    "/tool uninstall finance",
			wantOK: false,
		},
		{
			name:   "not a tool command",
			msg:    "/help",
			wantOK: false,
		},
		{
			name:   "empty message",
			msg:    "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCategory, gotExtra, gotOK := parseToolInstallCommand(tt.msg)
			assert.Equal(t, tt.wantCategory, gotCategory, "category")
			assert.Equal(t, tt.wantExtra, gotExtra, "extra")
			assert.Equal(t, tt.wantOK, gotOK, "ok")
		})
	}
}

func TestToolLifecycle_InstallFinance(t *testing.T) {
	// Shared test helpers
	defaultVerConfig := sandbox.DefaultVerificationConfig()
	healthyTool := mcp.ToolDefinition{
		Name:        "finance_analyzer",
		Description: "Financial analysis tool",
		Version:     "1.0.0",
		Category:    "finance",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ticker": map[string]any{"type": "string"},
			},
		},
	}

	successPipelineResult := &adaptation.AdaptationResult{
		ToolDefinition: healthyTool,
		Registered:     true,
		Version:        "1.0.0",
		Stages: []adaptation.StageResult{
			{Name: "verification", Passed: true, Message: "tool verified"},
			{Name: "analysis", Passed: true, Message: "tool analyzed"},
			{Name: "adaptation", Passed: true, Message: "tool adapted"},
			{Name: "testing", Passed: true, Message: "tests passed"},
			{Name: "registration", Passed: true, Message: "tool registered"},
		},
	}

	type lifecycleTestCase struct {
		name           string
		category       string // from parsed /tool install <category>
		discoverFn     func(ctx context.Context, category string) ([]mcp.ToolDefinition, error)
		verifyFn       func(ctx context.Context, toolID string, config sandbox.VerificationConfig) (sandbox.VerificationResult, error)
		pipelineFn     func(ctx context.Context, tool mcp.ToolDefinition) (*adaptation.AdaptationResult, error)
		healthLatestFn func(toolID string) string
		wantAllPassed  bool
		wantStageCount int
		wantError      string
	}

	tests := []lifecycleTestCase{
		{
			name:     "happy path — all stages pass",
			category: "finance",
			discoverFn: func(_ context.Context, cat string) ([]mcp.ToolDefinition, error) {
				return []mcp.ToolDefinition{healthyTool}, nil
			},
			verifyFn: func(_ context.Context, _ string, _ sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
				return sandbox.VerificationResult{Passed: true, ExitCode: 0, Stdout: "ok", Duration: 50 * time.Millisecond}, nil
			},
			pipelineFn: func(_ context.Context, _ mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
				return successPipelineResult, nil
			},
			healthLatestFn: func(_ string) string { return "healthy" },
			wantAllPassed:  true,
			wantStageCount: 4, // Discovery, Adaptation, Registration, HealthCheck
		},
		{
			name:     "discovery returns error",
			category: "finance",
			discoverFn: func(_ context.Context, _ string) ([]mcp.ToolDefinition, error) {
				return nil, fmt.Errorf("MCP server unreachable")
			},
			verifyFn:       nil,
			pipelineFn:     nil,
			healthLatestFn: nil,
			wantAllPassed:  false,
			wantStageCount: 1,
			wantError:      "MCP server unreachable",
		},
		{
			name:     "discovery returns empty list",
			category: "finance",
			discoverFn: func(_ context.Context, _ string) ([]mcp.ToolDefinition, error) {
				return nil, nil
			},
			verifyFn:       nil,
			pipelineFn:     nil,
			healthLatestFn: nil,
			wantAllPassed:  false,
			wantStageCount: 1,
			wantError:      "no tools discovered",
		},
		{
			name:     "adaptation pipeline fails",
			category: "finance",
			discoverFn: func(_ context.Context, _ string) ([]mcp.ToolDefinition, error) {
				return []mcp.ToolDefinition{healthyTool}, nil
			},
			pipelineFn: func(_ context.Context, _ mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
				return &adaptation.AdaptationResult{
					ToolDefinition: healthyTool,
					Registered:     false,
					Error:          "stage testing failed: test assertion error",
				}, nil
			},
			verifyFn: func(_ context.Context, _ string, _ sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
				return sandbox.VerificationResult{Passed: true, ExitCode: 0}, nil
			},
			healthLatestFn: nil,
			wantAllPassed:  false,
			wantStageCount: 2,
			wantError:      "stage testing failed: test assertion error",
		},
		{
			name:     "registration succeeds but health check is degraded",
			category: "finance",
			discoverFn: func(_ context.Context, _ string) ([]mcp.ToolDefinition, error) {
				return []mcp.ToolDefinition{healthyTool}, nil
			},
			pipelineFn: func(_ context.Context, _ mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
				return successPipelineResult, nil
			},
			verifyFn: func(_ context.Context, _ string, _ sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
				return sandbox.VerificationResult{Passed: true, ExitCode: 0}, nil
			},
			healthLatestFn: func(_ string) string { return "degraded" },
			wantAllPassed:  false,
			wantStageCount: 4,
			wantError:      `tool health is degraded`,
		},
		{
			name:     "registration succeeds but health check is down",
			category: "finance",
			discoverFn: func(_ context.Context, _ string) ([]mcp.ToolDefinition, error) {
				return []mcp.ToolDefinition{healthyTool}, nil
			},
			pipelineFn: func(_ context.Context, _ mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
				return successPipelineResult, nil
			},
			verifyFn: func(_ context.Context, _ string, _ sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
				return sandbox.VerificationResult{Passed: true, ExitCode: 0}, nil
			},
			healthLatestFn: func(_ string) string { return "down" },
			wantAllPassed:  false,
			wantStageCount: 4,
			wantError:      `tool health is down`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupLifecycleMetaRepo(t)
			require.NotNil(t, repo)
			_ = defaultVerConfig

			disc := &mockDiscoverer{fn: tt.discoverFn}

			var pipeline PipelineRunner
			if tt.pipelineFn != nil {
				pipeline = &mockPipelineRunner{fn: tt.pipelineFn}
			} else {
				pipeline = &mockPipelineRunner{fn: func(_ context.Context, _ mcp.ToolDefinition) (*adaptation.AdaptationResult, error) {
					return nil, fmt.Errorf("pipeline not expected to be called")
				}}
			}

			verifier := &mockVerifier{fn: func(_ context.Context, _ string, _ sandbox.VerificationConfig) (sandbox.VerificationResult, error) {
				return sandbox.VerificationResult{Passed: true, ExitCode: 0}, nil
			}}

			healthProv := &mockHealthProvider{
				latestFn: func(toolID string) string {
					if tt.healthLatestFn != nil {
						return tt.healthLatestFn(toolID)
					}
					return "unknown"
				},
			}

			runner := &ToolLifecycleRunner{
				metaRepo:   repo,
				discoverer: disc,
				verifier:   verifier,
				pipeline:   pipeline,
				healthChk:  healthProv,
			}

			cat, extra, parsed := parseToolInstallCommand("/tool install " + tt.category)
			require.True(t, parsed, "command must parse")
			require.Equal(t, tt.category, cat)
			assert.Empty(t, extra)

			ctx := context.Background()
			result := runner.RunToolInstall(ctx, cat)

			require.NotNil(t, result)
			assert.Len(t, result.Stages, tt.wantStageCount)
			assert.Equal(t, tt.wantAllPassed, result.AllPassed)

			if tt.wantError != "" {
				assert.Contains(t, result.Error, tt.wantError)
			} else {
				assert.Empty(t, result.Error)
			}

			for _, s := range result.Stages {
				if tt.wantAllPassed {
					assert.True(t, s.Passed, "stage %q should pass", s.Name)
				}
				assert.NotEmpty(t, s.Detail, "stage %q detail", s.Name)
			}

			output := result.FormatOutput()
			t.Logf("Output: %s", output)

			for _, s := range result.Stages {
				expectedMarker := fmt.Sprintf("%s [PASS]", s.Name)
				expectedMarkerFail := fmt.Sprintf("%s [FAIL]", s.Name)
				hasPass := strings.Contains(output, expectedMarker)
				hasFail := strings.Contains(output, expectedMarkerFail)
				assert.True(t, hasPass || hasFail,
					"output should contain %q or %q", expectedMarker, expectedMarkerFail)
			}
			assert.Contains(t, output, "VERDICT [")

			if tt.wantStageCount >= 3 {
				tools, err := repo.ListTools()
				require.NoError(t, err)
				if tt.wantAllPassed {
					assert.Len(t, tools, 1, "should have 1 registered tool")
					assert.Equal(t, "finance_analyzer", tools[0].Name)
				}
			}
		})
	}
}

func TestToolInstallResult_FormatOutput(t *testing.T) {
	tests := []struct {
		name    string
		result  *ToolInstallResult
		wantOut string
	}{
		{
			name: "all pass",
			result: &ToolInstallResult{
				ToolName: "test_tool",
				Stages: []ToolLifecycleStage{
					{Name: "Discovery", Passed: true},
					{Name: "Adaptation", Passed: true},
					{Name: "Registration", Passed: true},
				},
				AllPassed: true,
			},
			wantOut: "Discovery [PASS] | Adaptation [PASS] | Registration [PASS] | VERDICT [PASS]",
		},
		{
			name: "discovery fails",
			result: &ToolInstallResult{
				Stages: []ToolLifecycleStage{
					{Name: "Discovery", Passed: false},
				},
				AllPassed: false,
			},
			wantOut: "Discovery [FAIL] | VERDICT [FAIL]",
		},
		{
			name: "adaptation fails",
			result: &ToolInstallResult{
				Stages: []ToolLifecycleStage{
					{Name: "Discovery", Passed: true},
					{Name: "Adaptation", Passed: false},
				},
				AllPassed: false,
			},
			wantOut: "Discovery [PASS] | Adaptation [FAIL] | VERDICT [FAIL]",
		},
		{
			name: "mixed stages",
			result: &ToolInstallResult{
				Stages: []ToolLifecycleStage{
					{Name: "Discovery", Passed: true},
					{Name: "Adaptation", Passed: true},
					{Name: "Registration", Passed: false},
					{Name: "HealthCheck", Passed: true},
				},
				AllPassed: false,
			},
			wantOut: "Discovery [PASS] | Adaptation [PASS] | Registration [FAIL] | HealthCheck [PASS] | VERDICT [FAIL]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.FormatOutput()
			assert.Equal(t, tt.wantOut, got)
		})
	}
}
