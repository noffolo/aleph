package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentHandler_ListModels_DefaultURL_UnavailableV2(t *testing.T) {
	h := &AgentHandler{ollamaBaseURL: ""}
	req := connect.NewRequest(&v1.ListModelsRequest{})
	_, err := h.ListModels(context.Background(), req)
	require.Error(t, err)
}

func TestIngestionHandler_NewIngestionHandler_NotNil(t *testing.T) {
	h := NewIngestionHandler("/tmp", nil, nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/tmp", h.projectsRoot)
}

func TestIngestionHandler_GetTaskLogs_LogNotFound(t *testing.T) {
	h := &IngestionHandler{projectsRoot: t.TempDir(), metaRepo: setupMetaRepoExtended(t)}
	resp, err := h.GetTaskLogs(context.Background(), connect.NewRequest(&v1.GetTaskLogsRequest{
		ProjectId: "p1", TaskId: "nonexistent",
	}))
	require.NoError(t, err)
	assert.Equal(t, "No logs found.", resp.Msg.Logs)
}

func TestSSEHandler_Stream_UnauthenticatedV2(t *testing.T) {
	h := NewSSEHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	h.Stream(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProjectHandler_DeleteProject_NotFoundV2(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)

	_, err := h.DeleteProject(context.Background(), connect.NewRequest(&v1.DeleteProjectRequest{
		Id: "nonexistent",
	}))
	require.Error(t, err)
}

func TestProjectHandler_SaveOntology_ProjectNotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)

	_, err := h.SaveOntology(context.Background(), connect.NewRequest(&v1.SaveOntologyRequest{
		ProjectId:       "nonexistent",
		AlephDefinition: `{"objects":[]}`,
	}))
	require.Error(t, err)
}

func TestProjectHandler_GetOntology_ProjectNotFound(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewProjectHandler("/tmp", nil)
	h.SetMetaRepo(repo)

	_, err := h.GetOntology(context.Background(), connect.NewRequest(&v1.GetOntologyRequest{
		ProjectId: "nonexistent",
	}))
	require.Error(t, err)
}

func TestProjectHandler_EmergeOntology_NilDB(t *testing.T) {
	t.Skip("requires full DuckDB setup, would panic on nil db")
}

func TestToolHandler_ListTools_EmptyV2(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	resp, err := h.ListTools(context.Background(), connect.NewRequest(&v1.ListToolsRequest{}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Tools, 0)
}

func TestToolHandler_CreateTool_AutoID(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewToolHandler("/tmp", repo)

	resp, err := h.CreateTool(context.Background(), connect.NewRequest(&v1.CreateToolRequest{
		Tool: &v1.Tool{
			Name:        "auto-id-tool",
			Description: "Test tool with auto-generated ID",
			Code:        "execute()",
		},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Tool.Id)
	assert.Equal(t, "auto-id-tool", resp.Msg.Tool.Name)
}

func TestSkillHandler_DeleteSkill_NotFoundV2(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewSkillHandler("/tmp", repo)

	resp, err := h.DeleteSkill(context.Background(), connect.NewRequest(&v1.DeleteSkillRequest{
		Id: "nonexistent", ProjectId: "p1",
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestToolExecutor_ExecuteTool_UnknownV2(t *testing.T) {
	e := &toolExecutor{}
	result, needsConf, err := e.ExecuteTool(context.Background(), "unknown_tool", map[string]any{}, "p1", "a1")
	assert.NoError(t, err)
	assert.True(t, needsConf)
	assert.Contains(t, result, "unknown_tool")
}

func TestToolExecutor_ExecuteSearchData_MissingObject(t *testing.T) {
	e := &toolExecutor{
		executeQuery: func(ctx context.Context, req *connect.Request[v1.ExecuteQueryRequest]) (*connect.Response[v1.ExecuteQueryResponse], error) {
			return connect.NewResponse(&v1.ExecuteQueryResponse{}), nil
		},
	}
	_, _, err := e.executeSearchData(context.Background(), map[string]any{}, "p1")
	require.Error(t, err)
}

func TestCodeFlowHandler_NewCodeFlowNil(t *testing.T) {
	h := NewCodeFlowHandler(nil)
	assert.NotNil(t, h)
}

func TestBreakerClient_StreamPredictions_NilClientV2(t *testing.T) {
	cb := NewCircuitBreakerClient(nil, nil)
	_, err := cb.StreamPredictions(context.Background(), connect.NewRequest(&nlpv1.StreamPredictionsRequest{}))
	require.Error(t, err)
}

func TestBreakerClient_AnalyzeSentiment_NilClientV2(t *testing.T) {
	cb := NewCircuitBreakerClient(nil, nil)
	_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{Text: "test"}))
	require.Error(t, err)
}

func TestIngestionHandler_RunTask_WithTaskRecord(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	repo.CreateTask(&repository.IngestionTaskRecord{
		ID: "t-run-1", ProjectID: "p1", Name: "run-task", SourceType: "csv",
		ConfigJSON: `{}`, Schedule: "* * *", Status: "idle", Progress: 0,
	})
	h := &IngestionHandler{projectsRoot: t.TempDir(), metaRepo: repo}

	resp, err := h.RunTask(context.Background(), connect.NewRequest(&v1.RunTaskRequest{
		ProjectId: "p1", TaskId: "t-run-1",
	}))
	require.NoError(t, err)
	assert.Equal(t, "started", resp.Msg.Status)
}

func TestIngestionHandler_CreateTask_WithAutoID(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := &IngestionHandler{metaRepo: repo}

	resp, err := h.CreateTask(context.Background(), connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "p1",
		Task:      &v1.IngestionTask{Name: "auto-task", SourceType: "rss", Schedule: "@daily"},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Task.Id)
}

func TestAgentHandler_CreateAgent_WithMaxLimit(t *testing.T) {
	repo := setupMetaRepoExtended(t)
	h := NewAgentHandler("/tmp", repo, "")
	h.SetMaxAgentsPerProject(2)

	_, err := h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Name: "a0", Provider: "ollama", Model: "llama3"},
	}))
	require.NoError(t, err)
	_, err = h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Name: "a1", Provider: "ollama", Model: "llama3"},
	}))
	require.NoError(t, err)

	_, err = h.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "p1",
		Agent:     &v1.Agent{Name: "a2", Provider: "ollama", Model: "llama3"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit reached")
}
