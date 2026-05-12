package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func okTool(name, category string) ToolDefinition {
	return ToolDefinition{
		Name:        name,
		Category:    category,
		Description: "test tool " + name,
		Execute: func(ctx context.Context, params map[string]any) (any, error) {
			return "ok", nil
		},
	}
}

// ---------------------------------------------------------------------------
// NewToolRegistry
// ---------------------------------------------------------------------------

func TestNewToolRegistry(t *testing.T) {
	r := NewToolRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.List(""))
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r := NewToolRegistry()
		err := r.Register(okTool("test-tool", "test-cat"))
		require.NoError(t, err)
		def, ok := r.Get("test-cat", "test-tool")
		assert.True(t, ok)
		assert.Equal(t, "test-tool", def.Name)
	})

	t.Run("empty name", func(t *testing.T) {
		r := NewToolRegistry()
		err := r.Register(ToolDefinition{
			Name:     "",
			Category: "cat",
			Execute:  func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool name is required")
	})

	t.Run("empty category", func(t *testing.T) {
		r := NewToolRegistry()
		err := r.Register(ToolDefinition{
			Name:     "mytool",
			Category: "",
			Execute:  func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool category is required")
	})

	t.Run("nil execute", func(t *testing.T) {
		r := NewToolRegistry()
		err := r.Register(ToolDefinition{
			Name:     "mytool",
			Category: "cat",
			Execute:  nil,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil Execute")
	})

	t.Run("duplicate registration", func(t *testing.T) {
		r := NewToolRegistry()
		def := okTool("dup", "cat")
		require.NoError(t, r.Register(def))
		err := r.Register(def)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate tool registration")
	})
}

// ---------------------------------------------------------------------------
// RegisterAll
// ---------------------------------------------------------------------------

func TestRegisterAll(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		r := NewToolRegistry()
		defs := []ToolDefinition{
			okTool("a", "cat"),
			okTool("b", "cat"),
		}
		err := r.RegisterAll(defs)
		require.NoError(t, err)
		assert.Len(t, r.List(""), 2)
	})

	t.Run("duplicate within batch returns error and rolls back", func(t *testing.T) {
		r := NewToolRegistry()
		defs := []ToolDefinition{
			okTool("a", "cat"),
			okTool("a", "cat"), // duplicate
			okTool("b", "cat"),
		}
		err := r.RegisterAll(defs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate tool in batch")
		// Atomic rollback: none should be registered
		_, ok := r.Get("cat", "a")
		assert.False(t, ok, "tool 'a' should NOT be registered after batch rollback")
	})

	t.Run("empty name in batch", func(t *testing.T) {
		r := NewToolRegistry()
		defs := []ToolDefinition{
			okTool("a", "cat"),
			{Name: "", Category: "cat", Execute: func(ctx context.Context, params map[string]any) (any, error) { return nil, nil }},
		}
		err := r.RegisterAll(defs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty name")
		_, ok := r.Get("cat", "a")
		assert.False(t, ok, "tool 'a' should NOT be registered after batch rollback")
	})

	t.Run("duplicate with existing registration", func(t *testing.T) {
		r := NewToolRegistry()
		require.NoError(t, r.Register(okTool("a", "cat")))
		err := r.RegisterAll([]ToolDefinition{okTool("a", "cat")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate tool in batch")
	})
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	t.Run("empty registry returns empty slice", func(t *testing.T) {
		r := NewToolRegistry()
		assert.Empty(t, r.List(""))
	})

	t.Run("list all when category is empty", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("a", "cat1"))
		_ = r.Register(okTool("b", "cat2"))
		all := r.List("")
		assert.Len(t, all, 2)
	})

	t.Run("filter by category", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("a", "finance"))
		_ = r.Register(okTool("b", "osint"))
		_ = r.Register(okTool("c", "finance"))
		fin := r.List("finance")
		assert.Len(t, fin, 2)
		for _, def := range fin {
			assert.Equal(t, "finance", def.Category)
		}
	})

	t.Run("returns empty for unknown category", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("a", "finance"))
		assert.Empty(t, r.List("nonexistent"))
	})
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet(t *testing.T) {
	t.Run("existing tool", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("my-tool", "my-cat"))
		def, ok := r.Get("my-cat", "my-tool")
		assert.True(t, ok)
		assert.Equal(t, "my-tool", def.Name)
		assert.Equal(t, "my-cat", def.Category)
	})

	t.Run("missing tool returns false", func(t *testing.T) {
		r := NewToolRegistry()
		_, ok := r.Get("cat", "nonexistent")
		assert.False(t, ok)
	})

	t.Run("wrong category returns false", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("t", "cat-a"))
		_, ok := r.Get("cat-b", "t")
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

func TestExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("greet", "test"))
		result, err := r.Execute(context.Background(), "test", "greet", map[string]any{"name": "world"})
		require.NoError(t, err)
		assert.Equal(t, "ok", result)
	})

	t.Run("tool not found", func(t *testing.T) {
		r := NewToolRegistry()
		_, err := r.Execute(context.Background(), "cat", "missing", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool not found")
	})

	t.Run("execute error propagates", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(ToolDefinition{
			Name:        "fail",
			Category:    "test",
			Description: "always fails",
			Execute: func(ctx context.Context, params map[string]any) (any, error) {
				return nil, errors.New("execution failed")
			},
		})
		_, err := r.Execute(context.Background(), "test", "fail", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution failed")
	})
}

// ---------------------------------------------------------------------------
// ExecuteContext
// ---------------------------------------------------------------------------

func TestExecuteContext(t *testing.T) {
	t.Run("passes context and returns result", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("echo", "test"))
		ctx := context.Background()
		result, err := r.ExecuteContext(ctx, "test", "echo", nil)
		require.NoError(t, err)
		assert.Equal(t, "ok", result)
	})

	t.Run("context cancellation propagates", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(ToolDefinition{
			Name:        "slow",
			Category:    "test",
			Description: "slow tool",
			Execute: func(ctx context.Context, params map[string]any) (any, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(10 * time.Second):
					return "done", nil
				}
			},
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel
		_, err := r.ExecuteContext(ctx, "test", "slow", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("tool not found", func(t *testing.T) {
		r := NewToolRegistry()
		_, err := r.ExecuteContext(context.Background(), "cat", "missing", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool not found")
	})
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

func TestCategories(t *testing.T) {
	t.Run("empty registry returns empty slice", func(t *testing.T) {
		r := NewToolRegistry()
		assert.Empty(t, r.Categories())
	})

	t.Run("single category", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("a", "finance"))
		_ = r.Register(okTool("b", "finance"))
		cats := r.Categories()
		assert.Equal(t, []string{"finance"}, cats)
	})

	t.Run("multiple categories", func(t *testing.T) {
		r := NewToolRegistry()
		_ = r.Register(okTool("a", "finance"))
		_ = r.Register(okTool("b", "osint"))
		_ = r.Register(okTool("c", "human-ecosystems"))
		cats := r.Categories()
		assert.ElementsMatch(t, []string{"finance", "osint", "human-ecosystems"}, cats)
	})
}

// ---------------------------------------------------------------------------
// FinanceToolDef
// ---------------------------------------------------------------------------

func TestFinanceToolDef(t *testing.T) {
	executed := false
	def := FinanceToolDef("forecast", "Financial forecast tool",
		func(ctx context.Context, params map[string]any) (any, error) {
			executed = true
			return map[string]any{"prediction": 42}, nil
		},
	)
	assert.Equal(t, "forecast", def.Name)
	assert.Equal(t, "finance", def.Category)
	assert.Equal(t, "Financial forecast tool", def.Description)
	require.NotNil(t, def.Execute)

	result, err := def.Execute(context.Background(), map[string]any{"periods": 5})
	require.NoError(t, err)
	assert.True(t, executed)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 42, m["prediction"])
}

// ---------------------------------------------------------------------------
// OSINTToolDef
// ---------------------------------------------------------------------------

func TestOSINTToolDef(t *testing.T) {
	t.Run("happy path returns parsed JSON", func(t *testing.T) {
		def := OSINTToolDef("threat-scan", "Threat scanner",
			func(ctx context.Context, argsJSON string) (string, error) {
				return `{"threats": 3, "severity": "high"}`, nil
			},
		)
		assert.Equal(t, "threat-scan", def.Name)
		assert.Equal(t, "osint", def.Category)

		result, err := def.Execute(context.Background(), map[string]any{"target": "example.com"})
		require.NoError(t, err)
		m, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(3), m["threats"])
		assert.Equal(t, "high", m["severity"])
	})

	t.Run("non-JSON result returns raw string", func(t *testing.T) {
		def := OSINTToolDef("raw", "raw output",
			func(ctx context.Context, argsJSON string) (string, error) {
				return "plain text output", nil
			},
		)
		result, err := def.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "plain text output", result)
	})

	t.Run("inner error propagates", func(t *testing.T) {
		def := OSINTToolDef("bad", "broken tool",
			func(ctx context.Context, argsJSON string) (string, error) {
				return "", errors.New("api unavailable")
			},
		)
		_, err := def.Execute(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api unavailable")
	})
}

// ---------------------------------------------------------------------------
// HEToolDef
// ---------------------------------------------------------------------------

type mockHETool struct {
	name        string
	description string
	executeErr  error
}

func (m *mockHETool) Execute(ctx context.Context, args map[string]any) (any, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return map[string]any{"result": "he-data"}, nil
}

func (m *mockHETool) Name() string        { return m.name }
func (m *mockHETool) Description() string { return m.description }

func TestHEToolDef(t *testing.T) {
	t.Run("converts HETool to ToolDefinition", func(t *testing.T) {
		mock := &mockHETool{name: "ecosystem-map", description: "Maps human ecosystems"}
		def := HEToolDef(mock)
		assert.Equal(t, "ecosystem-map", def.Name)
		assert.Equal(t, "human-ecosystems", def.Category)
		assert.Equal(t, "Maps human ecosystems", def.Description)

		result, err := def.Execute(context.Background(), map[string]any{"region": "EU"})
		require.NoError(t, err)
		m, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "he-data", m["result"])
	})

	t.Run("propagates execute error", func(t *testing.T) {
		mock := &mockHETool{name: "broken", description: "broken", executeErr: errors.New("internal error")}
		def := HEToolDef(mock)
		_, err := def.Execute(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
	})
}

// ---------------------------------------------------------------------------
// Concurrency safety (non-deterministic, but exercises the RWMutex)
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	r := NewToolRegistry()

	// Register from multiple goroutines
	const n = 20
	errs := make(chan error, n)
	for i := range n {
		i := i
		go func() {
			err := r.Register(okTool(
				"tool-"+string(rune('a'+i)),
				"finance",
			))
			errs <- err
		}()
	}

	for range n {
		err := <-errs
		if err != nil {
			// Duplicates are expected — we're testing there's no data race
			assert.Contains(t, err.Error(), "duplicate")
		}
	}

	// Concurrent reads
	done := make(chan struct{}, 10)
	for range 10 {
		go func() {
			r.List("")
			r.Get("finance", "tool-a")
			r.Categories()
			r.Execute(context.Background(), "finance", "tool-a", nil)
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}
}
