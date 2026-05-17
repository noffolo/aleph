package decision

import (
	"context"

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

// Replan constants define the type of replan suggested by the Reflect phase.
const (
	ReplanNone    = ""        // no replan needed
	ReplanPartial = "partial" // only some steps need correction
	ReplanFull    = "full"    // the entire plan must be redone
)

// PlanResult is the output of the Plan phase.
type PlanResult struct {
	Intent          Intent
	Steps           []PlannedStep
	CanProceed      bool
	Reason          string
	CorrectionSteps []PlannedStep // alternative steps suggested by Reflect for failed steps
	ReplanType      string        // ReplanNone, ReplanPartial, or ReplanFull
}

// PlannedStep is a single atomic action in the plan.
type PlannedStep struct {
	ToolName              string
	Arguments             map[string]interface{}
	ExpectedOutcome       string
	RequiresConfirmation  bool
	Depends               []string // tool names that must succeed before this step executes
	Rationale             string   // why this step was chosen (runtime-only, not in proto)
	Fallback              string   // fallback action if this step fails (runtime-only, not in proto)
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
	Provider              llm.Provider
	ProviderBaseURL       string           // base URL used to create the provider (fallback for PlanWithProvider)
	MetaRepo              ToolRepository   // interface for tool/chat history operations
	Executor              ToolExecutor     // interface for executing tools
	Registry              PluginRegistry   // interface for registry validation
	MaxAttempts           int              // max loop iterations (default 5)
	LinkPredictor         LinkPredictor    // optional GNN link predictor for confidence blending
	Graph                 *gnn.Graph       // optional workspace knowledge graph for link prediction
	Reflector             Reflector        // optional custom reflector (default: DefaultReflector)
	ConfirmationThreshold float64          // 0.0-1.0: if step requiresConfirmation and threshold > 0, auto-skip; 0 = no check
	TruncationThreshold   int              // max output length before truncation signal (default 1900)
}

// DecisionEngine orchestrates the Plan→Act→Observe→Reflect→Admit loop.
type DecisionEngine interface {
	Plan(ctx context.Context, msg string, projectID string, agentID string, ontContent []byte, agent *alephv1.Agent) (*PlanResult, error)
	PlanWithProvider(ctx context.Context, msg string, projectID string, agentID string, ontContent []byte, agent *alephv1.Agent, provider llm.Provider) (*PlanResult, error)
	Act(ctx context.Context, step PlannedStep, projectID string) (*ActResult, error)
	Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error)
	Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error)
	Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error)
	ShouldAutoSkip(step PlannedStep) bool
	BuildToolsMap(ctx context.Context) []map[string]interface{}
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


