package tracker

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrackingInterceptor(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	interceptor := NewTrackingInterceptor(tr)
	assert.NotNil(t, interceptor)
}

func TestWrapStreamingClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	interceptor := NewTrackingInterceptor(tr)

	called := false
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		called = true
		return nil
	}
	wrapped := interceptor.WrapStreamingClient(next)
	_ = wrapped(context.Background(), connect.Spec{Procedure: "test"})
	assert.True(t, called)
}

func TestWrapStreamingHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	interceptor := NewTrackingInterceptor(tr)

	called := false
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		called = true
		return nil
	}
	wrapped := interceptor.WrapStreamingHandler(next)
	err := wrapped(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWrapUnary_NoProjectID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	interceptor := NewTrackingInterceptor(tr)

	called := false
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	}
	wrapped := interceptor.WrapUnary(next)
	resp, err := wrapped(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, called)
	// resp may be nil because our next returns nil — that's fine
	_ = resp
}

func TestExtractToolName_NoLeadingSlash(t *testing.T) {
	spec := connect.Spec{Procedure: "aleph.v1.ToolService/ExecuteTool"}
	got := extractToolName(spec)
	assert.Equal(t, "toolservice.executeTool", got)
}

func TestExtractToolName_SingleSegment(t *testing.T) {
	spec := connect.Spec{Procedure: "ToolService"}
	got := extractToolName(spec)
	assert.Equal(t, "ToolService", got)
}

func TestExtractToolName_SingleSegmentWithSlash(t *testing.T) {
	spec := connect.Spec{Procedure: "/ToolService"}
	got := extractToolName(spec)
	assert.Equal(t, "ToolService", got)
}

func TestExtractToolName_MultiDotService(t *testing.T) {
	spec := connect.Spec{Procedure: "/com.example.v1.UserService/GetUser"}
	got := extractToolName(spec)
	assert.Equal(t, "userservice.getUser", got)
}

func TestLowerFirst(t *testing.T) {
	assert.Equal(t, "", lowerFirst(""))
	assert.Equal(t, "hello", lowerFirst("Hello"))
	assert.Equal(t, "world", lowerFirst("World"))
	assert.Equal(t, "a", lowerFirst("A"))
	assert.Equal(t, "123", lowerFirst("123"))
	assert.Equal(t, "_underscore", lowerFirst("_underscore"))
}

func TestRecord_AutoTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	usage := ToolUsage{
		ID:        "explicit-id",
		UserID:    "user1",
		ProjectID: "proj1",
		ToolName:  "auto_ts",
		Success:   true,
	}
	err := tr.Record(ctx, usage)
	require.NoError(t, err)

	var storedTs sql.NullString
	err = db.QueryRow("SELECT timestamp FROM tool_usage WHERE id = 'explicit-id'").Scan(&storedTs)
	require.NoError(t, err)
	assert.True(t, storedTs.Valid, "timestamp should be auto-filled when zero")
	assert.NotEmpty(t, storedTs.String)
}

func TestRecord_AutoID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	usage1 := ToolUsage{UserID: "user1", ProjectID: "p1", ToolName: "tool"}
	usage2 := ToolUsage{UserID: "user1", ProjectID: "p1", ToolName: "tool"}

	err := tr.Record(ctx, usage1)
	require.NoError(t, err)
	err = tr.Record(ctx, usage2)
	require.NoError(t, err)

	var id1, id2 string
	rows, err := db.Query("SELECT id FROM tool_usage WHERE user_id = 'user1' ORDER BY timestamp")
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	rows.Scan(&id1)
	require.True(t, rows.Next())
	rows.Scan(&id2)

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestMostUsedTools_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)

	var zeroTime time.Time
	stats, err := tr.MostUsedTools(context.Background(), "no_user", 10, zeroTime)
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestToolSequences_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)

	seqs, err := tr.ToolSequences(context.Background(), "no_user", 10)
	require.NoError(t, err)
	assert.Empty(t, seqs)
}

func TestToolSequences_SingleEntry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	err := tr.Record(ctx, ToolUsage{
		ID: "single-1", UserID: "u1", ProjectID: "p1",
		ToolName: "lonely", Success: true,
	})
	require.NoError(t, err)

	seqs, err := tr.ToolSequences(ctx, "u1", 10)
	require.NoError(t, err)
	require.Len(t, seqs, 1)
	assert.Equal(t, []string{"lonely"}, seqs[0])
}

func TestRecord_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	usage := ToolUsage{
		ID:        "err-test",
		UserID:    "user1",
		ProjectID: "proj1",
		ToolName:  "error_tool",
		Success:   false,
		ErrorMsg:  "something went wrong",
	}
	err := tr.Record(ctx, usage)
	require.NoError(t, err)

	var success bool
	var errMsg string
	err = db.QueryRow("SELECT success, error_msg FROM tool_usage WHERE id = 'err-test'").Scan(&success, &errMsg)
	require.NoError(t, err)
	assert.False(t, success)
	assert.Equal(t, "something went wrong", errMsg)
}
