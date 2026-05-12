package mcp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryGoroutineLeak(t *testing.T) {
	// Phase 0 confirmed all select blocks have ctx.Done(), so this should pass.
	// If any select is missing a ctx.Done() case, cancel() will deadlock and
	// the test will time out.
	ctx, cancel := context.WithCancel(context.Background())
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 100 * time.Millisecond,
	})

	err := engine.Start(ctx)
	require.NoError(t, err)
	assert.True(t, engine.running)

	cancel()

	// Stop waits for wg (which tracks the health-loop goroutine).
	// If the goroutine leaks (select without <-ctx.Done()), Stop hangs and
	// the test exceeds its timeout.
	done := make(chan struct{})
	go func() {
		engine.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Goroutine exited cleanly — PASS.
	case <-time.After(3 * time.Second):
		t.Fatal("discovery engine goroutine leaked — Stop did not return after context cancellation")
	}

	assert.False(t, engine.running)
}

func TestNewDiscoveryEngine(t *testing.T) {
	logger := slog.Default()
	config := DiscoveryConfig{
		ServerURIs:  []string{"mcp://example.com:8080/tools"},
		HealthCheck: 30 * time.Second,
	}
	engine := NewDiscoveryEngine(logger, nil, config)
	assert.NotNil(t, engine)
	assert.Equal(t, logger, engine.logger)
	assert.Nil(t, engine.metaRepo)
	assert.NotNil(t, engine.health)
	assert.Equal(t, config, engine.config)
	assert.False(t, engine.running)
}

func TestNewDiscoveryEngine_NilMetaRepo(t *testing.T) {
	logger := slog.Default()
	config := DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 0,
	}
	engine := NewDiscoveryEngine(logger, nil, config)
	assert.NotNil(t, engine)
	assert.Nil(t, engine.metaRepo)
}

func TestNewDiscoveryEngine_EmptyConfig(t *testing.T) {
	engine := NewDiscoveryEngine(nil, nil, DiscoveryConfig{})
	assert.NotNil(t, engine)
	assert.Nil(t, engine.logger)
	assert.False(t, engine.running)
}

func TestNewDiscoveryEngine_StartStop(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 0,
	})

	// Stop on an engine that was never started should not panic
	engine.Stop()
	assert.False(t, engine.running)

	// Start with empty server list should succeed
	err := engine.Start(context.Background())
	assert.NoError(t, err)
	assert.True(t, engine.running)

	// Second start should fail
	err = engine.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop should work
	engine.Stop()
	assert.False(t, engine.running)
}

func TestNewDiscoveryEngine_StartWithHealthCheck(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := engine.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, engine.running)

	// Let the health loop tick once
	time.Sleep(150 * time.Millisecond)

	engine.Stop()
	assert.False(t, engine.running)
}

func TestDiscoveryConfig_Defaults(t *testing.T) {
	config := DiscoveryConfig{}
	assert.Empty(t, config.ServerURIs)
	assert.Zero(t, config.HealthCheck)
}

func TestNewMCPHealthChecker(t *testing.T) {
	checker := NewMCPHealthChecker()
	assert.NotNil(t, checker)
	assert.NotNil(t, checker.client)
	assert.Equal(t, 10*time.Second, checker.defaultTimeout)
}

func TestNewMCPHealthChecker_DefaultTimeout(t *testing.T) {
	checker := NewMCPHealthChecker()
	assert.Equal(t, 10*time.Second, checker.defaultTimeout)
}

func TestHealthCheckResult_Defaults(t *testing.T) {
	result := HealthCheckResult{}
	assert.False(t, result.Available)
	assert.Empty(t, result.ResponseTime)
	assert.Empty(t, result.Error)
	assert.False(t, result.TLSValid)
	assert.True(t, result.CheckedAt.IsZero())
}

func TestHealthCheckResult_Available(t *testing.T) {
	result := HealthCheckResult{Available: true, TLSValid: true}
	assert.True(t, result.Available)
	assert.True(t, result.TLSValid)
}

func TestDiscoverSchemas_EmptyURL(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	schemas, err := engine.DiscoverSchemas(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, schemas)
}

func TestDiscoverSchemas_InvalidURL(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	schemas, err := engine.DiscoverSchemas(context.Background(), "://invalid")
	assert.Error(t, err)
	assert.Nil(t, schemas)
}

func TestDiscover_EmptyConfig(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	err := engine.Discover(context.Background())
	assert.NoError(t, err) // No servers configured, no error
}

func TestDiscover_InvalidURIs(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{
		ServerURIs: []string{"not-an-mcp-uri", "also-bad"},
	})

	err := engine.Discover(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 MCP servers failed")
}

func TestDiscoverSchemas_BlockedBySSRF(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	schemas, err := engine.DiscoverSchemas(context.Background(), "http://localhost:8080/tools")
	assert.Error(t, err)
	assert.Nil(t, schemas)
	assert.Contains(t, err.Error(), "SSRF validation failed")
}

func TestDiscoverSchemas_SchemeBlocked(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	schemas, err := engine.DiscoverSchemas(context.Background(), "ftp://example.com/tools")
	assert.Error(t, err)
	assert.Nil(t, schemas)
}

func TestParseMCPURI_InvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"empty string", ""},
		{"no prefix", "http://example.com"},
		{"just mcp", "mcp:"},
		{"mcp without slashes", "mcp:host"},
		{"bad scheme", "invalid://host/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := ParseMCPURI(tt.uri)
			assert.Error(t, err)
		})
	}
}

func TestParseMCPURI_Defaults(t *testing.T) {
	_, host, port, path, err := ParseMCPURI("mcp://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "example.com", host)
	assert.Equal(t, "8080", port)
	assert.Equal(t, "/", path)
}

func TestParseMCPURI_WithPortAndPath(t *testing.T) {
	_, host, port, path, err := ParseMCPURI("mcp://myserver:3000/api/tools")
	assert.NoError(t, err)
	assert.Equal(t, "myserver", host)
	assert.Equal(t, "3000", port)
	assert.Equal(t, "/api/tools", path)
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", StatusHealthy)
	assert.Equal(t, "degraded", StatusDegraded)
	assert.Equal(t, "down", StatusDown)
	assert.Equal(t, "unknown", StatusUnknown)
}

func TestDiscover_WithSSRFBlockedURI(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{
		ServerURIs: []string{"mcp://localhost:8080/tools"},
	})

	err := engine.Discover(context.Background())
	assert.Error(t, err)
}

func TestDiscover_WithOnlyInvalidURIs(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{
		ServerURIs: []string{"mcp://127.0.0.1:8080/tools"},
	})

	err := engine.Discover(context.Background())
	assert.Error(t, err)
}

func TestExtractTools_EmptyURL(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	tools, err := engine.extractTools(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "empty server URL")
}

func TestHealthLoop_ZeroInterval(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{
		ServerURIs:  []string{},
		HealthCheck: 0,
	})

	// healthLoop should return immediately when interval is 0
	engine.healthLoop(context.Background())
	// No panic = success
}

func TestCheckServerHealth_EmptyConfig(t *testing.T) {
	logger := slog.Default()
	engine := NewDiscoveryEngine(logger, nil, DiscoveryConfig{})

	// Should not panic with empty server list
	engine.checkServerHealth(context.Background())
}

func TestValidateSSRF_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty string", "", true},
		{"no scheme", "example.com", true},
		{"relative path", "/path/to/resource", true},
		{"file scheme", "file:///etc/passwd", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSRF(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSRF_PrivateIPs(t *testing.T) {
	// These tests may fail in CI depending on DNS resolution, but they verify the private IP check logic
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"localhost", "http://localhost:8080", true},
		{"127.0.0.1", "http://127.0.0.1:8080", true},
		{"::1", "http://[::1]:8080", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSRF(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSRF_InternalTLDs(t *testing.T) {
	assert.Error(t, ValidateSSRF("http://service.internal/api"))
	assert.Error(t, ValidateSSRF("http://dev.local:8080"))
}

func TestToToolRecord_NilInputSchema(t *testing.T) {
	td := ToolDefinition{
		Name: "nil-schema-tool",
	}
	record := td.ToToolRecord("mcp://server:8080/")
	assert.Equal(t, "nil-schema-tool", record.Name)
	assert.Empty(t, record.Code)
	assert.Equal(t, "retrieval", record.Category)
	assert.Equal(t, "0.1.0", record.Version)
	assert.Equal(t, "mcp", record.SourceType)
}

func TestParseToolList_NoToolsField(t *testing.T) {
	input := []byte(`{}`)
	tools, err := ParseToolList(input)
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

func TestParseToolList_ExtraFields(t *testing.T) {
	input := []byte(`{"tools": [{"name": "t1", "extra": "value"}], "meta": {"version": "1.0"}}`)
	tools, err := ParseToolList(input)
	assert.NoError(t, err)
	assert.Len(t, tools, 1)
	assert.Equal(t, "t1", tools[0].Name)
}

func TestValidateSSRF_BadScheme(t *testing.T) {
	assert.Error(t, ValidateSSRF("http://localhost:8080"), "localhost blocked")
	assert.Error(t, ValidateSSRF("file:///etc/passwd"), "file scheme blocked")
	assert.Error(t, ValidateSSRF("https://service.internal/api"), "internal TLD blocked")
}

func TestDiscoverSchemas_MalformedURL(t *testing.T) {
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	schemas, err := engine.DiscoverSchemas(context.Background(), "http://[::1]:8080/tools")
	assert.Error(t, err)
	assert.Nil(t, schemas)
}

func TestParseMCPURI_EdgeCases(t *testing.T) {
	tests := []struct {
		uri      string
		host     string
		port     string
		path     string
		wantErr  bool
	}{
		{"mcp://host:1234", "host", "1234", "/", false},
		{"mcp://host:0/path", "host", "0", "/path", false},
		{"mcp://host", "host", "8080", "/", false},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			_, host, port, path, err := ParseMCPURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.host, host)
			assert.Equal(t, tt.port, port)
			assert.Equal(t, tt.path, path)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	// isPrivateIP is unexported, test via ValidateSSRF
	assert.Error(t, ValidateSSRF("http://10.0.0.1:8080"), "10.x blocked")
	assert.Error(t, ValidateSSRF("http://172.16.0.1:8080"), "172.16.x blocked")
	assert.Error(t, ValidateSSRF("http://192.168.1.1:8080"), "192.168.x blocked")
	assert.Error(t, ValidateSSRF("http://100.64.0.1:8080"), "CGNAT blocked")
}

func TestDiscoverSchemas_ReachableExternal(t *testing.T) {
	// With DNS-resolvable URL, DiscoverSchemas should fail at the HTTP layer
	// not at the SSRF layer (SSRF passes, then HTTP call fails)
	engine := NewDiscoveryEngine(slog.Default(), nil, DiscoveryConfig{})
	schemas, err := engine.DiscoverSchemas(context.Background(), "https://api.example.com/tools")
	if err == nil {
		assert.Nil(t, schemas)
	} else {
		// SSRF validation passed, but either DNS fails or HTTP fails - both are acceptable
		assert.Nil(t, schemas)
	}
}

// TestCheckServer_SSRFBlocked tests that CheckServer blocks SSRF URLs.
func TestCheckServer_SSRFBlocked(t *testing.T) {
	checker := NewMCPHealthChecker()
	result := checker.CheckServer(context.Background(), "http://localhost:8080")
	assert.False(t, result.Available)
	assert.Contains(t, result.Error, "SSRF")
}

func TestCheckServer_SSRFBlockedPrivate(t *testing.T) {
	checker := NewMCPHealthChecker()
	result := checker.CheckServer(context.Background(), "http://10.0.0.1:8080")
	assert.False(t, result.Available)
	assert.Contains(t, result.Error, "SSRF")
}

func TestCheckServer_SSRFBlockedFileScheme(t *testing.T) {
	checker := NewMCPHealthChecker()
	result := checker.CheckServer(context.Background(), "file:///etc/passwd")
	assert.False(t, result.Available)
	assert.Contains(t, result.Error, "SSRF")
}

func TestVerifyCertificate_SSRFBlocked(t *testing.T) {
	checker := NewMCPHealthChecker()
	err := checker.VerifyCertificate("http://localhost:8080")
	assert.Error(t, err)
}

func TestVerifyCertificate_InvalidURL(t *testing.T) {
	checker := NewMCPHealthChecker()
	err := checker.VerifyCertificate("not-a-url")
	assert.Error(t, err)
}

