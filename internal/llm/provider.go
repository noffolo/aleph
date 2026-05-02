package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
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

// NewProvider creates a new LLM provider. Returns an error for unknown providers.
// The timeout parameter sets per-request timeout (0 = no timeout).
func NewProvider(provider string, baseURL string, httpClient *http.Client, timeout time.Duration) (Provider, error) {
	if provider == "" && strings.Contains(baseURL, "11434") {
		return &OllamaProvider{client: httpClient, timeout: timeout}, nil
	}

	switch provider {
	case "ollama":
		return &OllamaProvider{client: httpClient, timeout: timeout}, nil
	case "anthropic":
		return &AnthropicProvider{client: httpClient, timeout: timeout}, nil
	case "openai":
		return &OpenAIProvider{client: httpClient, timeout: timeout}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %q (supported: ollama, openai, anthropic)", provider)
	}
}