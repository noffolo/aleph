package handler

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/google/uuid"
)

type SkillHandler struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
}

func NewSkillHandler(projectsRoot string, metaRepo *repository.MetadataRepository) *SkillHandler {
	return &SkillHandler{projectsRoot: projectsRoot, metaRepo: metaRepo}
}

func (h *SkillHandler) ListSkills(
	ctx context.Context,
	req *connect.Request[v1.ListSkillsRequest],
) (*connect.Response[v1.ListSkillsResponse], error) {
	projectID := req.Msg.ProjectId
	skills, err := h.metaRepo.ListSkills(projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var result []*v1.Skill
	for _, s := range skills {
		skill := &v1.Skill{Id: s.ID, Name: s.Name, Description: s.Description}
		if s.ToolIDsJSON != "" {
			json.Unmarshal([]byte(s.ToolIDsJSON), &skill.ToolIds)
		}
		result = append(result, skill)
	}
	return connect.NewResponse(&v1.ListSkillsResponse{Skills: result}), nil
}

func (h *SkillHandler) CreateSkill(
	ctx context.Context,
	req *connect.Request[v1.CreateSkillRequest],
) (*connect.Response[v1.CreateSkillResponse], error) {
	projectID := req.Msg.ProjectId
	skill := req.Msg.Skill
	if skill == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("skill is required"))
	}
	if skill.Id == "" {
		skill.Id = uuid.NewString()
	}
	toolIDs, _ := json.Marshal(skill.ToolIds)

	err := h.metaRepo.CreateSkill(&repository.SkillRecord{
		ID: skill.Id, ProjectID: projectID, Name: skill.Name,
		Description: skill.Description, ToolIDsJSON: string(toolIDs),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateSkillResponse{Skill: skill}), nil
}

func (h *SkillHandler) UpdateSkill(
	ctx context.Context,
	req *connect.Request[v1.UpdateSkillRequest],
) (*connect.Response[v1.UpdateSkillResponse], error) {
	projectID := req.Msg.ProjectId
	skill := req.Msg.Skill
	if skill == nil || skill.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("skill id is required"))
	}
	toolIDs, _ := json.Marshal(skill.ToolIds)

	err := h.metaRepo.UpdateSkill(&repository.SkillRecord{
		ID: skill.Id, ProjectID: projectID, Name: skill.Name,
		Description: skill.Description, ToolIDsJSON: string(toolIDs),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateSkillResponse{Skill: skill}), nil
}

func (h *SkillHandler) DeleteSkill(
	ctx context.Context,
	req *connect.Request[v1.DeleteSkillRequest],
) (*connect.Response[v1.DeleteSkillResponse], error) {
	err := h.metaRepo.DeleteSkill(req.Msg.Id, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteSkillResponse{Success: true}), nil
}
