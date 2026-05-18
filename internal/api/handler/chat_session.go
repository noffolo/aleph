package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/errors"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// ChatSession owns the state for a single chat interaction.
// Created per-request by Chat(), manages the LLM loop, tool execution, and streaming.
type ChatSession struct {
	ctx              context.Context
	stream           *connect.ServerStream[v1.ChatResponse]
	handler          *QueryHandler
	projectID        string
	agentID          string
	fullSystemPrompt string
	chatMessages     []map[string]any
	tools            []map[string]any
	agent            AgentInfo
	baseURL          string
	needsPlanning    bool

	// Decision loop state
	engine       decision.DecisionEngine
	plan         *decision.PlanResult
	observations []decision.Observation
	actResults   []*decision.ActResult
	maxAttempts  int
	replanState  string // "" = normal, "partial" = use correction steps, "full" = regenerate plan
}

// AgentInfo holds the resolved agent configuration for a session.
type AgentInfo struct {
	Provider     string
	Model        string
	ApiKey       string
	SystemPrompt string
	BaseURL      string
}

// NewChatSession creates a new ChatSession with resolved agent config and tools.
func NewChatSession(
	ctx context.Context,
	stream *connect.ServerStream[v1.ChatResponse],
	h *QueryHandler,
	projectID string,
	agentID string,
	msg string,
	agent AgentInfo,
	ontContent []byte,
	fullSystemPrompt string,
) *ChatSession {
	chatMessages := []map[string]any{
		{"role": "system", "content": fullSystemPrompt},
	}

	// Load chat history
	history, histErr := h.metaRepo.GetChatMessages(ctx, projectID, agentID)
	if histErr == nil {
		for _, m := range history {
			if m.Role == "user" {
				chatMessages = append(chatMessages, map[string]any{"role": "user", "content": m.Content})
			} else if m.Role == "assistant" && m.ToolCall == "" {
				chatMessages = append(chatMessages, map[string]any{"role": "assistant", "content": m.Content})
			}
		}
	}
	chatMessages = append(chatMessages, map[string]any{"role": "user", "content": msg})

	// Build tool definitions
	var tools []map[string]any
	if h.engine != nil {
		tools = h.engine.BuildToolsMap(ctx)
	} else {
		tools = buildMinimalToolsMap(ctx, h.metaRepo)
	}

	return &ChatSession{
		ctx:              ctx,
		stream:           stream,
		handler:          h,
		projectID:        projectID,
		agentID:          agentID,
		fullSystemPrompt: fullSystemPrompt,
		chatMessages:     chatMessages,
		tools:            tools,
		agent:            agent,
		baseURL:          strings.TrimRight(agent.BaseURL, "/"),
		needsPlanning:    true,
		engine:           h.engine,
		maxAttempts:      5,
	}
}

// Run executes the chat loop: up to maxAttempts iterations of LLM call, tool dispatch, and decision loop.
func (s *ChatSession) Run() error {
	for i := 0; i < s.maxAttempts; i++ {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		// DECISION LOOP: Plan phase (first iteration OR full replan only)
		if s.engine != nil && s.needsPlanning {
			s.needsPlanning = false

			// Partial replan: use correction steps directly, skip PlanWithProvider
			if s.replanState == "partial" && s.plan != nil && len(s.plan.CorrectionSteps) > 0 {
				s.plan.Steps = s.plan.CorrectionSteps
				s.plan.Reason = "replanning using correction steps from reflection"
				s.replanState = ""
				if err := s.executePlanSteps(i); err != nil {
					return err
				}
				summary := s.buildMultiStepSummary()
				if summary != "" {
					if err := s.streamResponse(summary); err != nil {
						return err
					}
				}
				break
			}

			provider, err := llm.NewProvider(s.agent.Provider, s.baseURL, s.handler.httpClient, s.handler.llmTimeout)
			if err != nil {
				slog.Warn("decision engine: NewProvider failed", "error", err)
			} else if provider != nil {
				plan, err := s.engine.PlanWithProvider(s.ctx, s.lastUserMessage(),
					s.projectID, s.agentID, nil, nil, provider)
				if err == nil {
					s.plan = plan
				} else {
					slog.Warn("decision engine: PlanWithProvider failed", "error", err)
				}
				if s.plan != nil && !s.plan.CanProceed && i == 0 {
					slog.Warn("decision engine: plan indicates cannot proceed", "reason", s.plan.Reason)
				}
			}

			// Multi-step execution: if the plan has multiple steps, execute them
			// in dependency order before the first LLM call.
			if s.plan != nil && len(s.plan.Steps) > 0 {
				if err := s.executePlanSteps(i); err != nil {
					return err
				}
				// If the plan executed all steps, skip the LLM round and go to reflect.
				// We still stream a summary to the user.
				summary := s.buildMultiStepSummary()
				if summary != "" {
					if err := s.streamResponse(summary); err != nil {
						return err
					}
				}
				// After executing all planned steps, go to reflect/admit
				break
			}
		}

		responseContent, toolCalls, err := s.callLLM()
		if err != nil {
			return err
		}

		if responseContent != "" {
			if err := s.streamResponse(responseContent); err != nil {
				return err
			}
		}

		if len(toolCalls) == 0 {
			break
		}

		s.appendToolCallToMessages(toolCalls, responseContent, i)

		for _, tc := range toolCalls {
			if err := s.executeAndStreamTool(tc, i); err != nil {
				return err
			}
		}

		// DECISION LOOP: Reflect phase after all tools
		if s.engine != nil && len(s.observations) > 0 {
			reflected, err := s.engine.Reflect(s.ctx, s.plan, s.observations)
			if err == nil && reflected != nil {
				s.plan = reflected
				if reflected.ReplanType != "" && reflected.ReplanType != decision.ReplanNone {
					if reflected.ReplanType == decision.ReplanFull {
						slog.Info("decision engine: full replan requested", "reason", reflected.Reason)
						s.replanState = "full"
						s.needsPlanning = true
						s.plan = nil
						s.observations = nil
						continue
					} else {
						slog.Info("decision engine: partial replan requested", "reason", reflected.Reason)
						s.replanState = "partial"
						s.needsPlanning = true
						continue
					}
				}
				if !reflected.CanProceed {
					slog.Warn("decision engine: reflect says stop", "reason", reflected.Reason)
					break
				}
			}
		}
	}

	// DECISION LOOP: Admit phase at loop end
	if s.engine != nil && len(s.actResults) > 0 {
		// Goal-achieved detection: all plan steps completed successfully
		if s.plan != nil && len(s.plan.Steps) > 0 && len(s.actResults) >= len(s.plan.Steps) {
			allSucceeded := true
			for _, r := range s.actResults {
				if r.Error != "" {
					allSucceeded = false
					break
				}
			}
			if allSucceeded && len(s.actResults) > 0 {
				slog.Debug("decision engine: all plan steps completed, stopping")
				return nil
			}
		}
		s.engine.Admit(s.ctx, s.actResults, s.maxAttempts)
	}

	return nil
}

// executePlanSteps executes all planned steps in dependency order.
// Skips steps whose dependencies failed and steps requiring confirmation
// when the auto-skip threshold is active. Feeds results back into
// the chat message history for LLM context.
func (s *ChatSession) executePlanSteps(iteration int) error {
	steps := decision.SortStepsByDependencies(s.plan.Steps)

	// Track which tools succeeded and failed for dependency resolution
	failedDeps := make(map[string]bool)

	for _, step := range steps {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		// Check if dependencies failed
		if decision.ShouldSkipStep(step, failedDeps) {
			skipMsg := fmt.Sprintf("Skipping tool %s: dependency failed", step.ToolName)
			slog.Warn("decision engine: skipping step due to failed dependency", "tool", step.ToolName, "deps", step.Depends)
			if err := s.stream.Send(&v1.ChatResponse{ToolCall: skipMsg}); err != nil {
				return err
			}
			s.appendToolResult(iteration, "SKIPPED: "+skipMsg)
			s.actResults = append(s.actResults, &decision.ActResult{
				Step:   step,
				Output: "SKIPPED: dependency failed",
			})
			continue
		}

		// Check confirmation threshold for auto-skip
		if s.engine != nil && s.engine.ShouldAutoSkip(step) {
			skipMsg := fmt.Sprintf("Skipping tool %s: requires confirmation (threshold active)", step.ToolName)
			slog.Warn("decision engine: skipping step requiring confirmation", "tool", step.ToolName)
			if err := s.stream.Send(&v1.ChatResponse{ToolCall: skipMsg}); err != nil {
				return err
			}
			s.appendToolResult(iteration, "SKIPPED: "+skipMsg)
			s.actResults = append(s.actResults, &decision.ActResult{
				Step:   step,
				Output: "SKIPPED: requires confirmation",
			})
			continue
		}

		// Execute the step
		reasoning := fmt.Sprintf("Executing planned tool: %s", step.ToolName)
		if err := s.stream.Send(&v1.ChatResponse{ToolCall: reasoning}); err != nil {
			return err
		}
		s.handler.metaRepo.SaveChatMessage(s.ctx, s.projectID, s.agentID, "assistant", "", reasoning)

		actResult, err := s.engine.Act(s.ctx, step, s.projectID)
		var resultStr string
		if err != nil {
			resultStr = "Errore: " + err.Error()
		} else {
			resultStr = actResult.Output
			if actResult.Error != "" {
				resultStr = "Errore: " + actResult.Error
			}
		}

		if actResult != nil && actResult.Step.RequiresConfirmation {
			s.stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
		}

		// Feed result back: append to chat history for LLM context
		s.appendToolResult(iteration, resultStr)
		s.actResults = append(s.actResults, actResult)

		// DECISION LOOP: Observe
		if s.engine != nil && actResult != nil {
			obs, obsErr := s.engine.Observe(s.ctx, step, actResult)
			if obsErr == nil && obs != nil {
				s.observations = append(s.observations, *obs)
			}
		}

		// Track failures for dependency chain skipping
		if actResult.Error != "" {
			failedDeps[step.ToolName] = true
		}
	}
	return nil
}

// buildMultiStepSummary creates a human-readable summary of multi-step execution results.
func (s *ChatSession) buildMultiStepSummary() string {
	if len(s.actResults) == 0 {
		return ""
	}
	var total, succeeded, skipped, failed int
	for _, r := range s.actResults {
		total++
		if r.Output == "SKIPPED: dependency failed" || r.Output == "SKIPPED: requires confirmation" {
			skipped++
		} else if r.Error != "" {
			failed++
		} else {
			succeeded++
		}
	}
	summary := fmt.Sprintf("Multi-step execution complete: %d steps (%d succeeded, %d failed, %d skipped)", total, succeeded, failed, skipped)
	slog.Info("decision engine: multi-step execution", "total", total, "succeeded", succeeded, "failed", failed, "skipped", skipped)
	return summary
}

// callLLM sends the current messages to the LLM and returns the response.
func (s *ChatSession) callLLM() (string, []llm.ToolCall, error) {
	provider, err := llm.NewProvider(s.agent.Provider, s.baseURL, s.handler.httpClient, s.handler.llmTimeout)
	if err != nil {
		return "", nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("unsupported provider: %s", s.agent.Provider))
	}

	var systemPrompt string
	if s.agent.Provider == "anthropic" {
		systemPrompt = s.fullSystemPrompt
	}

	req := llm.CompletionRequest{
		Model:        s.agent.Model,
		Messages:     s.chatMessages,
		Tools:        s.tools,
		SystemPrompt: systemPrompt,
		ApiKey:       s.agent.ApiKey,
		BaseURL:      s.baseURL,
	}

	completion, err := provider.Complete(s.ctx, req)
	if err != nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errors.NewAPIErrorWithMeta(
			errors.ErrUnavailable, "LLM completion failed", err,
			"llm", "complete", true, 10000,
		))
	}

	return completion.Content, completion.ToolCalls, nil
}

// streamResponse sends a text token to the client and saves to history.
func (s *ChatSession) streamResponse(content string) error {
	if err := s.stream.Send(&v1.ChatResponse{Token: content}); err != nil {
		return err
	}
	return s.handler.metaRepo.SaveChatMessage(s.ctx, s.projectID, s.agentID, "assistant", content, "")
}

// executeAndStreamTool executes a single tool call and streams the result.
// Uses engine.Act() for telemetry/timing when engine is available.
func (s *ChatSession) executeAndStreamTool(tc llm.ToolCall, iteration int) error {
	reasoning := fmt.Sprintf("Executing tool: %s", tc.Name)
	if err := s.stream.Send(&v1.ChatResponse{ToolCall: reasoning}); err != nil {
		return err
	}
	s.handler.metaRepo.SaveChatMessage(s.ctx, s.projectID, s.agentID, "assistant", "", reasoning)

	var resultStr string
	var actResult *decision.ActResult

	if s.engine != nil {
		// Use engine.Act() for execution with telemetry and timing
		step := decision.PlannedStep{
			ToolName:  tc.Name,
			Arguments: tc.Arguments,
		}
		var engineErr error
		actResult, engineErr = s.engine.Act(s.ctx, step, s.projectID)
		if engineErr != nil {
			resultStr = "Errore: " + engineErr.Error()
		} else {
			resultStr = actResult.Output
			if actResult.Error != "" {
				resultStr = "Errore: " + actResult.Error
			}
		}
		if actResult != nil && actResult.Step.RequiresConfirmation {
			s.stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
		}
	} else if s.handler.executor != nil {
		// Fallback: direct executor call (no engine)
		result, requiresConfirmation, execErr := s.handler.executor.ExecuteTool(
			s.ctx, tc.Name, tc.Arguments, s.projectID, s.agentID)
		if execErr != nil {
			resultStr = "Errore: " + execErr.Error()
		} else {
			resultStr = result
		}
		if requiresConfirmation {
			s.stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
		}
		actResult = &decision.ActResult{
			Step: decision.PlannedStep{
				ToolName:             tc.Name,
				Arguments:            tc.Arguments,
				RequiresConfirmation: requiresConfirmation,
			},
			Output: resultStr,
		}
		if execErr != nil {
			actResult.Error = resultStr
		}
	} else {
		s.stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
		resultStr = fmt.Sprintf("Proposta azione '%s' in attesa di conferma.", tc.Name)
		actResult = &decision.ActResult{
			Step: decision.PlannedStep{
				ToolName:             tc.Name,
				Arguments:            tc.Arguments,
				RequiresConfirmation: true,
			},
			Output: resultStr,
		}
	}

	s.appendToolResult(iteration, resultStr)

	// DECISION LOOP: Observe
	if s.engine != nil && actResult != nil {
		s.actResults = append(s.actResults, actResult)
		obs, err := s.engine.Observe(s.ctx, actResult.Step, actResult)
		if err == nil && obs != nil {
			s.observations = append(s.observations, *obs)
		}
	}

	return nil
}

// appendToolCallToMessages adds the assistant's tool call message to the chat history.
func (s *ChatSession) appendToolCallToMessages(toolCalls []llm.ToolCall, responseContent string, iteration int) {
	assistantMsg := map[string]any{"role": "assistant", "content": responseContent}

	var apiToolCalls []map[string]any
	for j, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		apiToolCalls = append(apiToolCalls, map[string]any{
			"id":   fmt.Sprintf("call_%d_%d", iteration, j),
			"type": "function",
			"function": map[string]any{
				"name":      tc.Name,
				"arguments": string(argsJSON),
			},
		})
	}
	assistantMsg["tool_calls"] = apiToolCalls
	s.chatMessages = append(s.chatMessages, assistantMsg)
}

// appendToolResult adds the tool execution result to the message history.
func (s *ChatSession) appendToolResult(iteration int, resultStr string) {
	s.chatMessages = append(s.chatMessages, map[string]any{
		"role":         "tool",
		"content":      resultStr,
		"tool_call_id": fmt.Sprintf("call_%d_tools_0", iteration),
	})
}

// lastUserMessage returns the most recent user message from chat history.
func (s *ChatSession) lastUserMessage() string {
	for i := len(s.chatMessages) - 1; i >= 0; i-- {
		if role, ok := s.chatMessages[i]["role"].(string); ok && role == "user" {
			if content, ok := s.chatMessages[i]["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

// buildMinimalToolsMap creates a minimal tool definition map from registered tools only.
// Used in degraded mode when no engine is available.
func buildMinimalToolsMap(ctx context.Context, metaRepo *repository.MetadataRepository) []map[string]any {
	if metaRepo == nil {
		return nil
	}
	tools, err := metaRepo.ListTools()
	if err != nil {
		return nil
	}
	result := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		toolDef := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
			},
		}
		if t.Code != "" {
			var params map[string]any
			if json.Unmarshal([]byte(t.Code), &params) == nil {
				toolDef["function"].(map[string]any)["parameters"] = params
			}
		}
		result = append(result, toolDef)
	}
	return result
}
