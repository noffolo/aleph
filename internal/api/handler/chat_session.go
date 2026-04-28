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
	chatMessages     []map[string]interface{}
	tools            []map[string]interface{}
	agent            AgentInfo
	baseURL          string
	needsPlanning    bool

	// Decision loop state
	engine      decision.DecisionEngine
	plan        *decision.PlanResult
	observations []decision.Observation
	actResults   []*decision.ActResult
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
	chatMessages := []map[string]interface{}{
		{"role": "system", "content": fullSystemPrompt},
	}

	// Load chat history
	history, histErr := h.metaRepo.GetChatMessages(ctx, projectID, agentID)
	if histErr == nil {
		for _, m := range history {
			if m.Role == "user" {
				chatMessages = append(chatMessages, map[string]interface{}{"role": "user", "content": m.Content})
			} else if m.Role == "assistant" && m.ToolCall == "" {
				chatMessages = append(chatMessages, map[string]interface{}{"role": "assistant", "content": m.Content})
			}
		}
	}
	chatMessages = append(chatMessages, map[string]interface{}{"role": "user", "content": msg})

	// Build tool definitions
	var tools []map[string]interface{}
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
	}
}

// Run executes the chat loop: up to 5 iterations of LLM call, tool dispatch, and decision loop.
func (s *ChatSession) Run() error {
	for i := 0; i < 5; i++ {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		// DECISION LOOP: Plan phase (first iteration only)
		if s.engine != nil && s.needsPlanning {
			provider := llm.NewProvider(s.agent.Provider, s.baseURL, s.handler.httpClient)
			if provider != nil {
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
			s.needsPlanning = false
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
				if !reflected.CanProceed {
					slog.Warn("decision engine: reflect says stop", "reason", reflected.Reason)
					break
				}
			}
		}
	}

	// DECISION LOOP: Admit phase at loop end
	if s.engine != nil && len(s.actResults) > 0 {
		s.engine.Admit(s.ctx, s.actResults, 5)
	}

	return nil
}

// callLLM sends the current messages to the LLM and returns the response.
func (s *ChatSession) callLLM() (string, []llm.ToolCall, error) {
	provider := llm.NewProvider(s.agent.Provider, s.baseURL, s.handler.httpClient)
	if provider == nil {
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
		return "", nil, connect.NewError(connect.CodeUnavailable, err)
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
func (s *ChatSession) executeAndStreamTool(tc llm.ToolCall, iteration int) error {
	reasoning := fmt.Sprintf("Executing tool: %s", tc.Name)
	if err := s.stream.Send(&v1.ChatResponse{ToolCall: reasoning}); err != nil {
		return err
	}
	s.handler.metaRepo.SaveChatMessage(s.ctx, s.projectID, s.agentID, "assistant", "", reasoning)

	var resultStr string
	if s.handler.executor != nil {
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
	} else {
		s.stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
		resultStr = fmt.Sprintf("Proposta azione '%s' in attesa di conferma.", tc.Name)
	}

	s.appendToolResult(iteration, resultStr)

	// DECISION LOOP: Observe
	if s.engine != nil {
		step := decision.PlannedStep{
			ToolName:  tc.Name,
			Arguments: tc.Arguments,
		}
		actResult := &decision.ActResult{
			Step:   step,
			Output: resultStr,
		}
		if strings.HasPrefix(resultStr, "Errore:") {
			actResult.Error = resultStr
		}
		s.actResults = append(s.actResults, actResult)

		obs, err := s.engine.Observe(s.ctx, step, actResult)
		if err == nil && obs != nil {
			s.observations = append(s.observations, *obs)
		}
	}

	return nil
}

// appendToolCallToMessages adds the assistant's tool call message to the chat history.
func (s *ChatSession) appendToolCallToMessages(toolCalls []llm.ToolCall, responseContent string, iteration int) {
	assistantMsg := map[string]interface{}{"role": "assistant", "content": responseContent}

	var apiToolCalls []map[string]interface{}
	for j, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		apiToolCalls = append(apiToolCalls, map[string]interface{}{
			"id":   fmt.Sprintf("call_%d_%d", iteration, j),
			"type": "function",
			"function": map[string]interface{}{
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
	s.chatMessages = append(s.chatMessages, map[string]interface{}{
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
func buildMinimalToolsMap(ctx context.Context, metaRepo *repository.MetadataRepository) []map[string]interface{} {
	if metaRepo == nil {
		return nil
	}
	tools, err := metaRepo.ListTools()
	if err != nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		toolDef := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
			},
		}
		if t.Code != "" {
			var params map[string]interface{}
			if json.Unmarshal([]byte(t.Code), &params) == nil {
				toolDef["function"].(map[string]interface{})["parameters"] = params
			}
		}
		result = append(result, toolDef)
	}
	return result
}
