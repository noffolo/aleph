package decision

import (
	"context"

	"connectrpc.com/connect"
	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/ff3300/aleph-v2/internal/llm"
)

// Intent represents the parsed user request after planning.
type Intent struct {
	PrimaryGoal   string   // what the user wants to accomplish
	NeededTools   []string // tool names required
	TargetObjects []string // ontology object references
	Confidence    float64  // 0.0-1.0 how confident the engine is about this intent
}

// PlanResult is the output of the Plan phase.
type PlanResult struct {
	Intent     Intent
	Steps      []PlannedStep
	CanProceed bool
	Reason     string
}

// PlannedStep is a single atomic action in the plan.
type PlannedStep struct {
	ToolName              string
	Arguments             map[string]interface{}
	ExpectedOutcome       string
	RequiresConfirmation  bool
}

// ActResult is the output of executing a single step.
type ActResult struct {
	Step       PlannedStep
	Output     string
	Error      string
	DurationMs int64
}

// Observation is the result of evaluating an act.
type Observation struct {
	Step       PlannedStep
	ActResult  ActResult
	Success    bool
	TrustDelta float64  // how trust score changed
	Issues     []string // any problems found
}

// EngineConfig holds dependencies needed by the DecisionEngine.
type EngineConfig struct {
	Provider      llm.Provider
	MetaRepo      ToolRepository   // interface for tool/chat history operations
	Executor      ToolExecutor     // interface for executing tools
	Registry      PluginRegistry   // interface for registry validation
	MaxAttempts   int              // max loop iterations (default 5)
	LinkPredictor LinkPredictor    // optional GNN link predictor for confidence blending
	Graph         *gnn.Graph       // optional workspace knowledge graph for link prediction
}

// DecisionEngine orchestrates the Plan→Act→Observe→Reflect→Admit loop.
type DecisionEngine interface {
	Plan(ctx context.Context, msg string, projectID string, agentID string, ontContent []byte, agent *alephv1.Agent) (*PlanResult, error)
	Act(ctx context.Context, step PlannedStep, projectID string) (*ActResult, error)
	Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error)
	Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error)
	Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error)
}

// ToolRepository is the minimal interface decision engine needs from metadata repo.
type ToolRepository interface {
	SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error
	GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error)
	ListTools(ctx context.Context) ([]ToolDef, error)
}

// ToolExecutor executes tool calls (the dispatch switch).
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID string, agentID string) (string, bool, error)
}

// PluginRegistry validates tool names against available plugins.
type PluginRegistry interface {
	GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error)
}

// ChatMessage is a simplified chat message struct.
type ChatMessage struct {
	Role     string
	Content  string
	ToolCall string
}

// ToolDef is a simplified tool definition.
type ToolDef struct {
	Name        string
	Description string
	Code        string
}

// ComponentMetadata is the minimal registry info needed for validation.
type ComponentMetadata struct {
	ID       string
	Name     string
	Category string
	Status   string
}

// ToolDefinition is a typed tool definition for LLM function calling.
type ToolDefinition struct {
	Type       string           `json:"type"`
	Function   FunctionDef      `json:"function"`
}

// FunctionDef is the function descriptor inside a tool definition.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  *ParameterDef   `json:"parameters,omitempty"`
}

// ParameterDef describes the JSON schema for tool parameters.
type ParameterDef struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertyDef    `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertyDef describes a single property in a tool parameter.
type PropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// GetToolExecutor returns the configured executor or nil.
func GetToolExecutor() ToolExecutor {
	if NewToolExecutor == nil {
		return nil
	}
	return NewToolExecutor(nil, nil, nil, nil)
}

// NewToolExecutor creates a tool executor that wraps the handler's dispatch logic.
// This is called from the handler package to bridge to the decision engine.
var NewToolExecutor func(
	executeQuery func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error),
	analyzeSentiment func(ctx context.Context, text string) (string, error),
	getTrustScore func(ctx context.Context, entityID string) (string, error),
	getComponentByID func(id string) (*ComponentMetadata, error),
) ToolExecutor
