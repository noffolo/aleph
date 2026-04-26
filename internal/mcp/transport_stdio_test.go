package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type nopWC struct{ *bytes.Buffer }

func (nopWC) Close() error { return nil }

type nopRC struct{ *bytes.Buffer }

func (nopRC) Close() error { return nil }

// fakeStdio returns stdin (WriteCloser), stdout (*bytes.Buffer), stderr (ReadCloser).
func fakeStdio() (io.WriteCloser, *bytes.Buffer, io.ReadCloser) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return nopWC{stdin}, stdout, nopRC{stderr}
}

func TestNewMCPStdioTransport(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	if tr == nil {
		t.Fatal("transport should not be nil")
	}
	if tr.IsClosed() {
		t.Fatal("new transport should not be closed")
	}
	tr.Close()
}

func TestMCPStdioTransport_SendReceive(t *testing.T) {
	stdin, stdoutBuf, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdoutBuf, stderr)
	defer tr.Close()

	resp := JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      intPtr(1),
		Result:  json.RawMessage(`{"status":"ok"}`),
	}
	respRaw, _ := json.Marshal(resp)
	stdoutBuf.WriteString(string(respRaw) + "\n")

	req, err := NewRequest("test/method", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	respGot, err := tr.SendRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if respGot.Error != nil {
		t.Errorf("unexpected error: %+v", respGot.Error)
	}
}

func TestMCPStdioTransport_Close(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !tr.IsClosed() {
		t.Fatal("transport should be closed after Close()")
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("double Close: %v", err)
	}
}

func TestMCPStdioTransport_SendOnClosed(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	tr.Close()

	req, _ := NewRequest("test/method", nil)
	_, err := tr.SendRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on closed transport")
	}
}

func TestMCPStdioTransport_ContextCancelled(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	defer tr.Close()

	req, _ := NewRequest("test/method", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tr.SendRequest(ctx, req)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

func TestMCPStdioTransport_EmptyStdout(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	defer tr.Close()

	req, _ := NewRequest("test/method", nil)
	_, err := tr.SendRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when stdout has no data")
	}
}

func TestMCPStdioPinger_Ping(t *testing.T) {
	stdin, stdoutBuf, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdoutBuf, stderr)
	defer tr.Close()

	pinger := NewMCPStdioPinger(tr, "test-pinger")
	defer pinger.Close()

	resp := JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      intPtr(1),
		Result:  json.RawMessage(`{}`),
	}
	respRaw, _ := json.Marshal(resp)
	stdoutBuf.WriteString(string(respRaw) + "\n")

	if err := pinger.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestMCPStdioPinger_Ping_ErrorResponse(t *testing.T) {
	stdin, stdoutBuf, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdoutBuf, stderr)
	defer tr.Close()

	pinger := NewMCPStdioPinger(tr, "test-pinger")
	defer pinger.Close()

	resp := JSONRPCResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      intPtr(1),
		Error:   &JSONRPCError{Code: InternalError, Message: "subprocess error"},
	}
	respRaw, _ := json.Marshal(resp)
	stdoutBuf.WriteString(string(respRaw) + "\n")

	if err := pinger.Ping(context.Background()); err == nil {
		t.Fatal("expected error from ping error response")
	}
}

func TestMCPStdioPinger_Ping_Timeout(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	defer tr.Close()

	pinger := NewMCPStdioPinger(tr, "test-pinger")
	defer pinger.Close()

	// Don't write to stdout — ping hangs until timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	if err := pinger.Ping(ctx); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestMCPStdioTransport_ReadStderr(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	defer tr.Close()

	if rc, ok := stderr.(nopRC); ok {
		rc.Buffer.WriteString("log: info message\nlog: warning\n")
	}

	output, err := tr.ReadStderr()
	if err != nil {
		t.Fatalf("ReadStderr: %v", err)
	}
	if !strings.Contains(output, "info message") {
		t.Errorf("stderr missing 'info message': got %q", output)
	}
}

func TestRunSubprocess(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := RunSubprocess(stdin, stdout, stderr)
	if tr == nil {
		t.Fatal("RunSubprocess returned nil")
	}
	tr.Close()
}

func TestNewStdioPinger(t *testing.T) {
	stdin, stdout, stderr := fakeStdio()
	tr := NewMCPStdioTransport(stdin, stdout, stderr)
	defer tr.Close()

	p := NewStdioPinger(tr, "my-pinger")
	if p == nil {
		t.Fatal("NewStdioPinger returned nil")
	}
	p.Close()
}

func TestMCPStdioTransport_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stdin, stdoutBuf, stderr := fakeStdio()
			tr := NewMCPStdioTransport(stdin, stdoutBuf, stderr)
			defer tr.Close()

			resp := JSONRPCResponse{
				Jsonrpc: JSONRPCVersion,
				ID:      intPtr(1),
				Result:  json.RawMessage(`{"ok":true}`),
			}
			respRaw, _ := json.Marshal(resp)
			stdoutBuf.WriteString(string(respRaw) + "\n")

			req, _ := NewRequest("test/conc", nil)
			_, err := tr.SendRequest(context.Background(), req)
			if err != nil {
				t.Errorf("concurrent send: %v", err)
			}
		}()
	}
	wg.Wait()
}

func intPtr(n int) *int { return &n }
