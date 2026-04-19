package handler

import (
	"context"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type IngestionHandler struct {
	projectsRoot string
	engine       *ingestion.Engine
	metaRepo     *repository.MetadataRepository
}

func NewIngestionHandler(projectsRoot string, engine *ingestion.Engine, metaRepo *repository.MetadataRepository) *IngestionHandler {
	return &IngestionHandler{
		projectsRoot: projectsRoot,
		engine:       engine,
		metaRepo:     metaRepo,
	}
}

func (h *IngestionHandler) GetProgress(
	ctx context.Context,
	req *connect.Request[v1.GetProgressRequest],
) (*connect.Response[v1.GetProgressResponse], error) {
	taskID := req.Msg.TaskId
	var progress int32
	err := h.metaRepo.DB().QueryRow("SELECT progress FROM system_tasks WHERE id = $1", taskID).Scan(&progress)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.GetProgressResponse{Progress: progress}), nil
}

func (h *IngestionHandler) ListTasks(
	ctx context.Context,
	req *connect.Request[v1.ListTasksRequest],
) (*connect.Response[v1.ListTasksResponse], error) {
	projectID := req.Msg.ProjectId
	var tasks []*v1.IngestionTask
	rows, err := h.metaRepo.DB().Query("SELECT id, name, source_type, config_json, status, progress FROM system_tasks WHERE project_id = $1", projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t v1.IngestionTask
			rows.Scan(&t.Id, &t.Name, &t.SourceType, &t.ConfigJson, &t.Status, &t.Progress)
			tasks = append(tasks, &t)
		}
	}
	return connect.NewResponse(&v1.ListTasksResponse{Tasks: tasks}), nil
}

func (h *IngestionHandler) CreateTask(
	ctx context.Context,
	req *connect.Request[v1.CreateTaskRequest],
) (*connect.Response[v1.CreateTaskResponse], error) {
	projectID := req.Msg.ProjectId
	task := req.Msg.Task

	_, err := h.metaRepo.DB().Exec(
		"INSERT INTO system_tasks (id, project_id, name, source_type, config_json, status, progress) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		task.Id, projectID, task.Name, task.SourceType, task.ConfigJson, "idle", 0,
	)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	return connect.NewResponse(&v1.CreateTaskResponse{Task: task}), nil
}

func (h *IngestionHandler) RunTask(
	ctx context.Context,
	req *connect.Request[v1.RunTaskRequest],
) (*connect.Response[v1.RunTaskResponse], error) {
	projectID := req.Msg.ProjectId
	taskID := req.Msg.TaskId

	var task v1.IngestionTask
	err := h.metaRepo.DB().QueryRow("SELECT id, name, source_type, config_json FROM system_tasks WHERE id = $1", taskID).Scan(&task.Id, &task.Name, &task.SourceType, &task.ConfigJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	go func() {
		h.engine.RunTask(context.Background(), projectID, &task)
	}()

	return connect.NewResponse(&v1.RunTaskResponse{Status: "started"}), nil
}

func (h *IngestionHandler) GetTaskLogs(
	ctx context.Context,
	req *connect.Request[v1.GetTaskLogsRequest],
) (*connect.Response[v1.GetTaskLogsResponse], error) {
	projectID := req.Msg.ProjectId
	taskID := req.Msg.TaskId
	logPath := filepath.Join(h.projectsRoot, projectID, "logs", taskID+".log")
	data, err := os.ReadFile(logPath)
	if err != nil { return connect.NewResponse(&v1.GetTaskLogsResponse{Logs: "No logs found."}), nil }
	return connect.NewResponse(&v1.GetTaskLogsResponse{Logs: string(data)}), nil
}

func (h *IngestionHandler) DeleteTask(
	ctx context.Context,
	req *connect.Request[v1.DeleteTaskRequest],
) (*connect.Response[v1.DeleteTaskResponse], error) {
	projectID := req.Msg.ProjectId
	id := req.Msg.Id
	_, err := h.metaRepo.DB().Exec("DELETE FROM system_tasks WHERE project_id = $1 AND id = $2", projectID, id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteTaskResponse{Success: true}), nil
}
