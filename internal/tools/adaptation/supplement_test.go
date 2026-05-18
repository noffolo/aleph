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
)

// ---------------------------------------------------------------------------
// templates.go supplement
// ---------------------------------------------------------------------------

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "HelloWorld"},
		{"my_tool_name", "MyToolName"},
		{"simple", "Simple"},
		{"alreadyPascal", "AlreadyPascal"},
		{"", ""},
		{"a_b_c", "ABC"},
		{"one-two", "OneTwo"},
		{"has space", "HasSpace"},
		{"__leading__underscores__", "LeadingUnderscores"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toPascalCase(tt.input))
		})
	}
}

func TestCountSchemaProps(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   int
	}{
		{"nil schema", nil, 0},
		{"empty schema", map[string]any{}, 0},
		{"no properties key", map[string]any{"type": "object"}, 0},
		{"properties not a map", map[string]any{"properties": "invalid"}, 0},
		{
			"single property",
			map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			1,
		},
		{
			"three properties",
			map[string]any{
				"properties": map[string]any{
					"name":     map[string]any{"type": "string"},
					"age":      map[string]any{"type": "integer"},
					"location": map[string]any{"type": "string"},
				},
			},
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countSchemaProps(tt.schema))
		})
	}
}

// ---------------------------------------------------------------------------
// suggestion.go supplement
// ---------------------------------------------------------------------------

func TestMatchQuality(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		toolName   string
		desc       string
		wantScore  float64
		wantReason string
	}{
		{
			name:       "empty query",
			query:      "",
			toolName:   "anything",
			desc:       "anything",
			wantScore:  0.0,
			wantReason: "empty query",
		},
		{
			name:       "exact name match",
			query:      "my-tool",
			toolName:   "my-tool",
			desc:       "does stuff",
			wantScore:  0.95,
			wantReason: "exact tool name match",
		},
		{
			name:       "exact description match",
			query:      "does stuff",
			toolName:   "other-name",
			desc:       "does stuff",
			wantScore:  0.70,
			wantReason: "exact tool description match",
		},
		{
			name:       "partial name match",
			query:      "my",
			toolName:   "my-tool",
			desc:       "desc",
			wantScore:  0.70,
			wantReason: "partial tool name match",
		},
		{
			name:       "partial description match",
			query:      "stuff",
			toolName:   "tool",
			desc:       "does stuff",
			wantScore:  0.50,
			wantReason: "partial tool description match",
		},
		{
			name:       "no match",
			query:      "zzz",
			toolName:   "tool",
			desc:       "desc",
			wantScore:  0.0,
			wantReason: "no match",
		},
		{
			name:       "name match wins over desc match",
			query:      "over",
			toolName:   "overlap",
			desc:       "over lap",
			wantScore:  0.70,
			wantReason: "partial tool name match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, reason := matchQuality(tt.query, tt.toolName, tt.desc)
			assert.InDelta(t, tt.wantScore, score, 0.01)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

// ---------------------------------------------------------------------------
// pipeline.go utility supplement
// ---------------------------------------------------------------------------

func TestExtractGoImports(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		count int
	}{
		{"no imports", "package main\nfunc main() {}", 0},
		{
			"single import",
			`package main
import "fmt"
func main() { fmt.Println("ok") }`,
			1,
		},
		{
			"two imports",
			`package main
import "fmt"
import "os"
func main() {}`,
			2,
		},
		{
			"deduplicates",
			`package main
import "fmt"
import "fmt"
func main() {}`,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := extractGoImports(tt.code)
			assert.Len(t, imports, tt.count)
		})
	}
}

func TestExtractPythonImports(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		count int
	}{
		{"no imports", "# python\ndef run(): pass", 0},
		{
			"import and from",
			"import os\nfrom sys import argv\n\ndef run(): pass",
			2,
		},
		{
			"deduplicates",
			"import os\nimport os\nfrom sys import argv",
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := extractPythonImports(tt.code)
			assert.Len(t, imports, tt.count)
		})
	}
}

func TestDetectTemplateType(t *testing.T) {
	tests := []struct {
		name string
		code string
		def  mcp.ToolDefinition
		want TemplateType
	}{
		{
			name: "adapter when InputSchema has properties",
			def: mcp.ToolDefinition{
				InputSchema: map[string]any{"type": "object"},
			},
			want: TemplateAdapter,
		},
		{
			name: "python code → wrapper",
			code: "# python\ndef run(): pass",
			def:  mcp.ToolDefinition{Name: "py-tool"},
			want: TemplateWrapper,
		},
		{
			name: "library go (func but no main) → decorator",
			code: "package lib\nfunc DoSomething() int { return 1 }",
			def:  mcp.ToolDefinition{Name: "lib-tool"},
			want: TemplateDecorator,
		},
		{
			name: "go with main → wrapper",
			code: "package main\nfunc main() {}",
			def:  mcp.ToolDefinition{Name: "main-tool"},
			want: TemplateWrapper,
		},
		{
			name: "go with main args → wrapper",
			code: "package main\nfunc main(args []string) {}",
			def:  mcp.ToolDefinition{Name: "main-args-tool"},
			want: TemplateWrapper,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectTemplateType(tt.code, tt.def))
		})
	}
}

func TestDetectIssues(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		lang       string
		hasIssues  bool
		issueCount int
	}{
		{"empty code, no issues", "", "go", false, 0},
		{"clean go code", "package main\nfunc main() {}", "go", false, 0},
		{"clean python", "# python\ndef run(): pass", "python", false, 0},
		{"go with panic", "package main\nfunc main() { panic(\"argh\") }", "go", true, 1},
		{"go with log.Fatal", `package main
import "log"
func main() { log.Fatal("dead") }`, "go", true, 1},
		{"python with panic (not checked)", `# python
def run():
    raise Exception("panic")`, "python", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := detectIssues(tt.code, tt.lang)
			if tt.hasIssues {
				assert.NotEmpty(t, issues)
			} else {
				assert.Empty(t, issues)
			}
			if tt.issueCount > 0 {
				assert.Len(t, issues, tt.issueCount)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"hello", 5, "hello"},
		{"hi", 2, "hi"},
		{"hi", 0, "..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.input, tt.maxLen))
		})
	}
}

// ---------------------------------------------------------------------------
// suggestion.go struct supplement
// ---------------------------------------------------------------------------

func setupSuggesterDB(t *testing.T) *repository.MetadataRepository {
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

func TestNewSuggester(t *testing.T) {
	repo := setupSuggesterDB(t)
	s := NewSuggester(repo, nil)
	assert.NotNil(t, s)
	assert.NotNil(t, s.metaRepo)
	assert.Nil(t, s.discovery)
}

func TestSuggester_Suggest(t *testing.T) {
	repo := setupSuggesterDB(t)
	mustCreateTool(t, repo, "alpha", "searches things")
	mustCreateTool(t, repo, "beta", "processes data")

	s := NewSuggester(repo, nil)

	t.Run("matches tools by name", func(t *testing.T) {
		results, err := s.Suggest(context.Background(), "alpha")
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "alpha", results[0].ToolDef.Name)
		assert.InDelta(t, 0.95, results[0].Confidence, 0.01)
	})

	t.Run("matches tools by partial name", func(t *testing.T) {
		results, err := s.Suggest(context.Background(), "bet")
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "beta", results[0].ToolDef.Name)
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		results, err := s.Suggest(context.Background(), "zzz")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestNewVersioningRollback(t *testing.T) {
	repo := setupSuggesterDB(t)
	vr := NewVersioningRollback(repo)
	assert.NotNil(t, vr)
	assert.NotNil(t, vr.metaRepo)
	assert.Empty(t, vr.versions)
}

func TestVersioningRollback_Snapshot(t *testing.T) {
	vr := NewVersioningRollback(nil)
	def := mcp.ToolDefinition{Name: "tool-1", Version: "1.0.0"}
	vr.Snapshot(def, "initial")

	assert.Len(t, vr.versions, 1)
	assert.Equal(t, "v1", vr.versions[0].Version)
	assert.Equal(t, "initial", vr.versions[0].Reason)

	vr.Snapshot(def, "updated")
	assert.Len(t, vr.versions, 2)
	assert.Equal(t, "v2", vr.versions[1].Version)
}

func TestVersioningRollback_ListVersions(t *testing.T) {
	vr := NewVersioningRollback(nil)

	_, err := vr.ListVersions()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no versions")

	def := mcp.ToolDefinition{Name: "tool-1"}
	vr.Snapshot(def, "first")
	vr.Snapshot(def, "second")

	result, err := vr.ListVersions()
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "v1", result[0].Version)
	assert.Equal(t, "v2", result[1].Version)
}

func TestVersioningRollback_Rollback(t *testing.T) {
	vr := NewVersioningRollback(nil)

	err := vr.Rollback("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version required")

	def := mcp.ToolDefinition{Name: "tool-1", Version: "1.0.0"}
	vr.Snapshot(def, "initial")
	vr.Snapshot(def, "updated")

	err = vr.Rollback("v1")
	assert.NoError(t, err)

	err = vr.Rollback("v99")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func mustCreateTool(t *testing.T, repo *repository.MetadataRepository, name, desc string) {
	t.Helper()
	err := repo.CreateTool(&repository.ToolRecord{
		ID:          name + "-id",
		Name:        name,
		Description: desc,
		Code:        "package main\nfunc main() {}",
		Category:    "test",
		Version:     "1.0.0",
		SourceType:  "builtin",
	})
	require.NoError(t, err)
}

func TestContainsSuspiciousOutput(t *testing.T) {
	tests := []struct {
		input   string
		suspect bool
	}{
		{"normal output", false},
		{"everything ok", false},
		{"found /etc/passwd", true},
		{"reading /etc/shadow", true},
		{"user root: authenticated", true},
		{"run sudo rm", true},
		{"chmod 777 all the things", true},
		{"clean up with rm -rf /tmp", true},
		{"case insensitive: /ETC/PASSWD", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.suspect, containsSuspiciousOutput(tt.input))
		})
	}
}

func setupMetaRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	metaRepo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return metaRepo
}

func TestNewFactory(t *testing.T) {
	metaRepo := setupMetaRepo(t)
	factory := NewFactory(metaRepo, nil)

	assert.NotNil(t, factory)
	assert.NotNil(t, factory.metaRepo)
	assert.Nil(t, factory.discovery)
}

func TestFactory_CreatePipeline(t *testing.T) {
	metaRepo := setupMetaRepo(t)
	factory := NewFactory(metaRepo, nil)

	pipeline := factory.CreatePipeline()
	assert.NotNil(t, pipeline)
}

func TestFactory_CreateSuggester(t *testing.T) {
	metaRepo := setupMetaRepo(t)
	factory := NewFactory(metaRepo, nil)

	suggester := factory.CreateSuggester()
	assert.NotNil(t, suggester)
}

func TestFactory_CreateVersioningRollback(t *testing.T) {
	metaRepo := setupMetaRepo(t)
	factory := NewFactory(metaRepo, nil)

	rollback := factory.CreateVersioningRollback()
	assert.NotNil(t, rollback)
}
