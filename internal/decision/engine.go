package decision

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/telemetry"
)

// toolExecutionHistory tracks success/failure counts per tool for TrustDelta calculation.
type toolExecutionHistory struct {
	successes int
	failures  int
}

// total returns the total number of recorded executions.
func (h *toolExecutionHistory) total() int {
	return h.successes + h.failures
}

// Engine is the concrete implementation of DecisionEngine.
type Engine struct {
	provider              llm.Provider
	providerBaseURL       string
	metaRepo              ToolRepository
	executor              ToolExecutor
	registry              PluginRegistry
	maxAttempts           int
	linkPredictor         LinkPredictor
	graph                 *gnn.Graph
	reflector             Reflector
	confirmationThreshold float64
	truncationThreshold   int
	toolHistory           map[string]*toolExecutionHistory // per-tool success/failure tracking
}

// compile-time interface check
var _ DecisionEngine = (*Engine)(nil)

// NewEngine creates a new DecisionEngine with the given config.
func NewEngine(cfg EngineConfig) *Engine {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	reflector := cfg.Reflector
	if reflector == nil {
		reflector = NewDefaultReflector()
	}
	confThreshold := cfg.ConfirmationThreshold
	if confThreshold < 0 {
		confThreshold = 0
	}
	if confThreshold > 1 {
		confThreshold = 1
	}
	truncThreshold := cfg.TruncationThreshold
	if truncThreshold <= 0 {
		truncThreshold = 1900
	}
	return &Engine{
		provider:              cfg.Provider,
		providerBaseURL:       cfg.ProviderBaseURL,
		metaRepo:              cfg.MetaRepo,
		executor:              cfg.Executor,
		registry:              cfg.Registry,
		maxAttempts:           maxAttempts,
		linkPredictor:         cfg.LinkPredictor,
		graph:                 cfg.Graph,
		reflector:             reflector,
		confirmationThreshold: confThreshold,
		truncationThreshold:   truncThreshold,
		toolHistory:           make(map[string]*toolExecutionHistory),
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
		if e.providerBaseURL != "" {
			baseURL = e.providerBaseURL
		} else {
			baseURL = "http://localhost:11434"
		}
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
		PrimaryGoal:   msg,
		TargetObjects: e.extractObjectReferencesWithOntology(msg, ontContent),
		Confidence:    0.7,
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
				TargetObjects: e.extractObjectReferencesWithOntology(msg, ontContent),
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
		TargetObjects: e.extractObjectReferencesWithOntology(msg, ontContent),
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

	telemetry.RecordPAORACycle("plan", "success")
	return &PlanResult{
		Intent:     intent,
		Steps:      steps,
		CanProceed: true,
		Reason:     "plan ready",
	}, nil
}

// Act implements DecisionEngine.Act.
// Executes a single tool via the executor and records the outcome in tool history
// for subsequent TrustDelta computation during Observe.
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

	// Record execution outcome in tool history for TrustDelta calculation
	hist := e.toolHistory[step.ToolName]
	if hist == nil {
		hist = &toolExecutionHistory{}
		e.toolHistory[step.ToolName] = hist
	}
	if err != nil {
		hist.failures++
	} else {
		hist.successes++
	}

	// If the tool requires confirmation, set requiresConfirmation
	if requiresConfirmation {
		step.RequiresConfirmation = true
		result.Step = step
	}

	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	telemetry.RecordPAORACycle("act", outcome)

	return result, nil
}

// Observe implements DecisionEngine.Observe.
// Evaluates the result of a tool execution.
// TrustDelta is computed from recorded tool history: (success/total) - 0.5 baseline,
// producing -0.5..+0.5. Initial executions with no history default to a neutral 0.05
// if successful, -0.1 if failed.
func (e *Engine) Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error) {
	// Compute TrustDelta from tool execution history
	hist := e.toolHistory[step.ToolName]
	var trustDelta float64
	if hist != nil && hist.total() > 0 {
		// Baseline = 0.5. Rate = successes / total.
		// Positive delta when success rate exceeds baseline.
		rate := float64(hist.successes) / float64(hist.total())
		trustDelta = rate - 0.5
	} else if result.Error == "" {
		// No history yet, first execution succeeded — small positive delta
		trustDelta = 0.05
	} else {
		// No history yet, first execution failed — small negative delta
		trustDelta = -0.1
	}

	obs := &Observation{
		Step:       step,
		ActResult:  *result,
		Success:    result.Error == "",
		TrustDelta: trustDelta,
		Issues:     []string{},
	}

	if result.Error != "" {
		obs.Success = false
		obs.Issues = append(obs.Issues, result.Error)
	}

	if result.Output == "" && result.Error == "" {
		obs.Issues = append(obs.Issues, "tool returned empty output")
	}

	// Configurable truncation threshold from EngineConfig
	threshold := e.truncationThreshold
	if threshold <= 0 {
		threshold = 1900
	}
	if len(result.Output) > threshold {
		obs.Issues = append(obs.Issues, fmt.Sprintf("output was truncated due to context limits (%d > %d)", len(result.Output), threshold))
	}

	outcome := "success"
	if !obs.Success {
		outcome = "error"
	}
	telemetry.RecordPAORACycle("observe", outcome)

	return obs, nil
}

// Reflect implements DecisionEngine.Reflect.
// Delegates to the configured Reflector (DefaultReflector by default) which
// analyzes ALL observations, classifies each as expected/unexpected/critical,
// and produces a structured reflection with actionable insights.
func (e *Engine) Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error) {
	result, err := e.reflector.Reflect(ctx, plan, observations)
	if err != nil {
		telemetry.RecordPAORACycle("reflect", "error")
		return plan, fmt.Errorf("reflection failed: %w", err)
	}
	if result != nil && !result.CanProceed {
		telemetry.RecordPAORACycle("reflect", "error")
	} else {
		telemetry.RecordPAORACycle("reflect", "success")
	}
	return result, err
}

// Admit implements DecisionEngine.Admit.
// Decides whether to continue the loop or stop.
func (e *Engine) Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = e.maxAttempts
	}

	if len(results) >= maxAttempts {
		telemetry.RecordPAORACycle("admit", "max_attempts")
		return true, nil
	}

	if len(results) > 0 {
		last := results[len(results)-1]
		if last.Error != "" {
			telemetry.RecordPAORACycle("admit", "error")
			return true, nil
		}
	}

	telemetry.RecordPAORACycle("admit", "continue")
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

// SortStepsByDependencies topologically sorts steps so that steps listed in
// Depends execute before their dependents. Circular dependencies are resolved
// by placing the step in its original position. Stable sort: order is preserved
// for independent steps.
func SortStepsByDependencies(steps []PlannedStep) []PlannedStep {
	if len(steps) <= 1 {
		return steps
	}
	// Build name→index map
	nameIndex := make(map[string]int, len(steps))
	for i, s := range steps {
		nameIndex[s.ToolName] = i
	}
	// Track visited states for cycle detection (0=unvisited, 1=visiting, 2=done)
	state := make([]int, len(steps))
	var sorted []PlannedStep
	var visit func(int)
	visit = func(i int) {
		if state[i] == 2 {
			return
		}
		if state[i] == 1 {
			// Cycle detected — skip to avoid infinite recursion
			return
		}
		state[i] = 1
		for _, dep := range steps[i].Depends {
			if depIdx, ok := nameIndex[dep]; ok {
				visit(depIdx)
			}
			// Unknown deps are tolerated (may reference tools not in this plan)
		}
		state[i] = 2
		sorted = append(sorted, steps[i])
	}
	for i := range steps {
		if state[i] == 0 {
			visit(i)
		}
	}
	return sorted
}

// FailedStepNames returns the set of tool names that failed during execution.
func FailedStepNames(results []*ActResult) map[string]bool {
	failed := make(map[string]bool)
	for _, r := range results {
		if r.Error != "" {
			failed[r.Step.ToolName] = true
		}
	}
	return failed
}

// ShouldSkipStep returns true if any of the step's dependencies failed.
func ShouldSkipStep(step PlannedStep, failedDeps map[string]bool) bool {
	for _, dep := range step.Depends {
		if failedDeps[dep] {
			return true
		}
	}
	return false
}

// ShouldAutoSkip returns true if the step requires confirmation and
// the engine's confirmation threshold is > 0 (meaning auto-skip).
func (e *Engine) ShouldAutoSkip(step PlannedStep) bool {
	return step.RequiresConfirmation && e.confirmationThreshold > 0
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
// Uses a simple heuristic: split msg by spaces, look for words that start with
// an uppercase letter (suggesting object names).
func (e *Engine) extractObjectReferences(msg string) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, word := range strings.Fields(msg) {
		// Strip trailing punctuation for cleaner matching
		word = strings.TrimRight(word, ".,;:!?()[]{}'\"")
		if word == "" {
			continue
		}
		// Words starting with uppercase are candidate object references
		r := []rune(word)
		if r[0] >= 'A' && r[0] <= 'Z' && !seen[word] {
			seen[word] = true
			refs = append(refs, word)
		}
	}
	return refs
}

// extractObjectReferencesWithOntology checks the msg against named objects
// parsed from ontology DSL content. If ontContent is empty/nil, falls back
// to the heuristic extractObjectReferences. Otherwise parses the DSL,
// collects ObjectDefinition names, and returns those found in the message
// (case-insensitive, whole-word matching).
func (e *Engine) extractObjectReferencesWithOntology(msg string, ontContent []byte) []string {
	if len(ontContent) == 0 {
		return e.extractObjectReferences(msg)
	}

	prog, err := dsl.Parse(string(ontContent))
	if err != nil {
		return e.extractObjectReferences(msg)
	}

	var objNames []string
	for _, stmt := range prog.Statements {
		if stmt.Object != nil {
			objNames = append(objNames, stmt.Object.Name)
		}
	}

	if len(objNames) == 0 {
		return nil
	}

	lowerMsg := strings.ToLower(msg)
	seen := make(map[string]bool)
	var matched []string

	for _, name := range objNames {
		if seen[name] {
			continue
		}
		// Case-insensitive whole-word matching using \b word boundaries
		pattern := `\b` + regexp.QuoteMeta(name) + `\b`
		matchedRE, err := regexp.MatchString("(?i)"+pattern, lowerMsg)
		if err == nil && matchedRE {
			seen[name] = true
			matched = append(matched, name)
		}
	}

	return matched
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
