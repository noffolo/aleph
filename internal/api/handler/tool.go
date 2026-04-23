package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type ToolHandler struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
}

func NewToolHandler(projectsRoot string, metaRepo *repository.MetadataRepository) *ToolHandler {
	return &ToolHandler{projectsRoot: projectsRoot, metaRepo: metaRepo}
}

func (h *ToolHandler) ListTools(
	ctx context.Context,
	req *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	tools, err := h.metaRepo.ListTools()
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	var result []*v1.Tool
	for _, t := range tools {
		result = append(result, &v1.Tool{Id: t.ID, Name: t.Name, Description: t.Description, Code: t.Code})
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: result}), nil
}

func (h *ToolHandler) CreateTool(
	ctx context.Context,
	req *connect.Request[v1.CreateToolRequest],
) (*connect.Response[v1.CreateToolResponse], error) {
	t := req.Msg.Tool
	err := h.metaRepo.CreateTool(&repository.ToolRecord{ID: t.Id, Name: t.Name, Description: t.Description, Code: t.Code})
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.CreateToolResponse{Tool: t}), nil
}

func (h *ToolHandler) DeleteTool(
	ctx context.Context,
	req *connect.Request[v1.DeleteToolRequest],
) (*connect.Response[v1.DeleteToolResponse], error) {
	err := h.metaRepo.DeleteTool(req.Msg.Id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteToolResponse{Success: true}), nil
}