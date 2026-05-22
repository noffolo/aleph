package handler

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"connectrpc.com/connect"
)

func setupIngestionMetaRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE system_tasks (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		name TEXT,
		source_type TEXT,
		config_json TEXT,
		schedule TEXT,
		status TEXT,
		progress INT
	)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

func seedIngestionTask(t *testing.T, repo *repository.MetadataRepository, id, projectID, name, status string, progress int32) {
	t.Helper()
	err := repo.CreateTask(&repository.IngestionTaskRecord{
		ID:         id,
		ProjectID:  projectID,
		Name:       name,
		SourceType: "csv",
		ConfigJSON: `{"path":"/data/test.csv"}`,
		Schedule:   "daily",
		Status:     status,
		Progress:   progress,
	})
	require.NoError(t, err)
}

// ─── GetProgress tests ──────────────────────────────────────────────────────

func TestIngestionHandler_GetProgress_HappyPath(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "task-1", "proj-1", "Test Task", "running", 50)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.GetProgressRequest{TaskId: "task-1"})
	resp, err := h.GetProgress(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int32(50), resp.Msg.Progress)
}

func TestIngestionHandler_GetProgress_MissingTask(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.GetProgressRequest{TaskId: "nonexistent"})
	_, err := h.GetProgress(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestIngestionHandler_GetProgress_ZeroProgress(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "task-init", "proj-1", "Init Task", "idle", 0)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.GetProgressRequest{TaskId: "task-init"})
	resp, err := h.GetProgress(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Msg.Progress)
}

// ─── ListTasks tests ────────────────────────────────────────────────────────

func TestIngestionHandler_ListTasks_HappyPath(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "t1", "proj-1", "Task A", "running", 30)
	seedIngestionTask(t, repo, "t2", "proj-1", "Task B", "done", 100)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.ListTasksRequest{ProjectId: "proj-1"})
	resp, err := h.ListTasks(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Msg.Tasks, 2)
	assert.Equal(t, "t1", resp.Msg.Tasks[0].Id)
	assert.Equal(t, "Task B", resp.Msg.Tasks[1].Name)
}

func TestIngestionHandler_ListTasks_EmptyProject(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.ListTasksRequest{ProjectId: "empty-proj"})
	resp, err := h.ListTasks(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Msg.Tasks, 0)
}

func TestIngestionHandler_ListTasks_MultipleProjects(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "a", "proj-1", "P1 Task", "idle", 0)
	seedIngestionTask(t, repo, "b", "proj-2", "P2 Task", "idle", 0)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.ListTasksRequest{ProjectId: "proj-1"})
	resp, err := h.ListTasks(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Msg.Tasks, 1)
	assert.Equal(t, "P1 Task", resp.Msg.Tasks[0].Name)
}

// ─── CreateTask tests ──────────────────────────────────────────────────────

func TestIngestionHandler_CreateTask_HappyPath(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "proj-1",
		Task: &v1.IngestionTask{
			Id:         "task-abc",
			Name:       "New Task",
			SourceType: "csv",
			ConfigJson: `{"path":"/data/in.csv"}`,
			Schedule:   "hourly",
		},
	})
	resp, err := h.CreateTask(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp.Msg.Task)
	assert.Equal(t, "task-abc", resp.Msg.Task.Id)
	assert.Equal(t, "New Task", resp.Msg.Task.Name)
}

func TestIngestionHandler_CreateTask_EmptyID_GeneratesAuto(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "proj-1",
		Task: &v1.IngestionTask{
			Name:       "Auto Task",
			SourceType: "api",
			ConfigJson: `{"url":"https://example.com/data.json"}`,
			Schedule:   "manual",
		},
	})
	resp, err := h.CreateTask(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Task.Id, "auto-generated ID must not be empty")
	assert.Len(t, resp.Msg.Task.Id, 32, "hex-encoded 16 bytes = 32 hex chars")
}

func TestIngestionHandler_CreateTask_DuplicateID(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "dup-id", "proj-1", "Existing", "idle", 0)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "proj-1",
		Task: &v1.IngestionTask{
			Id:         "dup-id",
			Name:       "Duplicate",
			SourceType: "csv",
			ConfigJson: `{}`,
			Schedule:   "daily",
		},
	})
	_, err := h.CreateTask(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

// ─── RunTask tests ─────────────────────────────────────────────────────────

func TestIngestionHandler_RunTask_HappyPath(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "run-me", "proj-1", "Run Test", "idle", 0)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.RunTaskRequest{
		ProjectId: "proj-1",
		TaskId:    "run-me",
	})
	resp, err := h.RunTask(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "started", resp.Msg.Status)
}

func TestIngestionHandler_RunTask_NotFound(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.RunTaskRequest{
		ProjectId: "proj-1",
		TaskId:    "no-such-task",
	})
	_, err := h.RunTask(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestIngestionHandler_RunTask_EmptyRequest(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.RunTaskRequest{
		ProjectId: "",
		TaskId:    "",
	})
	_, err := h.RunTask(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// ─── GetTaskLogs tests ─────────────────────────────────────────────────────

func TestIngestionHandler_GetTaskLogs_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "proj-1", "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))
	logPath := filepath.Join(logsDir, "log-me.log")
	require.NoError(t, os.WriteFile(logPath, []byte("line 1\nline 2\nerror: something\n"), 0644))

	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{
		projectsRoot: tmpDir,
		metaRepo:     repo,
	}

	req := connect.NewRequest(&v1.GetTaskLogsRequest{
		ProjectId: "proj-1",
		TaskId:    "log-me",
	})
	resp, err := h.GetTaskLogs(context.Background(), req)

	require.NoError(t, err)
	assert.Contains(t, resp.Msg.Logs, "line 1")
	assert.Contains(t, resp.Msg.Logs, "error: something")
}

func TestIngestionHandler_GetTaskLogs_NoFile_ReturnsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{
		projectsRoot: tmpDir,
		metaRepo:     repo,
	}

	req := connect.NewRequest(&v1.GetTaskLogsRequest{
		ProjectId: "proj-1",
		TaskId:    "no-log",
	})
	resp, err := h.GetTaskLogs(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "No logs found.", resp.Msg.Logs)
}

func TestIngestionHandler_GetTaskLogs_EmptyLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "proj-1", "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))
	logPath := filepath.Join(logsDir, "empty.log")
	require.NoError(t, os.WriteFile(logPath, []byte(""), 0644))

	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{
		projectsRoot: tmpDir,
		metaRepo:     repo,
	}

	req := connect.NewRequest(&v1.GetTaskLogsRequest{
		ProjectId: "proj-1",
		TaskId:    "empty",
	})
	resp, err := h.GetTaskLogs(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "", resp.Msg.Logs)
}

// ─── DeleteTask tests ──────────────────────────────────────────────────────

func TestIngestionHandler_DeleteTask_HappyPath(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "del-me", "proj-1", "Delete Test", "done", 100)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.DeleteTaskRequest{
		ProjectId: "proj-1",
		Id:        "del-me",
	})
	resp, err := h.DeleteTask(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

func TestIngestionHandler_DeleteTask_NonExistent(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.DeleteTaskRequest{
		ProjectId: "proj-1",
		Id:        "no-such",
	})
	resp, err := h.DeleteTask(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Msg.Success, "delete should return OK even for nonexistent")
}

func TestIngestionHandler_DeleteTask_DifferentProject(t *testing.T) {
	repo := setupIngestionMetaRepo(t)
	seedIngestionTask(t, repo, "task-x", "proj-A", "Task X", "idle", 0)
	h := &IngestionHandler{metaRepo: repo}

	req := connect.NewRequest(&v1.DeleteTaskRequest{
		ProjectId: "proj-B",
		Id:        "task-x",
	})
	resp, err := h.DeleteTask(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)

	// Verify task still exists in proj-A
	listReq := connect.NewRequest(&v1.ListTasksRequest{ProjectId: "proj-A"})
	listResp, listErr := h.ListTasks(context.Background(), listReq)
	require.NoError(t, listErr)
	assert.Len(t, listResp.Msg.Tasks, 1)
}

func TestNewIngestionHandler(t *testing.T) {
	h := NewIngestionHandler("/tmp/projects", nil, &repository.MetadataRepository{})
	assert.NotNil(t, h)
	assert.Equal(t, "/tmp/projects", h.projectsRoot)
}
