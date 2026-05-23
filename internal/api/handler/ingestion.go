package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/errors"
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
	progress, err := h.metaRepo.GetTaskProgress(req.Msg.TaskId)
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
	tasks, err := h.metaRepo.ListTasks(projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var result []*v1.IngestionTask
	for _, t := range tasks {
		result = append(result, &v1.IngestionTask{
			Id: t.ID, Name: t.Name, SourceType: t.SourceType,
			ConfigJson: t.ConfigJSON, Schedule: t.Schedule,
			Status: t.Status, Progress: t.Progress,
		})
	}
	return connect.NewResponse(&v1.ListTasksResponse{Tasks: result}), nil
}

func (h *IngestionHandler) CreateTask(
	ctx context.Context,
	req *connect.Request[v1.CreateTaskRequest],
) (*connect.Response[v1.CreateTaskResponse], error) {
	projectID := req.Msg.ProjectId
	task := req.Msg.Task
	if task.Id == "" {
		b := make([]byte, 16)
		rand.Read(b)
		task.Id = fmt.Sprintf("%x", b)
	}

	err := h.metaRepo.CreateTask(&repository.IngestionTaskRecord{
		ID: task.Id, ProjectID: projectID, Name: task.Name,
		SourceType: task.SourceType, ConfigJSON: task.ConfigJson,
		Schedule: task.Schedule, Status: "idle", Progress: 0,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreateTaskResponse{Task: task}), nil
}

func (h *IngestionHandler) RunTask(
	ctx context.Context,
	req *connect.Request[v1.RunTaskRequest],
) (*connect.Response[v1.RunTaskResponse], error) {
	projectID := req.Msg.ProjectId
	taskID := req.Msg.TaskId

	t, err := h.metaRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.NewAPIErrorWithMeta(
			errors.ErrNotFound, "ingestion task not found", err,
			"ingestion", "query", false, 0,
		))
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ingestion RunTask goroutine panic", "projectID", projectID, "taskID", t.ID, "recover", r)
			}
		}()
		configJSON := t.ConfigJSON
		if overrides := req.Msg.ConfigOverrides; overrides != nil && *overrides != "" {
			merged, err := mergeConfigOverrides(t.ConfigJSON, *overrides)
			if err != nil {
				slog.Error("failed to merge config_overrides", "taskID", t.ID, "error", err)
			} else {
				configJSON = merged
			}
		}
		v1Task := &v1.IngestionTask{Id: t.ID, Name: t.Name, SourceType: t.SourceType, ConfigJson: configJSON}
		taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		if err := h.engine.RunTask(taskCtx, projectID, v1Task); err != nil {
			slog.Error("ingestion task failed", "projectID", projectID, "taskID", v1Task.Id, "error", err)
		}
	}()

	return connect.NewResponse(&v1.RunTaskResponse{Status: "started"}), nil
}

func (h *IngestionHandler) GetTaskLogs(
	ctx context.Context,
	req *connect.Request[v1.GetTaskLogsRequest],
) (*connect.Response[v1.GetTaskLogsResponse], error) {
	projectID := req.Msg.ProjectId
	taskID := req.Msg.TaskId
	logPath, err := sanitizePath(h.projectsRoot, projectID, "logs", taskID+".log")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return connect.NewResponse(&v1.GetTaskLogsResponse{Logs: "No logs found."}), nil
	}
	return connect.NewResponse(&v1.GetTaskLogsResponse{Logs: string(data)}), nil
}

func (h *IngestionHandler) DeleteTask(
	ctx context.Context,
	req *connect.Request[v1.DeleteTaskRequest],
) (*connect.Response[v1.DeleteTaskResponse], error) {
	err := h.metaRepo.DeleteTask(req.Msg.Id, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteTaskResponse{Success: true}), nil
}

// mergeConfigOverrides shallow-merges the overrides JSON into the base config JSON.
// overrides fields replace base fields at the top level. Returns the merged JSON string.
func mergeConfigOverrides(baseJSON, overridesJSON string) (string, error) {
	var base map[string]any
	if err := json.Unmarshal([]byte(baseJSON), &base); err != nil {
		return baseJSON, fmt.Errorf("unmarshal base config: %w", err)
	}
	var overrides map[string]any
	if err := json.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		return baseJSON, fmt.Errorf("unmarshal config_overrides: %w", err)
	}
	for k, v := range overrides {
		base[k] = v
	}
	merged, err := json.Marshal(base)
	if err != nil {
		return baseJSON, fmt.Errorf("marshal merged config: %w", err)
	}
	return string(merged), nil
}
