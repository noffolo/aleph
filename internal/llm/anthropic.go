package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AnthropicProvider struct {
	client  *http.Client
	timeout time.Duration
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	callCtx := ctx
	if p.timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	requestBody := map[string]interface{}{
		"model":      req.Model,
		"max_tokens":  4096,
		"system":     req.SystemPrompt,
		"messages":   req.Messages,
		"tools":      req.Tools,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := strings.TrimRight(req.BaseURL, "/")
	endpoint := baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(callCtx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", req.ApiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response body: %w", err)
		}
		return nil, fmt.Errorf("Anthropic API error: %d - %s", resp.StatusCode, string(body))
	}

	var anthropicResp struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var contentText strings.Builder
	var toolCalls []ToolCall

	for _, item := range anthropicResp.Content {
		switch item.Type {
		case "text":
			if item.Text != "" {
				contentText.WriteString(item.Text)
			}
		case "tool_use":
			var args map[string]interface{}
			if len(item.Input) > 0 && string(item.Input) != "null" {
				if err := json.Unmarshal(item.Input, &args); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tool input: %w", err)
				}
			}
			if args == nil {
				args = make(map[string]interface{})
			}
			toolCalls = append(toolCalls, ToolCall{
				Name:      item.Name,
				Arguments: args,
			})
		}
	}

	return &CompletionResponse{
		Content:   contentText.String(),
		ToolCalls: toolCalls,
	}, nil
}