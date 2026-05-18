package mcp

import (
	"encoding/json"
	"testing"
)

// TestJSONRPCRequestRoundTrip verifies that a JSONRPCRequest can be marshalled
// to JSON and unmarshalled back with all fields preserved.
func TestJSONRPCRequestRoundTrip(t *testing.T) {
	id := 42
	params := map[string]any{
		"query": "test",
		"limit": 10,
	}
	rawParams, _ := json.Marshal(params)

	original := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "list_tools",
		Params:  rawParams,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if decoded.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", decoded.Jsonrpc, JSONRPCVersion)
	}
	if decoded.ID == nil {
		t.Fatal("ID is nil")
	}
	if *decoded.ID != id {
		t.Errorf("ID = %d, want %d", *decoded.ID, id)
	}
	if decoded.Method != "list_tools" {
		t.Errorf("Method = %q, want %q", decoded.Method, "list_tools")
	}

	// Verify params round-trip
	var gotParams map[string]any
	if err := json.Unmarshal(decoded.Params, &gotParams); err != nil {
		t.Fatalf("unmarshal decoded params: %v", err)
	}
	if gotParams["query"] != "test" {
		t.Errorf("params.query = %v, want %v", gotParams["query"], "test")
	}
	if int(gotParams["limit"].(float64)) != 10 {
		t.Errorf("params.limit = %v, want %v", gotParams["limit"], 10)
	}
}

// TestJSONRPCResponseRoundTrip_Success verifies a successful response round-trip.
func TestJSONRPCResponseRoundTrip_Success(t *testing.T) {
	id := 7
	result := map[string]any{
		"status": "ok",
		"count":  3,
	}
	rawResult, _ := json.Marshal(result)

	original := &JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Result:  rawResult,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if decoded.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", decoded.Jsonrpc, JSONRPCVersion)
	}
	if decoded.ID == nil {
		t.Fatal("ID is nil")
	}
	if *decoded.ID != id {
		t.Errorf("ID = %d, want %d", *decoded.ID, id)
	}
	if decoded.Error != nil {
		t.Errorf("unexpected error in success response: %+v", decoded.Error)
	}

	// Verify result round-trip
	var gotResult map[string]any
	if err := json.Unmarshal(decoded.Result, &gotResult); err != nil {
		t.Fatalf("unmarshal decoded result: %v", err)
	}
	if gotResult["status"] != "ok" {
		t.Errorf("result.status = %v, want %v", gotResult["status"], "ok")
	}
}

// TestJSONRPCResponseRoundTrip_Error verifies an error response round-trip.
func TestJSONRPCResponseRoundTrip_Error(t *testing.T) {
	id := 99
	original := &JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Error: &JSONRPCError{
			Code:    MethodNotFound,
			Message: "method not found: unknown_tool",
			Data:    "tool 'unknown_tool' is not registered",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if decoded.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", decoded.Jsonrpc, JSONRPCVersion)
	}
	if decoded.ID == nil {
		t.Fatal("ID is nil")
	}
	if *decoded.ID != id {
		t.Errorf("ID = %d, want %d", *decoded.ID, id)
	}
	if decoded.Result != nil {
		t.Errorf("unexpected result in error response: %s", string(decoded.Result))
	}
	if decoded.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if decoded.Error.Code != MethodNotFound {
		t.Errorf("error.code = %d, want %d", decoded.Error.Code, MethodNotFound)
	}
	if decoded.Error.Message != "method not found: unknown_tool" {
		t.Errorf("error.message = %q, want %q", decoded.Error.Message, "method not found: unknown_tool")
	}
}

// TestParseErrorCode verifies the ParseError constant.
func TestParseErrorCode(t *testing.T) {
	if ParseError != -32700 {
		t.Errorf("ParseError = %d, want %d", ParseError, -32700)
	}
}

// TestInvalidRequestCode verifies the InvalidRequest constant.
func TestInvalidRequestCode(t *testing.T) {
	if InvalidRequest != -32600 {
		t.Errorf("InvalidRequest = %d, want %d", InvalidRequest, -32600)
	}
}

// TestMethodNotFoundCode verifies the MethodNotFound constant.
func TestMethodNotFoundCode(t *testing.T) {
	if MethodNotFound != -32601 {
		t.Errorf("MethodNotFound = %d, want %d", MethodNotFound, -32601)
	}
}

// TestInvalidParamsCode verifies the InvalidParams constant.
func TestInvalidParamsCode(t *testing.T) {
	if InvalidParams != -32602 {
		t.Errorf("InvalidParams = %d, want %d", InvalidParams, -32602)
	}
}

// TestInternalErrorCode verifies the InternalError constant.
func TestInternalErrorCode(t *testing.T) {
	if InternalError != -32603 {
		t.Errorf("InternalError = %d, want %d", InternalError, -32603)
	}
}

// TestNewRequest_Valid creates a request via NewRequest and verifies it.
func TestNewRequest_Valid(t *testing.T) {
	params := map[string]string{"key": "value"}
	req, err := NewRequest("ping", params)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	if req.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", req.Jsonrpc, JSONRPCVersion)
	}
	if req.ID == nil {
		t.Fatal("ID is nil")
	}
	if *req.ID != 1 {
		t.Errorf("ID = %d, want 1", *req.ID)
	}
	if req.Method != "ping" {
		t.Errorf("Method = %q, want %q", req.Method, "ping")
	}
	if req.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestNewRequest_EmptyMethod verifies NewRequest rejects an empty method.
func TestNewRequest_EmptyMethod(t *testing.T) {
	_, err := NewRequest("", nil)
	if err == nil {
		t.Fatal("expected error for empty method, got nil")
	}
}

// TestNewRequest_NilParams creates a request with nil params and verifies Params is nil.
func TestNewRequest_NilParams(t *testing.T) {
	req, err := NewRequest("ping", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req.Params != nil {
		t.Error("expected nil Params for nil input")
	}
}

// TestNewResponse_Valid creates a success response via NewResponse and verifies it.
func TestNewResponse_Valid(t *testing.T) {
	result := map[string]string{"status": "ok"}
	resp, err := NewResponse(5, result)
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}

	if resp.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", resp.Jsonrpc, JSONRPCVersion)
	}
	if resp.ID == nil {
		t.Fatal("ID is nil")
	}
	if *resp.ID != 5 {
		t.Errorf("ID = %d, want 5", *resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("Result is nil")
	}

	// Verify result content
	var got map[string]string
	if err := json.Unmarshal(resp.Result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("result.status = %q, want %q", got["status"], "ok")
	}
}

// TestNewResponse_NilResult creates a response with nil result and verifies Result is nil.
func TestNewResponse_NilResult(t *testing.T) {
	resp, err := NewResponse(1, nil)
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}
	if resp.Result != nil {
		t.Error("expected nil Result for nil input")
	}
}

// TestNewError creates an error response via NewError and verifies it.
func TestNewError(t *testing.T) {
	data := map[string]string{"detail": "something went wrong"}
	resp := NewError(3, InternalError, "internal server error", data)

	if resp.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", resp.Jsonrpc, JSONRPCVersion)
	}
	if resp.ID == nil {
		t.Fatal("ID is nil")
	}
	if *resp.ID != 3 {
		t.Errorf("ID = %d, want 3", *resp.ID)
	}
	if resp.Result != nil {
		t.Errorf("unexpected result in error response: %s", string(resp.Result))
	}
	if resp.Error == nil {
		t.Fatal("Error is nil")
	}
	if resp.Error.Code != InternalError {
		t.Errorf("error.code = %d, want %d", resp.Error.Code, InternalError)
	}
	if resp.Error.Message != "internal server error" {
		t.Errorf("error.message = %q, want %q", resp.Error.Message, "internal server error")
	}
}

// TestRequestValidate_Valid verifies Validate passes on a well-formed request.
func TestRequestValidate_Valid(t *testing.T) {
	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "list_tools",
	}
	if err := req.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// TestRequestValidate_WrongVersion verifies Validate rejects wrong jsonrpc version.
func TestRequestValidate_WrongVersion(t *testing.T) {
	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: "1.0",
		ID:      &id,
		Method:  "list_tools",
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for wrong jsonrpc version, got nil")
	}
}

// TestRequestValidate_NilID verifies Validate rejects nil ID.
func TestRequestValidate_NilID(t *testing.T) {
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		Method:  "list_tools",
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for nil ID, got nil")
	}
}

// TestRequestValidate_EmptyMethod verifies Validate rejects empty method.
func TestRequestValidate_EmptyMethod(t *testing.T) {
	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for empty method, got nil")
	}
}

// TestErrorSerialization verifies that error responses marshal to the expected
// JSON shape and back.
func TestErrorSerialization(t *testing.T) {
	data := "additional context"
	resp := NewError(42, ParseError, "parse error", data)

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	// Verify JSON structure
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("unmarshal raw response: %v", err)
	}

	if rawMap["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", rawMap["jsonrpc"])
	}

	// Verify error object
	errObj, ok := rawMap["error"].(map[string]any)
	if !ok {
		t.Fatal("error field is not an object")
	}
	if int(errObj["code"].(float64)) != ParseError {
		t.Errorf("error.code = %v, want %d", errObj["code"], ParseError)
	}
	if errObj["message"] != "parse error" {
		t.Errorf("error.message = %v, want %q", errObj["message"], "parse error")
	}
	if errObj["data"] != "additional context" {
		t.Errorf("error.data = %v, want %q", errObj["data"], "additional context")
	}

	// Verify result is absent
	if _, hasResult := rawMap["result"]; hasResult {
		t.Error("unexpected result field in error response")
	}
}

// TestEnvelopeRoundTrip verifies JSONRPCEnvelope can carry raw messages.
func TestEnvelopeRoundTrip(t *testing.T) {
	id := 1
	params := json.RawMessage(`{"query":"test"}`)

	env := &JSONRPCEnvelope{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "ping",
		Params:  params,
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var decoded JSONRPCEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if decoded.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", decoded.Jsonrpc, JSONRPCVersion)
	}
	if decoded.ID == nil {
		t.Fatal("ID is nil")
	}
	if *decoded.ID != id {
		t.Errorf("ID = %d, want %d", *decoded.ID, id)
	}
	if decoded.Method != "ping" {
		t.Errorf("Method = %q, want %q", decoded.Method, "ping")
	}
}

// TestRequestMarshalUnmarshalNullParams verifies that a request with null
// params handles correctly.
func TestRequestMarshalUnmarshalNullParams(t *testing.T) {
	id := 1
	req := &JSONRPCRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
		Method:  "no_params",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Method != "no_params" {
		t.Errorf("Method = %q, want %q", decoded.Method, "no_params")
	}
}

// TestResponseMarshalUnmarshalNullResult verifies that a response with
// null/nil result fields handles correctly.
func TestResponseMarshalUnmarshalNullResult(t *testing.T) {
	id := 1
	resp := &JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      &id,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Jsonrpc != JSONRPCVersion {
		t.Errorf("jsonrpc = %q, want %q", decoded.Jsonrpc, JSONRPCVersion)
	}
	if decoded.Error != nil {
		t.Errorf("unexpected error: %+v", decoded.Error)
	}
}

// TestNewRequestRoundTrip verifies NewRequest → marshal → unmarshal preserves fields.
func TestNewRequestRoundTrip(t *testing.T) {
	params := map[string]string{"symbol": "AAPL", "interval": "1d"}
	original, err := NewRequest("get_quote", params)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Method != "get_quote" {
		t.Errorf("Method = %q, want %q", decoded.Method, "get_quote")
	}

	var gotParams map[string]string
	if err := json.Unmarshal(decoded.Params, &gotParams); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if gotParams["symbol"] != "AAPL" {
		t.Errorf("symbol = %q, want %q", gotParams["symbol"], "AAPL")
	}
	if gotParams["interval"] != "1d" {
		t.Errorf("interval = %q, want %q", gotParams["interval"], "1d")
	}
}

// TestNewResponseRoundTrip verifies NewResponse → marshal → unmarshal preserves fields.
func TestNewResponseRoundTrip(t *testing.T) {
	result := map[string]any{
		"price":  150.25,
		"change": -2.5,
	}
	original, err := NewResponse(42, result)
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID == nil || *decoded.ID != 42 {
		t.Errorf("ID = %v, want 42", decoded.ID)
	}

	var gotResult map[string]any
	if err := json.Unmarshal(decoded.Result, &gotResult); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if gotResult["price"] != 150.25 {
		t.Errorf("price = %v, want %v", gotResult["price"], 150.25)
	}
	if gotResult["change"] != -2.5 {
		t.Errorf("change = %v, want %v", gotResult["change"], -2.5)
	}
}

// TestNewErrorRoundTrip verifies NewError → marshal → unmarshal preserves fields.
func TestNewErrorRoundTrip(t *testing.T) {
	data := map[string]string{"reason": "rate limit exceeded"}
	original := NewError(100, InvalidRequest, "invalid request", data)

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("expected error")
	}
	if decoded.Error.Code != InvalidRequest {
		t.Errorf("code = %d, want %d", decoded.Error.Code, InvalidRequest)
	}
	if decoded.Error.Message != "invalid request" {
		t.Errorf("message = %q, want %q", decoded.Error.Message, "invalid request")
	}

	gotData, ok := decoded.Error.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", decoded.Error.Data)
	}
	if gotData["reason"] != "rate limit exceeded" {
		t.Errorf("data.reason = %v, want %q", gotData["reason"], "rate limit exceeded")
	}
}
