package handler

import (
	"context"
	"encoding/json"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"connectrpc.com/connect"
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
	rows, err := h.metaRepo.DB().Query("SELECT id, name, description FROM system_skills WHERE project_id = $1", projectID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	defer rows.Close()

	var skills []*v1.Skill
	for rows.Next() {
		var s v1.Skill
		rows.Scan(&s.Id, &s.Name, &s.Description)
		skills = append(skills, &s)
	}
	return connect.NewResponse(&v1.ListSkillsResponse{Skills: skills}), nil
}

func (h *SkillHandler) CreateSkill(
	ctx context.Context,
	req *connect.Request[v1.CreateSkillRequest],
) (*connect.Response[v1.CreateSkillResponse], error) {
	projectID := req.Msg.ProjectId
	skill := req.Msg.Skill
	
	toolIDs, _ := json.Marshal(skill.ToolIds)
	
	_, err := h.metaRepo.DB().Exec(
		"INSERT INTO system_skills (id, project_id, name, description, tool_ids) VALUES ($1, $2, $3, $4, $5)",
		skill.Id, projectID, skill.Name, skill.Description, string(toolIDs),
	)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.CreateSkillResponse{Skill: skill}), nil
}

func (h *SkillHandler) DeleteSkill(
	ctx context.Context,
	req *connect.Request[v1.DeleteSkillRequest],
) (*connect.Response[v1.DeleteSkillResponse], error) {
	projectID := req.Msg.ProjectId
	id := req.Msg.Id
	_, err := h.metaRepo.DB().Exec("DELETE FROM system_skills WHERE project_id = $1 AND id = $2", projectID, id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteSkillResponse{Success: true}), nil
}
