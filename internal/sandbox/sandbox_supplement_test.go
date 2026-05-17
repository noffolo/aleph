package sandbox

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerSandbox_RunSkill_ErrorHandling(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)
	assert.NotNil(t, cs)

	result, err := cs.RunSkill(context.Background(), "skill1", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestExecutionResult_ZeroValue(t *testing.T) {
	r := ExecutionResult{}
	assert.Equal(t, "", r.Stdout)
	assert.Equal(t, "", r.Stderr)
	assert.Equal(t, 0, r.ExitCode)
	assert.Equal(t, "", r.Error)
	assert.Equal(t, "", r.Metrics)
}

func TestExecutionResult_WithError(t *testing.T) {
	r := ExecutionResult{
		ExitCode: 1,
		Error:    "something went wrong",
		Stderr:   "traceback follows",
	}
	assert.Equal(t, 1, r.ExitCode)
	assert.Contains(t, r.Error, "something went wrong")
	assert.Contains(t, r.Stderr, "traceback")
}

func TestExecutionResult_Success(t *testing.T) {
	r := ExecutionResult{
		Stdout:   `{"result": "ok"}`,
		ExitCode: 0,
	}
	assert.Equal(t, 0, r.ExitCode)
	assert.Equal(t, `{"result": "ok"}`, r.Stdout)
	assert.Empty(t, r.Error)
}

func TestHealthCheckResult_Struct(t *testing.T) {
	result := HealthCheckResult{
		Available:    true,
		HasRunsc:     true,
		DockerPingOK: true,
	}
	assert.True(t, result.Available)
	assert.True(t, result.HasRunsc)
	assert.True(t, result.DockerPingOK)
	assert.Empty(t, result.Error)
}

func TestHealthCheckResult_Unavailable(t *testing.T) {
	result := HealthCheckResult{
		Error: "docker binary not found in PATH",
	}
	assert.False(t, result.Available)
	assert.False(t, result.HasRunsc)
	assert.False(t, result.DockerPingOK)
	assert.NotEmpty(t, result.Error)
}

func TestContainerConfig_Fields(t *testing.T) {
	cfg := ContainerConfig{
		PythonImage:    "python:3.12-slim",
		GoImage:        "golang:1.24-bookworm",
		DefaultTimeout: 60 * time.Second,
		MaxMemoryMB:    512,
		MaxCPUSeconds:  15.0,
		NetworkBlocked: true,
	}
	assert.Equal(t, "python:3.12-slim", cfg.PythonImage)
	assert.Equal(t, "golang:1.24-bookworm", cfg.GoImage)
	assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
	assert.Equal(t, 512, cfg.MaxMemoryMB)
	assert.Equal(t, 15.0, cfg.MaxCPUSeconds)
	assert.True(t, cfg.NetworkBlocked)
}

func TestNewExecSandbox(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")
	assert.NotNil(t, sb)
	assert.Equal(t, "python3", sb.pythonCmd)
	assert.Equal(t, "go", sb.goCmd)
	assert.Nil(t, sb.mockDeps)
	assert.False(t, sb.profileEnabled)
}

func TestExecSandbox_WithMockDep_Chaining(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithMockDep("a", "code a")
	sb3 := sb2.WithMockDep("b", "code b")

	assert.Len(t, sb.mockDeps, 0) // original unchanged
	assert.Len(t, sb2.mockDeps, 1)
	assert.Len(t, sb3.mockDeps, 2)
}

func TestExecSandbox_WithProfiling_Disable(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithProfiling(false, "")
	assert.NotNil(t, sb2)
	assert.False(t, sb2.profileEnabled)
}

func TestWriteMockFiles_ReadOnlyDir(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")
	sb2 := sb.WithMockDep("test", "code")

	// Write to a non-existent directory should fail
	err := sb2.WriteMockFiles("/nonexistent/dir/that/cannot/be/created")
	assert.Error(t, err)
}

func TestWriteMockFiles_ExistingDir(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")
	sb2 := sb.WithMockDep("svc", "# python\nprint('hi')\n")

	tmpDir := t.TempDir()
	err := sb2.WriteMockFiles(tmpDir)
	require.NoError(t, err)

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "mock_svc.py", entries[0].Name())
}

func TestDefaultContainerConfig_Struct(t *testing.T) {
	cfg := DefaultContainerConfig()
	assert.NotZero(t, cfg.MaxCPUSeconds)
	assert.Greater(t, cfg.MaxCPUSeconds, float64(0))
}

func TestContainerConfig_DefaultTimeout(t *testing.T) {
	cfg := DefaultContainerConfig()
	assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
}

func TestErrContainerUnavailable_IsError(t *testing.T) {
	assert.Implements(t, (*error)(nil), ErrContainerUnavailable)
}

func TestSandboxManager_Interface(t *testing.T) {
	// Verify SandboxManager interface is satisfied by ExecSandbox and ContainerSandbox
	var _ SandboxManager = (*ExecSandbox)(nil)
	var _ SandboxManager = (*ContainerSandbox)(nil)
}
