package sandbox

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultContainerConfig(t *testing.T) {
	cfg := DefaultContainerConfig()
	assert.Equal(t, DefaultPythonImage, cfg.PythonImage)
	assert.Equal(t, DefaultGoImage, cfg.GoImage)
	assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
	assert.Equal(t, 256, cfg.MaxMemoryMB)
	assert.True(t, cfg.NetworkBlocked)
}

func TestDefaultContainerConfig_CPULimit(t *testing.T) {
	cfg := DefaultContainerConfig()
	assert.Equal(t, float64(0.5)*float64(30*time.Second/time.Second), cfg.MaxCPUSeconds)
}

func TestDockerAvailable(t *testing.T) {
	available := dockerAvailable()
	t.Logf("docker available: %v", available)
	assert.NotPanics(t, func() { dockerAvailable() })
}

func TestNewContainerSandbox_NoFallback(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)
	require.NotNil(t, cs)
	assert.Equal(t, logger, cs.logger)
	assert.Equal(t, cs.config, DefaultContainerConfig())
}

func TestContainerSandbox_ExecuteTool_NoMetaRepo(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "nonexistent", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestContainerSandbox_ExecuteTool_DockerUnavailable_NoFallback(t *testing.T) {
	if dockerAvailable() {
		t.Skip("docker is available, cannot test unavailable path")
	}
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "tool1", map[string]interface{}{})
	require.NoError(t, err)
	assert.Contains(t, result.Error, "container runtime unavailable")
}

func TestContainerSandbox_HealthCheck(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	ctx := context.Background()
	result := cs.HealthCheck(ctx)

	if !result.Available {
		t.Skipf("docker not available: %s", result.Error)
	}

	assert.True(t, result.DockerPingOK)
	t.Logf("gVisor runsc available: %v", result.HasRunsc)
}

func TestContainerSandbox_RunSkill_NoMetaRepo(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	ctx := context.Background()
	result, err := cs.RunSkill(ctx, "nonexistent", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestErrContainerUnavailable(t *testing.T) {
	assert.Equal(t, "container runtime unavailable: docker daemon not reachable", ErrContainerUnavailable.Error())
}

func TestContainerSandbox_RunscDetection(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)
	assert.True(t, cs.runscCached, "expected runsc detection to be cached")
	t.Logf("hasRunsc: %v", cs.hasRunsc)
}