package decision

import (
	"context"
	"fmt"
	"strings"
	"time"

	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/ff3300/aleph-v2/internal/llm"
)

// Engine is the concrete implementation of DecisionEngine.
type Engine struct {
	provider      llm.Provider
	metaRepo      ToolRepository
	executor      ToolExecutor
	registry      PluginRegistry
	maxAttempts   int
	linkPredictor LinkPredictor
	graph         *gnn.Graph
}

// compile-time interface check
var _ DecisionEngine = (*Engine)(nil)

// NewEngine creates a new DecisionEngine with the given config.
func NewEngine(cfg EngineConfig) *Engine {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &Engine{
		provider:      cfg.Provider,
		metaRepo:      cfg.MetaRepo,
		executor:      cfg.Executor,
		registry:      cfg.Registry,
		maxAttempts:   maxAttempts,
		linkPredictor: cfg.LinkPredictor,
		graph:         cfg.Graph,
	}
}

// PlanWithProvider creates a plan using the given provider.
// If provider is nil, falls back to keyword-based heuristic planning.
func (e *Engine) PlanWithProvider(
	ctx context.Context,
	msg string,
	projectID string,
	agentID string,
	ontContent []byte,
	agent *alephv1.Agent,
	provider llm.Provider,
) (*PlanResult, error) {
	if provider == nil {
		return e.Plan(ctx, msg, projectID, agentID, ontContent, agent)
	}

	useTools := e.BuildToolsMap(ctx)
	systemPrompt := "You are a planning agent. Analyze the user's request and determine which tools to call."

	var model, apiKey, baseURL string
	if agent != nil {
		model = agent.Model
		apiKey = agent.ApiKey
		baseURL = agent.BaseUrl
	}
	if model == "" {
		model = "llama3"
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	req := llm.CompletionRequest{
		Model:        model,
		Messages:     []map[string]interface{}{{"role": "user", "content": msg}},
		Tools:        useTools,
		SystemPrompt: systemPrompt,
		ApiKey:       apiKey,
		BaseURL:      baseURL,
	}

	completion, err := provider.Complete(ctx, req)
	if err != nil {
		return e.Plan(ctx, msg, projectID, agentID, ontContent, agent)
	}

	intent := Intent{
		PrimaryGoal: msg,
		Confidence:  0.7,
	}

	steps := make([]PlannedStep, 0, len(completion.ToolCalls))
	for _, tc := range completion.ToolCalls {
		steps = append(steps, PlannedStep{
			ToolName:  tc.Name,
			Arguments: tc.Arguments,
		})
	}

	return &PlanResult{
		Intent:     intent,
		Steps:      steps,
		CanProceed: true,
		Reason:     "planned via LLM",
	}, nil
}

// Plan implements DecisionEngine.Plan.
// It builds tool definitions from hardcoded + registered tools,
// creates the initial plan with the user's intent.
// When provider is nil, operates in degraded mode with heuristic-based planning.
func (e *Engine) Plan(ctx context.Context, msg string, projectID string, agentID string, ontContent []byte, agent *alephv1.Agent) (*PlanResult, error) {
	if e.provider == nil {
		return &PlanResult{
			Intent: Intent{
				PrimaryGoal:   msg,
				NeededTools:   e.inferToolsFromMessage(ctx, msg, e.buildToolDefinitions(ctx)),
				TargetObjects: nil,
				Confidence:    0.5,
			},
			Steps: []PlannedStep{{
				ToolName:  "query_dispatch",
				Arguments: map[string]interface{}{"query": msg},
			}},
			CanProceed: true,
			Reason:     "degraded mode: heuristic planning (no LLM provider)",
		}, nil
	}
	// Build tool definitions for LLM use
	toolDefs := e.buildToolDefinitions(ctx)

	// Parse intent from the user message
	neededTools := e.inferToolsFromMessage(ctx, msg, toolDefs)
	intent := Intent{
		PrimaryGoal:   msg,
		NeededTools:   neededTools,
		TargetObjects: e.extractObjectReferences(msg),
		Confidence:    0.8,
	}

	// Blend GNN link prediction scores into confidence.
	// If a LinkPredictor and workspace graph are available, run inference
	// on detected tools/objects to adjust keyword-based confidence.
	if e.linkPredictor != nil && e.graph != nil && e.linkPredictor.IsTrained() && len(neededTools) > 0 {
		var totalScore float64
		var count int
		for _, tool := range neededTools {
			scores, err := e.linkPredictor.PredictLinks(ctx, e.graph, tool)
			if err == nil {
				totalScore += ConfidenceFromPredictions(scores)
				count++
			}
		}
		if count > 0 {
			gnnConf := totalScore / float64(count)
			// Blend: 70% keyword, 30% GNN
			intent.Confidence = 0.7*intent.Confidence + 0.3*gnnConf
		}
	}

	// Create a single planned step for the first action
	// The loop in Chat will handle iteration; we create an initial plan
	// that triggers the LLM to decide which tool to call.
	steps := []PlannedStep{}
	if len(intent.NeededTools) > 0 {
		for _, toolName := range intent.NeededTools {
			step := PlannedStep{
				ToolName:             toolName,
				Arguments:            map[string]interface{}{},
				ExpectedOutcome:      fmt.Sprintf("execute %s to fulfill user request", toolName),
				RequiresConfirmation: !e.isKnownTool(ctx, toolName),
			}
			steps = append(steps, step)
		}
	}

	return &PlanResult{
		Intent:     intent,
		Steps:      steps,
		CanProceed: true,
		Reason:     "plan ready",
	}, nil
}

// Act implements DecisionEngine.Act.
// Executes a single tool via the executor.
func (e *Engine) Act(ctx context.Context, step PlannedStep, projectID string) (*ActResult, error) {
	start := time.Now()

	output, requiresConfirmation, err := e.executor.ExecuteTool(ctx, step.ToolName, step.Arguments, projectID, "")
	durationMs := time.Since(start).Milliseconds()

	result := &ActResult{
		Step:       step,
		DurationMs: durationMs,
	}

	if err != nil {
		result.Error = err.Error()
		result.Output = ""
	} else {
		result.Output = output
	}

	// If the tool requires confirmation, set requiresConfirmation
	if requiresConfirmation {
		step.RequiresConfirmation = true
		result.Step = step
	}

	return result, nil
}

// Observe implements DecisionEngine.Observe.
// Evaluates the result of a tool execution.
func (e *Engine) Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error) {
	obs := &Observation{
		Step:       step,
		ActResult:  *result,
		Success:    result.Error == "",
		TrustDelta: 0,
		Issues:     []string{},
	}

	if result.Error != "" {
		obs.Success = false
		obs.Issues = append(obs.Issues, result.Error)
	}

	if result.Output == "" && result.Error == "" {
		obs.Issues = append(obs.Issues, "tool returned empty output")
	}

	return obs, nil
}

// Reflect implements DecisionEngine.Reflect.
// Determines next steps based on observations from previous actions.
func (e *Engine) Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error) {
	// Check if we already have a plan result that says we can't proceed
	if !plan.CanProceed {
		return plan, nil
	}

	// If the last observation failed, mark as unable to proceed
	if len(observations) > 0 {
		lastObs := observations[len(observations)-1]
		if !lastObs.Success {
			return &PlanResult{
				Intent:     plan.Intent,
				Steps:      plan.Steps,
				CanProceed: false,
				Reason:     fmt.Sprintf("tool execution failed: %s", strings.Join(lastObs.Issues, "; ")),
			}, nil
		}
	}

	// No reflection needed — Chat loop handles iteration
	return plan, nil
}

// Admit implements DecisionEngine.Admit.
// Decides whether to continue the loop or stop.
func (e *Engine) Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = e.maxAttempts
	}

	// If we've used all attempts, stop
	if len(results) >= maxAttempts {
		return true, nil
	}

	// If the last result had an error, admit (stop the loop)
	if len(results) > 0 {
		last := results[len(results)-1]
		if last.Error != "" {
			return true, nil
		}
	}

	return false, nil
}

// TrainLinkModel trains the GNN link predictor on the workspace graph.
// If the engine does not have a LinkPredictor configured, this is a no-op.
// The graph must have at least 1 node and 1 edge for meaningful training.
func (e *Engine) TrainLinkModel(ctx context.Context, graph *gnn.Graph, epochs int) error {
	if e.linkPredictor == nil {
		return nil
	}
	if graph == nil || graph.NumNodes() == 0 {
		return fmt.Errorf("decision: cannot train link model on empty graph")
	}
	return e.linkPredictor.TrainFromGraph(ctx, graph, epochs)
}

// isKnownTool checks if a tool name is one of the built-in tools or registered in the registry.
func (e *Engine) isKnownTool(ctx context.Context, name string) bool {
	switch name {
	case "search_data", "analyze_sentiment", "get_trust_score":
		return true
	}
	return validateToolName(ctx, name, e.registry)
}

// inferToolsFromMessage guesses which tools might be needed based on the message.
func (e *Engine) inferToolsFromMessage(ctx context.Context, msg string, toolDefs []ToolDefinition) []string {
	lower := strings.ToLower(msg)
	var tools []string

	if strings.Contains(lower, "search") || strings.Contains(lower, "find") || strings.Contains(lower, "query") || strings.Contains(lower, "show") || strings.Contains(lower, "data") || strings.Contains(lower, "object") {
		tools = append(tools, "search_data")
	}
	if strings.Contains(lower, "sentiment") || strings.Contains(lower, "feeling") || strings.Contains(lower, "opinion") {
		tools = append(tools, "analyze_sentiment")
	}
	if strings.Contains(lower, "trust") || strings.Contains(lower, "score") || strings.Contains(lower, "brier") || strings.Contains(lower, "prediction") {
		tools = append(tools, "get_trust_score")
	}

	// Also check registered tools
	if e.metaRepo != nil {
		registeredTools, err := e.metaRepo.ListTools(ctx)
		if err == nil {
			for _, t := range registeredTools {
				if strings.Contains(lower, strings.ToLower(t.Name)) {
					tools = append(tools, t.Name)
				}
			}
		}
	}

	return tools
}

// extractObjectReferences pulls potential ontology object names from the message.
func (e *Engine) extractObjectReferences(msg string) []string {
	// Simple heuristic: look for capitalized words that could be object names
	// The LLM handles actual object resolution — this is just a best guess
	return nil
}

// buildToolDefinitions creates typed tool definitions similar to the hardcoded
// map[string]interface{} definitions in the original Chat().
func (e *Engine) buildToolDefinitions(ctx context.Context) []ToolDefinition {
	defs := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "search_data",
				Description: "Search records from a specific business object defined in the ontology.",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"object_name": {Type: "string"},
						"limit":       {Type: "integer", Default: float64(10)},
					},
					Required: []string{"object_name"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "analyze_sentiment",
				Description: "Analyze the sentiment of text data. Returns a score from -1.0 (negative) to 1.0 (positive) and a label (positive/negative/neutral).",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"text": {Type: "string", Description: "The text to analyze"},
					},
					Required: []string{"text"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_trust_score",
				Description: "Get the trust score for a prediction entity. Returns the Brier score (0.0 = perfect, 1.0 = worst) and trust level.",
				Parameters: &ParameterDef{
					Type: "object",
					Properties: map[string]PropertyDef{
						"entity_id": {Type: "string", Description: "The entity ID to check trust for"},
					},
					Required: []string{"entity_id"},
				},
			},
		},
	}

	// Add registered tools from metadata repository
	if e.metaRepo != nil {
		registeredTools, err := e.metaRepo.ListTools(ctx)
		if err == nil {
			for _, t := range registeredTools {
				fn := FunctionDef{
					Name:        t.Name,
					Description: t.Description,
				}
				defs = append(defs, ToolDefinition{
					Type:     "function",
					Function: fn,
				})
			}
		}
	}

	return defs
}

// BuildToolsMap converts typed ToolDefinitions to map format expected by llm.Provider.
// This is used by the handler to pass tools to the LLM.
func (e *Engine) BuildToolsMap(ctx context.Context) []map[string]interface{} {
	defs := e.buildToolDefinitions(ctx)
	result := make([]map[string]interface{}, 0, len(defs))
	for _, d := range defs {
		fnMap := map[string]interface{}{
			"name":        d.Function.Name,
			"description": d.Function.Description,
		}
		if d.Function.Parameters != nil {
			props := make(map[string]interface{})
			for k, v := range d.Function.Parameters.Properties {
				p := map[string]interface{}{"type": v.Type}
				if v.Description != "" {
					p["description"] = v.Description
				}
				if v.Default != nil {
					p["default"] = v.Default
				}
				props[k] = p
			}
			fnMap["parameters"] = map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   d.Function.Parameters.Required,
			}
		}
		result = append(result, map[string]interface{}{
			"type":     "function",
			"function": fnMap,
		})
	}
	return result
}
