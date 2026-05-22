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
	Messages     []map[string]any
	Tools        []map[string]any
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
	Arguments map[string]any
}

// OllamaPort is the default Ollama port used for provider auto-detection.
// Override via config (OTTEL_OLLAMA_PORT env var) during application startup.
var OllamaPort = "11434"

// NewProvider creates a new LLM provider. Returns an error for unknown providers.
// The timeout parameter sets per-request timeout (0 = no timeout).
func NewProvider(provider string, baseURL string, httpClient *http.Client, timeout time.Duration) (Provider, error) {
	if provider == "" && strings.Contains(baseURL, OllamaPort) {
		return &OllamaProvider{client: httpClient, timeout: timeout}, nil
	}

	switch provider {
	case "ollama":
		return &OllamaProvider{client: httpClient, timeout: timeout}, nil
	case "ollama-cloud":
		return &OllamaCloudProvider{client: httpClient, timeout: timeout}, nil
	case "anthropic":
		return &AnthropicProvider{client: httpClient, timeout: timeout}, nil
	case "openai":
		return &OpenAIProvider{client: httpClient, timeout: timeout}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %q (supported: ollama, ollama-cloud, openai, anthropic)", provider)
	}
}
