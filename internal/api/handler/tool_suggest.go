package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
)

// ToolSuggestHandler orchestrates the tool suggestion workflow:
// MCP discovery → sandbox verification → adaptation pipeline → user approval → registration.
type ToolSuggestHandler struct {
	discoveryEngine *mcp.DiscoveryEngine
	pipeline        *adaptation.Pipeline
	mcpServerURIs   []string
	mu              sync.Mutex
	pending         map[string]*pendingSuggestion
	nextID          atomic.Int64
}

type pendingSuggestion struct {
	ToolDef   mcp.ToolDefinition
	Result    *adaptation.AdaptationResult
	CreatedAt time.Time
}

// NewToolSuggestHandler creates a handler that chains MCP discovery, sandbox
// verification, adaptation, and registration into a single suggest workflow.
func NewToolSuggestHandler(
	discoveryEngine *mcp.DiscoveryEngine,
	pipeline *adaptation.Pipeline,
	mcpServerURIs []string,
) *ToolSuggestHandler {
	return &ToolSuggestHandler{
		discoveryEngine: discoveryEngine,
		pipeline:        pipeline,
		mcpServerURIs:   mcpServerURIs,
		pending:         make(map[string]*pendingSuggestion),
	}
}

// ---------------------------------------------------------------------------
// Request/Response types
// ---------------------------------------------------------------------------

type suggestRequestBody struct {
	Name      string `json:"name"`
	ServerURL string `json:"server_url,omitempty"`
}

type approveRequestBody struct {
	Name         string `json:"name"`
	SuggestionID string `json:"suggestion_id"`
}

type stageResultJSON struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type analysisJSON struct {
	Language     string   `json:"language"`
	Dependencies []string `json:"dependencies"`
	Complexity   int      `json:"complexity"`
	TemplateType string   `json:"template_type"`
	HasTests     bool     `json:"has_tests"`
	Issues       []string `json:"issues"`
}

type suggestResponse struct {
	SuggestionID string            `json:"suggestion_id,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Version      string            `json:"version"`
	Stages       []stageResultJSON `json:"stages,omitempty"`
	Analysis     *analysisJSON     `json:"analysis,omitempty"`
	AdaptedCode  string            `json:"adapted_code,omitempty"`
	Approved     bool              `json:"approved"`
	Error        string            `json:"error,omitempty"`
}

type approveResponse struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Version string `json:"version"`
	ToolID  string `json:"tool_id"`
	Error   string `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// POST /api/v1/tools/suggest
// ---------------------------------------------------------------------------

// HandleSuggest runs MCP discovery → verification → adaptation for the named
// tool. Returns a suggestion_id that the caller can use to approve registration.
func (h *ToolSuggestHandler) HandleSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		writeError(w, "failed to read request", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	var req suggestRequestBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, suggestResponse{
			Error: fmt.Sprintf("invalid JSON: %v", err),
		})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, suggestResponse{
			Error: "name is required",
		})
		return
	}

	// Step 1: Discover the tool from MCP servers
	toolDef, err := h.discoverMCPTool(r.Context(), req.Name, req.ServerURL)
	if err != nil {
		writeJSON(w, http.StatusNotFound, suggestResponse{
			Name:  req.Name,
			Error: fmt.Sprintf("tool discovery failed: %v", err),
		})
		return
	}

	// Step 2: Run adaptation pipeline (verification → analysis → adaptation → testing)
	result, err := h.pipeline.RunSuggestion(r.Context(), toolDef)
	if err != nil {
		resp := suggestResponse{
			Name:  toolDef.Name,
			Error: fmt.Sprintf("pipeline error: %v", err),
		}
		if result != nil {
			resp.Stages = toStageResultJSON(result.Stages)
		}
		writeJSON(w, http.StatusInternalServerError, resp)
		return
	}

	// Check if any stage failed
	for _, s := range result.Stages {
		if !s.Passed {
			writeJSON(w, http.StatusUnprocessableEntity, suggestResponse{
				Name:   toolDef.Name,
				Error:  fmt.Sprintf("stage %s failed: %s", s.Name, s.Message),
				Stages: toStageResultJSON(result.Stages),
			})
			return
		}
	}

	// Step 3: Store pending suggestion for approval
	suggestionID := h.storePending(r.Context(), toolDef, result)

	resp := suggestResponse{
		SuggestionID: suggestionID,
		Name:         toolDef.Name,
		Description:  toolDef.Description,
		Version:      result.Version,
		Stages:       toStageResultJSON(result.Stages),
		AdaptedCode:  result.AdaptedCode,
		Approved:     false,
	}

	if result.Analysis.Language != "" {
		resp.Analysis = &analysisJSON{
			Language:     result.Analysis.Language,
			Dependencies: result.Analysis.Dependencies,
			Complexity:   result.Analysis.Complexity,
			TemplateType: string(result.Analysis.TemplateType),
			HasTests:     result.Analysis.HasTests,
			Issues:       result.Analysis.Issues,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// POST /api/v1/tools/suggest/approve
// ---------------------------------------------------------------------------

// HandleApprove registers a previously suggested tool after user approval.
func (h *ToolSuggestHandler) HandleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		writeError(w, "failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req approveRequestBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, approveResponse{
			Error: fmt.Sprintf("invalid JSON: %v", err),
		})
		return
	}
	if req.Name == "" || req.SuggestionID == "" {
		writeJSON(w, http.StatusBadRequest, approveResponse{
			Error: "name and suggestion_id are required",
		})
		return
	}

	// Look up pending suggestion
	h.mu.Lock()
	pending, ok := h.pending[req.SuggestionID]
	if !ok {
		h.mu.Unlock()
		writeJSON(w, http.StatusNotFound, approveResponse{
			Name:  req.Name,
			Error: "suggestion not found or expired",
		})
		return
	}
	delete(h.pending, req.SuggestionID)
	h.mu.Unlock()

	if pending.ToolDef.Name != req.Name {
		writeJSON(w, http.StatusBadRequest, approveResponse{
			Name:  req.Name,
			Error: fmt.Sprintf("suggestion name mismatch: got %q, expected %q", pending.ToolDef.Name, req.Name),
		})
		return
	}

	// Register the tool via the pipeline's registration stage
	if err := h.pipeline.RegisterFromSuggestion(r.Context(), pending.Result); err != nil {
		writeJSON(w, http.StatusInternalServerError, approveResponse{
			Name:  req.Name,
			Error: fmt.Sprintf("registration failed: %v", err),
		})
		return
	}

	toolID := req.Name + "-adapted"

	writeJSON(w, http.StatusOK, approveResponse{
		Success: true,
		Name:    req.Name,
		Version: pending.Result.Version,
		ToolID:  toolID,
	})
}

// ---------------------------------------------------------------------------
// Helper methods
// ---------------------------------------------------------------------------

// discoverMCPTool searches configured MCP servers (or a specific URL) for a
// tool matching the given name.
func (h *ToolSuggestHandler) discoverMCPTool(ctx context.Context, name, serverURL string) (mcp.ToolDefinition, error) {
	urls := h.mcpServerURIs
	if serverURL != "" {
		urls = []string{serverURL}
	}

	if len(urls) == 0 {
		return mcp.ToolDefinition{}, fmt.Errorf("no MCP servers configured and no server_url provided")
	}

	for _, raw := range urls {
		// Normalize URL — support both mcp:// URIs and direct http:// URLs
		discoverURL := raw
		scheme, host, port, path, parseErr := mcp.ParseMCPURI(raw)
		if parseErr == nil && scheme == "mcp" {
			discoverURL = fmt.Sprintf("http://%s:%s%s", host, port, path)
		}

		tools, err := h.discoveryEngine.DiscoverSchemas(ctx, discoverURL)
		if err != nil {
			continue
		}

		for _, t := range tools {
			if t.Name == name {
				return t, nil
			}
		}
	}

	return mcp.ToolDefinition{}, fmt.Errorf("tool %q not found on any MCP server", name)
}

// storePending saves a suggestion result for later approval.
// The provided ctx allows the cleanup goroutine to be interrupted on shutdown.
func (h *ToolSuggestHandler) storePending(ctx context.Context, toolDef mcp.ToolDefinition, result *adaptation.AdaptationResult) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := int(h.nextID.Add(1))
	suggestionID := fmt.Sprintf("sug-%d", id)

	h.pending[suggestionID] = &pendingSuggestion{
		ToolDef:   toolDef,
		Result:    result,
		CreatedAt: time.Now(),
	}

	// Schedule cleanup after 5 minutes — interruptible via ctx cancellation.
	go func(id string, ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("tool_suggest cleanup goroutine panic", "suggestionID", id, "recover", r)
			}
		}()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		select {
		case <-ticker.C:
			h.mu.Lock()
			delete(h.pending, id)
			h.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}(suggestionID, ctx)

	return suggestionID
}

func toStageResultJSON(stages []adaptation.StageResult) []stageResultJSON {
	out := make([]stageResultJSON, len(stages))
	for i, s := range stages {
		out[i] = stageResultJSON{
			Name:    s.Name,
			Passed:  s.Passed,
			Message: s.Message,
		}
	}
	return out
}

// Ensure *ToolSuggestHandler implements http.Handler for the suggest endpoint.
var (
	_ http.Handler = (*ToolSuggestHandler)(nil)
)

// ServeHTTP routes to the appropriate handler method based on URL path.
func (h *ToolSuggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/tools/suggest":
		h.HandleSuggest(w, r)
	case "/api/v1/tools/suggest/approve":
		h.HandleApprove(w, r)
	default:
		writeError(w, "not found", http.StatusNotFound)
	}
}
