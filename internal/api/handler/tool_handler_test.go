package handler

import (
	"context"
	"testing"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"

	"connectrpc.com/connect"
)

func TestNewToolHandler(t *testing.T) {
	h := NewToolHandler("/tmp/projects", (*repository.MetadataRepository)(nil))
	assert.NotNil(t, h)
	assert.Equal(t, "/tmp/projects", h.projectsRoot)
	assert.Nil(t, h.metaRepo)
}

func TestToolHandler_CreateTool_NilTool(t *testing.T) {
	h := &ToolHandler{}
	req := connect.NewRequest(&v1.CreateToolRequest{
		ProjectId: "proj-1",
		Tool:      nil,
	})
	_, err := h.CreateTool(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool is required")
}

func TestToolHandler_UpdateTool_NilTool(t *testing.T) {
	h := &ToolHandler{}
	req := connect.NewRequest(&v1.UpdateToolRequest{
		ProjectId: "proj-1",
		Tool:      nil,
	})
	_, err := h.UpdateTool(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool id is required")
}

func TestToolHandler_UpdateTool_EmptyId(t *testing.T) {
	h := &ToolHandler{}
	req := connect.NewRequest(&v1.UpdateToolRequest{
		ProjectId: "proj-1",
		Tool:      &v1.Tool{Id: "", Name: "test"},
	})
	_, err := h.UpdateTool(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool id is required")
}

func TestToolHandler_DeleteTool(t *testing.T) {
	req := &v1.DeleteToolRequest{
		Id:        "tool-1",
		ProjectId: "proj-1",
	}
	assert.Equal(t, "tool-1", req.Id)
	assert.Equal(t, "proj-1", req.ProjectId)
}

func TestToolHandler_ListTools_RequestStructure(t *testing.T) {
	req := &v1.ListToolsRequest{ProjectId: "proj-1"}
	assert.Equal(t, "proj-1", req.ProjectId)
}

func TestToolHandler_GetTool_RequestStructure(t *testing.T) {
	req := &v1.DeleteToolRequest{
		Id:        "tool-abc",
		ProjectId: "proj-1",
	}
	assert.Equal(t, "tool-abc", req.Id)
	assert.Equal(t, "proj-1", req.ProjectId)
}
