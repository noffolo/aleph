package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// MCPHealthChecker checks the availability and health of MCP servers.
type MCPHealthChecker struct {
	client         *http.Client
	defaultTimeout time.Duration
}

// NewMCPHealthChecker creates a health checker for MCP servers.
func NewMCPHealthChecker() *MCPHealthChecker {
	client := ssrf.NewClient()
	client.Timeout = 10 * time.Second
	return &MCPHealthChecker{
		client:         client,
		defaultTimeout: 10 * time.Second,
	}
}

// HealthCheckResult represents the result of an MCP server health check.
type HealthCheckResult struct {
	Available    bool      `json:"available"`
	ResponseTime string    `json:"response_time,omitempty"`
	Error        string    `json:"error,omitempty"`
	TLSValid     bool      `json:"tls_valid,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// CheckServer checks the health of an MCP server at the given HTTP(S) URL.
// The URL should be the base URL of the MCP server (after mcp:// conversion).
func (h *MCPHealthChecker) CheckServer(ctx context.Context, serverURL string) HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{
		CheckedAt: start,
	}

	// Validate URL against SSRF before connecting
	if err := ValidateSSRF(serverURL); err != nil {
		result.Error = fmt.Sprintf("SSRF validation failed: %v", err)
		return result
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, h.defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	elapsed := time.Since(start)
	result.ResponseTime = elapsed.String()

	if err != nil {
		result.Error = fmt.Sprintf("server unreachable: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Drain body to ensure connection reuse
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	result.Available = resp.StatusCode < 500
	result.TLSValid = true // If we got here with TLS, it's valid

	if resp.StatusCode >= 500 {
		result.Error = fmt.Sprintf("server error: HTTP %d", resp.StatusCode)
	}

	return result
}

// ─── STDIO Subprocess Health Monitor ────────────────────────────────────────────

// ToolHealthStatus represents the current health state of an MCP STDIO tool.
type ToolHealthStatus string

const (
	ToolStatusUp      ToolHealthStatus = "up"
	ToolStatusDown    ToolHealthStatus = "down"
	ToolStatusUnknown ToolHealthStatus = "unknown"
)

// ToolHealth holds the latest health snapshot for a single MCP STDIO tool.
type ToolHealth struct {
	Name         string           `json:"name"`
	Status       ToolHealthStatus `json:"status"`
	LastPing     time.Time        `json:"last_ping"`
	LastError    string           `json:"last_error,omitempty"`
	RestartCount int              `json:"restart_count"`
}

// Pinger is the interface for checking liveness of an MCP STDIO tool subprocess.
// Implementations wrap stdin/stdout pipes or any transport that supports a
// ping/pong exchange.
type Pinger interface {
	// Ping sends a liveness probe to the subprocess and returns nil if alive.
	Ping(ctx context.Context) error
	// Close terminates the subprocess and cleans up resources.
	Close() error
}

// Restarter is an optional interface for transports that support restarting the
// subprocess on failure. If a Pinger also implements Restarter, ToolHealthMonitor
// will call Restart instead of marking the tool as down.
type Restarter interface {
	// Restart kills the current subprocess and starts a new one.
	Restart(ctx context.Context) error
}

// ToolHealthMonitor runs periodic ping checks against MCP STDIO tool
// subprocesses. On 3 consecutive failures it either calls Restart (if the
// transport supports it) or marks the tool as down.
type ToolHealthMonitor struct {
	mu            sync.RWMutex
	tools         map[string]*ToolHealth
	pinger        Pinger
	logger        *slog.Logger
	checkInterval time.Duration
	consecutive   int // consecutive failures before acting
	toolFailCount int // consecutive ping failures counter

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewToolHealthMonitor creates a health monitor that pings the given transport
// at the specified interval. A tool name is required to identify the monitored
// subprocess in status reports.
func NewToolHealthMonitor(logger *slog.Logger, toolName string, pinger Pinger, interval time.Duration) *ToolHealthMonitor {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &ToolHealthMonitor{
		tools: map[string]*ToolHealth{
			toolName: {
				Name:   toolName,
				Status: ToolStatusUnknown,
			},
		},
		pinger:        pinger,
		logger:        logger.With("component", "tool-health-monitor", "tool", toolName),
		checkInterval: interval,
		consecutive:   3,
	}
}

// Start launches the background ping goroutine. The monitor runs until the
// context is cancelled or Stop is called.
func (m *ToolHealthMonitor) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ctx != nil {
		return // already started
	}
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wg.Add(1)
	go m.run()

	m.logger.Info("tool health monitor started", "interval", m.checkInterval)
}

// Stop terminates the background ping goroutine and waits for it to exit.
func (m *ToolHealthMonitor) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.wg.Wait()

	m.mu.Lock()
	m.ctx = nil
	m.mu.Unlock()

	m.logger.Info("tool health monitor stopped")
}

// Status returns a snapshot of all monitored tool health states.
func (m *ToolHealthMonitor) Status() []ToolHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ToolHealth, 0, len(m.tools))
	for _, h := range m.tools {
		result = append(result, *h)
	}
	return result
}

// UpdateToolName allows changing or adding a tool name after construction.
func (m *ToolHealthMonitor) UpdateToolName(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[name]; exists {
		return
	}
	m.tools[name] = &ToolHealth{
		Name:   name,
		Status: ToolStatusUnknown,
	}
}

// ConsecutiveFailures returns the current consecutive failure count for a tool.
func (m *ToolHealthMonitor) ConsecutiveFailures(toolName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	th, ok := m.tools[toolName]
	if !ok {
		return 0
	}

	// Count consecutive failures from the last N pings.
	// We track this via a simple counter internal to the monitor.
	return m.consecutiveFailuresLocked(th)
}

// GetLatestStatus returns the latest status for a tool.
func (m *ToolHealthMonitor) GetLatestStatus(toolName string) ToolHealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	th, ok := m.tools[toolName]
	if !ok {
		return ToolStatusUnknown
	}
	return th.Status
}

// ─── internal ────────────────────────────────────────────────────────────────────

func (m *ToolHealthMonitor) run() {
	defer m.wg.Done()

	// Initial check
	m.checkOnce()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkOnce()
		}
	}
}

func (m *ToolHealthMonitor) checkOnce() {
	// Determine which tool name we are monitoring.
	m.mu.RLock()
	var toolName string
	for name := range m.tools {
		toolName = name
		break
	}
	m.mu.RUnlock()

	if toolName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(m.ctx, m.checkInterval/2)
	defer cancel()

	start := time.Now()
	err := m.pinger.Ping(ctx)
	elapsed := time.Since(start)

	m.mu.Lock()
	defer m.mu.Unlock()

	th, ok := m.tools[toolName]
	if !ok {
		return
	}

	th.LastPing = time.Now()

	if err == nil {
		// Success — reset failure count
		th.Status = ToolStatusUp
		th.LastError = ""
		m.toolFailCount = 0

		m.logger.Debug("tool ping succeeded", "response_time", elapsed)
		return
	}

	// Failure
	m.toolFailCount++
	failures := m.toolFailCount

	th.LastError = err.Error()
	m.logger.Warn("tool ping failed",
		"error", err,
		"consecutive_failures", failures,
		"response_time", elapsed,
	)

	if failures < m.consecutive {
		th.Status = ToolStatusDown // transient down
		return
	}

	// 3+ consecutive failures — act
	th.Status = ToolStatusDown
	th.RestartCount++

	// Attempt restart if the transport supports it
	var restErr error
	if r, ok := m.pinger.(Restarter); ok {
		m.logger.Warn("attempting subprocess restart",
			"restart_attempt", th.RestartCount,
		)
		restErr = r.Restart(m.ctx)
		if restErr == nil {
			m.logger.Info("subprocess restarted successfully",
				"restart_attempt", th.RestartCount,
			)
			th.Status = ToolStatusUnknown // will be resolved by next ping
			m.toolFailCount = 0
			return
		}
		m.logger.Error("subprocess restart failed",
			"error", restErr,
			"restart_attempt", th.RestartCount,
		)
	}

	// No restarter or restart failed — mark as down
	if restErr != nil {
		th.LastError = fmt.Sprintf("restart failed: %v", restErr)
	} else {
		th.LastError = fmt.Sprintf("subprocess down after %d consecutive failures", failures)
	}
}

func (m *ToolHealthMonitor) consecutiveFailuresLocked(th *ToolHealth) int {
	return m.toolFailCount
}

// Ensure compile-time checks.
var (
	_ Pinger = (*nopPinger)(nil)
)

// nopPinger is a no-op pinger used for testing or graceful degradation.
type nopPinger struct{}

func (n *nopPinger) Ping(_ context.Context) error { return nil }
func (n *nopPinger) Close() error                 { return nil }

// ErrPinger is a pinger that always returns the given error.
type ErrPinger struct {
	Err error
}

func (e *ErrPinger) Ping(_ context.Context) error { return e.Err }
func (e *ErrPinger) Close() error                 { return nil }

// ErrRestartPinger is a pinger that always fails and also implements Restarter
// (which also fails). Useful for testing the full failure→restart→failure path.
type ErrRestartPinger struct {
	Err error
}

func (e *ErrRestartPinger) Ping(_ context.Context) error { return e.Err }
func (e *ErrRestartPinger) Close() error                 { return nil }
func (e *ErrRestartPinger) Restart(_ context.Context) error {
	return errors.New("restart not supported in this mode")
}

// VerifyCertificate checks that the TLS certificate for an HTTPS MCP server is valid.
func (h *MCPHealthChecker) VerifyCertificate(serverURL string) error {
	if err := ValidateSSRF(serverURL); err != nil {
		return fmt.Errorf("verifyCertificateSSRF: %w", err)
	}

	client := ssrf.NewClient()
	client.Timeout = 5 * time.Second
	defer client.CloseIdleConnections()

	resp, err := client.Get(serverURL)
	if err != nil {
		return fmt.Errorf("TLS verification failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	return nil
}
