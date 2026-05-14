package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewProvider constructor tests ---

func TestNewProvider_AllProviders(t *testing.T) {
	client := &http.Client{}
	timeout := 30 * time.Second

	tests := []struct {
		name     string
		provider string
		baseURL  string
		want     interface{}
	}{
		{"ollama explicit", "ollama", "http://localhost:11434", &OllamaProvider{}},
		{"openai explicit", "openai", "https://api.openai.com", &OpenAIProvider{}},
		{"anthropic explicit", "anthropic", "https://api.anthropic.com", &AnthropicProvider{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.provider, tt.baseURL, client, timeout)
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.IsType(t, tt.want, p)
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	p, err := NewProvider("groq", "https://api.groq.com", &http.Client{}, 0)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
	assert.Contains(t, err.Error(), "groq")
}

func TestNewProvider_EmptyProviderAutoDetect(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		ollamaPort string
		want       bool // true = should auto-detect Ollama
	}{
		{"default port match", "http://localhost:11434", "11434", true},
		{"custom port match", "http://localhost:8080", "8080", true},
		{"no match", "https://api.openai.com", "11434", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := OllamaPort
			OllamaPort = tt.ollamaPort
			defer func() { OllamaPort = original }()

			p, err := NewProvider("", tt.baseURL, &http.Client{}, 0)
			if tt.want {
				require.NoError(t, err)
				_, ok := p.(*OllamaProvider)
				assert.True(t, ok, "should auto-detect Ollama for baseURL containing port %s", tt.ollamaPort)
			} else {
				// empty provider without Ollama port → unsupported
				require.Error(t, err)
				assert.Nil(t, p)
			}
		})
	}
}

func TestNewProvider_ZeroTimeout(t *testing.T) {
	p, err := NewProvider("ollama", "http://localhost:11434", &http.Client{}, 0)
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestNewProvider_WithCustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	p, err := NewProvider("openai", "https://api.openai.com", customClient, 10*time.Second)
	require.NoError(t, err)
	require.NotNil(t, p)
	// Provider is created; functional test with request is in openai_test.go
}

// --- Struct field tests ---

func TestCompletionRequest_Fields(t *testing.T) {
	req := CompletionRequest{
		Model:        "gpt-4",
		Messages:     []map[string]interface{}{{"role": "user", "content": "test"}},
		Tools:        []map[string]interface{}{{"name": "calculator"}},
		SystemPrompt: "you are helpful",
		ApiKey:       "sk-test",
		BaseURL:      "https://api.openai.com",
	}
	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "you are helpful", req.SystemPrompt)
	assert.Equal(t, "sk-test", req.ApiKey)
	assert.Equal(t, "https://api.openai.com", req.BaseURL)
}

func TestCompletionResponse_Fields(t *testing.T) {
	resp := CompletionResponse{
		Content: "hello world",
		ToolCalls: []ToolCall{
			{Name: "search", Arguments: map[string]interface{}{"query": "test"}},
		},
	}
	assert.Equal(t, "hello world", resp.Content)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "search", resp.ToolCalls[0].Name)
}

func TestToolCall_Fields(t *testing.T) {
	tc := ToolCall{
		Name:      "execute",
		Arguments: map[string]interface{}{"cmd": "ls"},
	}
	assert.Equal(t, "execute", tc.Name)
	assert.NotNil(t, tc.Arguments)
}

// --- Helper to create mock servers ---

func newMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// --- Ollama mock response helper ---

func ollamaSuccessResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": content,
		},
	}
}

func ollamaSuccessWithToolCalls(content string, toolCalls []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"role":      "assistant",
			"content":   content,
			"tool_calls": toolCalls,
		},
	}
}

// --- OpenAI mock response helper ---

func openaiSuccessResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	}
}

func openaiSuccessWithToolCalls(content string, toolCalls []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":      "assistant",
					"content":   content,
					"tool_calls": toolCalls,
				},
			},
		},
	}
}

// --- Anthropic mock response helper ---

func anthropicSuccessResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": content},
		},
		"stop_reason": "end_turn",
	}
}

func anthropicSuccessWithToolCalls(textContent string, toolCalls []map[string]interface{}) map[string]interface{} {
	content := []map[string]interface{}{}
	if textContent != "" {
		content = append(content, map[string]interface{}{"type": "text", "text": textContent})
	}
	for _, tc := range toolCalls {
		content = append(content, tc)
	}
	return map[string]interface{}{
		"content":     content,
		"stop_reason": "end_turn",
	}
}

// --- OllamaProvider Complete with httptest ---

func TestOllamaProvider_Complete_Success(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body structure
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "llama3", body["model"])
		assert.Equal(t, false, body["stream"])

		json.NewEncoder(w).Encode(ollamaSuccessResponse("Hello from Ollama!"))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from Ollama!", resp.Content)
}

func TestOllamaProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedMessages []map[string]interface{}

	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		receivedMessages = toMessages(body["messages"])

		json.NewEncoder(w).Encode(ollamaSuccessResponse("system response"))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:        "llama3",
		BaseURL:      server.URL,
		SystemPrompt: "You are a helpful assistant",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
	})

	require.NoError(t, err)
	require.Len(t, receivedMessages, 2)
	assert.Equal(t, "system", receivedMessages[0]["role"])
	assert.Equal(t, "You are a helpful assistant", receivedMessages[0]["content"])
	assert.Equal(t, "user", receivedMessages[1]["role"])
}

func TestOllamaProvider_Complete_WithToolCalls(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ollamaSuccessWithToolCalls("calculating", []map[string]interface{}{
			{
				"function": map[string]interface{}{
					"name":      "calculator",
					"arguments": map[string]interface{}{"expr": "2+2"},
				},
			},
		}))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "what is 2+2?"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "calculating", resp.Content)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "calculator", resp.ToolCalls[0].Name)
	assert.Equal(t, "2+2", resp.ToolCalls[0].Arguments["expr"])
}

func TestOllamaProvider_Complete_HTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorMsg   string
	}{
		{"400 bad request", 400, "Ollama API error: 400"},
		{"401 unauthorized", 401, "Ollama API error: 401"},
		{"429 rate limited", 429, "Ollama API error: 429"},
		{"500 internal", 500, "Ollama API error: 500"},
		{"503 unavailable", 503, "Ollama API error: 503"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error body"))
			})
			defer server.Close()

			provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
			_, err := provider.Complete(context.Background(), CompletionRequest{
				Model:   "llama3",
				BaseURL: server.URL,
				Messages: []map[string]interface{}{
					{"role": "user", "content": "test"},
				},
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestOllamaProvider_Complete_ConnectionError(t *testing.T) {
	// Create a provider pointing to a non-existent server
	provider := &OllamaProvider{client: &http.Client{Timeout: 1 * time.Second}, timeout: 2 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "llama3",
		BaseURL: "http://localhost:1",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestOllamaProvider_Complete_Timeout(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode(ollamaSuccessResponse("late"))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 50 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Complete(ctx, CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
}

func TestOllamaProvider_Complete_ContextCancelled(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ollamaSuccessResponse("ok"))
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(ctx, CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
}

func TestOllamaProvider_Complete_InvalidResponseJSON(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL,
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestOllamaProvider_Complete_BaseURLTrimming(t *testing.T) {
	var receivedPath string
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewEncoder(w).Encode(ollamaSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &OllamaProvider{client: server.Client(), timeout: 5 * time.Second}
	// Ollama trims trailing slash from baseURL
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "llama3",
		BaseURL: server.URL + "/",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "/api/chat", receivedPath)
}

// --- OpenAIProvider Complete with httptest ---

func TestOpenAIProvider_Complete_Success(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

		json.NewEncoder(w).Encode(openaiSuccessResponse("Hello from GPT!"))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test-key",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from GPT!", resp.Content)
}

func TestOpenAIProvider_Complete_NoAPIKey(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		// When no API key, Authorization header should not be set
		assert.Empty(t, r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(openaiSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4",
		BaseURL:  server.URL,
		ApiKey:   "",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.NoError(t, err)
}

func TestOpenAIProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedMessages []map[string]interface{}

	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		receivedMessages = toMessages(body["messages"])
		json.NewEncoder(w).Encode(openaiSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:        "gpt-4",
		BaseURL:      server.URL,
		SystemPrompt: "You are helpful",
		Messages:     []map[string]interface{}{{"role": "user", "content": "hi"}},
		ApiKey:       "sk-test",
	})

	require.NoError(t, err)
	require.Len(t, receivedMessages, 2)
	assert.Equal(t, "system", receivedMessages[0]["role"])
	assert.Equal(t, "You are helpful", receivedMessages[0]["content"])
}

func TestOpenAIProvider_Complete_WithToolCalls(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiSuccessWithToolCalls("calling tool", []map[string]interface{}{
			{
				"id":   "call_123",
				"type": "function",
				"function": map[string]interface{}{
					"name":      "get_weather",
					"arguments": `{"city": "Berlin"}`,
				},
			},
		}))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "weather in Berlin?"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "calling tool", resp.Content)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Name)
	assert.Equal(t, "Berlin", resp.ToolCalls[0].Arguments["city"])
}

func TestOpenAIProvider_Complete_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorMsg   string
	}{
		{"400 bad request", 400, "OpenAI API error: 400"},
		{"401 unauthorized", 401, "OpenAI API error: 401"},
		{"429 rate limited", 429, "OpenAI API error: 429"},
		{"500 internal", 500, "OpenAI API error: 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error details"))
			})
			defer server.Close()

			provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
			_, err := provider.Complete(context.Background(), CompletionRequest{
				Model:   "gpt-4",
				BaseURL: server.URL,
				ApiKey:  "sk-test",
				Messages: []map[string]interface{}{
					{"role": "user", "content": "test"},
				},
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
		})
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestOpenAIProvider_Complete_InvalidToolCallArguments(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "using tool",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_bad",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "bad_tool",
									"arguments": `{invalid json`,
								},
							},
						},
					},
				},
			},
		})
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal tool arguments")
}

func TestOpenAIProvider_Complete_CustomEndpoint(t *testing.T) {
	var receivedPath string
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewEncoder(w).Encode(openaiSuccessResponse("ok"))
	})
	defer server.Close()

	// When BaseURL already ends with /completions, it should use it directly
	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-4",
		BaseURL:  server.URL + "/completions",
		ApiKey:   "sk-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.NoError(t, err)
	// Should use the base URL directly when it ends with /completions
	assert.Equal(t, "/completions", receivedPath)
}

// --- AnthropicProvider Complete with httptest ---

func TestAnthropicProvider_Complete_Success(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "sk-ant-test", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		json.NewEncoder(w).Encode(anthropicSuccessResponse("Hello from Claude!"))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "claude-3-sonnet",
		BaseURL: server.URL,
		ApiKey:  "sk-ant-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from Claude!", resp.Content)
}

func TestAnthropicProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(anthropicSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:        "claude-3-sonnet",
		BaseURL:      server.URL,
		ApiKey:       "sk-ant-test",
		SystemPrompt: "You are a coding assistant",
		Messages:     []map[string]interface{}{{"role": "user", "content": "hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "You are a coding assistant", receivedBody["system"])
	assert.Equal(t, 4096, int(receivedBody["max_tokens"].(float64)))
}

func TestAnthropicProvider_Complete_WithToolCalls(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(anthropicSuccessWithToolCalls("result text", []map[string]interface{}{
			{
				"type": "tool_use",
				"id":   "toolu_123",
				"name": "search_web",
				"input": map[string]interface{}{
					"query": "golang testing",
				},
			},
		}))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "claude-3-sonnet",
		BaseURL: server.URL,
		ApiKey:  "sk-ant-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "search for golang testing"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "result text", resp.Content)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "search_web", resp.ToolCalls[0].Name)
	assert.Equal(t, "golang testing", resp.ToolCalls[0].Arguments["query"])
}

func TestAnthropicProvider_Complete_ToolUseWithEmptyInput(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "tool_use",
					"id":   "toolu_456",
					"name": "ping",
					"input": nil,
				},
			},
			"stop_reason": "end_turn",
		})
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL,
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "ping"}},
	})

	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "ping", resp.ToolCalls[0].Name)
	// When Anthropic returns "input": null, json.Unmarshal produces nil map.
	// Accept either nil or empty map until provider is updated to coerce null → {}.
	assert.Empty(t, resp.ToolCalls[0].Arguments)
}

func TestAnthropicProvider_Complete_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorMsg   string
	}{
		{"400 bad request", 400, "Anthropic API error: 400"},
		{"401 unauthorized", 401, "Anthropic API error: 401"},
		{"429 rate limited", 429, "Anthropic API error: 429"},
		{"500 internal", 500, "Anthropic API error: 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error details"))
			})
			defer server.Close()

			provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
			_, err := provider.Complete(context.Background(), CompletionRequest{
				Model:    "claude-3-sonnet",
				BaseURL:  server.URL,
				ApiKey:   "sk-ant-test",
				Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestAnthropicProvider_Complete_InvalidJSON(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json at all`))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL,
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestAnthropicProvider_Complete_MultipleContentBlocks(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "First part. "},
				{"type": "text", "text": "Second part."},
			},
			"stop_reason": "end_turn",
		})
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL,
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "First part. Second part.", resp.Content)
}

func TestAnthropicProvider_Complete_BaseURLTrimming(t *testing.T) {
	var receivedPath string
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewEncoder(w).Encode(anthropicSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL + "/",
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/messages", receivedPath)
}

func TestAnthropicProvider_Complete_ConnectionError(t *testing.T) {
	provider := &AnthropicProvider{client: &http.Client{Timeout: 1 * time.Second}, timeout: 2 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  "http://localhost:1",
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

// --- RetryProvider functional tests ---

type mockProviderFunc func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

func (f mockProviderFunc) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return f(ctx, req)
}

func TestRetryProvider_SuccessFirstTry(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		return &CompletionResponse{Content: "ok"}, nil
	})

	rp, err := NewRetryProvider(inner, 2, 0)
	require.NoError(t, err)

	resp, err := rp.Complete(context.Background(), CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Content)
	assert.Equal(t, 1, callCount)
}

func TestRetryProvider_SuccessAfterRetry(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		if callCount == 1 {
			return nil, assert.AnError
		}
		return &CompletionResponse{Content: "recovered"}, nil
	})

	rp, err := NewRetryProvider(inner, 2, 0)
	require.NoError(t, err)

	resp, err := rp.Complete(context.Background(), CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "recovered", resp.Content)
	assert.Equal(t, 2, callCount)
}

func TestRetryProvider_MaxRetriesExceeded(t *testing.T) {
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		return nil, assert.AnError
	})

	rp, err := NewRetryProvider(inner, 2, 0)
	require.NoError(t, err)

	_, err = rp.Complete(context.Background(), CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
}

func TestRetryProvider_ZeroMaxRetries(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		return &CompletionResponse{Content: "ok"}, nil
	})

	rp, err := NewRetryProvider(inner, 0, 0)
	require.NoError(t, err)

	resp, err := rp.Complete(context.Background(), CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Content)
	assert.Equal(t, 1, callCount)
}

func TestRetryProvider_ContextCancelledDuringBackoff(t *testing.T) {
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		return nil, assert.AnError
	})

	rp, err := NewRetryProvider(inner, 3, 0)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = rp.Complete(ctx, CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetryProvider_CircuitBreaker_TripsOpen(t *testing.T) {
	failCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		failCount++
		return nil, assert.AnError
	})

	rp, err := NewRetryProvider(inner, 0, 0)
	require.NoError(t, err)

	req := CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	cbThreshold := 5
	for i := 0; i < cbThreshold; i++ {
		_, err = rp.Complete(context.Background(), req)
		require.Error(t, err)
	}

	_, err = rp.Complete(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker: open")
}

func TestRetryProvider_Cache_Hit(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		return &CompletionResponse{Content: "cached"}, nil
	})

	rp, err := NewRetryProvider(inner, 0, 10*time.Second)
	require.NoError(t, err)

	req := CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	resp1, err := rp.Complete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "cached", resp1.Content)
	assert.Equal(t, 1, callCount)

	resp2, err := rp.Complete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "cached", resp2.Content)
	assert.Equal(t, 1, callCount)
}

func TestRetryProvider_Cache_Disabled(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		return &CompletionResponse{Content: "fresh"}, nil
	})

	rp, err := NewRetryProvider(inner, 0, 0)
	require.NoError(t, err)

	req := CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	for i := 0; i < 3; i++ {
		resp, err := rp.Complete(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "fresh", resp.Content)
	}
	assert.Equal(t, 3, callCount)
}

func TestRetryProvider_Cache_Expired(t *testing.T) {
	callCount := 0
	inner := mockProviderFunc(func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
		callCount++
		return &CompletionResponse{Content: "new"}, nil
	})

	rp, err := NewRetryProvider(inner, 0, 1*time.Millisecond)
	require.NoError(t, err)

	req := CompletionRequest{
		Model: "test", Messages: []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	_, err = rp.Complete(context.Background(), req)
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	_, err = rp.Complete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

// --- OpenAI supplement tests (connection error, context cancel, timeout, base URL) ---

func TestOpenAIProvider_Complete_ConnectionError(t *testing.T) {
	provider := &OpenAIProvider{client: &http.Client{Timeout: 100 * time.Millisecond}, timeout: 200 * time.Millisecond}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: "http://localhost:1",
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestOpenAIProvider_Complete_ContextCancelled(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiSuccessResponse("ok"))
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(ctx, CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
}

func TestOpenAIProvider_Complete_Timeout(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		json.NewEncoder(w).Encode(openaiSuccessResponse("late"))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 50 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Complete(ctx, CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.Error(t, err)
}

func TestOpenAIProvider_Complete_BasePathConstruction(t *testing.T) {
	var receivedPath string
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewEncoder(w).Encode(openaiSuccessResponse("ok"))
	})
	defer server.Close()

	provider := &OpenAIProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Model:   "gpt-4",
		BaseURL: server.URL,
		ApiKey:  "sk-test",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/chat/completions", receivedPath)
}

// --- Anthropic supplement tests (timeout, context cancel) ---

func TestAnthropicProvider_Complete_Timeout(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		json.NewEncoder(w).Encode(anthropicSuccessResponse("late"))
	})
	defer server.Close()

	provider := &AnthropicProvider{client: server.Client(), timeout: 50 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Complete(ctx, CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL,
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.Error(t, err)
}

func TestAnthropicProvider_Complete_ContextCancelled(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(anthropicSuccessResponse("ok"))
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := &AnthropicProvider{client: server.Client(), timeout: 5 * time.Second}
	_, err := provider.Complete(ctx, CompletionRequest{
		Model:    "claude-3-sonnet",
		BaseURL:  server.URL,
		ApiKey:   "sk-ant-test",
		Messages: []map[string]interface{}{{"role": "user", "content": "test"}},
	})
	require.Error(t, err)
}

// --- helper ---

func toMessages(raw interface{}) []map[string]interface{} {
	if raw == nil {
		return nil
	}
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, len(slice))
	for i, v := range slice {
		result[i], _ = v.(map[string]interface{})
	}
	return result
}