package repository

import (
	"database/sql"
	"fmt"
)

type MetadataRepository struct {
	db *sql.DB
}

func NewMetadataRepository(db *sql.DB) (*MetadataRepository, error) {
	r := &MetadataRepository{db: db}
	if err := r.init(); err != nil { return nil, err }
	return r, nil
}

func (r *MetadataRepository) init() error {
	// Enable UUID extension if needed (Postgres 13+)
	r.db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"")

	queries := []string{
		`CREATE TABLE IF NOT EXISTS system_tasks (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, source_type TEXT, config_json TEXT, status TEXT, progress INTEGER, is_predictive INTEGER DEFAULT 0, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_simulations (
			id TEXT PRIMARY KEY, project_id TEXT, task_id TEXT, scenario_name TEXT, results_json TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_proposals (
			id TEXT PRIMARY KEY, project_id TEXT, action_name TEXT, object_id TEXT, params_json TEXT, status TEXT DEFAULT 'pending', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_history (
			id TEXT PRIMARY KEY, project_id TEXT, agent_id TEXT, role TEXT, content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_agents (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT, model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_skills (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_tools (
			id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_api_keys (
			id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_notification_channels (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, type TEXT, config_json TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil { return fmt.Errorf("failed to init system table: %v", err) }
	}
	return nil
}

type NotificationChannel struct {
	ID         string
	ProjectID  string
	Name       string
	Type       string
	ConfigJSON string
}

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

func (r *MetadataRepository) UpdateTaskProgress(id string, progress int32, status string) error {
	_, err := r.db.Exec("UPDATE system_tasks SET progress = $1, status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3", progress, status, id)
	return err
}

func (r *MetadataRepository) SaveChatMessage(projectID, agentID, role, content, toolCall string) error {
	// Using a simple UUID-like string generation for Postgres if hex(randomblob) is missing, 
	// or better, use a library in Go or gen_random_uuid() in SQL.
	// For now, let's use gen_random_uuid() directly in SQL.
	_, err := r.db.Exec("INSERT INTO system_chat_history (id, project_id, agent_id, role, content, tool_call) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)", projectID, agentID, role, content, toolCall)
	return err
}

func (r *MetadataRepository) DB() *sql.DB { return r.db }
