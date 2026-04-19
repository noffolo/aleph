package handler

import (
	"context"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"connectrpc.com/connect"
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
	rows, err := h.metaRepo.DB().Query("SELECT id, name, description, code FROM system_tools")
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	defer rows.Close()

	var tools []*v1.Tool
	for rows.Next() {
		var t v1.Tool
		rows.Scan(&t.Id, &t.Name, &t.Description, &t.Code)
		tools = append(tools, &t)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: tools}), nil
}

func (h *ToolHandler) CreateTool(
	ctx context.Context,
	req *connect.Request[v1.CreateToolRequest],
) (*connect.Response[v1.CreateToolResponse], error) {
	t := req.Msg.Tool
	_, err := h.metaRepo.DB().Exec(
		"INSERT INTO system_tools (id, name, description, code) VALUES ($1, $2, $3, $4)",
		t.Id, t.Name, t.Description, t.Code,
	)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.CreateToolResponse{Tool: t}), nil
}

func (h *ToolHandler) DeleteTool(
	ctx context.Context,
	req *connect.Request[v1.DeleteToolRequest],
) (*connect.Response[v1.DeleteToolResponse], error) {
	_, err := h.metaRepo.DB().Exec("DELETE FROM system_tools WHERE id = $1", req.Msg.Id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteToolResponse{Success: true}), nil
}
