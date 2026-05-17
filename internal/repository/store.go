// Package repository provides persistence interfaces and implementations
// for Aleph's metadata, audit, and ontology subsystems.
package repository

import (
	"context"
)

// ─── DataSource Entity ─────────────────────────────────────────────────────

// DataSourceRecord represents a registered data source (CSV, RSS, GitHub, IMAP, etc.).
type DataSourceRecord struct {
	ID          string
	ProjectID   string
	Name        string
	Type        string // csv, rss, github, imap, sitemap, etc.
	ConfigJSON  string
	Schedule    string
	Status      string // active, paused, errored
	LastRunAt   string
	NextRunAt   string
}

// ─── MetadataStore Interface ────────────────────────────────────────────────

// MetadataStore defines the contract for CRUD operations on Agent,
// ToolRecord, Skill, and DataSource entities, plus a health check.
//
// Implemented by [MetadataRepository].
type MetadataStore interface {
	// ─── Health ──────────────────────────────────────────────────────────
	//
	// Health returns nil if the underlying database connection is alive.
	Health(ctx context.Context) error

	// ─── Agent CRUD ──────────────────────────────────────────────────────

	// CreateAgent inserts a new agent record. The caller must provide a
	// unique ID and project scope.
	CreateAgent(a *AgentRecord) error

	// GetAgentForChat retrieves the minimal agent fields (provider, model,
	// api_key, system_prompt, skill_ids, base_url) needed for chat routing.
	// API keys are transparently decrypted when an encryption key is set.
	GetAgentForChat(agentID string) (*AgentRecord, error)

	// ListAgents returns all agent records scoped to the given project.
	ListAgents(projectID string) ([]*AgentRecord, error)

	// UpdateAgent replaces the mutable fields (name, provider, model,
	// api_key, system_prompt, skill_ids, base_url) for the given agent.
	UpdateAgent(a *AgentRecord) error

	// DeleteAgent removes the agent record identified by agentID and projectID.
	DeleteAgent(agentID, projectID string) error

	// ─── ToolRecord CRUD ─────────────────────────────────────────────────

	// CreateTool inserts a new tool record.
	CreateTool(t *ToolRecord) error

	// ListTools returns all registered tools. Results are cached via ToolCache.
	ListTools() ([]ToolRecord, error)

	// GetToolByCategory returns tools that match the given category string.
	GetToolByCategory(category string) ([]ToolRecord, error)

	// GetToolCode returns the source code stored for the given toolID.
	GetToolCode(ctx context.Context, toolID string) (string, error)

	// UpdateTool replaces the mutable fields (name, description, code,
	// category, version, health_status, source_type) for the given tool.
	UpdateTool(t *ToolRecord) error

	// DeleteTool removes the tool record identified by id.
	DeleteTool(id string) error

	// ─── Skill CRUD ──────────────────────────────────────────────────────

	// CreateSkill inserts a new skill record scoped to a project.
	CreateSkill(s *SkillRecord) error

	// ListSkills returns all skill records for the given project.
	ListSkills(projectID string) ([]SkillRecord, error)

	// UpdateSkill replaces the mutable fields (name, description, tool_ids)
	// for the given skill within its project scope.
	UpdateSkill(s *SkillRecord) error

	// DeleteSkill removes the skill record identified by id and projectID.
	DeleteSkill(id, projectID string) error

	// ─── DataSource CRUD ─────────────────────────────────────────────────

	// CreateDataSource inserts a new data source record.
	CreateDataSource(ds *DataSourceRecord) error

	// GetDataSource returns the data source record for the given id.
	GetDataSource(id string) (*DataSourceRecord, error)

	// ListDataSources returns all data source records for the given project.
	ListDataSources(projectID string) ([]DataSourceRecord, error)

	// UpdateDataSource replaces mutable fields for the given data source.
	UpdateDataSource(ds *DataSourceRecord) error

	// DeleteDataSource removes the data source record identified by id.
	DeleteDataSource(id string) error
}
