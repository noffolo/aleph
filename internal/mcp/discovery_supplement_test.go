package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

func TestToToolRecord_AllFields(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}

	td := ToolDefinition{
		Name:        "full-tool",
		Description: "A full tool definition",
		InputSchema: schema,
		Version:     "1.2.3",
		Category:    "analysis",
	}
	record := td.ToToolRecord("mcp://server:8080/")

	assert.Equal(t, "full-tool", record.Name)
	assert.Equal(t, "A full tool definition", record.Description)
	assert.Equal(t, "analysis", record.Category)
	assert.Equal(t, "1.2.3", record.Version)
	assert.Equal(t, StatusUnknown, record.HealthStatus)
	assert.Equal(t, "mcp", record.SourceType)
	assert.NotEmpty(t, record.Code)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(record.Code), &parsed); err != nil {
		t.Errorf("code should be valid JSON: %v", err)
	}
}

func TestToToolRecord_InputSchemaMarshalError(t *testing.T) {
	td := ToolDefinition{
		Name:        "bad-schema-tool",
		InputSchema: map[string]any{"channel": make(chan int)},
	}
	record := td.ToToolRecord("mcp://s:8080/")
	assert.Empty(t, record.Code)
	assert.Equal(t, "0.1.0", record.Version)
}

func TestParseToolList_ValidTools(t *testing.T) {
	input := []byte(`{
		"tools": [
			{"name": "tool1", "description": "desc1"},
			{"name": "tool2", "description": "desc2", "inputSchema": {"type": "object"}}
		]
	}`)
	tools, err := ParseToolList(input)
	require.NoError(t, err)
	assert.Len(t, tools, 2)
	assert.Equal(t, "tool1", tools[0].Name)
	assert.Equal(t, "desc1", tools[0].Description)
	assert.Nil(t, tools[0].InputSchema)
	assert.Equal(t, "tool2", tools[1].Name)
	assert.NotNil(t, tools[1].InputSchema)
}

func TestParseToolList_InvalidJSON(t *testing.T) {
	_, err := ParseToolList([]byte("not json"))
	assert.Error(t, err)
}

func TestParseToolList_EmptyArray(t *testing.T) {
	tools, err := ParseToolList([]byte(`{"tools": []}`))
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

func TestValidateMCPServers_AllValid(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs: []string{
			"mcp://example.com:8080/tools",
			"mcp://other.org:3000/api",
		},
	})
	assert.NoError(t, engine.ValidateMCPServers())
}

func TestValidateMCPServers_OneInvalid(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs: []string{
			"mcp://example.com:8080/tools",
			"not-an-mcp-uri",
		},
	})
	assert.Error(t, engine.ValidateMCPServers())
}

func TestValidateMCPServers_EmptyList(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	assert.NoError(t, engine.ValidateMCPServers())
}

func TestDiscoverSchemas_SSRFValidReachable(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	_, err := engine.DiscoverSchemas(context.Background(), "https://api.github.com/tools")
	assert.Error(t, err)
}

func TestDiscoverEngine_StopWithoutStart(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	engine.Stop()
	assert.False(t, engine.running)
}

func TestRetryConfig_Defaults(t *testing.T) {
	assert.Equal(t, 3, defaultRetryConfig.maxAttempts)
	assert.Len(t, defaultRetryConfig.delays, 3)
	assert.Equal(t, 2*time.Second, defaultRetryConfig.delays[0])
	assert.Equal(t, 4*time.Second, defaultRetryConfig.delays[1])
	assert.Equal(t, 8*time.Second, defaultRetryConfig.delays[2])
}

func TestDiscoverServerWithRetry_SSRFBlocks(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	tools, err := engine.discoverServerWithRetry(
		context.Background(),
		"http://localhost:8080/tools",
		"mcp://localhost:8080/tools",
	)
	assert.Error(t, err)
	assert.Nil(t, tools)
}

func TestDiscoverServerWithRetry_CancelledContext(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tools, err := engine.discoverServerWithRetry(ctx, "http://example.com:8080/tools", "mcp://example.com:8080/tools")
	assert.Error(t, err)
	assert.Nil(t, tools)
}

func TestDiscover_WithMetadataRepoIntegration(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS system_tools (
		id VARCHAR PRIMARY KEY,
		name VARCHAR,
		description TEXT,
		code TEXT,
		category VARCHAR,
		version VARCHAR,
		health_status VARCHAR,
		source_type VARCHAR
	)`)
	require.NoError(t, err)

	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs: []string{"mcp://example.com:8080/tools"},
	})
	err = engine.Discover(context.Background())
	assert.Error(t, err)
}

func TestHealthLoop_SkipsWhenZeroHealthCheck(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 0,
	})

	ctx := context.Background()
	engine.healthLoop(ctx)
}

func TestCheckServerHealth_InvalidURISkipped(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs: []string{
			"not-a-uri",
		},
	})
	engine.checkServerHealth(context.Background())
}

func TestErrToolNotFound_IsError(t *testing.T) {
	assert.Error(t, ErrToolNotFound)
	assert.Equal(t, "tool not found", ErrToolNotFound.Error())
}

func TestNewDiscoveryEngine_WithLoggingSetup(t *testing.T) {
	logger := slog.With("component", "test")
	config := DiscoveryConfig{
		ServerURIs:  []string{"mcp://example.com:9090/v1"},
		HealthCheck: 15 * time.Second,
	}
	engine := NewDiscoveryEngine(logger, nil, config)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.health)
	assert.NotNil(t, engine.httpClient)
	assert.False(t, engine.running)
	assert.Equal(t, config, engine.config)
}

func TestMCPListToolsResponse_Empty(t *testing.T) {
	resp := MCPListToolsResponse{}
	assert.Empty(t, resp.Tools)
}

func TestToolDefinition_ZeroValue(t *testing.T) {
	td := ToolDefinition{}
	assert.Empty(t, td.Name)
	assert.Empty(t, td.Description)
	assert.Nil(t, td.InputSchema)
	assert.Empty(t, td.Version)
	assert.Empty(t, td.Category)
}

func TestToolRecord_ZeroValue(t *testing.T) {
	tr := repository.ToolRecord{}
	assert.Empty(t, tr.ID)
	assert.Empty(t, tr.Name)
	assert.Empty(t, tr.Description)
	assert.Empty(t, tr.Code)
	assert.Empty(t, tr.Category)
	assert.Empty(t, tr.Version)
	assert.Empty(t, tr.HealthStatus)
	assert.Empty(t, tr.SourceType)
}

func TestExtractTools_InvalidServerURL(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	tools, err := engine.extractTools(context.Background(), "://invalid-url")
	assert.Error(t, err)
	assert.Nil(t, tools)
}

func TestDiscover_OneValidOneInvalidURI(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{
		ServerURIs: []string{
			"mcp://192.0.2.1:9999/tools",
			"not-a-uri",
		},
	})
	err := engine.Discover(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 MCP servers failed")
}

func TestValidatePrivateRanges_Noop(t *testing.T) {
	assert.NoError(t, ValidatePrivateRanges())
}
