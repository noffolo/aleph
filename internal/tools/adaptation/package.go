package adaptation

import (
	"context"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// Factory creates adaptation pipeline components.
type Factory struct {
	metaRepo  *repository.MetadataRepository
	discovery *mcp.DiscoveryEngine
}

// NewFactory creates a new adaptation factory.
func NewFactory(metaRepo *repository.MetadataRepository, discovery *mcp.DiscoveryEngine) *Factory {
	return &Factory{
		metaRepo:  metaRepo,
		discovery: discovery,
	}
}

// CreatePipeline creates a new adaptation pipeline.
func (f *Factory) CreatePipeline() *Pipeline {
	return NewPipeline(f.metaRepo)
}

// CreateSuggester creates a new suggestion engine.
func (f *Factory) CreateSuggester() *Suggester {
	return NewSuggester(f.metaRepo, f.discovery)
}

// CreateVersioningRollback creates a versioning rollback handler.
func (f *Factory) CreateVersioningRollback() *VersioningRollback {
	return NewVersioningRollback(f.metaRepo)
}

// Candidate holds the full input for a pipeline adaptation run.
type Candidate struct {
	ToolDef      mcp.ToolDefinition
	Code         string       // candidate source code to adapt
	TemplateType TemplateType // optional override of template type
}

// AnalysisDetail holds detailed results from the analysis stage.
type AnalysisDetail struct {
	Language     string       // "go" or "python"
	Dependencies []string     // detected import dependencies
	Complexity   int          // estimated cyclomatic complexity
	TemplateType TemplateType // detected or assigned template type
	HasTests     bool         // whether test patterns are present
	Issues       []string     // code quality issues found
}

// AdaptationResult holds the result of an adaptation run.
type AdaptationResult struct {
	ToolDefinition mcp.ToolDefinition
	Stages         []StageResult
	Version        string
	Registered     bool
	Error          string
	CandidateCode  string         // original candidate code
	AdaptedCode    string         // code after adaptation
	TemplateType   TemplateType   // template type used
	Analysis       AnalysisDetail // analysis results
}

// StageResult holds the result of a single pipeline stage.
type StageResult struct {
	Name    string
	Passed  bool
	Message string
}

// Suggestion represents a tool adaptation suggestion.
type Suggestion struct {
	Query        string
	ToolDef      mcp.ToolDefinition
	Confidence   float64
	Reason       string
	TemplateType TemplateType
}

// TemplateType defines the type of adaptation template.
type TemplateType string

const (
	TemplateWrapper   TemplateType = "wrapper"
	TemplateAdapter   TemplateType = "adapter"
	TemplateDecorator TemplateType = "decorator"
)

// PipelineStage defines the interface for pipeline stages.
// Each stage receives the candidate and the accumulating result, allowing
// stages to pass data downstream (e.g. AnalysisStage writes to result.Analysis,
// AdaptationStage reads candidate.Code and writes result.AdaptedCode).
type PipelineStage interface {
	Execute(ctx context.Context, candidate *Candidate, result *AdaptationResult) (StageResult, error)
}

//go:generate mockgen -source=package.go -destination=mocks/mock_package.go -package=mocks
