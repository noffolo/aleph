// Package mcp provides MCP (Model Context Protocol) types and utilities.
//
// This file implements the JSON-RPC 2.0 method dispatch for MCP protocol
// tools/list and tools/call. It is transport-agnostic — no STDIO, no HTTP.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/tools"
)

// Default execution timeout for tools/call.
const toolExecTimeout = 30 * time.Second

// MCPService dispatches JSON-RPC 2.0 requests to the ToolRegistry.
// It is transport-agnostic — call HandleRequest with any JSON-RPC request
// and receive a JSON-RPC response.
type MCPService struct {
	registry *tools.ToolRegistry
	mu       sync.RWMutex
}

// NewMCPService creates an MCPService backed by the given ToolRegistry.
func NewMCPService(registry *tools.ToolRegistry) *MCPService {
	return &MCPService{
		registry: registry,
	}
}

// HandleRequest dispatches a single JSON-RPC 2.0 request.
//
// Supported methods:
//   - "tools/list" — returns the full tool manifest
//   - "tools/call" — executes a tool by name
//
// Unknown methods return a -32601 MethodNotFound error.
func (s *MCPService) HandleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	if err := req.Validate(); err != nil {
		return NewError(requestID(req), InvalidRequest, "invalid request: "+err.Error(), nil)
	}

	id := requestID(req)

	switch req.Method {
	case "tools/list":
		return s.handleToolsList(id)
	case "tools/call":
		return s.handleToolsCall(ctx, id, req.Params)
	default:
		return NewError(id, MethodNotFound, fmt.Sprintf("method not found: %s", req.Method), nil)
	}
}

// ---------------------------------------------------------------------------
// tools/list
// ---------------------------------------------------------------------------

// toolManifestEntry is a single entry in the tools/list response.
type toolManifestEntry struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

// inputSchema is a JSON Schema object describing tool parameters.
type inputSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]paramProperty `json:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}

// paramProperty describes a single parameter property in JSON Schema.
type paramProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// toolsListResult is the top-level result for tools/list.
type toolsListResult struct {
	Tools []toolManifestEntry `json:"tools"`
}

func (s *MCPService) handleToolsList(id int) *JSONRPCResponse {
	s.mu.RLock()
	defs := s.registry.List("")
	s.mu.RUnlock()

	entries := make([]toolManifestEntry, 0, len(defs))
	for _, def := range defs {
		props := make(map[string]paramProperty, len(def.Params))
		required := make([]string, 0, len(def.Params))
		for _, p := range def.Params {
			props[p.Name] = paramProperty{
				Type:        p.Type,
				Description: p.Description,
			}
			if p.Required {
				required = append(required, p.Name)
			}
		}
		entries = append(entries, toolManifestEntry{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: inputSchema{
				Type:       "object",
				Properties: props,
				Required:   required,
			},
		})
	}

	resp, err := NewResponse(id, toolsListResult{Tools: entries})
	if err != nil {
		slog.Error("failed to marshal tools/list response", "error", err)
		return NewError(id, InternalError, "failed to marshal response", nil)
	}
	return resp
}

// ---------------------------------------------------------------------------
// tools/call
// ---------------------------------------------------------------------------

// toolsCallParams is the expected shape of the params object for tools/call.
type toolsCallParams struct {
	// Method is the tool name (e.g. "finance.prophet_forecast", "osint_region_dossier").
	Method string `json:"method"`
	// Name is an alternative field for the tool name (MCP spec uses "name").
	Name string `json:"name"`
	// Params holds tool-specific arguments.
	Params map[string]any `json:"params"`
	// Arguments is an alternative field (MCP spec uses "arguments").
	Arguments map[string]any `json:"arguments"`
}

func (s *MCPService) handleToolsCall(ctx context.Context, id int, rawParams json.RawMessage) *JSONRPCResponse {
	if len(rawParams) == 0 {
		return NewError(id, InvalidParams, "params is required for tools/call", nil)
	}

	var params toolsCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return NewError(id, InvalidParams, fmt.Sprintf("invalid params JSON: %v", err), nil)
	}

	// Resolve tool name: "method" or "name"
	toolName := params.Method
	if toolName == "" {
		toolName = params.Name
	}
	if toolName == "" {
		return NewError(id, InvalidParams, "tool name is required (provide \"method\" or \"name\")", nil)
	}

	// Resolve tool arguments: "params" or "arguments"
	execParams := params.Params
	if execParams == nil {
		execParams = params.Arguments
	}
	if execParams == nil {
		execParams = make(map[string]any)
	}

	// Resolve the tool: try dot notation first, then flat name search
	category, name := s.resolveToolName(toolName)
	if name == "" {
		return NewError(id, InvalidParams, fmt.Sprintf("tool not found: %s", toolName), nil)
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, toolExecTimeout)
	defer cancel()

	resultCh := make(chan execResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("tool execution goroutine panic", "recover", r)
				resultCh <- execResult{err: fmt.Errorf("tool execution panic: %v", r)}
			}
		}()
		s.mu.RLock()
		result, err := s.registry.ExecuteContext(execCtx, category, name, execParams)
		s.mu.RUnlock()
		resultCh <- execResult{result: result, err: err}
	}()

	select {
	case <-execCtx.Done():
		if ctx.Err() != nil {
			return NewError(id, InternalError, "request cancelled", nil)
		}
		return NewError(id, InternalError, "tool execution timed out", nil)
	case r := <-resultCh:
		if r.err != nil {
			return NewError(id, InternalError, fmt.Sprintf("tool execution failed: %v", r.err), nil)
		}

		// tools/call result wraps in { "content": [...] } per MCP spec
		content := []map[string]any{
			{
				"type": "text",
				"text": mustMarshalJSON(r.result),
			},
		}
		resp, err := NewResponse(id, map[string]any{"content": content})
		if err != nil {
			slog.Error("failed to marshal tools/call response", "error", err)
			return NewError(id, InternalError, "failed to marshal response", nil)
		}
		return resp
	}
}

// resolveToolName resolves a tool name to (category, name).
//
// Strategy:
//  1. If the name contains a dot (e.g. "finance.prophet_forecast"), split and look up.
//  2. If not found, search all categories for a tool with the given name.
func (s *MCPService) resolveToolName(toolName string) (category, name string) {
	// Try dot notation first
	for i := 0; i < len(toolName); i++ {
		if toolName[i] == '.' {
			cat, n := toolName[:i], toolName[i+1:]
			s.mu.RLock()
			_, ok := s.registry.Get(cat, n)
			s.mu.RUnlock()
			if ok {
				return cat, n
			}
			break
		}
	}

	// Fallback: search all categories
	s.mu.RLock()
	defs := s.registry.List("")
	s.mu.RUnlock()

	for _, def := range defs {
		if def.Name == toolName {
			return def.Category, def.Name
		}
	}

	return "", ""
}

// execResult carries the result or error from a tool execution goroutine.
type execResult struct {
	result any
	err    error
}

// mustMarshalJSON marshals v to a JSON string. If marshalling fails,
// it returns a string representation of v instead.
func mustMarshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// requestID extracts the request ID from a request, defaulting to 0 if nil.
func requestID(req *JSONRPCRequest) int {
	if req.ID != nil {
		return *req.ID
	}
	return 0
}
