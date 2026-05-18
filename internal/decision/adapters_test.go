package decision

import (
	"context"
	"errors"
	"testing"

	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMetaRepo struct {
	saveMsgFn   func(ctx context.Context, projectID, agentID, role, content, toolCall string) error
	getMsgsFn   func(ctx context.Context, projectID, agentID string) ([]repository.ChatMessage, error)
	listToolsFn func(ctx context.Context) ([]repository.ToolRecord, error)
}

func (m *mockMetaRepo) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	if m.saveMsgFn != nil {
		return m.saveMsgFn(ctx, projectID, agentID, role, content, toolCall)
	}
	return nil
}

func (m *mockMetaRepo) GetChatMessages(ctx context.Context, projectID, agentID string) ([]repository.ChatMessage, error) {
	if m.getMsgsFn != nil {
		return m.getMsgsFn(ctx, projectID, agentID)
	}
	return nil, nil
}

func (m *mockMetaRepo) ListTools() ([]repository.ToolRecord, error) {
	if m.listToolsFn != nil {
		return m.listToolsFn(context.Background())
	}
	return nil, nil
}

func TestMetaRepoAdapter_SaveChatMessage(t *testing.T) {
	t.Parallel()
	var called bool
	repo := &mockMetaRepo{
		saveMsgFn: func(_ context.Context, projectID, agentID, role, content, toolCall string) error {
			called = true
			assert.Equal(t, "proj-1", projectID)
			assert.Equal(t, "agent-1", agentID)
			assert.Equal(t, "user", role)
			assert.Equal(t, "hello", content)
			assert.Equal(t, "call-1", toolCall)
			return nil
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	err := adapter.SaveChatMessage(context.Background(), "proj-1", "agent-1", "user", "hello", "call-1")
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestMetaRepoAdapter_SaveChatMessage_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("db error")
	repo := &mockMetaRepo{
		saveMsgFn: func(_ context.Context, _, _, _, _, _ string) error {
			return expectedErr
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	err := adapter.SaveChatMessage(context.Background(), "p", "a", "user", "msg", "")
	assert.ErrorIs(t, err, expectedErr)
}

func TestMetaRepoAdapter_GetChatMessages_Empty(t *testing.T) {
	t.Parallel()
	repo := &mockMetaRepo{
		getMsgsFn: func(_ context.Context, _, _ string) ([]repository.ChatMessage, error) {
			return []repository.ChatMessage{}, nil
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	msgs, err := adapter.GetChatMessages(context.Background(), "proj-1", "agent-1")
	assert.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestMetaRepoAdapter_GetChatMessages_Conversion(t *testing.T) {
	t.Parallel()
	repoMsgs := []repository.ChatMessage{
		{Role: "user", Content: "hello", ToolCall: ""},
		{Role: "assistant", Content: "response", ToolCall: "search_data"},
	}
	repo := &mockMetaRepo{
		getMsgsFn: func(_ context.Context, _, _ string) ([]repository.ChatMessage, error) {
			return repoMsgs, nil
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	msgs, err := adapter.GetChatMessages(context.Background(), "proj-1", "agent-1")
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "hello", msgs[0].Content)
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "response", msgs[1].Content)
	assert.Equal(t, "search_data", msgs[1].ToolCall)
}

func TestMetaRepoAdapter_GetChatMessages_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("db error")
	repo := &mockMetaRepo{
		getMsgsFn: func(_ context.Context, _, _ string) ([]repository.ChatMessage, error) {
			return nil, expectedErr
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	msgs, err := adapter.GetChatMessages(context.Background(), "p", "a")
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, msgs)
}

func TestMetaRepoAdapter_ListTools_Empty(t *testing.T) {
	t.Parallel()
	repo := &mockMetaRepo{
		listToolsFn: func(_ context.Context) ([]repository.ToolRecord, error) {
			return []repository.ToolRecord{}, nil
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	tools, err := adapter.ListTools(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

func TestMetaRepoAdapter_ListTools_Conversion(t *testing.T) {
	t.Parallel()
	records := []repository.ToolRecord{
		{Name: "search_data", Description: "Search tool", Code: `{"type":"object"}`},
		{Name: "analyze", Description: "Analysis tool", Code: ""},
	}
	repo := &mockMetaRepo{
		listToolsFn: func(_ context.Context) ([]repository.ToolRecord, error) {
			return records, nil
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	tools, err := adapter.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 2)
	assert.Equal(t, "search_data", tools[0].Name)
	assert.Equal(t, "Search tool", tools[0].Description)
	assert.Equal(t, `{"type":"object"}`, tools[0].Code)
	assert.Equal(t, "analyze", tools[1].Name)
	assert.Equal(t, "Analysis tool", tools[1].Description)
}

func TestMetaRepoAdapter_ListTools_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("list error")
	repo := &mockMetaRepo{
		listToolsFn: func(_ context.Context) ([]repository.ToolRecord, error) {
			return nil, expectedErr
		},
	}
	adapter := &MetaRepoAdapter{Repo: repo}
	tools, err := adapter.ListTools(context.Background())
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, tools)
}

type mockRegistryClient struct {
	getCompFn func(ctx context.Context, id string) (*registry.ComponentMetadata, error)
}

func (m *mockRegistryClient) GetComponentByID(ctx context.Context, id string) (*registry.ComponentMetadata, error) {
	if m.getCompFn != nil {
		return m.getCompFn(ctx, id)
	}
	return nil, nil
}

func TestRegistryAdapter_GetComponentByID_NotFound(t *testing.T) {
	t.Parallel()
	reg := &mockRegistryClient{
		getCompFn: func(_ context.Context, _ string) (*registry.ComponentMetadata, error) {
			return nil, nil
		},
	}
	adapter := &RegistryAdapter{Reg: reg}
	comp, err := adapter.GetComponentByID(context.Background(), "missing")
	assert.NoError(t, err)
	assert.Nil(t, comp)
}

func TestRegistryAdapter_GetComponentByID_Success(t *testing.T) {
	t.Parallel()
	record := &registry.ComponentMetadata{
		ID:       "comp-1",
		Name:     "Test Component",
		Category: "tool",
		Status:   "active",
	}
	reg := &mockRegistryClient{
		getCompFn: func(_ context.Context, id string) (*registry.ComponentMetadata, error) {
			assert.Equal(t, "comp-1", id)
			return record, nil
		},
	}
	adapter := &RegistryAdapter{Reg: reg}
	comp, err := adapter.GetComponentByID(context.Background(), "comp-1")
	require.NoError(t, err)
	require.NotNil(t, comp)
	assert.Equal(t, "comp-1", comp.ID)
	assert.Equal(t, "Test Component", comp.Name)
	assert.Equal(t, "tool", comp.Category)
	assert.Equal(t, "active", comp.Status)
}

func TestRegistryAdapter_GetComponentByID_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("registry error")
	reg := &mockRegistryClient{
		getCompFn: func(_ context.Context, _ string) (*registry.ComponentMetadata, error) {
			return nil, expectedErr
		},
	}
	adapter := &RegistryAdapter{Reg: reg}
	comp, err := adapter.GetComponentByID(context.Background(), "comp-1")
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, comp)
}
