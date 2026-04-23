package llm

import (
	"context"
	"net/http"
	"strings"
)

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type CompletionRequest struct {
	Model        string
	Messages     []map[string]interface{}
	Tools        []map[string]interface{}
	SystemPrompt string
	ApiKey       string
	BaseURL      string
}

type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	Name      string
	Arguments map[string]interface{}
}

func NewProvider(provider string, baseURL string, httpClient *http.Client) Provider {
	if provider == "" && strings.Contains(baseURL, "11434") {
		return &OllamaProvider{client: httpClient}
	}
	
	switch provider {
	case "ollama":
		return &OllamaProvider{client: httpClient}
	case "anthropic":
		return &AnthropicProvider{client: httpClient}
	case "openai":
		return &OpenAIProvider{client: httpClient}
	default:
		return nil
	}
}