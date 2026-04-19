package handler

import (
	"context"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"connectrpc.com/connect"
	"net/http"
	"encoding/json"
)

type AgentHandler struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
}

func NewAgentHandler(projectsRoot string, metaRepo *repository.MetadataRepository) *AgentHandler {
	return &AgentHandler{projectsRoot: projectsRoot, metaRepo: metaRepo}
}

func (h *AgentHandler) ListAgents(
	ctx context.Context,
	req *connect.Request[v1.ListAgentsRequest],
) (*connect.Response[v1.ListAgentsResponse], error) {
	projectID := req.Msg.ProjectId
	rows, err := h.metaRepo.DB().Query("SELECT id, name, model, system_prompt FROM system_agents WHERE project_id = $1", projectID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	defer rows.Close()

	var agents []*v1.Agent
	for rows.Next() {
		var a v1.Agent
		rows.Scan(&a.Id, &a.Name, &a.Model, &a.SystemPrompt)
		agents = append(agents, &a)
	}
	return connect.NewResponse(&v1.ListAgentsResponse{Agents: agents}), nil
}

func (h *AgentHandler) CreateAgent(
	ctx context.Context,
	req *connect.Request[v1.CreateAgentRequest],
) (*connect.Response[v1.CreateAgentResponse], error) {
	projectID := req.Msg.ProjectId
	agent := req.Msg.Agent
	_, err := h.metaRepo.DB().Exec(
		"INSERT INTO system_agents (id, project_id, name, model, system_prompt) VALUES ($1, $2, $3, $4, $5)",
		agent.Id, projectID, agent.Name, agent.Model, agent.SystemPrompt,
	)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.CreateAgentResponse{Agent: agent}), nil
}


func (h *AgentHandler) ListModels(
	ctx context.Context,
	req *connect.Request[v1.ListModelsRequest],
) (*connect.Response[v1.ListModelsResponse], error) {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil { return nil, connect.NewError(connect.CodeUnavailable, err) }
	defer resp.Body.Close()
	var ollamaResp struct {
		Models []struct { Name string `json:"name"` } `json:"models"`
	}
	json.NewDecoder(resp.Body).Decode(&ollamaResp)
	var models []string
	for _, m := range ollamaResp.Models { models = append(models, m.Name) }
	return connect.NewResponse(&v1.ListModelsResponse{Models: models}), nil
}

func (h *AgentHandler) DeleteAgent(
	ctx context.Context,
	req *connect.Request[v1.DeleteAgentRequest],
) (*connect.Response[v1.DeleteAgentResponse], error) {
	projectID := req.Msg.ProjectId
	id := req.Msg.Id
	_, err := h.metaRepo.DB().Exec("DELETE FROM system_agents WHERE project_id = $1 AND id = $2", projectID, id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteAgentResponse{Success: true}), nil
}
