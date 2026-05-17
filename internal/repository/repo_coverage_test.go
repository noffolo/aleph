package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupRepoDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func setupFullSchema(t *testing.T) (*sql.DB, *MetadataRepository) {
	t.Helper()
	db := setupRepoDB(t)

	_, err := db.Exec(`
		CREATE TABLE system_api_keys (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			label TEXT,
			key TEXT,
			role TEXT DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_projects (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_agents (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			provider TEXT,
			model TEXT,
			api_key TEXT,
			system_prompt TEXT,
			skill_ids TEXT,
			base_url TEXT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_tools (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			code TEXT,
			category TEXT,
			version TEXT,
			health_status TEXT,
			source_type TEXT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_skills (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			description TEXT,
			tool_ids TEXT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			source_type TEXT,
			config_json TEXT,
			schedule TEXT,
			status TEXT,
			progress INT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_notification_channels (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			type TEXT,
			config_json TEXT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_chat_history (
			id UUID PRIMARY KEY,
			project_id TEXT,
			agent_id TEXT,
			role TEXT,
			content TEXT,
			tool_call TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE system_chat_sessions (
			id TEXT PRIMARY KEY,
			project_id TEXT
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE ontology_versions (
			version_id TEXT PRIMARY KEY,
			project_id TEXT,
			parent_version_id TEXT,
			diff_json TEXT,
			core_aleph_snapshot TEXT,
			status TEXT DEFAULT 'pending',
			source_description TEXT,
			rationale TEXT,
			confidence DOUBLE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			modified_at TIMESTAMP
		)
	`)
	require.NoError(t, err)

	repo, err := NewMetadataRepository(db)
	require.NoError(t, err)
	return db, repo
}

func TestCreateProjectRecord(t *testing.T) {
	t.Parallel()
	_, repo := setupFullSchema(t)

	err := repo.CreateProjectRecord("proj-1", "Test Project")
	assert.NoError(t, err)

	count, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCreateProjectRecord_Idempotent(t *testing.T) {
	t.Parallel()
	_, repo := setupFullSchema(t)

	err := repo.CreateProjectRecord("proj-1", "First")
	require.NoError(t, err)

	err = repo.CreateProjectRecord("proj-1", "Second")
	assert.NoError(t, err) // ON CONFLICT DO NOTHING

	count, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCountProjects(t *testing.T) {
	t.Parallel()
	_, repo := setupFullSchema(t)

	count, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.CreateProjectRecord("p1", "P1"))
	require.NoError(t, repo.CreateProjectRecord("p2", "P2"))
	require.NoError(t, repo.CreateProjectRecord("p3", "P3"))

	count, err = repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountProjectAgents(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	count, err := repo.CountProjectAgents("proj-1")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	_, err = db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ('a1', 'proj-1', 'Agent1', 'ollama', 'llama3', '', '', '', '')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ('a2', 'proj-1', 'Agent2', 'ollama', 'llama3', '', '', '', '')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ('a3', 'proj-2', 'Agent3', 'ollama', 'llama3', '', '', '', '')")
	require.NoError(t, err)

	count, err = repo.CountProjectAgents("proj-1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = repo.CountProjectAgents("proj-2")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetAPIKeyByID(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	_, err := db.Exec("INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ('key-01', 'proj-x', 'my-key', 'hashed-secret', 'admin')")
	require.NoError(t, err)

	hashedKey, projectID, role, err := repo.GetAPIKeyByID("key-01")
	require.NoError(t, err)
	assert.Equal(t, "hashed-secret", hashedKey)
	assert.Equal(t, "proj-x", projectID)
	assert.Equal(t, "admin", role)
}

func TestGetAPIKeyByID_NotFound(t *testing.T) {
	t.Parallel()
	_, repo := setupFullSchema(t)

	_, _, _, err := repo.GetAPIKeyByID("nonexistent")
	assert.Error(t, err)
}

func TestDeleteProjectCascade(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	// Create project-scoped data that uses project_id column
	require.NoError(t, repo.CreateProjectRecord("proj-del", "To Delete"))
	_, err := db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ('a-del', 'proj-del', 'Test', 'ollama', 'llama3', '', '', '', '')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO system_skills (id, project_id, name, description, tool_ids) VALUES ('s-del', 'proj-del', 'Skill', 'desc', '')")
	require.NoError(t, err)

	// Verify pre-deletion state
	agentCount, err := repo.CountProjectAgents("proj-del")
	require.NoError(t, err)
	assert.Equal(t, 1, agentCount)
}

func TestScheduleSchemaCleanup(t *testing.T) {
	ScheduleSchemaCleanup("proj-deferred-1")
	ScheduleSchemaCleanup("proj-deferred-2")

	ids := DrainSchemaCleanupQueue()
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "proj-deferred-1")
	assert.Contains(t, ids, "proj-deferred-2")

	// Queue should be empty after drain
	assert.Empty(t, DrainSchemaCleanupQueue())
}

func TestUpdateHealthStatus(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	_, err := db.Exec("INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('tool-h1', 'TestTool', 'desc', 'code', 'cat', '1.0', 'unknown', 'builtin')")
	require.NoError(t, err)

	err = repo.UpdateHealthStatus("tool-h1", "healthy")
	assert.NoError(t, err)
}

func TestUpdateTaskProgress(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	// Override task table to include updated_at column that UpdateTaskProgress needs
	db.Exec("DROP TABLE system_tasks")
	_, err := db.Exec(`
		CREATE TABLE system_tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			source_type TEXT,
			config_json TEXT,
			schedule TEXT,
			status TEXT,
			progress INT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO system_tasks (id, project_id, name, source_type, config_json, schedule, status, progress) VALUES ('task-1', 'proj-1', 'Task', 'rss', '{}', '', 'pending', 0)")
	require.NoError(t, err)

	err = repo.UpdateTaskProgress("task-1", 50, "running")
	assert.NoError(t, err)

	progress, err := repo.GetTaskProgress("task-1")
	require.NoError(t, err)
	assert.Equal(t, int32(50), progress)
}

func TestListNotificationChannels(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	_, err := db.Exec("INSERT INTO system_notification_channels (id, project_id, name, type, config_json) VALUES ('nc-1', 'proj-1', 'Email', 'email', '{}')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO system_notification_channels (id, project_id, name, type, config_json) VALUES ('nc-2', 'proj-1', 'Webhook', 'webhook', '{}')")
	require.NoError(t, err)

	channels, err := repo.ListNotificationChannels("proj-1")
	require.NoError(t, err)
	assert.Len(t, channels, 2)
}

func TestEncryptionKey(t *testing.T) {
	t.Parallel()
	repo, err := NewMetadataRepository(setupRepoDB(t))
	require.NoError(t, err)

	assert.Nil(t, repo.EncryptionKey())

	repo.SetEncryptionKey([]byte("test-key-32-bytes-xxxxxxxxxx"))
	assert.Equal(t, []byte("test-key-32-bytes-xxxxxxxxxx"), repo.EncryptionKey())

	repo.SetEncryptionKey(nil)
	assert.Nil(t, repo.EncryptionKey())
}

func TestValidateAPIKey_Deprecated(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	_, err := db.Exec("INSERT INTO system_api_keys (id, project_id, label, key, role) VALUES ('vk-1', 'proj-v', 'Key', 'exact-hash', 'user')")
	require.NoError(t, err)

	projectID, err := repo.ValidateAPIKey("exact-hash")
	require.NoError(t, err)
	assert.Equal(t, "proj-v", projectID)

	_, err = repo.ValidateAPIKey("wrong-hash")
	assert.Error(t, err)
}

func TestConfirmAgentInProject(t *testing.T) {
	t.Parallel()
	db, repo := setupFullSchema(t)

	found, err := repo.ConfirmAgentInProject("no-such-agent", "proj-1")
	require.NoError(t, err)
	assert.False(t, found)

	_, err = db.Exec("INSERT INTO system_agents (id, project_id, name, provider, model, api_key, system_prompt, skill_ids, base_url) VALUES ('ca-1', 'proj-1', 'CA', 'ollama', 'llama3', '', '', '', '')")
	require.NoError(t, err)

	found, err = repo.ConfirmAgentInProject("ca-1", "proj-1")
	require.NoError(t, err)
	assert.True(t, found)

	found, err = repo.ConfirmAgentInProject("ca-1", "wrong-proj")
	require.NoError(t, err)
	assert.False(t, found)
}

// ─── AuditRepository ────────────────────────────────────────────────────────





// ─── OntologyRepository ──────────────────────────────────────────────────────

func TestNewOntologyRepository(t *testing.T) {
	t.Parallel()
	db := setupRepoDB(t)
	or := NewOntologyRepository(db)
	assert.NotNil(t, or)
}

func TestProposeOntologyDiff(t *testing.T) {
	t.Parallel()
	db, _ := setupFullSchema(t)
	or := NewOntologyRepository(db)

	versionID, err := or.ProposeOntologyDiff(context.Background(), "proj-1", "",
		`{"added": ["field new_field"]}`,
		"snapshot-content",
		"auto-generated",
		"testing ontology diff",
		0.85,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, versionID)
}

func TestAcceptDiff(t *testing.T) {
	t.Parallel()
	db, _ := setupFullSchema(t)
	or := NewOntologyRepository(db)

	versionID, err := or.ProposeOntologyDiff(context.Background(), "proj-1", "",
		`{}`, "snap", "test", "test", 0.5,
	)
	require.NoError(t, err)

	err = or.AcceptDiff(context.Background(), versionID)
	assert.NoError(t, err)
}

func TestRejectDiff(t *testing.T) {
	t.Parallel()
	db, _ := setupFullSchema(t)
	or := NewOntologyRepository(db)

	versionID, err := or.ProposeOntologyDiff(context.Background(), "proj-1", "",
		`{}`, "snap", "test", "test", 0.5,
	)
	require.NoError(t, err)

	err = or.RejectDiff(context.Background(), versionID, "not approved by reviewer")
	assert.NoError(t, err)
}

func TestListVersions(t *testing.T) {
	t.Parallel()
	db, _ := setupFullSchema(t)
	or := NewOntologyRepository(db)

	// Insert a few versions
	_, err := or.ProposeOntologyDiff(context.Background(), "proj-1", "", `{}`, "s1", "test", "test", 0.5)
	require.NoError(t, err)
	_, err = or.ProposeOntologyDiff(context.Background(), "proj-1", "", `{}`, "s2", "test", "test", 0.5)
	require.NoError(t, err)

	versions, err := or.ListVersions(context.Background(), "proj-1", 5)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
}

func TestListVersions_DefaultLimit(t *testing.T) {
	t.Parallel()
	db, _ := setupFullSchema(t)
	or := NewOntologyRepository(db)

	versions, err := or.ListVersions(context.Background(), "proj-empty", 0)
	require.NoError(t, err)
	assert.Empty(t, versions)
}
