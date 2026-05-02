package llm

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewProvider_KnownProviders(t *testing.T) {
	client := &http.Client{}
	timeout := 30 * time.Second

	tests := []struct {
		name     string
		provider string
		baseURL  string
	}{
		{"ollama", "ollama", "http://localhost:11434"},
		{"openai", "openai", "https://api.openai.com"},
		{"anthropic", "anthropic", "https://api.anthropic.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.provider, tt.baseURL, client, timeout)
			assert.NoError(t, err, "provider should not error for %s", tt.provider)
			assert.NotNil(t, p, "provider should not be nil for %s", tt.provider)
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	client := &http.Client{}
	p, err := NewProvider("unknown", "http://localhost", client, 30*time.Second)
	assert.Error(t, err, "unknown provider should return error")
	assert.Nil(t, p, "unknown provider should return nil")
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

func TestNewProvider_CaseInsensitivity(t *testing.T) {
	client := &http.Client{}
	timeout := 30 * time.Second

	p1, err := NewProvider("ollama", "http://localhost:11434", client, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, p1)

	p2, err := NewProvider("openai", "https://api.openai.com", client, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	p3, err := NewProvider("anthropic", "https://api.anthropic.com", client, timeout)
	assert.NoError(t, err)
	assert.NotNil(t, p3)
}

func TestNewRetryProvider_NilInner(t *testing.T) {
	p, err := NewRetryProvider(nil, 3, 0)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "cannot wrap nil provider")
}

func TestCompletionRequest_Struct(t *testing.T) {
	req := CompletionRequest{
		Model:        "test-model",
		Messages:     []map[string]interface{}{{"role": "user", "content": "hello"}},
		Tools:        []map[string]interface{}{{"name": "test"}},
		SystemPrompt: "system prompt",
		ApiKey:       "api-key",
		BaseURL:      "http://localhost",
	}

	assert.Equal(t, "test-model", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "system prompt", req.SystemPrompt)
	assert.Equal(t, "api-key", req.ApiKey)
	assert.Equal(t, "http://localhost", req.BaseURL)
}

func TestCompletionResponse_Struct(t *testing.T) {
	resp := CompletionResponse{
		Content: "response content",
		ToolCalls: []ToolCall{
			{Name: "tool1", Arguments: map[string]interface{}{"arg": "val"}},
		},
	}

	assert.Equal(t, "response content", resp.Content)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "tool1", resp.ToolCalls[0].Name)
}

func TestToolCall_Struct(t *testing.T) {
	tc := ToolCall{
		Name:      "test-tool",
		Arguments: map[string]interface{}{"key": "value"},
	}
	assert.Equal(t, "test-tool", tc.Name)
	assert.NotNil(t, tc.Arguments)
}

func TestNewProvider_TimeoutZero(t *testing.T) {
	client := &http.Client{}
	p, err := NewProvider("ollama", "http://localhost:11434", client, 0)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_EmptyProviderWithOllamaPort(t *testing.T) {
	client := &http.Client{}
	p, err := NewProvider("", "http://localhost:11434", client, 30*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	_, ok := p.(*OllamaProvider)
	assert.True(t, ok, "should auto-detect Ollama")
}
