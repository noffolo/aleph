package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/tools"
)

// mockTool is a simple tool that returns its name and params.
func mockToolExecute(ctx context.Context, params map[string]any) (any, error) {
	return map[string]any{
		"tool":   "mock_tool",
		"params": params,
	}, nil
}

// slowTool simulates a long-running tool for timeout testing.
func slowToolExecute(ctx context.Context, params map[string]any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return map[string]any{"status": "done"}, nil
	}
}

// errTool returns an error for error-path testing.
func errToolExecute(ctx context.Context, params map[string]any) (any, error) {
	return nil, &mockError{"execution failed: something went wrong"}
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

// newTestService creates an MCPService with a registry containing mock tools.
func newTestService(t *testing.T) *MCPService {
	t.Helper()

	reg := tools.NewToolRegistry()

	err := reg.Register(tools.ToolDefinition{
		Name:        "mock_tool",
		Category:    "test",
		Description: "A mock tool for testing",
		Params: []tools.ParamDef{
			{Name: "input", Type: "string", Description: "Input value", Required: true},
			{Name: "count", Type: "number", Description: "Count value", Required: false},
		},
		Execute: mockToolExecute,
	})
	if err != nil {
		t.Fatalf("failed to register mock_tool: %v", err)
	}

	err = reg.Register(tools.ToolDefinition{
		Name:        "slow_tool",
		Category:    "test",
		Description: "A slow tool for timeout testing",
		Execute:     slowToolExecute,
	})
	if err != nil {
		t.Fatalf("failed to register slow_tool: %v", err)
	}

	err = reg.Register(tools.ToolDefinition{
		Name:        "err_tool",
		Category:    "test",
		Description: "A tool that returns an error",
		Execute:     errToolExecute,
	})
	if err != nil {
		t.Fatalf("failed to register err_tool: %v", err)
	}

	// Register a tool with dot-notation name (category.name)
	err = reg.Register(tools.ToolDefinition{
		Name:        "finance.prophet_forecast",
		Category:    "finance",
		Description: "Time-series forecasting",
		Params: []tools.ParamDef{
			{Name: "symbol", Type: "string", Description: "Stock symbol", Required: true},
		},
		Execute: func(ctx context.Context, params map[string]any) (any, error) {
			return map[string]any{"forecast": "up", "symbol": params["symbol"]}, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to register finance.prophet_forecast: %v", err)
	}

	return NewMCPService(reg)
}

// ---------------------------------------------------------------------------
// tools/list
// ---------------------------------------------------------------------------

func TestHandleToolsList_ReturnsTools(t *testing.T) {
	svc := newTestService(t)
	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/list",
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}

	var result toolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(result.Tools) == 0 {
		t.Fatal("expected at least one tool in manifest")
	}

	// Verify mock_tool entry
	var mockEntry *toolManifestEntry
	for i := range result.Tools {
		if result.Tools[i].Name == "mock_tool" {
			mockEntry = &result.Tools[i]
			break
		}
	}
	if mockEntry == nil {
		t.Fatal("expected mock_tool in manifest")
	}
	if mockEntry.Description == "" {
		t.Error("expected non-empty description")
	}
	if mockEntry.InputSchema.Type != "object" {
		t.Errorf("inputSchema.type = %q, want %q", mockEntry.InputSchema.Type, "object")
	}
	if _, ok := mockEntry.InputSchema.Properties["input"]; !ok {
		t.Error("expected 'input' property in inputSchema")
	}
	if len(mockEntry.InputSchema.Required) != 1 || mockEntry.InputSchema.Required[0] != "input" {
		t.Errorf("required = %v, want [input]", mockEntry.InputSchema.Required)
	}
}

func TestHandleToolsList_IDPreserved(t *testing.T) {
	svc := newTestService(t)
	id := 42
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/list",
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.ID == nil {
		t.Fatal("expected non-nil ID")
	}
	if *resp.ID != 42 {
		t.Errorf("ID = %d, want 42", *resp.ID)
	}
}

// ---------------------------------------------------------------------------
// tools/call
// ---------------------------------------------------------------------------

func TestHandleToolsCall_ValidParams(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"method": "mock_tool",
		"params": map[string]any{
			"input": "hello",
			"count": 42,
		},
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}

	var callResult map[string]any
	if err := json.Unmarshal(resp.Result, &callResult); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	content, ok := callResult["content"]
	if !ok {
		t.Fatal("expected 'content' field in result")
	}
	contentArr, ok := content.([]any)
	if !ok || len(contentArr) == 0 {
		t.Fatal("expected non-empty content array")
	}
	firstItem, ok := contentArr[0].(map[string]any)
	if !ok {
		t.Fatal("expected content[0] to be an object")
	}
	if firstItem["type"] != "text" {
		t.Errorf("content[0].type = %v, want 'text'", firstItem["type"])
	}
}

func TestHandleToolsCall_WithNameField(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"name": "mock_tool",
		"arguments": map[string]any{
			"input": "test",
		},
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestHandleToolsCall_DotNotation(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"method": "finance.prophet_forecast",
		"params": map[string]any{
			"symbol": "AAPL",
		},
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestHandleToolsCall_InvalidTool(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"method": "nonexistent_tool",
		"params": map[string]any{},
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for invalid tool")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InvalidParams)
	}
	if resp.Error.Message != "tool not found: nonexistent_tool" {
		t.Errorf("error.message = %q, want %q", resp.Error.Message, "tool not found: nonexistent_tool")
	}
}

func TestHandleToolsCall_MissingParams(t *testing.T) {
	svc := newTestService(t)
	id := 1

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for missing params")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InvalidParams)
	}
}

func TestHandleToolsCall_MissingToolName(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{"params": map[string]any{}}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for missing tool name")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InvalidParams)
	}
}

// ---------------------------------------------------------------------------
// Unknown method
// ---------------------------------------------------------------------------

func TestHandleUnknownMethod(t *testing.T) {
	svc := newTestService(t)
	id := 1

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "unknown_method",
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, MethodNotFound)
	}
}

// ---------------------------------------------------------------------------
// Invalid request
// ---------------------------------------------------------------------------

func TestHandleInvalidRequest(t *testing.T) {
	svc := newTestService(t)

	req := &JSONRPCRequest{
		Jsonrpc: "1.0",
		Method:  "tools/list",
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for invalid request")
	}
	if resp.Error.Code != InvalidRequest {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InvalidRequest)
	}
}

// ---------------------------------------------------------------------------
// Concurrent calls (race test)
// ---------------------------------------------------------------------------

func TestConcurrentCallsAreSafe(t *testing.T) {
	svc := newTestService(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			id := idx
			req := &JSONRPCRequest{
				Jsonrpc: JSONRPCVersion,
				ID:      &id,
				Method:  "tools/list",
			}

			resp := svc.HandleRequest(context.Background(), req)
			if resp.Error != nil {
				return
			}
			if resp.Result == nil {
				return
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		id := 100
		params := map[string]any{
			"method": "mock_tool",
			"params": map[string]any{"input": "concurrent"},
		}
		rawParams, _ := json.Marshal(params)

		req := &JSONRPCRequest{
			Jsonrpc: JSONRPCVersion,
			ID:      &id,
			Method:  "tools/call",
			Params:  rawParams,
		}

		resp := svc.HandleRequest(context.Background(), req)
		if resp.Error != nil {
			return
		}
	}()

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestHandleToolsCall_ContextCancelled(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"method": "slow_tool",
		"params": map[string]any{},
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(ctx, req)

	if resp.Error == nil {
		t.Fatal("expected error for cancelled context")
	}
	if resp.Error.Code != InternalError {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InternalError)
	}
	// Allow either cancellation or timeout message
	if resp.Error.Message != "request cancelled" && resp.Error.Message != "tool execution timed out" {
		t.Errorf("error.message = %q, want %q or %q", resp.Error.Message, "request cancelled", "tool execution timed out")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestHandleToolsCall_EmptyParams(t *testing.T) {
	svc := newTestService(t)
	id := 1

	params := map[string]any{
		"method": "mock_tool",
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  rawParams,
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestHandleToolsCall_InvalidJSON(t *testing.T) {
	svc := newTestService(t)
	id := 1

	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "tools/call",
		Params:  json.RawMessage(`{invalid`),
	}

	resp := svc.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InvalidParams)
	}
}
