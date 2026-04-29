package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/ssrf"
)

type AgentHandler struct {
	projectsRoot  string
	metaRepo      *repository.MetadataRepository
	ollamaBaseURL string
}

func NewAgentHandler(projectsRoot string, metaRepo *repository.MetadataRepository, ollamaBaseURL string) *AgentHandler {
	return &AgentHandler{projectsRoot: projectsRoot, metaRepo: metaRepo, ollamaBaseURL: ollamaBaseURL}
}

func (h *AgentHandler) ListAgents(
	ctx context.Context,
	req *connect.Request[v1.ListAgentsRequest],
) (*connect.Response[v1.ListAgentsResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agentRecs, err := h.metaRepo.ListAgents(projectID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	var agents []*v1.Agent
	for _, a := range agentRecs {
		agent := &v1.Agent{
			Id: a.ID, Name: a.Name, Provider: a.Provider, Model: a.Model,
			SystemPrompt: a.SystemPrompt, BaseUrl: a.BaseURL,
		}
		if a.ApiKey != "" {
			runes := []rune(a.ApiKey)
			if len(runes) > 8 { agent.ApiKey = string(runes[:8]) + "****" } else { agent.ApiKey = "****" }
		}
		if a.SkillIDsJSON != "" {
			json.Unmarshal([]byte(a.SkillIDsJSON), &agent.SkillIds)
		}
		agents = append(agents, agent)
	}
	return connect.NewResponse(&v1.ListAgentsResponse{Agents: agents}), nil
}

func (h *AgentHandler) CreateAgent(
	ctx context.Context,
	req *connect.Request[v1.CreateAgentRequest],
) (*connect.Response[v1.CreateAgentResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agent := req.Msg.Agent
	if agent == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agent is required"))
	}
	if agent.Id == "" {
		agent.Id = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	skillIDsJSON, _ := json.Marshal(agent.SkillIds)
	rec := &repository.AgentRecord{
		ID: agent.Id, ProjectID: projectID, Name: agent.Name, Provider: agent.Provider,
		Model: agent.Model, ApiKey: agent.ApiKey, SystemPrompt: agent.SystemPrompt,
		SkillIDsJSON: string(skillIDsJSON), BaseURL: agent.BaseUrl,
	}
	if err := h.metaRepo.CreateAgent(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if agent.ApiKey != "" {
		runes := []rune(agent.ApiKey)
		if len(runes) > 8 { agent.ApiKey = string(runes[:8]) + "****" } else { agent.ApiKey = "****" }
	}
	return connect.NewResponse(&v1.CreateAgentResponse{Agent: agent}), nil
}


func (h *AgentHandler) ListModels(
	ctx context.Context,
	req *connect.Request[v1.ListModelsRequest],
) (*connect.Response[v1.ListModelsResponse], error) {
	baseURL := h.ollamaBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := ssrf.NewClient()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	defer resp.Body.Close()

	var ollamaResp struct {
		Models []struct { Name string `json:"name"` } `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var models []string
	for _, m := range ollamaResp.Models {
		models = append(models, m.Name)
	}
	return connect.NewResponse(&v1.ListModelsResponse{Models: models}), nil
}

func (h *AgentHandler) DeleteAgent(
	ctx context.Context,
	req *connect.Request[v1.DeleteAgentRequest],
) (*connect.Response[v1.DeleteAgentResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	id := req.Msg.Id
	if err := h.metaRepo.DeleteAgent(id, projectID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteAgentResponse{Success: true}), nil
}

func (h *AgentHandler) UpdateAgent(
	ctx context.Context,
	req *connect.Request[v1.UpdateAgentRequest],
) (*connect.Response[v1.UpdateAgentResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agent := req.Msg.Agent
	if agent == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agent is required"))
	}
	skillIDsJSON, _ := json.Marshal(agent.SkillIds)
	rec := &repository.AgentRecord{
		ID: agent.Id, ProjectID: projectID, Name: agent.Name, Provider: agent.Provider,
		Model: agent.Model, ApiKey: agent.ApiKey, SystemPrompt: agent.SystemPrompt,
		SkillIDsJSON: string(skillIDsJSON), BaseURL: agent.BaseUrl,
	}
	if err := h.metaRepo.UpdateAgent(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if agent.ApiKey != "" {
		runes := []rune(agent.ApiKey)
		if len(runes) > 8 { agent.ApiKey = string(runes[:8]) + "****" } else { agent.ApiKey = "****" }
	}
	return connect.NewResponse(&v1.UpdateAgentResponse{Agent: agent}), nil
}
