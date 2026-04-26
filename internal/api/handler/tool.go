package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

// ─── Raw HTTP Handlers ───────────────────────────────────────────────────────

// ServeHTTP dispatches tool routes based on request path suffix.
//   - /api/v1/tools/intelligence → tool intel listing
//   - /api/v1/tools/recommendations → tool recommendations
//   - /api/v1/tools/health → tool health status
//   - /api/v1/tools → default listing
func (h *ToolHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch {
	case strings.HasSuffix(r.URL.Path, "/intelligence"):
		h.HandleIntelligence(w, r)
	case strings.HasSuffix(r.URL.Path, "/recommendations"):
		h.HandleRecommendations(w, r)
	case strings.HasSuffix(r.URL.Path, "/health"):
		h.HandleHealth(w, r)
	default:
		h.HandleListAll(w, r)
	}
}

// HandleIntelligence returns tool intelligence as a JSON array.
func (h *ToolHandler) HandleIntelligence(w http.ResponseWriter, r *http.Request) {
	tools, err := h.metaRepo.ListTools()
	if err != nil {
		writeError(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, tools)
}

// HandleRecommendations returns tool recommendations as a JSON array.
func (h *ToolHandler) HandleRecommendations(w http.ResponseWriter, r *http.Request) {
	tools, err := h.metaRepo.ListTools()
	if err != nil {
		writeError(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, tools)
}

// HandleHealth returns health status for all tools.
func (h *ToolHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	tools, err := h.metaRepo.ListTools()
	if err != nil {
		writeError(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	type toolHealth struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		HealthStatus string `json:"health_status"`
	}
	result := make([]toolHealth, len(tools))
	for i, t := range tools {
		result[i] = toolHealth{ID: t.ID, Name: t.Name, HealthStatus: t.HealthStatus}
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleVerify verifies a tool. POST /api/v1/tools/verify with {tool_id}.
func (h *ToolHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ToolID string `json:"tool_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ToolID == "" {
		writeError(w, "tool_id is required", http.StatusBadRequest)
		return
	}
	// Verify the tool exists in the repository
	_, err := h.metaRepo.GetToolCode(r.Context(), req.ToolID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"tool_id": req.ToolID,
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tool_id": req.ToolID,
		"valid":   true,
	})
}

// HandleHealthHistory returns health history for a tool.
// GET /api/v1/tools/{id} or POST with {tool_id}
func (h *ToolHandler) HandleHealthHistory(w http.ResponseWriter, r *http.Request) {
	var toolID string
	if r.Method == http.MethodPost {
		var req struct {
			ToolID string `json:"tool_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		toolID = req.ToolID
	} else {
		// Extract tool ID from path: /api/v1/tools/{id}
		// Use path segment after /api/v1/tools/
		toolID = r.PathValue("id")
	}

	if toolID == "" {
		writeError(w, "tool_id is required", http.StatusBadRequest)
		return
	}

	// Get current tool data
	tools, err := h.metaRepo.ListTools()
	if err != nil {
		writeError(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for _, t := range tools {
		if t.ID == toolID {
			writeJSON(w, http.StatusOK, map[string]any{
				"tool_id":         t.ID,
				"name":            t.Name,
				"health_status":   t.HealthStatus,
				"version":         t.Version,
				"source_type":     t.SourceType,
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tool_id":       toolID,
		"health_status": "unknown",
		"error":         "tool not found",
	})
}

// HandleListAll lists all tools in JSON format.
func (h *ToolHandler) HandleListAll(w http.ResponseWriter, r *http.Request) {
	tools, err := h.metaRepo.ListTools()
	if err != nil {
		writeError(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, tools)
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