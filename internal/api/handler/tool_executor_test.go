package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockExecuteQuery(res *connect.Response[alephv1.ExecuteQueryResponse], err error) func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error) {
	return func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error) {
		return res, err
	}
}

func TestNewHandlerToolExecutor(t *testing.T) {
	exec := NewHandlerToolExecutor(nil, nil, nil)
	assert.NotNil(t, exec)
	_, ok := exec.(decision.ToolExecutor)
	assert.True(t, ok)
}

func TestToolExecutor_ExecuteTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]any
		wantErr  bool
		wantCfm  bool
	}{
		{name: "unknown tool requires confirmation", toolName: "unknown_tool", args: map[string]any{}, wantErr: false, wantCfm: true},
		{name: "search_data missing param", toolName: "search_data", args: map[string]any{}, wantErr: true},
		{name: "analyze_sentiment missing param", toolName: "analyze_sentiment", args: map[string]any{}, wantErr: true},
		{name: "get_trust_score missing param", toolName: "get_trust_score", args: map[string]any{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewHandlerToolExecutor(nil, nil, nil).(*toolExecutor)
			_, needsConfirm, err := exec.ExecuteTool(context.Background(), tt.toolName, tt.args, "proj-1", "agent-1")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCfm, needsConfirm)
		})
	}
}

func TestToolExecutor_ExecuteSearchData(t *testing.T) {
	mockResp := connect.NewResponse(&alephv1.ExecuteQueryResponse{
		Rows: []*alephv1.Row{
			{Values: map[string]string{"name": "alice"}},
		},
	})
	exec := NewHandlerToolExecutor(mockExecuteQuery(mockResp, nil), nil, nil).(*toolExecutor)

	result, needsConfirm, err := exec.ExecuteTool(context.Background(), "search_data", map[string]any{
		"object_name": "users",
		"limit":       float64(5),
	}, "proj-1", "agent-1")

	require.NoError(t, err)
	assert.False(t, needsConfirm)
	assert.Contains(t, result, "alice")
}

func TestToolExecutor_ExecuteSearchData_QueryError(t *testing.T) {
	exec := NewHandlerToolExecutor(mockExecuteQuery(nil, assert.AnError), nil, nil).(*toolExecutor)

	_, _, err := exec.ExecuteTool(context.Background(), "search_data", map[string]any{
		"object_name": "users",
	}, "proj-1", "agent-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error")
}

func TestToolExecutor_ExecuteAnalyzeSentiment_NilNLP(t *testing.T) {
	exec := NewHandlerToolExecutor(nil, nil, nil).(*toolExecutor)

	result, needsConfirm, err := exec.ExecuteTool(context.Background(), "analyze_sentiment", map[string]any{
		"text": "test message",
	}, "", "")

	require.NoError(t, err)
	assert.False(t, needsConfirm)
	assert.Contains(t, result, "unavailable")
}

func TestToolExecutor_ExecuteGetTrustScore_NilReg(t *testing.T) {
	exec := NewHandlerToolExecutor(nil, nil, nil).(*toolExecutor)

	result, needsConfirm, err := exec.ExecuteTool(context.Background(), "get_trust_score", map[string]any{
		"entity_id": "entity-1",
	}, "", "")

	require.NoError(t, err)
	assert.False(t, needsConfirm)
	assert.Contains(t, result, "unavailable")
}

func TestToolExecutor_ResultTruncation(t *testing.T) {
	longVal := make([]byte, 3000)
	for i := range longVal {
		longVal[i] = 'A'
	}
	mockResp := connect.NewResponse(&alephv1.ExecuteQueryResponse{
		Rows: []*alephv1.Row{
			{Values: map[string]string{"data": string(longVal)}},
		},
	})
	exec := NewHandlerToolExecutor(mockExecuteQuery(mockResp, nil), nil, nil).(*toolExecutor)

	result, _, err := exec.ExecuteTool(context.Background(), "search_data", map[string]any{
		"object_name": "test",
	}, "", "")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result), 2100)
	assert.Contains(t, result, "truncated")
}

func TestToolExecutor_InterfaceCompliance(t *testing.T) {
	exec := NewHandlerToolExecutor(nil, nil, nil)
	var _ decision.ToolExecutor = exec
	assert.NotNil(t, exec)
}
