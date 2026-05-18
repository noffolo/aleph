// Package mcp provides MCP (Model Context Protocol) types and utilities.
//
// This file implements the JSON-RPC 2.0 message envelope for MCP protocol
// communication. It provides pure data types, serialization helpers, and
// validation — no I/O, no transport, no business logic.
package mcp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 protocol version constant.
const JSONRPCVersion = "2.0"

// Standard JSON-RPC 2.0 error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// JSONRPCEnvelope represents the top-level structure of any JSON-RPC 2.0
// message. It contains only the fields common to both requests and responses.
type JSONRPCEnvelope struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request message.
type JSONRPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response message.
// Exactly one of Result or Error must be present.
type JSONRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// NewRequest creates a JSON-RPC 2.0 request with an auto-assigned ID of 1.
// The params value is serialized to JSON. If params is nil, Params is left nil.
func NewRequest(method string, params any) (*JSONRPCRequest, error) {
	if method == "" {
		return nil, fmt.Errorf("method must not be empty")
	}

	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  method,
	}

	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = raw
	}

	return req, nil
}

// NewResponse creates a JSON-RPC 2.0 success response. The result value is
// serialized to JSON. The id is set from the id parameter.
func NewResponse(id int, result any) (*JSONRPCResponse, error) {
	resp := &JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
	}

	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resp.Result = raw
	}

	return resp, nil
}

// NewError creates a JSON-RPC 2.0 error response with the given error details.
func NewError(id int, code int, message string, data any) *JSONRPCResponse {
	resp := &JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	return resp
}

// Validate checks that the request conforms to JSON-RPC 2.0 specification:
//   - jsonrpc must be "2.0"
//   - ID must be non-nil (notifications excluded — this is for requests)
//   - method must be non-empty
func (r *JSONRPCRequest) Validate() error {
	if r.Jsonrpc != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version %q: expected %q", r.Jsonrpc, JSONRPCVersion)
	}
	if r.ID == nil {
		return fmt.Errorf("request ID must not be nil")
	}
	if r.Method == "" {
		return fmt.Errorf("method must not be empty")
	}
	return nil
}
