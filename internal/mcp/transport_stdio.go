package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultReadTimeout  = 30 * time.Second
	defaultPingInterval = 30 * time.Second
	healthMethod        = "health/ping"
)

// MCPStdioTransport implements line-delimited JSON-RPC 2.0 communication over
// stdin/stdout for MCP tool subprocesses.
type MCPStdioTransport struct {
	mu          sync.RWMutex
	stdin       io.WriteCloser
	stdout      *bufio.Scanner
	stderr      io.ReadCloser
	readTimeout time.Duration
	pongTimeout time.Duration
	pinger      *MCPStdioPinger
	closed      bool
}

// MCPStdioPinger wraps a transport to implement the Pinger interface for health
// monitoring. It is registered with ToolHealthMonitor via NewToolHealthMonitor.
type MCPStdioPinger struct {
	transport *MCPStdioTransport
	name      string
}

// NewMCPStdioTransport creates a transport from stdin/stdout/stderr handles.
// Typically used with os/exec.Cmd.StdinPipe/StdoutPipe/StderrPipe.
func NewMCPStdioTransport(stdin io.WriteCloser, stdout io.Reader, stderr io.ReadCloser) *MCPStdioTransport {
	return &MCPStdioTransport{
		stdin:       stdin,
		stdout:      bufio.NewScanner(stdout),
		stderr:      stderr,
		readTimeout: defaultReadTimeout,
		pongTimeout: 5 * time.Second,
	}
}

// NewMCPStdioPinger creates a Pinger-compatible wrapper for the transport.
func NewMCPStdioPinger(transport *MCPStdioTransport, name string) *MCPStdioPinger {
	return &MCPStdioPinger{transport: transport, name: name}
}

// Ping sends a health/ping request and waits for a response.
func (p *MCPStdioPinger) Ping(ctx context.Context) error {
	req, err := NewRequest(healthMethod, nil)
	if err != nil {
		return fmt.Errorf("create ping request: %w", err)
	}

	resp, err := p.transport.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("ping send: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("ping error: %s", resp.Error.Message)
	}
	return nil
}

// Close closes the transport.
func (p *MCPStdioPinger) Close() error {
	return p.transport.Close()
}

// SendRequest sends a JSON-RPC request and waits for a response.
func (t *MCPStdioTransport) SendRequest(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	respCh := make(chan *JSONRPCResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("mcp transport goroutine panic", "recover", r)
				select {
				case errCh <- fmt.Errorf("transport goroutine panic: %v", r):
				default:
				}
			}
		}()
		t.mu.Lock()
		defer t.mu.Unlock()

		if t.closed {
			errCh <- fmt.Errorf("transport closed")
			return
		}

		raw, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("marshal request: %w", err)
			return
		}

		if _, err := fmt.Fprintf(t.stdin, "%s\n", string(raw)); err != nil {
			errCh <- fmt.Errorf("write stdin: %w", err)
			return
		}

		slog.Debug("mcp transport sent", "method", req.Method, "id", req.ID)

		if t.stdout.Scan() {
			line := t.stdout.Text()
			var resp JSONRPCResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				errCh <- fmt.Errorf("unmarshal response: %w", err)
				return
			}
			respCh <- &resp
		} else {
			if err := t.stdout.Err(); err != nil {
				errCh <- fmt.Errorf("stdout read error: %w", err)
			} else {
				errCh <- fmt.Errorf("stdout closed")
			}
		}
	}()

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close shuts down the transport.
func (t *MCPStdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	var errs []error
	if err := t.stdin.Close(); err != nil {
		errs = append(errs, fmt.Errorf("stdin close: %w", err))
	}
	if err := t.stderr.Close(); err != nil {
		errs = append(errs, fmt.Errorf("stderr close: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("transport close errors: %v", errs)
	}
	return nil
}

// ReadStderr reads all bytes from stderr (for diagnostics/logs).
func (t *MCPStdioTransport) ReadStderr() (string, error) {
	data, err := io.ReadAll(t.stderr)
	if err != nil {
		return "", fmt.Errorf("read stderr: %w", err)
	}
	return string(data), nil
}

// IsClosed returns whether the transport has been closed.
func (t *MCPStdioTransport) IsClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}

// RunSubprocess starts a subprocess and returns an MCPStdioTransport connected
// to its stdin/stdout/stderr, plus a Pinger for health monitoring.
// The caller is responsible for calling cmd.Wait() after the transport is closed.
//
// Example:
//
//	cmd := exec.CommandContext(ctx, "my-mcp-server")
//	stdin, _ := cmd.StdinPipe()
//	stdout, _ := cmd.StdoutPipe()
//	stderr, _ := cmd.StderrPipe()
//	cmd.Start()
//	transport, pinger := RunSubprocess(stdin, stdout, stderr, "my-server")
//	defer transport.Close()
func RunSubprocess(stdin io.WriteCloser, stdout io.Reader, stderr io.ReadCloser) *MCPStdioTransport {
	return NewMCPStdioTransport(stdin, stdout, stderr)
}

// NewStdioPinger creates a subprocess health monitor using the given transport.
// It wraps the transport in a ToolHealthMonitor for periodic health checks.
//
// Example:
//
//	transport := RunSubprocess(stdin, stdout, stderr)
//	pinger := NewStdioPinger(transport, "my-server")
//	monitor := NewToolHealthMonitor("my-server", pinger, 30*time.Second)
//	monitor.Start(ctx)
//	defer monitor.Stop()
func NewStdioPinger(transport *MCPStdioTransport, name string) *MCPStdioPinger {
	return NewMCPStdioPinger(transport, name)
}
