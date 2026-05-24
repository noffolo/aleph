package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Compile-time assertion: OllamaCloudProvider implements Provider.
var _ Provider = (*OllamaCloudProvider)(nil)

// OllamaCloudProvider — OpenAI-compatible client for Ollama Cloud API (https://ollama.com).
type OllamaCloudProvider struct {
	client  *http.Client
	timeout time.Duration
}

func (p *OllamaCloudProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	callCtx := ctx
	if p.timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	if deadline, ok := ctx.Deadline(); ok {
		slog.Debug("ollamacloud: parent context deadline", "remaining", time.Until(deadline).Round(time.Second).String())
		slog.Debug("ollamacloud: client timeout", "timeout", p.timeout.String())
	} else {
		slog.Debug("ollamacloud: parent context has no deadline")
	}

	messages := req.Messages
	if req.SystemPrompt != "" {
		messages = append([]map[string]any{{
			"role":    "system",
			"content": req.SystemPrompt,
		}}, messages...)
	}

	requestBody := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
	}

	// Map Ollama tool format to OpenAI tool format if present
	if len(req.Tools) > 0 {
		requestBody["tools"] = req.Tools
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := strings.TrimRight(req.BaseURL, "/")
	// Use OpenAI-compatible endpoint
	endpoint := baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(callCtx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.ApiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.ApiKey)
	}

	slog.Debug("ollamacloud: sending request", "endpoint", endpoint, "model", req.Model, "msgs", len(messages), "tools", len(req.Tools), "body_len", len(bodyJSON))
	startTime := time.Now()
	resp, err := p.client.Do(httpReq)
	elapsed := time.Since(startTime)
	if err != nil {
		slog.Error("ollamacloud: request failed", "elapsed", elapsed.Round(time.Millisecond).String(), "error", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	slog.Debug("ollamacloud: response received", "status", resp.StatusCode, "elapsed", elapsed.Round(time.Millisecond).String())

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OllamaCloud API error: %d - %s", resp.StatusCode, string(body))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OllamaCloud API returned no choices: %s", string(respBody))
	}

	msg := openAIResp.Choices[0].Message
	var toolCalls []ToolCall
	for _, tc := range msg.ToolCalls {
		// OpenAI/DeepSeek returns arguments as a JSON-encoded string.
		// Step 1: decode the RawMessage into a string.
		var argsStr string
		if err := json.Unmarshal(tc.Function.Arguments, &argsStr); err == nil {
			// Step 2: parse the string as JSON object.
			var args map[string]any
			if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool call arguments string: %w (raw: %s)", err, argsStr)
			}
			toolCalls = append(toolCalls, ToolCall{
				Name:      tc.Function.Name,
				Arguments: args,
			})
		} else {
			// Fallback: arguments might be a JSON object directly.
			var args map[string]any
			if err := json.Unmarshal(tc.Function.Arguments, &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool call arguments: %w (raw: %s)", err, string(tc.Function.Arguments))
			}
			toolCalls = append(toolCalls, ToolCall{
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
	}

	return &CompletionResponse{
		Content:   msg.Content,
		ToolCalls: toolCalls,
	}, nil
}
