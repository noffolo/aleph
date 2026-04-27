package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MetadataRepository provides CRUD operations for system metadata tables
// (agents, tools, skills, tasks, API keys, chat history).
type MetadataRepository struct {
	db       *sql.DB
	toolCache *ToolCache
}

// NewMetadataRepository creates a MetadataRepository backed by the given *sql.DB.
func NewMetadataRepository(db *sql.DB) (*MetadataRepository, error) {
	return &MetadataRepository{
		db:        db,
		toolCache: NewToolCache(),
	}, nil
}

// SetToolCache replaces the default tool cache (used by tests).
func (r *MetadataRepository) SetToolCache(tc *ToolCache) {
	r.toolCache = tc
}

// ─── Notification Channels ────────────────────────────────────────────────

// NotificationChannel represents a notification channel record.
type NotificationChannel struct {
	ID         string
	ProjectID  string
	Name       string
	Type       string
	ConfigJSON string
}

// ListNotificationChannels returns all channels for a project.
func (r *MetadataRepository) ListNotificationChannels(projectID string) ([]NotificationChannel, error) {
	rows, err := r.db.Query("SELECT id, project_id, name, type, config_json FROM system_notification_channels WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var channels []NotificationChannel
	for rows.Next() {
		var c NotificationChannel
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.ConfigJSON); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, nil
}

// ─── Tasks ─────────────────────────────────────────────────────────────────

// IngestionTaskRecord represents a system_tasks row.
type IngestionTaskRecord struct {
	ID         string
	ProjectID  string
	Name       string
	SourceType string
	ConfigJSON string
	Schedule   string
	Status     string
	Progress   int32
}

func (r *MetadataRepository) UpdateTaskProgress(id string, progress int32, status string) error {
	_, err := r.db.Exec("UPDATE system_tasks SET progress = $1, status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3", progress, status, id)
	return fmt.Errorf("updateTaskProgress: %w", err)
}

func (r *MetadataRepository) GetTaskProgress(taskID string) (int32, error) {
	var progress int32
	err := r.db.QueryRow("SELECT progress FROM system_tasks WHERE id = $1", taskID).Scan(&progress)
	if err != nil {
		return 0, err
	}
	return progress, nil
}

func (r *MetadataRepository) ListTasks(projectID string) ([]IngestionTaskRecord, error) {
	rows, err := r.db.Query("SELECT id, name, source_type, config_json, schedule, status, progress FROM system_tasks WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []IngestionTaskRecord
	for rows.Next() {
		var t IngestionTaskRecord
		if err := rows.Scan(&t.ID, &t.Name, &t.SourceType, &t.ConfigJSON, &t.Schedule, &t.Status, &t.Progress); err != nil {
			continue
		}
		t.ProjectID = projectID
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *MetadataRepository) CreateTask(t *IngestionTaskRecord) error {
	_, err := r.db.Exec(
		"INSERT INTO system_tasks (id, project_id, name, source_type, config_json, schedule, status, progress) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		t.ID, t.ProjectID, t.Name, t.SourceType, t.ConfigJSON, t.Schedule, t.Status, t.Progress,
	)
	return fmt.Errorf("createTask: %w", err)
}

func (r *MetadataRepository) GetTaskByID(taskID string) (*IngestionTaskRecord, error) {
	var t IngestionTaskRecord
	err := r.db.QueryRow("SELECT id, name, source_type, config_json FROM system_tasks WHERE id = $1", taskID).Scan(&t.ID, &t.Name, &t.SourceType, &t.ConfigJSON)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MetadataRepository) DeleteTask(id, projectID string) error {
	_, err := r.db.Exec("DELETE FROM system_tasks WHERE project_id = $1 AND id = $2", projectID, id)
	return fmt.Errorf("deleteTask: %w", err)
}

// ─── Chat Messages ─────────────────────────────────────────────────────────

// ChatMessage represents a chat history entry.
type ChatMessage struct {
	Role      string
	Content   string
	ToolCall  string
	CreatedAt time.Time
}

func (r *MetadataRepository) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO system_chat_history (id, project_id, agent_id, role, content, tool_call) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)", projectID, agentID, role, content, toolCall)
	return fmt.Errorf("saveChatMessage: %w", err)
}

func (r *MetadataRepository) GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT role, content, tool_call, created_at FROM system_chat_history WHERE project_id = $1 AND agent_id = $2 ORDER BY created_at ASC LIMIT 20",
		projectID, agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolCall, &m.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *MetadataRepository) GetChatHistory(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	return r.GetChatMessages(ctx, projectID, agentID)
}

// ─── Agents ────────────────────────────────────────────────────────────────

// AgentRecord represents a system_agents row.
type AgentRecord struct {
	ID           string
	ProjectID    string
	Name         string
	Provider     string
	Model        string
	ApiKey       string
	SystemPrompt string
	SkillIDsJSON string
	BaseURL      string
}

func (r *MetadataRepository) ConfirmAgentInProject(agentID, projectID string) (bool, error) {
	var id string
	err := r.db.QueryRow("SELECT id FROM system_agents WHERE id = $1 AND project_id = $2", agentID, projectID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *MetadataRepository) GetAgentForChat(agentID string) (*AgentRecord, error) {
	var a AgentRecord
	err := r.db.QueryRow("SELECT provider, model, api_key, system_prompt, skill_ids, base_url FROM system_agents WHERE id = $1", agentID).
		Scan(&a.Provider, &a.Model, &a.ApiKey, &a.SystemPrompt, &a.SkillIDsJSON, &a.BaseURL)
	if err != nil {
		return nil, err
	}
	a.ID = agentID
	return &a, nil
}

func (r *MetadataRepository) ListAgents(projectID string) ([]*AgentRecord, error) {
	rows, err := r.db.Query("SELECT id, name, provider, model, api_key, system_prompt, skill_ids, base_url FROM system_agents WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []*AgentRecord
	for rows.Next() {
		var a AgentRecord
		if err := rows.Scan(&a.ID, &a.Name, &a.Provider, &a.Model, &a.ApiKey, &a.SystemPrompt, &a.SkillIDsJSON, &a.BaseURL); err != nil {
			continue
		}
		a.ProjectID = projectID
		agents = append(agents, &a)
	}
	return agents, nil
}

func (r *MetadataRepository) ListAgentsCursor(projectID, cursor string, limit int) ([]*AgentRecord, error) {
	var args []interface{}
	args = append(args, projectID)
	query := "SELECT id, name, provider, model, api_key, system_prompt, skill_ids, base_url FROM system_agents WHERE project_id = $1"
	if cursor != "" {
		query += " AND id > $2"
		args = append(args, cursor)
		query += fmt.Sprintf(" ORDER BY id LIMIT $%d", len(args)+1)
	} else {
		query += " ORDER BY id LIMIT $2"
	}
	args = append(args, limit)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []*AgentRecord
	for rows.Next() {
		var a AgentRecord
		if err := rows.Scan(&a.ID, &a.Name, &a.Provider, &a.Model, &a.ApiKey, &a.SystemPrompt, &a.SkillIDsJSON, &a.BaseURL); err != nil {
			continue
		}
		a.ProjectID = projectID
		agents = append(agents, &a)
	}
	return agents, nil
}

func (r *MetadataRepository) CreateAgent(a *AgentRecord) error {
	_, err := r.db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", a.ID, a.ProjectID, a.Name, a.Provider, a.Model, a.ApiKey, a.SystemPrompt, a.SkillIDsJSON, a.BaseURL)
	return fmt.Errorf("createAgent: %w", err)
}

func (r *MetadataRepository) DeleteAgent(agentID, projectID string) error {
	_, err := r.db.Exec("DELETE FROM system_agents WHERE project_id = $1 AND id = $2", projectID, agentID)
	return fmt.Errorf("deleteAgent: %w", err)
}

func (r *MetadataRepository) UpdateAgent(a *AgentRecord) error {
	_, err := r.db.Exec("UPDATE system_agents SET name = $1, provider = $2, model = $3, api_key = $4, system_prompt = $5, skill_ids = $6, base_url = $7 WHERE id = $8 AND project_id = $9", a.Name, a.Provider, a.Model, a.ApiKey, a.SystemPrompt, a.SkillIDsJSON, a.BaseURL, a.ID, a.ProjectID)
	return fmt.Errorf("updateAgent: %w", err)
}

// ─── Tools ─────────────────────────────────────────────────────────────────

// ToolRecord represents a system_tools row.
type ToolRecord struct {
	ID           string
	Name         string
	Description  string
	Code         string
	Category     string
	Version      string
	HealthStatus string
	SourceType   string
}

func (r *MetadataRepository) ListTools() ([]ToolRecord, error) {
	const cacheKey = "list_tools"
	if v, ok := r.toolCache.Get(cacheKey); ok {
		if tools, ok := v.([]ToolRecord); ok {
			return tools, nil
		}
	}

	rows, err := r.db.Query("SELECT id, name, description, code, category, version, health_status, source_type FROM system_tools")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tools []ToolRecord
	for rows.Next() {
		var t ToolRecord
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Code, &t.Category, &t.Version, &t.HealthStatus, &t.SourceType); err != nil {
			continue
		}
		tools = append(tools, t)
	}
	r.toolCache.Set(cacheKey, tools)
	return tools, nil
}

func (r *MetadataRepository) ListToolsCursor(cursor string, limit int) ([]ToolRecord, error) {
	var args []interface{}
	query := "SELECT id, name, description, code, category, version, health_status, source_type FROM system_tools"
	if cursor != "" {
		query += " WHERE id > $1"
		args = append(args, cursor)
		query += fmt.Sprintf(" ORDER BY id LIMIT $%d", len(args)+1)
	} else {
		query += " ORDER BY id LIMIT $1"
	}
	args = append(args, limit)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tools []ToolRecord
	for rows.Next() {
		var t ToolRecord
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Code, &t.Category, &t.Version, &t.HealthStatus, &t.SourceType); err != nil {
			continue
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func (r *MetadataRepository) GetToolByCategory(category string) ([]ToolRecord, error) {
	rows, err := r.db.Query("SELECT id, name, description, code, category, version, health_status, source_type FROM system_tools WHERE category = $1", category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tools []ToolRecord
	for rows.Next() {
		var t ToolRecord
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Code, &t.Category, &t.Version, &t.HealthStatus, &t.SourceType); err != nil {
			continue
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func (r *MetadataRepository) CreateTool(t *ToolRecord) error {
	_, err := r.db.Exec("INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		t.ID, t.Name, t.Description, t.Code, t.Category, t.Version, t.HealthStatus, t.SourceType)
	if err == nil {
		r.toolCache.Invalidate("list_tools")
	}
	return fmt.Errorf("createTool: %w", err)
}

func (r *MetadataRepository) UpdateToolCode(ctx context.Context, toolID, code string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE system_tools SET code = $1 WHERE id = $2", code, toolID)
	if err == nil {
		r.toolCache.Invalidate("list_tools")
	}
	return fmt.Errorf("updateToolCode: %w", err)
}

func (r *MetadataRepository) UpdateHealthStatus(toolID, status string) error {
	_, err := r.db.Exec("UPDATE system_tools SET health_status = $1 WHERE id = $2", status, toolID)
	if err == nil {
		r.toolCache.Invalidate("list_tools")
	}
	return fmt.Errorf("updateHealthStatus: %w", err)
}

func (r *MetadataRepository) GetToolCode(ctx context.Context, toolID string) (string, error) {
	var code string
	err := r.db.QueryRowContext(ctx, "SELECT code FROM system_tools WHERE id = $1", toolID).Scan(&code)
	if err != nil {
		return "", err
	}
	return code, nil
}

func (r *MetadataRepository) DeleteTool(id string) error {
	_, err := r.db.Exec("DELETE FROM system_tools WHERE id = $1", id)
	if err == nil {
		r.toolCache.Invalidate("list_tools")
	}
	return fmt.Errorf("deleteTool: %w", err)
}

// ─── Skills ────────────────────────────────────────────────────────────────

// SkillRecord represents a system_skills row.
type SkillRecord struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	ToolIDsJSON string
}

func (r *MetadataRepository) ListSkills(projectID string) ([]SkillRecord, error) {
	rows, err := r.db.Query("SELECT id, name, description, tool_ids FROM system_skills WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var skills []SkillRecord
	for rows.Next() {
		var s SkillRecord
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.ToolIDsJSON); err != nil {
			continue
		}
		s.ProjectID = projectID
		skills = append(skills, s)
	}
	return skills, nil
}

func (r *MetadataRepository) ListSkillsCursor(projectID, cursor string, limit int) ([]SkillRecord, error) {
	var args []interface{}
	args = append(args, projectID)
	query := "SELECT id, name, description, tool_ids FROM system_skills WHERE project_id = $1"
	if cursor != "" {
		query += " AND id > $2"
		args = append(args, cursor)
		query += fmt.Sprintf(" ORDER BY id LIMIT $%d", len(args)+1)
	} else {
		query += " ORDER BY id LIMIT $2"
	}
	args = append(args, limit)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var skills []SkillRecord
	for rows.Next() {
		var s SkillRecord
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.ToolIDsJSON); err != nil {
			continue
		}
		s.ProjectID = projectID
		skills = append(skills, s)
	}
	return skills, nil
}

func (r *MetadataRepository) CreateSkill(s *SkillRecord) error {
	_, err := r.db.Exec("INSERT INTO system_skills (id, project_id, name, description, tool_ids) VALUES ($1, $2, $3, $4, $5)", s.ID, s.ProjectID, s.Name, s.Description, s.ToolIDsJSON)
	return fmt.Errorf("createSkill: %w", err)
}

func (r *MetadataRepository) GetSkillToolIDs(skillID string) (string, error) {
	var toolIDsJSON string
	err := r.db.QueryRow("SELECT tool_ids FROM system_skills WHERE id = $1", skillID).Scan(&toolIDsJSON)
	if err != nil {
		return "", err
	}
	return toolIDsJSON, nil
}

func (r *MetadataRepository) DeleteSkill(id, projectID string) error {
	_, err := r.db.Exec("DELETE FROM system_skills WHERE project_id = $1 AND id = $2", projectID, id)
	return fmt.Errorf("deleteSkill: %w", err)
}

// ─── API Keys ──────────────────────────────────────────────────────────────

// APIKeyRecord represents a system_api_keys row (key is hashed).
type APIKeyRecord struct {
	ID        string
	ProjectID string
	Label     string
	Key       string
	CreatedAt time.Time
}

func (r *MetadataRepository) ListAPIKeys(projectID string) ([]APIKeyRecord, error) {
	rows, err := r.db.Query("SELECT id, label, key, created_at FROM system_api_keys WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKeyRecord
	for rows.Next() {
		var k APIKeyRecord
		if err := rows.Scan(&k.ID, &k.Label, &k.Key, &k.CreatedAt); err != nil {
			continue
		}
		k.ProjectID = projectID
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *MetadataRepository) CreateAPIKey(id, projectID, label, hashedKey string) error {
	_, err := r.db.Exec("INSERT INTO system_api_keys (id, project_id, label, key) VALUES ($1, $2, $3, $4)", id, projectID, label, hashedKey)
	return fmt.Errorf("createAPIKey: %w", err)
}

func (r *MetadataRepository) ValidateAPIKey(hashedKey string) (string, error) {
	var projectID string
	err := r.db.QueryRow("SELECT project_id FROM system_api_keys WHERE key = $1", hashedKey).Scan(&projectID)
	if err != nil {
		return "", err
	}
	return projectID, nil
}

func (r *MetadataRepository) DeleteAPIKey(id, projectID string) error {
	_, err := r.db.Exec("DELETE FROM system_api_keys WHERE project_id = $1 AND id = $2", projectID, id)
	return fmt.Errorf("deleteAPIKey: %w", err)
}
