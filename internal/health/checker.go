package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// ToolChecker defines how to check if a tool is healthy.
type ToolChecker interface {
	CheckToolHealth(ctx context.Context, toolID string) HealthRecord
}

// MCPHealthProvider provides health status for MCP STDIO tool subprocesses.
// Implementations should return the current health snapshot of all monitored
// MCP tools. This interface decouples the health checker from the MCP package.
type MCPHealthProvider interface {
	// Status returns a snapshot of current tool health states.
	Status() []MCPToolHealth
}

// MCPToolHealth mirrors mcp.ToolHealth for the health checker without
// importing the mcp package directly.
type MCPToolHealth struct {
	Name         string `json:"name"`
	Status       string `json:"status"` // "up", "down", "unknown"
	LastPing     string `json:"last_ping,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	RestartCount int    `json:"restart_count"`
}

// HealthChecker runs periodic health checks on tools.
type HealthChecker struct {
	logger     *slog.Logger
	metaRepo   *repository.MetadataRepository
	history    *HistoryStore
	checker    ToolChecker
	interval   time.Duration
	ctx        context.Context
	cancel      context.CancelFunc
	alertCount int          // number of consecutive failures before alerting
	mcpHealth  MCPHealthProvider // optional MCP subprocess health monitor
}

// NewHealthChecker creates a new periodic health checker.
// The parentCtx is used as the initial context for health check operations
// until Start(parentCtx) replaces it with the actual loop context.
func NewHealthChecker(parentCtx context.Context, logger *slog.Logger, metaRepo *repository.MetadataRepository, opts ...HealthCheckerOption) *HealthChecker {
	cfg := HealthCheckerConfig{
		Interval:   5 * time.Minute,
		HistoryLen: 10,
		AlertCount: 3,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// The initial ctx/cancel pair ensures the struct is always valid.
	// Start(parentCtx) replaces this context when the checker is launched.
	ctx, cancel := context.WithCancel(parentCtx)
	history := NewHistoryStore(cfg.HistoryLen)

	builtinChecker := NewBuiltinChecker(metaRepo)

	return &HealthChecker{
		logger:     logger,
		metaRepo:   metaRepo,
		history:    history,
		checker:    builtinChecker,
		interval:   cfg.Interval,
		ctx:        ctx,
		cancel:     cancel,
		alertCount: cfg.AlertCount,
		mcpHealth:  cfg.MCPHealth,
	}
}

// HealthCheckerConfig holds configuration for the health checker.
type HealthCheckerConfig struct {
	Interval   time.Duration
	HistoryLen int
	AlertCount int
	MCPHealth  MCPHealthProvider
}

// HealthCheckerOption is a functional option for HealthChecker.
type HealthCheckerOption func(*HealthCheckerConfig)

// WithInterval sets the health check interval.
func WithInterval(d time.Duration) HealthCheckerOption {
	return func(cfg *HealthCheckerConfig) { cfg.Interval = d }
}

// WithHistoryLen sets the maximum number of health records per tool.
func WithHistoryLen(n int) HealthCheckerOption {
	return func(cfg *HealthCheckerConfig) { cfg.HistoryLen = n }
}

// WithAlertCount sets the number of consecutive failures before alerting.
func WithAlertCount(n int) HealthCheckerOption {
	return func(cfg *HealthCheckerConfig) { cfg.AlertCount = n }
}

// WithMCPHealth attaches an MCP STDIO tool subprocess health monitor.
// The monitor's status is checked alongside builtin tool health checks.
func WithMCPHealth(m MCPHealthProvider) HealthCheckerOption {
	return func(cfg *HealthCheckerConfig) { cfg.MCPHealth = m }
}

// Start begins the periodic health check loop.
func (hc *HealthChecker) Start(parentCtx context.Context) {
	hc.ctx, hc.cancel = context.WithCancel(parentCtx)
	go hc.run()
	hc.logger.Info("health checker started", "interval", hc.interval)
}

// Stop cancels the health check loop.
func (hc *HealthChecker) Stop() {
	hc.cancel()
	hc.logger.Info("health checker stopped")
}

// GetHistory returns health history for a tool.
func (hc *HealthChecker) GetHistory(toolID string) []HealthRecord {
	return hc.history.GetHistory(toolID)
}

// GetAllHistory returns a map of all tool health histories.
func (hc *HealthChecker) GetAllHistory() map[string][]HealthRecord {
	toolIDs := hc.history.GetToolIDs()
	result := make(map[string][]HealthRecord)
	for _, id := range toolIDs {
		result[id] = hc.history.GetHistory(id)
	}
	return result
}

func (hc *HealthChecker) ConsecutiveFailures(toolID string) int {
	return hc.history.ConsecutiveFailures(toolID)
}

func (hc *HealthChecker) GetLatestStatus(toolID string) string {
	records := hc.history.GetHistory(toolID)
	if len(records) == 0 {
		return StatusUnknown
	}
	return string(records[len(records)-1].Status)
}

func (hc *HealthChecker) run() {
	// Run initial check immediately
	hc.checkAll()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.checkAll()
		}
	}
}

func (hc *HealthChecker) checkAll() {
	// Check MCP subprocess health if configured
	if hc.mcpHealth != nil {
		hc.checkMCPHealth()
	}

	tools, err := hc.metaRepo.ListTools()
	if err != nil {
		hc.logger.Error("failed to list tools for health check", "error", err)
		return
	}

	if len(tools) == 0 {
		return
	}

	for _, tool := range tools {
		if tool.ID == "" {
			continue
		}

		record := hc.checker.CheckToolHealth(hc.ctx, tool.ID)

		// Update status in repository
		if err := hc.metaRepo.UpdateHealthStatus(tool.ID, record.Status); err != nil {
			hc.logger.Warn("failed to update tool health status", "tool_id", tool.ID, "error", err)
		}

		// Record in history
		hc.history.Add(tool.ID, record)

		// Check for consecutive failures
		consecutive := hc.history.ConsecutiveFailures(tool.ID)
		if consecutive >= hc.alertCount {
			hc.logger.Warn("tool health alert",
				"tool_id", tool.ID,
				"consecutive_failures", consecutive,
				"alert_threshold", hc.alertCount,
				"last_error", record.Error,
			)
		}

		hc.logger.Debug("tool health checked",
			"tool_id", tool.ID,
			"status", record.Status,
			"response_time", record.ResponseTime,
		)
	}
}

// checkMCPHealth queries the MCP subprocess health monitor and records
// health history entries for each monitored MCP tool.
func (hc *HealthChecker) checkMCPHealth() {
	mcpStatuses := hc.mcpHealth.Status()
	for _, mcpTool := range mcpStatuses {
		if mcpTool.Name == "" {
			continue
		}

		status := StatusHealthy
		switch mcpTool.Status {
		case "down":
			status = StatusDown
		case "unknown":
			status = StatusUnknown
		case "up":
			status = StatusHealthy
		default:
			status = StatusUnknown
		}

		record := HealthRecord{
			ToolID:       mcpTool.Name,
			Status:       status,
			CheckedAt:    time.Now(),
			ResponseTime: mcpTool.LastPing,
			Error:        mcpTool.LastError,
		}

		hc.history.Add(mcpTool.Name, record)

		// Alert on MCP subprocess failures
		if status == StatusDown {
			consecutive := hc.history.ConsecutiveFailures(mcpTool.Name)
			if consecutive >= hc.alertCount && consecutive%hc.alertCount == 0 {
				hc.logger.Warn("MCP subprocess health alert",
					"tool_name", mcpTool.Name,
					"consecutive_failures", consecutive,
					"alert_threshold", hc.alertCount,
					"last_error", mcpTool.LastError,
					"restart_count", mcpTool.RestartCount,
				)
			}
		}
	}
}

// BuiltinChecker checks health of builtin tools (always healthy if code exists).
type BuiltinChecker struct {
	metaRepo *repository.MetadataRepository
}

// NewBuiltinChecker creates a checker for builtin tools.
func NewBuiltinChecker(metaRepo *repository.MetadataRepository) *BuiltinChecker {
	return &BuiltinChecker{metaRepo: metaRepo}
}

// CheckToolHealth checks if a builtin tool's code exists.
func (c *BuiltinChecker) CheckToolHealth(ctx context.Context, toolID string) HealthRecord {
	start := time.Now()
	code, err := c.metaRepo.GetToolCode(ctx, toolID)
	elapsed := time.Since(start)

	if ctx.Err() != nil {
		return HealthRecord{
			ToolID:       toolID,
			Status:       StatusDown,
			CheckedAt:    time.Now(),
			ResponseTime: elapsed.String(),
			Error:        "context cancelled",
		}
	}

	if err != nil {
		return HealthRecord{
			ToolID:       toolID,
			Status:       StatusDown,
			CheckedAt:    time.Now(),
			ResponseTime: elapsed.String(),
			Error:        err.Error(),
		}
	}

	if code == "" {
		return HealthRecord{
			ToolID:       toolID,
			Status:       StatusDegraded,
			CheckedAt:    time.Now(),
			ResponseTime: elapsed.String(),
			Error:        "tool code is empty",
		}
	}

	return HealthRecord{
		ToolID:       toolID,
		Status:       StatusHealthy,
		CheckedAt:    time.Now(),
		ResponseTime: elapsed.String(),
	}
}