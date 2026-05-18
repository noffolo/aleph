package handler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"google.golang.org/protobuf/types/known/structpb"
)

// mockSandboxManager implements sandbox.SandboxManager for testing.
type mockSandboxManager struct {
	executeResult  sandbox.ExecutionResult
	executeErr     error
	runSkillResult sandbox.ExecutionResult
	runSkillErr    error
}

func (m *mockSandboxManager) ExecuteTool(ctx context.Context, toolID string, input map[string]any) (sandbox.ExecutionResult, error) {
	return m.executeResult, m.executeErr
}

func (m *mockSandboxManager) RunSkill(ctx context.Context, skillID string, input map[string]any) (sandbox.ExecutionResult, error) {
	return m.runSkillResult, m.runSkillErr
}

func TestNewSandboxServiceHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(&mockSandboxManager{}, logger)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestExecuteTool_Success(t *testing.T) {
	mock := &mockSandboxManager{
		executeResult: sandbox.ExecutionResult{
			Stdout:   "hello world",
			Stderr:   "",
			ExitCode: 0,
			Error:    "",
			Metrics:  `{"duration_ms": 42}`,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	params, _ := structpb.NewStruct(map[string]any{"arg1": "val1"})
	req := connect.NewRequest(&v1.ExecuteToolRequest{
		ToolId:      "test-tool",
		InputParams: params,
	})

	resp, err := h.ExecuteTool(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Msg.Result.Stdout != "hello world" {
		t.Errorf("expected stdout 'hello world', got %q", resp.Msg.Result.Stdout)
	}
	if resp.Msg.Result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", resp.Msg.Result.ExitCode)
	}
	if resp.Msg.Result.MetricsJson != `{"duration_ms": 42}` {
		t.Errorf("expected metrics, got %q", resp.Msg.Result.MetricsJson)
	}
}

func TestExecuteTool_Error(t *testing.T) {
	mock := &mockSandboxManager{
		executeErr: errors.New("sandbox crashed"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	req := connect.NewRequest(&v1.ExecuteToolRequest{
		ToolId: "test-tool",
	})

	_, err := h.ExecuteTool(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("expected CodeInternal, got %v", connect.CodeOf(err))
	}
}

func TestExecuteTool_NilInputParams(t *testing.T) {
	mock := &mockSandboxManager{
		executeResult: sandbox.ExecutionResult{ExitCode: 0},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	req := connect.NewRequest(&v1.ExecuteToolRequest{
		ToolId:      "test-tool",
		InputParams: nil,
	})

	resp, err := h.ExecuteTool(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error with nil params, got %v", err)
	}
	if resp.Msg.Result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", resp.Msg.Result.ExitCode)
	}
}

func TestExecuteTool_ErrorWithStderr(t *testing.T) {
	mock := &mockSandboxManager{
		executeResult: sandbox.ExecutionResult{
			Stdout:   "",
			Stderr:   "syntax error at line 5",
			ExitCode: 1,
			Error:    "compilation failed",
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	req := connect.NewRequest(&v1.ExecuteToolRequest{
		ToolId: "test-tool",
	})

	resp, err := h.ExecuteTool(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error (error in result), got %v", err)
	}
	if resp.Msg.Result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", resp.Msg.Result.ExitCode)
	}
	if resp.Msg.Result.Stderr != "syntax error at line 5" {
		t.Errorf("expected stderr, got %q", resp.Msg.Result.Stderr)
	}
}

func TestRunSkill_Success(t *testing.T) {
	mock := &mockSandboxManager{
		runSkillResult: sandbox.ExecutionResult{
			Stdout:   "skill output",
			ExitCode: 0,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	params, _ := structpb.NewStruct(map[string]any{"x": 1})
	req := connect.NewRequest(&v1.RunSkillRequest{
		SkillId:     "skill-1",
		InputParams: params,
	})

	resp, err := h.RunSkill(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Msg.Result.Stdout != "skill output" {
		t.Errorf("expected stdout 'skill output', got %q", resp.Msg.Result.Stdout)
	}
}

func TestRunSkill_Error(t *testing.T) {
	mock := &mockSandboxManager{
		runSkillErr: errors.New("skill not found"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	req := connect.NewRequest(&v1.RunSkillRequest{
		SkillId: "nonexistent",
	})

	_, err := h.RunSkill(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("expected CodeInternal, got %v", connect.CodeOf(err))
	}
}

func TestRunSkill_NilInputParams(t *testing.T) {
	mock := &mockSandboxManager{
		runSkillResult: sandbox.ExecutionResult{ExitCode: 0},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewSandboxServiceHandler(mock, logger)

	req := connect.NewRequest(&v1.RunSkillRequest{
		SkillId:     "skill-1",
		InputParams: nil,
	})

	resp, err := h.RunSkill(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error with nil params, got %v", err)
	}
	if resp.Msg.Result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", resp.Msg.Result.ExitCode)
	}
}
