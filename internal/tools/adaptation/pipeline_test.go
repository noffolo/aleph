package adaptation

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
)

func setupTestDB(t *testing.T) *repository.MetadataRepository {
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

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

const pureGoCode = `package main

func main() {
	var x int
	x = 42
	_ = x
}`

const goCodeWithImports = `package main

import(
"fmt"
)

func main() {
	fmt.Println("ok")
}`

const blockedPythonCode = `# python
import subprocess
subprocess.run(["ls"])`

func TestVerificationStage_Execute(t *testing.T) {
	metaRepo := setupTestDB(t)
	verifier := sandbox.NewVerifier(nil, metaRepo, "python3", "go")
	stage := &VerificationStage{verifier: verifier}

	tests := []struct {
		name      string
		candidate Candidate
		wantPass  bool
		wantMsg   string
	}{
		{
			name: "valid go code passes",
			candidate: Candidate{
				Code:    pureGoCode,
				ToolDef: mcp.ToolDefinition{Name: "test-tool"},
			},
			wantPass: true,
		},
		{
			name: "valid python code passes",
			candidate: Candidate{
				Code:    "# python\ndef main():\n\treturn 1",
				ToolDef: mcp.ToolDefinition{Name: "py-tool"},
			},
			wantPass: true,
		},
		{
			name: "blocked python pattern fails",
			candidate: Candidate{
				Code:    blockedPythonCode,
				ToolDef: mcp.ToolDefinition{Name: "bad-py"},
			},
			wantPass: false,
			wantMsg:  "blocklisted",
		},
		{
			name: "empty code passes vacuously",
			candidate: Candidate{
				Code:    "",
				ToolDef: mcp.ToolDefinition{Name: "empty"},
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AdaptationResult{}
			sr, err := stage.Execute(context.Background(), &tt.candidate, result)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPass, sr.Passed)
			if tt.wantMsg != "" {
				assert.Contains(t, sr.Message, tt.wantMsg)
			}
		})
	}
}

func TestAnalysisStage_Execute(t *testing.T) {
	stage := &AnalysisStage{}

	adapterDef := mcp.ToolDefinition{
		Name:        "mcp-tool",
		Description: "MCP test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
		Version: "1.0.0",
	}

	decoratorCode := `package lib
func DoSomething(x int) int {
	return x * 2
}`

	purePython := "# python\ndef run():\n\treturn 42"

	tests := []struct {
		name      string
		candidate Candidate
		wantPass  bool
		checkFn   func(t *testing.T, result *AdaptationResult, sr StageResult)
	}{
		{
			name: "detects go code from multi-line imports",
			candidate: Candidate{
				Code:    goCodeWithImports,
				ToolDef: mcp.ToolDefinition{Name: "go-tool"},
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, "go", result.Analysis.Language)
				assert.Greater(t, result.Analysis.Complexity, 0)
			},
		},
		{
			name: "detects python code correctly",
			candidate: Candidate{
				Code:    purePython,
				ToolDef: mcp.ToolDefinition{Name: "py-tool"},
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, "python", result.Analysis.Language)
			},
		},
		{
			name: "mcp tool with schema detects adapter template",
			candidate: Candidate{
				Code:    pureGoCode,
				ToolDef: adapterDef,
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, TemplateAdapter, result.Analysis.TemplateType)
			},
		},
		{
			name: "library code (no main func) detects decorator",
			candidate: Candidate{
				Code:    decoratorCode,
				ToolDef: mcp.ToolDefinition{Name: "lib-tool"},
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, TemplateDecorator, result.Analysis.TemplateType)
			},
		},
		{
			name: "python code detects wrapper template",
			candidate: Candidate{
				Code:    purePython,
				ToolDef: mcp.ToolDefinition{Name: "wrapper-tool"},
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, TemplateWrapper, result.Analysis.TemplateType)
			},
		},
		{
			name: "go code defaults to wrapper template",
			candidate: Candidate{
				Code:    pureGoCode,
				ToolDef: mcp.ToolDefinition{Name: "go-wrapper"},
			},
			wantPass: true,
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.Equal(t, TemplateWrapper, result.Analysis.TemplateType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AdaptationResult{}
			sr, err := stage.Execute(context.Background(), &tt.candidate, result)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPass, sr.Passed)
			if tt.checkFn != nil {
				tt.checkFn(t, result, sr)
			}
		})
	}
}

func TestAdaptationStage_Execute(t *testing.T) {
	stage := &AdaptationStage{}

	tests := []struct {
		name      string
		candidate Candidate
		checkFn   func(t *testing.T, result *AdaptationResult, sr StageResult)
	}{
		{
			name: "wrapper template generates go code",
			candidate: Candidate{
				Code:         pureGoCode,
				TemplateType: TemplateWrapper,
				ToolDef:      mcp.ToolDefinition{Name: "wrapper-tool", Description: "A wrapper test tool"},
			},
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.True(t, sr.Passed)
				assert.Equal(t, TemplateWrapper, result.TemplateType)
				assert.Contains(t, result.AdaptedCode, "package main")
				assert.Contains(t, sr.Message, "wrapper")
			},
		},
		{
			name: "adapter template generates go code",
			candidate: Candidate{
				Code:         pureGoCode,
				TemplateType: TemplateAdapter,
				ToolDef: mcp.ToolDefinition{
					Name:        "adapter-tool",
					Description: "An adapter test tool",
					InputSchema: map[string]any{"type": "object"},
				},
			},
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.True(t, sr.Passed)
				assert.Equal(t, TemplateAdapter, result.TemplateType)
				assert.Contains(t, result.AdaptedCode, "AdapterTool")
				assert.Contains(t, result.AdaptedCode, "Adapt(")
			},
		},
		{
			name: "decorator template generates go code",
			candidate: Candidate{
				Code:         pureGoCode,
				TemplateType: TemplateDecorator,
				ToolDef:      mcp.ToolDefinition{Name: "decorator-tool", Description: "A decorator test tool"},
			},
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.True(t, sr.Passed)
				assert.Equal(t, TemplateDecorator, result.TemplateType)
				assert.Contains(t, result.AdaptedCode, "DecoratorTool")
				assert.Contains(t, result.AdaptedCode, "Run(")
			},
		},
		{
			name: "falls back to wrapper when no type set",
			candidate: Candidate{
				Code:    pureGoCode,
				ToolDef: mcp.ToolDefinition{Name: "default-tool"},
			},
			checkFn: func(t *testing.T, result *AdaptationResult, sr StageResult) {
				assert.True(t, sr.Passed)
				assert.Equal(t, TemplateWrapper, result.TemplateType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AdaptationResult{}
			sr, err := stage.Execute(context.Background(), &tt.candidate, result)
			assert.NoError(t, err)
			if tt.checkFn != nil {
				tt.checkFn(t, result, sr)
			}
		})
	}
}

func TestTestingStage_Execute(t *testing.T) {
	metaRepo := setupTestDB(t)
	stage := &TestingStage{
		metaRepo:  metaRepo,
		pythonCmd: "python3",
		goCmd:     "go",
	}

	validGoTool := `package main
func main() {}`

	validPythonTool := "# python\ndef run():\n\treturn 42"

	tests := []struct {
		name      string
		candidate Candidate
		result    *AdaptationResult
		wantPass  bool
	}{
		{
			name: "valid go code passes",
			candidate: Candidate{
				Code:    validGoTool,
				ToolDef: mcp.ToolDefinition{Name: "test-go"},
			},
			result:   &AdaptationResult{AdaptedCode: validGoTool},
			wantPass: true,
		},
		{
			name: "valid python code passes",
			candidate: Candidate{
				Code:    validPythonTool,
				ToolDef: mcp.ToolDefinition{Name: "test-py"},
			},
			result:   &AdaptationResult{AdaptedCode: validPythonTool},
			wantPass: true,
		},
		{
			name: "invalid go code fails",
			candidate: Candidate{
				Code: `package main
func main() { this is invalid }`,
				ToolDef: mcp.ToolDefinition{Name: "bad-go"},
			},
			result: &AdaptationResult{AdaptedCode: `package main
func main() { this is invalid }`},
			wantPass: false,
		},
		{
			name: "no code passes vacuously",
			candidate: Candidate{
				Code:    "",
				ToolDef: mcp.ToolDefinition{Name: "empty"},
			},
			result:   &AdaptationResult{},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr, err := stage.Execute(context.Background(), &tt.candidate, tt.result)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPass, sr.Passed)
		})
	}
}

func TestRegistrationStage_Execute(t *testing.T) {
	t.Run("registers new tool", func(t *testing.T) {
		metaRepo := setupTestDB(t)
		stage := &RegistrationStage{metaRepo: metaRepo}

		candidate := &Candidate{
			ToolDef: mcp.ToolDefinition{
				Name:        "new-tool",
				Description: "A new tool for testing",
				Category:    "test",
				Version:     "2.0.0",
			},
			Code: pureGoCode,
		}
		result := &AdaptationResult{
			AdaptedCode:   pureGoCode,
			CandidateCode: pureGoCode,
		}

		sr, err := stage.Execute(context.Background(), candidate, result)
		assert.NoError(t, err)
		assert.True(t, sr.Passed)
		assert.Contains(t, sr.Message, "registered")

		tools, err := metaRepo.ListTools()
		require.NoError(t, err)
		assert.Len(t, tools, 1)
		assert.Equal(t, "new-tool", tools[0].Name)
		assert.Equal(t, "2.0.0", tools[0].Version)
		assert.Equal(t, "test", tools[0].Category)
	})

	t.Run("updates existing tool", func(t *testing.T) {
		metaRepo := setupTestDB(t)
		stage := &RegistrationStage{metaRepo: metaRepo}

		existing := &repository.ToolRecord{
			ID: "existing-id", Name: "existing-tool",
			Description: "Old desc", Code: "old code",
			Category: "test", Version: "1.0.0",
			HealthStatus: "degraded", SourceType: "builtin",
		}
		err := metaRepo.CreateTool(existing)
		require.NoError(t, err)

		candidate := &Candidate{
			ToolDef: mcp.ToolDefinition{
				Name: "existing-tool", Description: "Updated desc",
				Category: "test", Version: "2.0.0",
			},
			Code: "updated code",
		}
		result := &AdaptationResult{
			AdaptedCode:   "updated code",
			CandidateCode: "updated code",
		}

		sr, err := stage.Execute(context.Background(), candidate, result)
		assert.NoError(t, err)
		assert.True(t, sr.Passed)
		assert.Contains(t, sr.Message, "updated")

		code, err := metaRepo.GetToolCode(context.Background(), "existing-id")
		require.NoError(t, err)
		assert.Equal(t, "updated code", code)
	})

	t.Run("nil metaRepo returns error", func(t *testing.T) {
		stage := &RegistrationStage{metaRepo: nil}
		candidate := &Candidate{
			ToolDef: mcp.ToolDefinition{Name: "no-repo"},
		}
		result := &AdaptationResult{}

		sr, err := stage.Execute(context.Background(), candidate, result)
		assert.NoError(t, err)
		assert.False(t, sr.Passed)
		assert.Contains(t, sr.Message, "not available")
	})
}

func TestPipeline_Run(t *testing.T) {
	metaRepo := setupTestDB(t)
	pipeline := NewPipeline(metaRepo)

	t.Run("full pipeline succeeds with valid input", func(t *testing.T) {
		result, err := pipeline.RunCandidate(context.Background(), &Candidate{
			Code: pureGoCode,
			ToolDef: mcp.ToolDefinition{
				Name:        "pipeline-test",
				Description: "Pipeline integration test",
				Version:     "1.0.0",
				Category:    "test",
			},
		})
		require.NoError(t, err)
		assert.Len(t, result.Stages, 5)
		assert.True(t, result.Registered)

		for _, s := range result.Stages {
			assert.True(t, s.Passed, "stage %s failed: %s", s.Name, s.Message)
		}

		assert.NotEmpty(t, result.AdaptedCode)
		assert.Contains(t, result.AdaptedCode, "package main")
	})

	t.Run("pipeline fails on blocked python code", func(t *testing.T) {
		result, err := pipeline.RunCandidate(context.Background(), &Candidate{
			Code: blockedPythonCode,
			ToolDef: mcp.ToolDefinition{
				Name:    "blocked-test",
				Version: "1.0.0",
			},
		})
		require.NoError(t, err)
		assert.False(t, result.Stages[0].Passed)
		assert.Contains(t, result.Error, "verification")
		assert.False(t, result.Registered)
	})

	t.Run("backward-compatible Run method works", func(t *testing.T) {
		result, err := pipeline.Run(context.Background(), mcp.ToolDefinition{
			Name:        "backward-compat",
			Description: "Testing backward compat",
			Version:     "0.5.0",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Stages, 5)
	})
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "Simple"},
		{"my-tool", "MyTool"},
		{"my.tool", "MyTool"},
		{"123tool", "_123tool"},
		{"", ""},
		{"spaced name", "SpacedName"},
		{"tool/with/slashes", "Toolwithslashes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
