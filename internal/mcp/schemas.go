package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// ToolDefinition represents a tool discovered from an MCP server.
// It maps from MCP JSON Schema to internal ToolRecord fields.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Category    string                 `json:"category,omitempty"`
}

// MCPListToolsResponse represents the response from an MCP server's list_tools method.
type MCPListToolsResponse struct {
	Tools []MCPToolEntry `json:"tools"`
}

// MCPToolEntry represents a single tool in an MCP list_tools response.
type MCPToolEntry struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ToToolRecord converts an MCP ToolDefinition to a repository ToolRecord.
func (td ToolDefinition) ToToolRecord(sourceURI string) repository.ToolRecord {
	category := td.Category
	if category == "" {
		category = "retrieval" // default category for MCP-discovered tools
	}

	version := td.Version
	if version == "" {
		version = "0.1.0" // default version for newly discovered tools
	}

	// Serialize input schema to JSON for storage
	var code string
	if td.InputSchema != nil {
		schemaBytes, err := json.Marshal(td.InputSchema)
		if err == nil {
			code = string(schemaBytes)
		}
	}

	return repository.ToolRecord{
		Name:         td.Name,
		Description:  td.Description,
		Code:         code,
		Category:     category,
		Version:      version,
		HealthStatus: StatusUnknown,
		SourceType:   "mcp",
	}
}

// ParseToolList parses a JSON response from an MCP server into ToolDefinition entries.
func ParseToolList(data []byte) ([]ToolDefinition, error) {
	var resp MCPListToolsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse MCP tool list: %w", err)
	}

	defs := make([]ToolDefinition, 0, len(resp.Tools))
	for _, tool := range resp.Tools {
		defs = append(defs, ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return defs, nil
}

// Health status constants for MCP discovery
const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusDown     = "down"
	StatusUnknown  = "unknown"
)