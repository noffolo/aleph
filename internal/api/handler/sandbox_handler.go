package handler

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/errors"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

type SandboxServiceHandler struct {
	sandboxMgr sandbox.SandboxManager
	logger     *slog.Logger
}

func NewSandboxServiceHandler(sbMgr sandbox.SandboxManager, logger *slog.Logger) *SandboxServiceHandler {
	return &SandboxServiceHandler{sandboxMgr: sbMgr, logger: logger}
}

func (h *SandboxServiceHandler) ExecuteTool(ctx context.Context, req *connect.Request[v1.ExecuteToolRequest]) (*connect.Response[v1.ExecuteToolResponse], error) {
	toolID := req.Msg.GetToolId()
	var inputMap map[string]interface{}
	if ip := req.Msg.GetInputParams(); ip != nil {
		inputMap = ip.AsMap()
	} else {
		inputMap = map[string]interface{}{}
	}

	result, err := h.sandboxMgr.ExecuteTool(ctx, toolID, inputMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.NewAPIErrorWithMeta(
			errors.ErrInternal, "tool execution failed", err,
			"sandbox", "execute", true, 0,
		))
	}

	ec := result.ExitCode
	if ec < -1<<31 || ec > 1<<31-1 {
		return nil, connect.NewError(connect.CodeInternal, errors.NewAPIErrorWithMeta(
			errors.ErrInternal, "exit code out of range", nil,
			"sandbox", "execute", true, 0,
		))
	}
	pbResult := &v1.ExecutionResult{
		Stdout:      result.Stdout,
		Stderr:      result.Stderr,
		ExitCode:    int32(ec),
		Error:       result.Error,
		MetricsJson: result.Metrics,
	}
	return connect.NewResponse(&v1.ExecuteToolResponse{Result: pbResult}), nil
}

func (h *SandboxServiceHandler) RunSkill(ctx context.Context, req *connect.Request[v1.RunSkillRequest]) (*connect.Response[v1.RunSkillResponse], error) {
	skillID := req.Msg.GetSkillId()
	var inputMap map[string]interface{}
	if ip := req.Msg.GetInputParams(); ip != nil {
		inputMap = ip.AsMap()
	} else {
		inputMap = map[string]interface{}{}
	}

	result, err := h.sandboxMgr.RunSkill(ctx, skillID, inputMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.NewAPIErrorWithMeta(
			errors.ErrInternal, "skill execution failed", err,
			"sandbox", "execute", true, 0,
		))
	}

	ec := result.ExitCode
	if ec < -1<<31 || ec > 1<<31-1 {
		return nil, connect.NewError(connect.CodeInternal, errors.NewAPIErrorWithMeta(
			errors.ErrInternal, "exit code out of range", nil,
			"sandbox", "execute", true, 0,
		))
	}
	pbResult := &v1.ExecutionResult{
		Stdout:      result.Stdout,
		Stderr:      result.Stderr,
		ExitCode:    int32(ec),
		Error:       result.Error,
		MetricsJson: result.Metrics,
	}
	return connect.NewResponse(&v1.RunSkillResponse{Result: pbResult}), nil
}
