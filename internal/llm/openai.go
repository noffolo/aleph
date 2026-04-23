package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenAIProvider struct {
	client *http.Client
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	endpoint := req.BaseURL + "/v1/chat/completions"
	if strings.HasSuffix(req.BaseURL, "/completions") || strings.HasSuffix(req.BaseURL, "/v4/chat/completions") {
		endpoint = req.BaseURL
	}

	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
		"tools":    req.Tools,
	}

	if req.SystemPrompt != "" {
		systemMsg := map[string]interface{}{
			"role":    "system",
			"content": req.SystemPrompt,
		}
		messages := make([]map[string]interface{}, 0, len(req.Messages)+1)
		messages = append(messages, systemMsg)
		messages = append(messages, req.Messages...)
		requestBody["messages"] = messages
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.ApiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.ApiKey)
	}

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
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var toolCalls []ToolCall
	for _, tc := range openaiResp.Choices[0].Message.ToolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
		}
		toolCalls = append(toolCalls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return &CompletionResponse{
		Content:   openaiResp.Choices[0].Message.Content,
		ToolCalls: toolCalls,
	}, nil
}