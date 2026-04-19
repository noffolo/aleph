package handler

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
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
	return connect.NewResponse(&v1.ExecuteToolResponse{}), nil
}

func (h *SandboxServiceHandler) RunSkill(ctx context.Context, req *connect.Request[v1.RunSkillRequest]) (*connect.Response[v1.RunSkillResponse], error) {
	return connect.NewResponse(&v1.RunSkillResponse{}), nil
}
