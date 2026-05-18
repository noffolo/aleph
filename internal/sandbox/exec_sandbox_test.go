package sandbox

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecSandbox_NoMetadataRepository(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	ctx := context.Background()
	result, err := sb.ExecuteTool(ctx, "nonexistent", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestExecSandbox_RunSkill_NoMetadataRepository(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	ctx := context.Background()
	result, err := sb.RunSkill(ctx, "nonexistent", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestExecSandbox_Timeout(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result, err := sb.ExecuteTool(ctx, "nonexistent", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "metadata repository not available")
}

func TestExecSandbox_WithMockDep(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithMockDep("db", `package tools
type DB struct{}
func (d *DB) Query(q string) string { return "mock_result" }
`)
	if sb2 == nil {
		t.Fatal("WithMockDep returned nil")
	}
	if len(sb2.mockDeps) != 1 {
		t.Errorf("expected 1 mock dep, got %d", len(sb2.mockDeps))
	}
	if _, ok := sb2.mockDeps["db"]; !ok {
		t.Error("expected 'db' mock dep")
	}

	if len(sb.mockDeps) != 0 {
		t.Error("WithMockDep should not mutate the original sandbox")
	}
}

func TestExecSandbox_WithProfiling(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithProfiling(true, "/tmp/profiles")
	if sb2 == nil {
		t.Fatal("WithProfiling returned nil")
	}
	if !sb2.profileEnabled {
		t.Error("expected profiling to be enabled")
	}
	if sb2.profileDir != "/tmp/profiles" {
		t.Errorf("expected profile dir /tmp/profiles, got %q", sb2.profileDir)
	}

	if sb.profileEnabled {
		t.Error("WithProfiling should not mutate the original sandbox")
	}
}

func TestWriteMockFiles_Empty(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	tmpDir := t.TempDir()
	err := sb.WriteMockFiles(tmpDir)
	if err != nil {
		t.Fatalf("WriteMockFiles() on empty mocks: %v", err)
	}
}

func TestWriteMockFiles_GoMock(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithMockDep("api", `package tools
type APIClient struct{}
func (c *APIClient) Get(url string) string { return "{}" }
`)

	tmpDir := t.TempDir()
	err := sb2.WriteMockFiles(tmpDir)
	if err != nil {
		t.Fatalf("WriteMockFiles(): %v", err)
	}

	mockPath := filepath.Join(tmpDir, "mock_api.go")
	if _, err := os.Stat(mockPath); os.IsNotExist(err) {
		t.Error("expected mock_api.go to exist")
	}
}

func TestWriteMockFiles_PythonMock(t *testing.T) {
	logger := slog.Default()
	sb := NewExecSandbox(logger, nil, nil, "python3", "go")

	sb2 := sb.WithMockDep("db", `# python
def query(sql):
    return []
`)

	tmpDir := t.TempDir()
	err := sb2.WriteMockFiles(tmpDir)
	if err != nil {
		t.Fatalf("WriteMockFiles(): %v", err)
	}

	mockPath := filepath.Join(tmpDir, "mock_db.py")
	if _, err := os.Stat(mockPath); os.IsNotExist(err) {
		t.Error("expected mock_db.py to exist")
	}
}
