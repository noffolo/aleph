package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Mock LLM HTTP Server ───────────────────────────────────────────────────

type mockLLMServer struct {
	server   *httptest.Server
	callCount atomic.Int32
}

type ollamaToolCall struct {
	Function struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"function"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

func newMockLLMServer(t *testing.T, responses []ollamaResponse) *mockLLMServer {
	t.Helper()
	var idx atomic.Int32
	m := &mockLLMServer{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		i := int(idx.Load())
		if i >= len(responses) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ollamaResponse{
				Message: ollamaMessage{Role: "assistant", Content: "No more responses."},
			})
			return
		}
		idx.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses[i])
	}))
	t.Cleanup(m.server.Close)
	m.callCount.Store(0)
	return m
}

func (m *mockLLMServer) URL() string { return m.server.URL }
func (m *mockLLMServer) Calls() int  { return int(m.callCount.Load()) }

func ollamaTextResponse(text string) ollamaResponse {
	return ollamaResponse{
		Message: ollamaMessage{Role: "assistant", Content: text},
	}
}

func ollamaToolResponse(text string, tools []struct {
	Name      string
	Arguments map[string]interface{}
}) ollamaResponse {
	resp := ollamaResponse{
		Message: ollamaMessage{Role: "assistant", Content: text},
	}
	for _, tc := range tools {
		otc := ollamaToolCall{}
		otc.Function.Name = tc.Name
		otc.Function.Arguments = tc.Arguments
		resp.Message.ToolCalls = append(resp.Message.ToolCalls, otc)
	}
	return resp
}

// ─── Mock DecisionEngine ────────────────────────────────────────────────────

type mockEngine struct {
	planFunc            func(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent) (*decision.PlanResult, error)
	planWithProvFunc    func(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent, provider llm.Provider) (*decision.PlanResult, error)
	actFunc             func(ctx context.Context, step decision.PlannedStep, projectID string) (*decision.ActResult, error)
	observeFunc         func(ctx context.Context, step decision.PlannedStep, result *decision.ActResult) (*decision.Observation, error)
	reflectFunc         func(ctx context.Context, plan *decision.PlanResult, observations []decision.Observation) (*decision.PlanResult, error)
	admitFunc           func(ctx context.Context, results []*decision.ActResult, maxAttempts int) (bool, error)
	buildToolsFunc      func(ctx context.Context) []map[string]interface{}
	shouldAutoSkipFunc  func(step decision.PlannedStep) bool
}

func (m *mockEngine) Plan(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent) (*decision.PlanResult, error) {
	if m.planFunc != nil {
		return m.planFunc(ctx, msg, projectID, agentID, ontContent, agent)
	}
	return &decision.PlanResult{CanProceed: true, Reason: "mock plan", Steps: []decision.PlannedStep{}}, nil
}

func (m *mockEngine) PlanWithProvider(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent, provider llm.Provider) (*decision.PlanResult, error) {
	if m.planWithProvFunc != nil {
		return m.planWithProvFunc(ctx, msg, projectID, agentID, ontContent, agent, provider)
	}
	return &decision.PlanResult{CanProceed: true, Reason: "mock plan", Steps: []decision.PlannedStep{}}, nil
}

func (m *mockEngine) Act(ctx context.Context, step decision.PlannedStep, projectID string) (*decision.ActResult, error) {
	if m.actFunc != nil {
		return m.actFunc(ctx, step, projectID)
	}
	return &decision.ActResult{Step: step, Output: "mock output"}, nil
}

func (m *mockEngine) Observe(ctx context.Context, step decision.PlannedStep, result *decision.ActResult) (*decision.Observation, error) {
	if m.observeFunc != nil {
		return m.observeFunc(ctx, step, result)
	}
	return &decision.Observation{Success: true, Step: step}, nil
}

func (m *mockEngine) Reflect(ctx context.Context, plan *decision.PlanResult, observations []decision.Observation) (*decision.PlanResult, error) {
	if m.reflectFunc != nil {
		return m.reflectFunc(ctx, plan, observations)
	}
	return &decision.PlanResult{CanProceed: true, Reason: "mock reflect"}, nil
}

func (m *mockEngine) Admit(ctx context.Context, results []*decision.ActResult, maxAttempts int) (bool, error) {
	if m.admitFunc != nil {
		return m.admitFunc(ctx, results, maxAttempts)
	}
	return true, nil
}

func (m *mockEngine) BuildToolsMap(ctx context.Context) []map[string]interface{} {
	if m.buildToolsFunc != nil {
		return m.buildToolsFunc(ctx)
	}
	return []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "search_data",
				"description": "Search data objects",
			},
		},
	}
}

func (m *mockEngine) ShouldAutoSkip(step decision.PlannedStep) bool {
	if m.shouldAutoSkipFunc != nil {
		return m.shouldAutoSkipFunc(step)
	}
	return false
}

// ─── Mock ToolExecutor ──────────────────────────────────────────────────────

type mockToolExecutor struct {
	execFunc func(ctx context.Context, toolName string, args map[string]interface{}, projectID, agentID string) (string, bool, error)
}

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID, agentID string) (string, bool, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, toolName, args, projectID, agentID)
	}
	return "mock executor result", false, nil
}

// ─── Test Harness Setup ─────────────────────────────────────────────────────

func setupChatSessionTest(t *testing.T, llmResponses []ollamaResponse, eng decision.DecisionEngine, exec decision.ToolExecutor) (v1connect.QueryServiceClient, string, string) {
	t.Helper()

	llmSrv := newMockLLMServer(t, llmResponses)

	h, projectsRoot := setupQueryHandler(t)

	projectID := "test-proj"
	agentID := "test-agent"
	createProjectWithOntology(t, projectsRoot, projectID, "// empty ontology\n")

	metaRepo := setupMetaRepoWithAgent(t, agentID, llmSrv.URL())
	h.metaRepo = metaRepo

	// Override SSRF-guarded httpClient with default client for mock LLM access
	h.httpClient = http.DefaultClient

	if eng != nil || exec != nil {
		if eng == nil {
			eng = &mockEngine{}
		}
		if exec == nil {
			exec = &mockToolExecutor{}
		}
		h.SetDecisionEngine(eng, exec)
	}

	_, handler := v1connect.NewQueryServiceHandler(h)
	httpSrv := httptest.NewServer(handler)
	t.Cleanup(httpSrv.Close)

	client := v1connect.NewQueryServiceClient(http.DefaultClient, httpSrv.URL)
	return client, projectID, agentID
}

func setupMetaRepoWithAgent(t *testing.T, agentID, llmURL string) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE system_tools (id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT, category TEXT DEFAULT '', version TEXT DEFAULT '', health_status TEXT DEFAULT 'unknown', last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, source_type TEXT DEFAULT 'builtin')`,
		`CREATE TABLE system_skills (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT)`,
		`CREATE TABLE system_agents (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT, model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT, base_url TEXT)`,
		`CREATE TABLE system_chat_history (id TEXT PRIMARY KEY, project_id TEXT, agent_id TEXT, role TEXT, content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	_, err = db.Exec(
		`INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, base_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		agentID, "test-proj", "Test Agent", "ollama", "test-model",
		"test-key", "You are a helpful test assistant.", llmURL,
	)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestChatSession_PAORACycle(t *testing.T) {
	// LLM iter 1: text + tool call; iter 2: final text, no tools
	responses := []ollamaResponse{
		ollamaToolResponse("I'll search the data for you.", []struct {
			Name      string
			Arguments map[string]interface{}
		}{
			{Name: "search_data", Arguments: map[string]interface{}{"object_name": "items", "limit": float64(10)}},
		}),
		ollamaTextResponse("Here are the results: found 42 items."),
	}

	var actCalled, observeCalled, reflectCalled, admitCalled bool

	eng := &mockEngine{
		planWithProvFunc: func(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent, provider llm.Provider) (*decision.PlanResult, error) {
			return &decision.PlanResult{
				CanProceed: true, Reason: "mock plan",
			}, nil
		},
		actFunc: func(ctx context.Context, step decision.PlannedStep, projectID string) (*decision.ActResult, error) {
			actCalled = true
			return &decision.ActResult{Step: step, Output: `[{"name":"item1","qty":10},{"name":"item2","qty":32}]`}, nil
		},
		observeFunc: func(ctx context.Context, step decision.PlannedStep, result *decision.ActResult) (*decision.Observation, error) {
			observeCalled = true
			return &decision.Observation{Success: true, Step: step}, nil
		},
		reflectFunc: func(ctx context.Context, plan *decision.PlanResult, observations []decision.Observation) (*decision.PlanResult, error) {
			reflectCalled = true
			return &decision.PlanResult{CanProceed: true, Reason: "all observations successful"}, nil
		},
		admitFunc: func(ctx context.Context, results []*decision.ActResult, maxAttempts int) (bool, error) {
			admitCalled = true
			return true, nil
		},
	}

	client, projectID, agentID := setupChatSessionTest(t, responses, eng, nil)
	stream, err := client.Chat(context.Background(), connect.NewRequest(&v1.ChatRequest{
		Message: "search for items with quantity > 5", ProjectId: projectID, AgentId: agentID,
	}))
	require.NoError(t, err)

	var tokens, toolCalls []string
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Token != "" {
			tokens = append(tokens, msg.Token)
		}
		if msg.ToolCall != "" {
			toolCalls = append(toolCalls, msg.ToolCall)
		}
	}
	require.NoError(t, stream.Err())

	assert.True(t, actCalled)
	assert.True(t, observeCalled)
	assert.True(t, reflectCalled)
	assert.True(t, admitCalled)
	assert.Contains(t, tokens[0], "search")
	assert.Contains(t, tokens[1], "42")
	assert.Len(t, toolCalls, 1)
	assert.Contains(t, toolCalls[0], "search_data")
}

func TestChatSession_DegradedMode(t *testing.T) {
	// nil engine = degraded mode (no Plan/Act/Observe/Reflect/Admit)
	responses := []ollamaResponse{
		ollamaTextResponse("Hello! I am operating in degraded mode."),
	}
	exec := &mockToolExecutor{
		execFunc: func(ctx context.Context, toolName string, args map[string]interface{}, projectID, agentID string) (string, bool, error) {
			return "executor result", false, nil
		},
	}

	client, projectID, agentID := setupChatSessionTest(t, responses, nil, exec)
	stream, err := client.Chat(context.Background(), connect.NewRequest(&v1.ChatRequest{
		Message: "hello", ProjectId: projectID, AgentId: agentID,
	}))
	require.NoError(t, err)

	var tokens []string
	for stream.Receive() {
		if msg := stream.Msg(); msg.Token != "" {
			tokens = append(tokens, msg.Token)
		}
	}
	require.NoError(t, stream.Err())
	assert.Contains(t, tokens[0], "degraded")
}

func TestChatSession_ToolExecution(t *testing.T) {
	responses := []ollamaResponse{
		ollamaToolResponse("", []struct {
			Name      string
			Arguments map[string]interface{}
		}{
			{Name: "search_data", Arguments: map[string]interface{}{"object_name": "items", "limit": float64(5)}},
		}),
		ollamaTextResponse("Found 5 items matching your criteria."),
	}

	eng := &mockEngine{
		planWithProvFunc: func(ctx context.Context, msg, projectID, agentID string, ontContent []byte, agent *v1.Agent, provider llm.Provider) (*decision.PlanResult, error) {
			return &decision.PlanResult{CanProceed: true, Reason: "mock plan"}, nil
		},
		actFunc: func(ctx context.Context, step decision.PlannedStep, projectID string) (*decision.ActResult, error) {
			assert.Equal(t, "search_data", step.ToolName)
			assert.Equal(t, "items", step.Arguments["object_name"])
			return &decision.ActResult{Step: step, Output: `[{"name":"widget","qty":5}]`}, nil
		},
		observeFunc: func(ctx context.Context, step decision.PlannedStep, result *decision.ActResult) (*decision.Observation, error) {
			return &decision.Observation{Success: true, Step: step}, nil
		},
		reflectFunc: func(ctx context.Context, plan *decision.PlanResult, observations []decision.Observation) (*decision.PlanResult, error) {
			return &decision.PlanResult{CanProceed: true, Reason: "ok"}, nil
		},
		admitFunc: func(ctx context.Context, results []*decision.ActResult, maxAttempts int) (bool, error) {
			return true, nil
		},
	}

	client, projectID, agentID := setupChatSessionTest(t, responses, eng, nil)
	stream, err := client.Chat(context.Background(), connect.NewRequest(&v1.ChatRequest{
		Message: "search for widgets", ProjectId: projectID, AgentId: agentID,
	}))
	require.NoError(t, err)

	var tokens, toolCalls []string
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Token != "" {
			tokens = append(tokens, msg.Token)
		}
		if msg.ToolCall != "" {
			toolCalls = append(toolCalls, msg.ToolCall)
		}
	}
	require.NoError(t, stream.Err())
	require.Len(t, toolCalls, 1)
	assert.Contains(t, toolCalls[0], "search_data")
	require.Len(t, tokens, 1)
	assert.Contains(t, tokens[0], "Found 5 items")
}

func TestChatSession_ContextCancellation(t *testing.T) {
	responses := []ollamaResponse{
		ollamaTextResponse("This should not be reached if ctx is cancelled."),
	}

	client, projectID, agentID := setupChatSessionTest(t, responses, &mockEngine{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream, err := client.Chat(ctx, connect.NewRequest(&v1.ChatRequest{
		Message: "hello", ProjectId: projectID, AgentId: agentID,
	}))
	if err != nil {
		return // context cancelled before request began
	}
	var received int
	for stream.Receive() {
		received++
	}
	err = stream.Err()
	assert.Error(t, err)
	t.Logf("received %d msgs before cancellation: %v", received, err)
}

func TestChatSession_EmptyToolCalls(t *testing.T) {
	responses := []ollamaResponse{
		ollamaTextResponse("The answer is 42."),
	}

	var actCalled bool
	eng := &mockEngine{
		actFunc: func(ctx context.Context, step decision.PlannedStep, projectID string) (*decision.ActResult, error) {
			actCalled = true
			return &decision.ActResult{Output: "should not be reached"}, nil
		},
	}

	client, projectID, agentID := setupChatSessionTest(t, responses, eng, nil)
	stream, err := client.Chat(context.Background(), connect.NewRequest(&v1.ChatRequest{
		Message: "what is the answer?", ProjectId: projectID, AgentId: agentID,
	}))
	require.NoError(t, err)

	var tokens []string
	for stream.Receive() {
		if msg := stream.Msg(); msg.Token != "" {
			tokens = append(tokens, msg.Token)
		}
	}
	require.NoError(t, stream.Err())
	require.Len(t, tokens, 1)
	assert.Contains(t, tokens[0], "42")
	assert.False(t, actCalled)
}

// Preserve existing stubs from original chat_test.go (duplicated for compatibility)
func TestChatHandler_RequiresProjectID(t *testing.T) {
	h, _ := setupQueryHandler(t)
	require.NotNil(t, h)
}

func TestChatHandler_HandlerNotNil(t *testing.T) {
	h, _ := setupQueryHandler(t)
	require.NotNil(t, h)
}

func TestChatHandler_ContextCancellation(t *testing.T) {
	_, _ = setupQueryHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Error(t, ctx.Err())
}
