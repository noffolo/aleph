package adaptation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
)

// Pipeline orchestrates the 5-stage tool adaptation workflow.
type Pipeline struct {
	stages   []PipelineStage
	metaRepo *repository.MetadataRepository
	verifier *sandbox.Verifier
}

// NewPipeline creates a new adaptation pipeline with default stages.
func NewPipeline(metaRepo *repository.MetadataRepository) *Pipeline {
	verifier := sandbox.NewVerifier(nil, metaRepo, "python3", "go")
	return &Pipeline{
		stages: []PipelineStage{
			&VerificationStage{verifier: verifier},
			&AnalysisStage{},
			&AdaptationStage{},
			&TestingStage{metaRepo: metaRepo, pythonCmd: "python3", goCmd: "go"},
			&RegistrationStage{metaRepo: metaRepo},
		},
		metaRepo: metaRepo,
		verifier: verifier,
	}
}

// Run executes the adaptation pipeline on a tool definition (backward-compatible).
func (p *Pipeline) Run(ctx context.Context, tool mcp.ToolDefinition) (*AdaptationResult, error) {
	return p.RunCandidate(ctx, &Candidate{ToolDef: tool})
}

// RunCandidate executes the adaptation pipeline on a full candidate with code.
func (p *Pipeline) RunCandidate(ctx context.Context, candidate *Candidate) (*AdaptationResult, error) {
	result := &AdaptationResult{
		ToolDefinition: candidate.ToolDef,
		Stages:         make([]StageResult, 0, len(p.stages)),
		CandidateCode:  candidate.Code,
		TemplateType:   candidate.TemplateType,
	}

	for _, stage := range p.stages {
		stageResult, err := stage.Execute(ctx, candidate, result)
		result.Stages = append(result.Stages, stageResult)

		if err != nil {
			result.Error = fmt.Sprintf("stage %s error: %v", stageResult.Name, err)
			return result, nil
		}

		if !stageResult.Passed {
			result.Error = fmt.Sprintf("stage %s failed: %s", stageResult.Name, stageResult.Message)
			return result, nil
		}
	}

	result.Registered = true
	result.Version = candidate.ToolDef.Version
	if result.Version == "" {
		result.Version = "1.0.0"
	}
	return result, nil
}

// RunSuggestion executes the adaptation pipeline up to (but not including)
// registration. Returns the adaptation result with adapted code and analysis.
// The caller can present the result for user approval, then call
// RegisterFromSuggestion to finalize registration.
func (p *Pipeline) RunSuggestion(ctx context.Context, tool mcp.ToolDefinition) (*AdaptationResult, error) {
	candidate := &Candidate{ToolDef: tool}
	result := &AdaptationResult{
		ToolDefinition: candidate.ToolDef,
		Stages:         make([]StageResult, 0, len(p.stages)-1),
	}

	// Run all stages except the last (registration)
	for i := 0; i < len(p.stages)-1; i++ {
		stage := p.stages[i]
		stageResult, err := stage.Execute(ctx, candidate, result)
		result.Stages = append(result.Stages, stageResult)
		if err != nil {
			result.Error = fmt.Sprintf("stage %s error: %v", stageResult.Name, err)
			return result, nil
		}
		if !stageResult.Passed {
			result.Error = fmt.Sprintf("stage %s failed: %s", stageResult.Name, stageResult.Message)
			return result, nil
		}
	}

	result.Version = candidate.ToolDef.Version
	if result.Version == "" {
		result.Version = "1.0.0"
	}
	return result, nil
}

// RegisterFromSuggestion registers a tool from a prior RunSuggestion result.
// It reconstructs the candidate from the AdaptationResult and runs the
// registration stage only.
func (p *Pipeline) RegisterFromSuggestion(ctx context.Context, result *AdaptationResult) error {
	if len(p.stages) == 0 {
		return fmt.Errorf("pipeline has no stages")
	}

	candidate := &Candidate{
		ToolDef: result.ToolDefinition,
		Code:    result.AdaptedCode,
	}

	// Last stage is RegistrationStage
	regStage := p.stages[len(p.stages)-1]
	sr, err := regStage.Execute(ctx, candidate, result)
	if err != nil {
		return fmt.Errorf("registration error: %w", err)
	}
	if !sr.Passed {
		return fmt.Errorf("registration failed: %s", sr.Message)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stage 1: VerificationStage
// ---------------------------------------------------------------------------

// VerificationStage validates the tool in sandbox.
type VerificationStage struct {
	verifier *sandbox.Verifier
}

func (s *VerificationStage) Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error) {
	code := candidate.Code

	// No code to verify — try to look up from repository by name
	if code == "" {
		if s.verifier == nil || candidate.ToolDef.Name == "" {
			return StageResult{
				Name: "verification", Passed: true,
				Message: "no code to verify",
			}, nil
		}
		vr := s.verifier.VerifyToolCode(candidate.ToolDef.Name)
		if vr.Passed {
			return StageResult{
				Name: "verification", Passed: true,
				Message: "static verification passed (no code in candidate)",
			}, nil
		}
		code = candidate.ToolDef.Name
	}

	// Static analysis via VerifyToolCode
	vr := s.verifier.VerifyToolCode(code)
	if !vr.Passed {
		return StageResult{
			Name: "verification", Passed: false,
			Message: fmt.Sprintf("static verification failed: %s", vr.Error),
		}, nil
	}

	msg := "static verification passed"

	// Full sandbox verification if tool already registered
	if s.verifier != nil && candidate.ToolDef.Name != "" {
		existingID := s.findToolIDByName(ctx, candidate.ToolDef.Name)
		if existingID != "" {
			fullResult, err := s.verifier.VerifyTool(ctx, existingID, sandbox.DefaultVerificationConfig())
			if err != nil {
				msg = fmt.Sprintf("static passed; sandbox unavailable: %v", err)
			} else if !fullResult.Passed {
				return StageResult{
					Name: "verification", Passed: false,
					Message: fmt.Sprintf("sandbox verification failed: %s", fullResult.Error),
				}, nil
			} else {
				msg = fmt.Sprintf("static+sandbox verified (exit:%d, dur:%v)", fullResult.ExitCode, fullResult.Duration)
			}
		}
	}

	return StageResult{
		Name: "verification", Passed: true, Message: msg,
	}, nil
}

func (s *VerificationStage) findToolIDByName(_ context.Context, name string) string {
	_ = name
	return ""
}

// ---------------------------------------------------------------------------
// Stage 2: AnalysisStage
// ---------------------------------------------------------------------------

// AnalysisStage analyzes tool structure and dependencies.
type AnalysisStage struct{}

func (s *AnalysisStage) Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error) {
	code := candidate.Code
	if code == "" {
		code = candidate.ToolDef.Name
	}

	analysis := AnalysisDetail{
		TemplateType: candidate.TemplateType,
	}

	// Detect language
	if sandbox.IsPythonCode(code) {
		analysis.Language = "python"
	} else {
		analysis.Language = "go"
	}

	// Extract imports/dependencies
	if analysis.Language == "go" {
		analysis.Dependencies = extractGoImports(code)
	} else {
		analysis.Dependencies = extractPythonImports(code)
	}

	// Estimate complexity
	analysis.Complexity = sandbox.EstimateComplexity(code)

	// Detect test patterns
	analysis.HasTests = strings.Contains(code, "func Test") ||
		strings.Contains(code, "assert.") ||
		strings.Contains(code, "def test_")

	// Detect template type if not already assigned
	if analysis.TemplateType == "" {
		analysis.TemplateType = detectTemplateType(code, candidate.ToolDef)
	}

	// Code quality issues
	analysis.Issues = detectIssues(code, analysis.Language)

	// Store analysis in result
	result.Analysis = analysis
	result.TemplateType = analysis.TemplateType

	msg := fmt.Sprintf("language=%s deps=%d complexity=%d template=%s",
		analysis.Language, len(analysis.Dependencies), analysis.Complexity, analysis.TemplateType)
	if len(analysis.Issues) > 0 {
		msg += fmt.Sprintf(" issues=%d", len(analysis.Issues))
	}

	passed := len(analysis.Issues) == 0 || analysis.Complexity < 50
	return StageResult{
		Name: "analysis", Passed: passed, Message: msg,
	}, nil
}

func extractGoImports(code string) []string {
	var deps []string
	seen := make(map[string]bool)
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import") {
			// Parse single-line import
			imp := strings.TrimPrefix(line, "import")
			imp = strings.Trim(imp, ` "`)
			if imp != "" && !seen[imp] {
				seen[imp] = true
				deps = append(deps, imp)
			}
		}
	}
	return deps
}

func extractPythonImports(code string) []string {
	var deps []string
	seen := make(map[string]bool)
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && !seen[parts[1]] {
				seen[parts[1]] = true
				deps = append(deps, parts[1])
			}
		}
		if strings.HasPrefix(line, "from ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && !seen[parts[1]] {
				seen[parts[1]] = true
				deps = append(deps, parts[1])
			}
		}
	}
	return deps
}

func detectTemplateType(code string, def mcp.ToolDefinition) TemplateType {
	// Heuristic: MCP tools have InputSchema → adapter
	if def.InputSchema != nil && len(def.InputSchema) > 0 {
		return TemplateAdapter
	}

	// Python code → likely needs wrapping
	if sandbox.IsPythonCode(code) {
		return TemplateWrapper
	}

	// Library-style code (no main function) → decorator
	if strings.Contains(code, "func ") && !strings.Contains(code, "func main()") &&
		!strings.Contains(code, "func main(") {
		return TemplateDecorator
	}

	// Default: wrapper
	return TemplateWrapper
}

func detectIssues(code, lang string) []string {
	var issues []string
	if code == "" {
		return issues
	}
	if len(code) > 100000 {
		issues = append(issues, "code exceeds 100KB")
	}
	if lang == "go" {
		if strings.Contains(code, "panic(") {
			issues = append(issues, "contains panic()")
		}
		if strings.Contains(code, "log.Fatal") {
			issues = append(issues, "contains log.Fatal()")
		}
	}
	if strings.Count(code, "\n") > 2000 {
		issues = append(issues, "code exceeds 2000 lines")
	}
	return issues
}

// ---------------------------------------------------------------------------
// Stage 3: AdaptationStage
// ---------------------------------------------------------------------------

// AdaptationStage applies adaptation template.
type AdaptationStage struct{}

func (s *AdaptationStage) Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error) {
	tmplType := result.TemplateType
	if tmplType == "" {
		tmplType = candidate.TemplateType
	}
	if tmplType == "" {
		tmplType = TemplateWrapper
	}
	result.TemplateType = tmplType

	code := candidate.Code
	if code == "" {
		code = "// tool: " + candidate.ToolDef.Name + "\n"
	}

	var adaptedCode string
	switch tmplType {
	case TemplateWrapper:
		adaptedCode = generateWrapperTemplate(code, candidate.ToolDef)
	case TemplateAdapter:
		adaptedCode = generateAdapterTemplate(code, candidate.ToolDef)
	case TemplateDecorator:
		adaptedCode = generateDecoratorTemplate(code, candidate.ToolDef)
	default:
		adaptedCode = generateWrapperTemplate(code, candidate.ToolDef)
	}

	result.AdaptedCode = adaptedCode

	return StageResult{
		Name:    "adaptation",
		Passed:  true,
		Message: fmt.Sprintf("applied %s template (%d bytes)", tmplType, len(adaptedCode)),
	}, nil
}

// ---------------------------------------------------------------------------
// Stage 4: TestingStage
// ---------------------------------------------------------------------------

// TestingStage runs tests on adapted tool.
type TestingStage struct {
	metaRepo  *repository.MetadataRepository
	pythonCmd string
	goCmd     string
}

func (s *TestingStage) Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error) {
	code := result.AdaptedCode
	if code == "" {
		code = result.CandidateCode
	}
	if code == "" {
		code = candidate.Code
	}
	if code == "" {
		return StageResult{
			Name: "testing", Passed: true,
			Message: "no code to test",
		}, nil
	}

	// Determine language: Go code starts with "package", Python with "# python".
	// Avoid sandbox.IsPythonCode which can misclassify Go import blocks as Python.
	trimmed := strings.TrimSpace(code)
	isGo := strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "package\t")

	// Validate code
	if isGo {
		if err := sandbox.ValidateGoCode(code); err != nil {
			return StageResult{
				Name: "testing", Passed: false,
				Message: fmt.Sprintf("go validation failed: %v", err),
			}, nil
		}
	} else {
		if err := sandbox.ValidatePythonCode(code); err != nil {
			return StageResult{
				Name: "testing", Passed: false,
				Message: fmt.Sprintf("python validation failed: %v", err),
			}, nil
		}
	}

	// Create temp directory and execute
	tmpDir, err := os.MkdirTemp("", "aleph-adapt-test-*")
	if err != nil {
		return StageResult{
			Name: "testing", Passed: false,
			Message: fmt.Sprintf("failed to create temp dir: %v", err),
		}, nil
	}
	defer os.RemoveAll(tmpDir)

	var (
		stdout, stderr strings.Builder
		exitCode       int
		execErr        error
	)

	if isGo {
		execErr = s.runGoCode(ctx, tmpDir, code, &stdout, &stderr, &exitCode)
	} else {
		execErr = s.runPythonCode(ctx, tmpDir, code, &stdout, &stderr, &exitCode)
	}

	if execErr != nil {
		return StageResult{
			Name: "testing", Passed: false,
			Message: fmt.Sprintf("execution error: %v; stderr: %s", execErr, truncate(stderr.String(), 200)),
		}, nil
	}

	if exitCode != 0 {
		return StageResult{
			Name: "testing", Passed: false,
			Message: fmt.Sprintf("exit code %d; stderr: %s", exitCode, truncate(stderr.String(), 200)),
		}, nil
	}

	// Check for suspicious patterns in output
	if containsSuspiciousOutput(stdout.String()) || containsSuspiciousOutput(stderr.String()) {
		return StageResult{
			Name: "testing", Passed: false,
			Message: "suspicious output patterns detected",
		}, nil
	}

	return StageResult{
		Name:    "testing",
		Passed:  true,
		Message: fmt.Sprintf("tests passed (exit:%d, stdout:%d bytes)", exitCode, stdout.Len()),
	}, nil
}

func (s *TestingStage) runPythonCode(ctx context.Context, tmpDir, code string, stdout, stderr *strings.Builder, exitCode *int) error {
	tmpFile := filepath.Join(tmpDir, "tool.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	pythonCmd := s.pythonCmd
	if pythonCmd == "" {
		pythonCmd = "python3"
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, pythonCmd, tmpFile)
	cmd.Dir = tmpDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + tmpDir}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			*exitCode = exitErr.ExitCode()
			return nil
		}
		return fmt.Errorf("runPythonCode: %w", err)
	}
	*exitCode = 0
	return nil
}

func (s *TestingStage) runGoCode(ctx context.Context, tmpDir, code string, stdout, stderr *strings.Builder, exitCode *int) error {
	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Create a minimal go.mod for the temp module
	modPath := filepath.Join(tmpDir, "go.mod")
	modContent := "module tooltest\n\ngo 1.24\n"
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	goCmd := s.goCmd
	if goCmd == "" {
		goCmd = "go"
	}

	execCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Build
	binPath := filepath.Join(tmpDir, "tool_bin")
	buildCmd := exec.CommandContext(execCtx, goCmd, "build", "-o", binPath, ".")
	buildCmd.Dir = tmpDir
	var buildStderr strings.Builder
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w; %s", err, truncate(buildStderr.String(), 200))
	}

	// Run
	runCmd := exec.CommandContext(execCtx, binPath)
	runCmd.Dir = tmpDir
	runCmd.Stdout = stdout
	runCmd.Stderr = stderr
	runCmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + tmpDir}

	err := runCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			*exitCode = exitErr.ExitCode()
			return nil
		}
		return fmt.Errorf("runGoCode: %w", err)
	}
	*exitCode = 0
	return nil
}

// ---------------------------------------------------------------------------
// Stage 5: RegistrationStage
// ---------------------------------------------------------------------------

// RegistrationStage registers adapted tool in metadata repository.
type RegistrationStage struct {
	metaRepo *repository.MetadataRepository
}

func (s *RegistrationStage) Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error) {
	if s.metaRepo == nil {
		return StageResult{
			Name: "registration", Passed: false,
			Message: "metadata repository not available",
		}, nil
	}

	def := candidate.ToolDef
	name := def.Name
	if name == "" {
		return StageResult{
			Name: "registration", Passed: true,
			Message: "no tool name provided, skipping registration",
		}, nil
	}

	adaptedCode := result.AdaptedCode
	if adaptedCode == "" {
		adaptedCode = result.CandidateCode
	}
	if adaptedCode == "" {
		adaptedCode = "// tool: " + name
	}

	version := def.Version
	if version == "" {
		version = "1.0.0"
	}

	category := def.Category
	if category == "" {
		category = "adaptation"
	}

	// Check if tool already exists by listing and matching name
	toolRecord := repository.ToolRecord{
		Name:         name,
		Description:  def.Description,
		Code:         adaptedCode,
		Category:     category,
		Version:      version,
		HealthStatus: "healthy",
		SourceType:   "adaptation",
	}

	existingID := s.findExistingToolID(name)
	if existingID != "" {
		// Update existing tool
		if err := s.metaRepo.UpdateToolCode(ctx, existingID, adaptedCode); err != nil {
			return StageResult{
				Name: "registration", Passed: false,
				Message: fmt.Sprintf("failed to update tool %q: %v", name, err),
			}, nil
		}
		toolRecord.ID = existingID
		_ = s.metaRepo.UpdateHealthStatus(existingID, "healthy")

		return StageResult{
			Name:    "registration",
			Passed:  true,
			Message: fmt.Sprintf("updated tool %q (id=%s, ver=%s)", name, existingID, version),
		}, nil
	}

	// Create new tool
	toolRecord.ID = name + "-adapted"
	if err := s.metaRepo.CreateTool(&toolRecord); err != nil {
		return StageResult{
			Name: "registration", Passed: false,
			Message: fmt.Sprintf("failed to create tool %q: %v", name, err),
		}, nil
	}

	return StageResult{
		Name:    "registration",
		Passed:  true,
		Message: fmt.Sprintf("registered tool %q (id=%s, ver=%s)", name, toolRecord.ID, version),
	}, nil
}

// findExistingToolID checks if a tool with the given name exists.
func (s *RegistrationStage) findExistingToolID(name string) string {
	tools, err := s.metaRepo.ListTools()
	if err != nil {
		return ""
	}
	for _, t := range tools {
		if t.Name == name {
			return t.ID
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func containsSuspiciousOutput(s string) bool {
	patterns := []string{
		"/etc/passwd", "/etc/shadow", "root:",
		"sudo ", "chmod 777", "rm -rf /",
	}
	lower := strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
