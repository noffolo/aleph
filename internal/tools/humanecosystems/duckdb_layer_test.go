package humanecosystems

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewDuckDBLayer
// =============================================================================

func TestNewDuckDBLayer(t *testing.T) {
	t.Run("happy: creates layer with nil db", func(t *testing.T) {
		dbl := NewDuckDBLayer(nil)
		require.NotNil(t, dbl)
		assert.False(t, dbl.IsAvailable())
	})

	t.Run("happy: creates layer with real DuckDB", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)
		require.NotNil(t, dbl)
		assert.True(t, dbl.IsAvailable())
	})

	t.Run("edge: multiple calls create independent instances", func(t *testing.T) {
		dbl1 := NewDuckDBLayer(nil)
		dbl2 := NewDuckDBLayer(nil)
		assert.NotSame(t, dbl1, dbl2)
	})
}

// =============================================================================
// QueryContext
// =============================================================================

func TestQueryContext(t *testing.T) {
	t.Run("error: nil db returns error", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		rows, err := dbl.QueryContext(context.Background(), "SELECT 1")
		assert.Error(t, err)
		assert.Nil(t, rows)
		assert.Contains(t, err.Error(), "duckdb not available")
	})

	t.Run("happy: real db executes valid query", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)

		_, err = dbl.ExecContext(context.Background(), "CREATE TABLE test_query (id INT)")
		require.NoError(t, err)

		rows, err := dbl.QueryContext(context.Background(), "SELECT 1")
		require.NoError(t, err)
		require.NotNil(t, rows)
		rows.Close()
	})

	t.Run("edge: invalid SQL on real db returns error", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)

		rows, err := dbl.QueryContext(context.Background(), "INVALID SQL SYNTAX !!!")
		assert.Error(t, err)
		assert.Nil(t, rows)
	})
}

// =============================================================================
// ExecContext
// =============================================================================

func TestExecContext(t *testing.T) {
	t.Run("error: nil db returns error", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		res, err := dbl.ExecContext(context.Background(), "CREATE TABLE x (a INT)")
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "duckdb not available")
	})

	t.Run("happy: real db executes valid DDL", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)

		res, err := dbl.ExecContext(context.Background(), "CREATE TABLE test_exec (id INT, name VARCHAR)")
		require.NoError(t, err)
		require.NotNil(t, res)
		n, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("edge: invalid SQL on real db returns error", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)

		res, err := dbl.ExecContext(context.Background(), "GARBAGE STATEMENT")
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

// =============================================================================
// IsAvailable
// =============================================================================

func TestIsAvailable(t *testing.T) {
	t.Run("happy: returns true when db is set", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Cleanup() })
		dbl := NewDuckDBLayer(db)
		assert.True(t, dbl.IsAvailable())
	})

	t.Run("happy: returns false when db is nil", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		assert.False(t, dbl.IsAvailable())
	})

	t.Run("edge: consistent across concurrent access", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				assert.False(t, dbl.IsAvailable())
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// =============================================================================
// SchemaContext
// =============================================================================

func TestSchemaContext(t *testing.T) {
	t.Run("happy: wraps context for valid project ID", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		ctx := dbl.SchemaContext(context.Background(), "proj-123")
		assert.NotNil(t, ctx)
		assert.NotEqual(t, context.Background(), ctx, "should return wrapped context")
	})

	t.Run("edge: empty projectID returns original context", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		orig := context.WithValue(context.Background(), "key", "val")
		ctx := dbl.SchemaContext(orig, "")
		assert.Equal(t, orig, ctx)
	})

	t.Run("edge: invalid projectID falls back to original context", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		orig := context.WithValue(context.Background(), "test_key", "test_val")
		ctx := dbl.SchemaContext(orig, "bad/project/id/with/slashes/../../etc")
		assert.NotNil(t, ctx)
		assert.Equal(t, "test_val", ctx.Value("test_key"))
	})
}

// TestSyntheticDuckDBLayer — already defined in sprint_final_test.go
// TestSyntheticRowCount — already defined in sprint_final_test.go
