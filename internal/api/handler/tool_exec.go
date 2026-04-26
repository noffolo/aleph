package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/tools"
	"github.com/ff3300/aleph-v2/internal/tools/finance"
	"github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
)

// ToolExecuteHandler provides HTTP handlers for listing and executing tools
// across the finance, OSINT, and human ecosystems categories.
type ToolExecuteHandler struct {
	metaRepo     *repository.MetadataRepository
	shadowbroker *osint.Shadowbroker
	duckdbLayer  *humanecosystems.DuckDBLayer
	registry     *tools.ToolRegistry
}

// NewToolExecuteHandler creates a new ToolExecuteHandler.
// shadowbroker and duckdbLayer may be nil for graceful degradation.
func NewToolExecuteHandler(
	metaRepo *repository.MetadataRepository,
	shadowbroker *osint.Shadowbroker,
	duckdbLayer *humanecosystems.DuckDBLayer,
) *ToolExecuteHandler {
	return &ToolExecuteHandler{
		metaRepo:     metaRepo,
		shadowbroker: shadowbroker,
		duckdbLayer:  duckdbLayer,
	}
}

// SetRegistry attaches a ToolRegistry to the handler. Called after construction
// because the registry is populated during app initialization.
func (h *ToolExecuteHandler) SetRegistry(reg *tools.ToolRegistry) {
	h.registry = reg
}

// Registry returns the attached ToolRegistry, initializing a default one if nil.
func (h *ToolExecuteHandler) Registry() *tools.ToolRegistry {
	if h.registry == nil {
		h.registry = populateDefaultRegistry(h.shadowbroker, h.duckdbLayer)
	}
	return h.registry
}

// populateDefaultRegistry creates and populates the registry with all known
// tools from finance, OSINT, and human-ecosystems packages.
func populateDefaultRegistry(broker *osint.Shadowbroker, dbl *humanecosystems.DuckDBLayer) *tools.ToolRegistry {
	reg := tools.NewToolRegistry()

	// Finance tools
	financeDefs := []tools.ToolDefinition{
		tools.FinanceToolDef("finance_prophet_forecast",
			"Time-series forecasting using SMA/linear regression (no Python Prophet dependency)",
			finance.NewProphetForecastTool().Execute),
		tools.FinanceToolDef("finance_openbb_market_data",
			"Market data via HTTP gateway with retry logic and structured mock fallback",
			finance.NewOpenBBMarketDataTool().Execute),
		tools.FinanceToolDef("finance_sentiment_analysis",
			"Financial sentiment analysis using NLP adapter or keyword-based fallback",
			finance.NewSentimentAnalysisFinTool().Execute),
	}
	for _, def := range financeDefs {
		if err := reg.Register(def); err != nil {
			slog.Warn("failed to register finance tool", "name", def.Name, "error", err)
		}
	}

	// OSINT tools (if broker is available)
	if broker != nil {
		osintDefs := []tools.ToolDefinition{
			tools.OSINTToolDef("osint_region_dossier",
				"Region dataset dossier from Shadowbroker (beta) | is_synthetic=true | privacy-preserving",
				osint.NewRegionDossierTool(broker).Execute),
			tools.OSINTToolDef("osint_threat_level",
				"Threat level assessment from Shadowbroker data (beta) | is_synthetic=true",
				osint.NewThreatLevelTool(broker).Execute),
			tools.OSINTToolDef("osint_vessel_tracking",
				"Vessel tracking intelligence from Shadowbroker (beta) | is_synthetic=true",
				osint.NewVesselTrackingTool(broker).Execute),
			tools.OSINTToolDef("osint_flight_tracking",
				"Flight tracking intelligence from Shadowbroker (beta) | is_synthetic=true",
				osint.NewFlightTrackingTool(broker).Execute),
			tools.OSINTToolDef("osint_correlation_alerts",
				"Cross-source OSINT correlation for threat alerts (beta) | is_synthetic=true",
				osint.NewCorrelationAlertsTool(broker).Execute),
		}
		for _, def := range osintDefs {
			if err := reg.Register(def); err != nil {
				slog.Warn("failed to register osint tool", "name", def.Name, "error", err)
			}
		}
	} else {
		slog.Warn("shadowbroker is nil, skipping OSINT tool registration")
	}

	// Human ecosystems tools
	for _, t := range humanecosystems.ListTools(dbl) {
		if err := reg.Register(tools.HEToolDef(t)); err != nil {
			slog.Warn("failed to register human-ecosystems tool", "name", t.Name(), "error", err)
		}
	}

	return reg
}

// HandleListCategories returns the available tool categories.
// GET /api/v1/tools/categories
func (h *ToolExecuteHandler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, h.Registry().Categories())
}

// HandleListToolsByCategory returns all tools in the given category.
// GET /api/v1/tools/execute/{category}/{name}  (GET ignores name, lists category)
func (h *ToolExecuteHandler) HandleListToolsByCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.PathValue("category")
	if category == "" {
		writeError(w, "category is required", http.StatusBadRequest)
		return
	}

	tools := h.Registry().List(category)
	if len(tools) == 0 {
		writeError(w, "unknown category: "+category, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, tools)
}

// HandleExecuteTool executes a specific tool by category and name.
// POST /api/v1/tools/execute/{category}/{name}
// Body: JSON object with tool-specific arguments.
func (h *ToolExecuteHandler) HandleExecuteTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.PathValue("category")
	name := r.PathValue("name")

	if category == "" || name == "" {
		writeError(w, "category and name are required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var argsMap map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &argsMap); err != nil {
			writeError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate category exists before looking up the tool
	catTools := h.Registry().List(category)
	if len(catTools) == 0 {
		writeError(w, "unknown category: "+category, http.StatusNotFound)
		return
	}

	ctx := r.Context()
	result, err := h.Registry().ExecuteContext(ctx, category, name, argsMap)
	if err != nil {
		slog.Error("tool execution failed", "category", category, "name", name, "error", err)
		writeError(w, "execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ServeHTTP dispatches to the appropriate handler based on path and method.
// Supports:
//
//	GET  /api/v1/tools/execute/{category}/{name} → list tools in category
//	POST /api/v1/tools/execute/{category}/{name} → execute tool
func (h *ToolExecuteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleListToolsByCategory(w, r)
	case http.MethodPost:
		h.HandleExecuteTool(w, r)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleCallTool executes a tool by qualified name (e.g. "finance.prophet_forecast").
// POST /api/v1/tools/call
// Body: {"tool": "category.name", "params": {...}}
//   or: {"category": "finance", "name": "prophet_forecast", "params": {...}}
func (h *ToolExecuteHandler) HandleCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Tool     string         `json:"tool"`
		Category string         `json:"category"`
		Name     string         `json:"name"`
		Params   map[string]any `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	category, name := req.Category, req.Name
	if req.Tool != "" {
		parts := strings.SplitN(req.Tool, ".", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			writeError(w, `invalid tool format, use "category.name"`, http.StatusBadRequest)
			return
		}
		category = parts[0]
		name = parts[1]
	}
	if category == "" || name == "" {
		writeError(w, `"tool" (e.g. "finance.prophet_forecast") or "category"+"name" required`, http.StatusBadRequest)
		return
	}

	catTools := h.Registry().List(category)
	if len(catTools) == 0 {
		writeError(w, "unknown category: "+category, http.StatusNotFound)
		return
	}

	ctx := r.Context()
	result, err := h.Registry().ExecuteContext(ctx, category, name, req.Params)
	if err != nil {
		slog.Error("tool call failed", "category", category, "name", name, "error", err)
		writeError(w, "execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/tools/register
func (h *ToolExecuteHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var registered []string
	var failed []string

	for _, t := range h.Registry().List("") {
		err := h.metaRepo.CreateTool(&repository.ToolRecord{
			ID:           t.Category + "_" + t.Name,
			Name:         t.Name,
			Description:  t.Description,
			Code:         "",
			Category:     t.Category,
			Version:      "1.0.0",
			HealthStatus: "unknown",
			SourceType:   "package",
		})
		if err != nil {
			slog.Error("failed to register tool", "category", t.Category, "name", t.Name, "error", err)
			failed = append(failed, t.Category+"/"+t.Name)
		} else {
			registered = append(registered, t.Category+"/"+t.Name)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"registered": registered,
		"failed":     failed,
		"count":      len(registered),
	})
}
