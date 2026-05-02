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

type OllamaProvider struct {
	client  *http.Client
	timeout time.Duration
}

func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	callCtx := ctx
	if p.timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	messages := req.Messages
	tools := req.Tools

	if req.SystemPrompt != "" {
		messages = append([]map[string]interface{}{{
			"role":    "system",
			"content": req.SystemPrompt,
		}}, messages...)
	}

	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
		"tools":    tools,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := strings.TrimRight(req.BaseURL, "/")
	endpoint := baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(callCtx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("Ollama API error: %d - %s", resp.StatusCode, string(body))
	}

	var ollamaResp struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var toolCalls []ToolCall
	for _, tc := range ollamaResp.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return &CompletionResponse{
		Content:   ollamaResp.Message.Content,
		ToolCalls: toolCalls,
	}, nil
}