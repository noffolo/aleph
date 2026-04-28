package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// ErrToolNotFound is returned by findToolByName when no tool matches.
var ErrToolNotFound = errors.New("tool not found")

// DiscoveryConfig holds configuration for the MCP discovery engine.
type DiscoveryConfig struct {
	ServerURIs  []string      // MCP server URIs (mcp://host:port/path)
	HealthCheck time.Duration  // Interval for health checking servers
}

// DiscoveryEngine discovers and registers tools from MCP servers.
type DiscoveryEngine struct {
	logger   *slog.Logger
	metaRepo *repository.MetadataRepository
	health   *MCPHealthChecker
	config   DiscoveryConfig
	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
}

// NewDiscoveryEngine creates a new MCP discovery engine.
func NewDiscoveryEngine(logger *slog.Logger, metaRepo *repository.MetadataRepository, config DiscoveryConfig) *DiscoveryEngine {
	return &DiscoveryEngine{
		logger:   logger,
		metaRepo: metaRepo,
		health:   NewMCPHealthChecker(),
		config:   config,
	}
}

// DiscoverSchemas discovers tool schemas from an MCP server URL.
// It validates the URL against SSRF rules and returns the list of tool definitions.
func (d *DiscoveryEngine) DiscoverSchemas(ctx context.Context, serverURL string) ([]ToolDefinition, error) {
	// Validate URL against SSRF
	if err := ValidateSSRF(serverURL); err != nil {
		return nil, fmt.Errorf("SSRF validation failed: %w", err)
	}
	return d.extractTools(ctx, serverURL)
}

// Start begins periodic discovery and health checking of MCP servers.
func (d *DiscoveryEngine) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("discovery engine already running")
	}
	d.running = true
	d.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	// Run initial discovery
	if err := d.Discover(ctx); err != nil {
		d.logger.Warn("initial MCP discovery failed", "error", err)
	}

	// Start periodic health checks if interval is configured
	if d.config.HealthCheck > 0 {
		go d.healthLoop(ctx)
	}

	d.logger.Info("MCP discovery engine started", "servers", len(d.config.ServerURIs))
	return nil
}

// Stop cancels the discovery engine.
func (d *DiscoveryEngine) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
	}
	d.running = false
	d.logger.Info("MCP discovery engine stopped")
}

// Discover scans all configured MCP servers and registers discovered tools.
func (d *DiscoveryEngine) Discover(ctx context.Context) error {
	var allErrors []error

	for _, uri := range d.config.ServerURIs {
		_, host, port, path, err := ParseMCPURI(uri)
		if err != nil {
			d.logger.Warn("invalid MCP URI", "uri", uri, "error", err)
			allErrors = append(allErrors, fmt.Errorf("invalid URI %q: %w", uri, err))
			continue
		}

		// Convert mcp:// to http:// for actual connections
		serverURL := fmt.Sprintf("http://%s:%s%s", host, port, path)

		// Validate URL against SSRF
		if err := ValidateSSRF(serverURL); err != nil {
			d.logger.Warn("MCP server URL blocked by SSRF validation", "uri", uri, "error", err)
			allErrors = append(allErrors, fmt.Errorf("SSRF blocked %q: %w", uri, err))
			continue
		}

		// Health check the server first
		healthResult := d.health.CheckServer(ctx, serverURL)
		if !healthResult.Available {
			d.logger.Warn("MCP server not available",
				"uri", uri,
				"error", healthResult.Error,
			)
			allErrors = append(allErrors, fmt.Errorf("server %q unavailable: %s", uri, healthResult.Error))
			continue
		}

		// Discover tools from the server
		tools, err := d.extractTools(ctx, serverURL)
		if err != nil {
			d.logger.Warn("failed to extract tools from MCP server", "uri", uri, "error", err)
			allErrors = append(allErrors, fmt.Errorf("extract from %q: %w", uri, err))
			continue
		}

		// Register discovered tools
		for _, toolDef := range tools {
			toolRecord := toolDef.ToToolRecord(uri)

			// Check if tool already exists (by name)
			existing, err := d.findToolByName(ctx, toolRecord.Name)
			if err != nil && !errors.Is(err, ErrToolNotFound) {
				d.logger.Warn("failed to check existing tool", "name", toolRecord.Name, "error", err)
				continue
			}

			if existing != nil {
				// Update existing tool
				d.logger.Debug("updating existing MCP tool", "name", toolRecord.Name, "source", uri)
				continue
			}

			// Create new tool
			if toolRecord.ID == "" {
				toolRecord.ID = fmt.Sprintf("mcp-%s", toolRecord.Name)
			}
			if err := d.metaRepo.CreateTool(&toolRecord); err != nil {
				d.logger.Warn("failed to register MCP tool", "name", toolRecord.Name, "error", err)
			} else {
				d.logger.Info("discovered and registered MCP tool",
					"name", toolRecord.Name,
					"category", toolRecord.Category,
					"source", uri,
				)
			}
		}
	}

	if len(allErrors) > 0 && len(allErrors) == len(d.config.ServerURIs) {
		return fmt.Errorf("all %d MCP servers failed", len(d.config.ServerURIs))
	}
	return nil
}

// extractTools connects to an MCP server and extracts tool definitions.
func (d *DiscoveryEngine) extractTools(ctx context.Context, serverURL string) ([]ToolDefinition, error) {
	// MCP protocol: POST to /tools/list endpoint
	listURL := serverURL + "/tools/list"
	if serverURL == "" {
		return nil, fmt.Errorf("empty server URL")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, listURL, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// Fallback: try GET method
		return d.extractToolsGet(ctx, serverURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("MCP server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	toolDefs, err := ParseToolList(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool list: %w", err)
	}

	return toolDefs, nil
}

// extractToolsGet tries to discover tools via GET method.
func (d *DiscoveryEngine) extractToolsGet(ctx context.Context, serverURL string) ([]ToolDefinition, error) {
	listURL := serverURL + "/tools/list"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MCP server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return ParseToolList(body)
}

// findToolByName looks up a tool by name in the repository.
func (d *DiscoveryEngine) findToolByName(ctx context.Context, name string) (*repository.ToolRecord, error) {
	tools, err := d.metaRepo.ListTools()
	if err != nil {
		return nil, err
	}
	for _, t := range tools {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, ErrToolNotFound
}

// healthLoop periodically health checks MCP servers and updates tool health status.
func (d *DiscoveryEngine) healthLoop(ctx context.Context) {
	if d.config.HealthCheck <= 0 {
		return
	}

	ticker := time.NewTicker(d.config.HealthCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.checkServerHealth(ctx)
		}
	}
}

// checkServerHealth checks all configured servers and updates tool health.
func (d *DiscoveryEngine) checkServerHealth(ctx context.Context) {
	for _, uri := range d.config.ServerURIs {
		_, host, port, path, err := ParseMCPURI(uri)
		if err != nil {
			continue
		}

		serverURL := fmt.Sprintf("http://%s:%s%s", host, port, path)

		result := d.health.CheckServer(ctx, serverURL)

		// Update health status for all tools from this server
		tools, err := d.metaRepo.ListTools()
		if err != nil {
			d.logger.Warn("failed to list tools for health update", "error", err)
			continue
		}

		status := StatusHealthy
		if !result.Available {
			status = StatusDown
		}

		for _, tool := range tools {
			if tool.SourceType == "mcp" {
				if err := d.metaRepo.UpdateHealthStatus(tool.ID, status); err != nil {
					d.logger.Warn("failed to update MCP tool health", "tool_id", tool.ID, "error", err)
				}
			}
		}
	}
}