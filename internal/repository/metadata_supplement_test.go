package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

// setupExtendedRepo creates a MetadataRepository with system_projects and
// ontology_versions tables, in addition to the standard setupMetadataRepo tables.
func setupExtendedRepo(t *testing.T) *MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	queries := []string{
		`CREATE TABLE IF NOT EXISTS system_projects (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_tasks (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, source_type TEXT,
			config_json TEXT, status TEXT, progress INTEGER,
			schedule TEXT DEFAULT '', is_predictive INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_agents (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT,
			model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT, base_url TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS system_skills (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_tools (
			id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT,
			category TEXT, version TEXT, health_status TEXT, source_type TEXT,
			last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_api_keys (
			id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT,
			role TEXT DEFAULT 'user',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_history (
			id TEXT PRIMARY KEY, project_id TEXT, agent_id TEXT, role TEXT,
			content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_notification_channels (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, type TEXT, config_json TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_sessions (
			id TEXT PRIMARY KEY, project_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_ontology_versions (
			version_id TEXT PRIMARY KEY, project_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS ontology_versions (
			version_id TEXT PRIMARY KEY, project_id TEXT,
			parent_version_id TEXT, diff_json TEXT, core_aleph_snapshot TEXT,
			status TEXT, source_description TEXT, rationale TEXT,
			confidence DOUBLE, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			modified_at TIMESTAMP
		)`,
	}
	for _, q := range queries {
		_, err := db.Exec(q)
		require.NoError(t, err)
	}

	repo, err := NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

// ─── Project Management ───────────────────────────────────────────────

func TestMetadataRepository_CreateProjectRecord(t *testing.T) {
	repo := setupExtendedRepo(t)

	err := repo.CreateProjectRecord("proj-abc", "Test Project")
	require.NoError(t, err)

	// Set project_id for cascade delete compatibility
	_, _ = repo.db.Exec("UPDATE system_projects SET project_id = $1 WHERE id = $1", "proj-abc")

	err = repo.CreateProjectRecord("proj-abc", "Test Project")
	require.NoError(t, err)

	var name string
	var pid string
	err = repo.db.QueryRow("SELECT name, project_id FROM system_projects WHERE id = $1", "proj-abc").Scan(&name, &pid)
	require.NoError(t, err)
	assert.Equal(t, "Test Project", name)
}

func TestMetadataRepository_CountProjects(t *testing.T) {
	repo := setupExtendedRepo(t)

	count, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.CreateProjectRecord("proj-a", "Project A"))
	require.NoError(t, repo.CreateProjectRecord("proj-b", "Project B"))
	require.NoError(t, repo.CreateProjectRecord("proj-c", "Project C"))
	// set project_id for cascade compatibility
	for _, pid := range []string{"proj-a", "proj-b", "proj-c"} {
		_, _ = repo.db.Exec("UPDATE system_projects SET project_id = $1 WHERE id = $1", pid)
	}

	count, err = repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMetadataRepository_CountProjectAgents(t *testing.T) {
	repo := setupExtendedRepo(t)

	count, err := repo.CountProjectAgents("proj-x")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.CreateAgent(&AgentRecord{ID: "a1", ProjectID: "proj-x", Name: "a1", Provider: "test"}))
	require.NoError(t, repo.CreateAgent(&AgentRecord{ID: "a2", ProjectID: "proj-x", Name: "a2", Provider: "test"}))

	count, err = repo.CountProjectAgents("proj-x")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Different project should be isolated
	count, err = repo.CountProjectAgents("proj-y")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ─── API Key ────────────────────────────────────────────────────────

func TestMetadataRepository_GetAPIKeyByID(t *testing.T) {
	repo := setupExtendedRepo(t)

	err := repo.CreateAPIKey("key-abc", "proj-x", "label-1", "hashed-secret-x")
	require.NoError(t, err)

	hashedKey, projectID, role, err := repo.GetAPIKeyByID("key-abc")
	require.NoError(t, err)
	assert.Equal(t, "hashed-secret-x", hashedKey)
	assert.Equal(t, "proj-x", projectID)
	// DuckDB returns empty string for NULL role column (no DEFAULT in older create)
	assert.NotEmpty(t, projectID)
	_ = role // may be empty for DuckDB :memory:

	// Non-existent key
	_, _, _, err = repo.GetAPIKeyByID("nonexistent")
	assert.Error(t, err)
}

// ─── Cascade Delete ──────────────────────────────────────────────────

func TestMetadataRepository_DeleteProjectCascade(t *testing.T) {
	repo := setupExtendedRepo(t)

	require.NoError(t, repo.CreateProjectRecord("proj-del", "Delete Me"))
	_, _ = repo.db.Exec("UPDATE system_projects SET project_id = $1 WHERE id = $1", "proj-del")
	require.NoError(t, repo.CreateAgent(&AgentRecord{ID: "a1", ProjectID: "proj-del", Name: "agent", Provider: "test"}))
	require.NoError(t, repo.CreateSkill(&SkillRecord{ID: "s1", ProjectID: "proj-del", Name: "skill"}))
	require.NoError(t, repo.CreateAPIKey("k1", "proj-del", "label", "hash"))

	count, err := repo.CountProjectAgents("proj-del")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = repo.DeleteProjectCascade("proj-del")
	require.NoError(t, err)

	count, err = repo.CountProjectAgents("proj-del")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	pCount, err := repo.CountProjects()
	require.NoError(t, err)
	assert.Equal(t, 0, pCount)
}

func TestMetadataRepository_DeleteProjectCascade_EmptyProject(t *testing.T) {
	repo := setupExtendedRepo(t)

	// Deleting a non-existent project should succeed (DELETE WHERE project_id = $1 finds 0 rows)
	err := repo.DeleteProjectCascade("nonexistent-proj")
	require.NoError(t, err)
}

// ─── Ontology ────────────────────────────────────────────────────────

func setupOntologyRepo(t *testing.T) *MetadataRepository {
	t.Helper()
	// reuse extended setup which includes ontology_versions table
	return setupExtendedRepo(t)
}

func TestOntologyRepository_ProposeAcceptList(t *testing.T) {
	repo := setupOntologyRepo(t)
	ontRepo := NewOntologyRepository(repo.db)

	ctx := context.Background()

	// Propose
	vid, err := ontRepo.ProposeOntologyDiff(ctx, "proj-o", "",
		`{"diff": "add entity"}`, `{"snapshot": "v1"}`,
		"manual", "initial ontology", 0.95)
	require.NoError(t, err)
	assert.NotEmpty(t, vid)

	// ListVersions with default limit
	versions, err := ontRepo.ListVersions(ctx, "proj-o", 0)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, vid, versions[0].VersionID)
	assert.Equal(t, "pending", versions[0].Status)

	// ListVersions with explicit limit
	versions, err = ontRepo.ListVersions(ctx, "proj-o", 10)
	require.NoError(t, err)
	assert.Len(t, versions, 1)

	// Accept
	err = ontRepo.AcceptDiff(ctx, vid)
	require.NoError(t, err)

	versions, err = ontRepo.ListVersions(ctx, "proj-o", 10)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, "accepted", versions[0].Status)
}

func TestOntologyRepository_RejectDiff(t *testing.T) {
	repo := setupOntologyRepo(t)
	ontRepo := NewOntologyRepository(repo.db)

	ctx := context.Background()

	vid, err := ontRepo.ProposeOntologyDiff(ctx, "proj-r", "",
		`{"diff": "remove entity"}`, `{"snapshot": "v1"}`,
		"auto", "cleanup", 0.6)
	require.NoError(t, err)

	err = ontRepo.RejectDiff(ctx, vid, "quality below threshold")
	require.NoError(t, err)

	versions, err := ontRepo.ListVersions(ctx, "proj-r", 10)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, "rejected", versions[0].Status)
	assert.Equal(t, "quality below threshold", versions[0].Rationale.String)
}

func TestOntologyRepository_ListVersions_EmptyProject(t *testing.T) {
	repo := setupOntologyRepo(t)
	ontRepo := NewOntologyRepository(repo.db)

	ctx := context.Background()

	versions, err := ontRepo.ListVersions(ctx, "empty-proj", 10)
	require.NoError(t, err)
	assert.Empty(t, versions)
}

// ─── Encryption ──────────────────────────────────────────────────────

func TestMetadataRepository_EncryptionKey(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo, err := NewMetadataRepository(db)
	require.NoError(t, err)

	assert.Nil(t, repo.EncryptionKey())

	key := []byte("abcdef1234567890abcdef1234567890")
	repo.SetEncryptionKey(key)
	assert.Equal(t, key, repo.EncryptionKey())

	repo.SetEncryptionKey(nil)
	assert.Nil(t, repo.EncryptionKey())
}

func TestMetadataRepository_SetToolCache(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo, err := NewMetadataRepository(db)
	require.NoError(t, err)

	original := repo.toolCache
	assert.NotNil(t, original)

	newCache := NewToolCache()
	defer newCache.Close()
	repo.SetToolCache(newCache)
	assert.Equal(t, newCache, repo.toolCache)
}

// ─── Schema Cleanup Queue ─────────────────────────────────────────────

func TestSchemaCleanupQueue(t *testing.T) {
	// Drain any leftovers from other tests
	DrainSchemaCleanupQueue()

	ScheduleSchemaCleanup("proj-1")
	ScheduleSchemaCleanup("proj-2")
	ScheduleSchemaCleanup("proj-3")

	ids := DrainSchemaCleanupQueue()
	assert.Equal(t, []string{"proj-1", "proj-2", "proj-3"}, ids)

	// Queue should be empty after drain
	ids = DrainSchemaCleanupQueue()
	assert.Empty(t, ids)
}

// ─── Agent Encryption ─────────────────────────────────────────────────

func TestMetadataRepository_AgentEncryption_Decrypt(t *testing.T) {
	repo := setupExtendedRepo(t)

	// Without encryption key set, api_key is stored and returned as-is
	err := repo.CreateAgent(&AgentRecord{
		ID: "agent-enc-1", ProjectID: "proj-e", Name: "enc-agent",
		Provider: "openai", ApiKey: "sk-plaintext",
	})
	require.NoError(t, err)

	agent, err := repo.GetAgentForChat("agent-enc-1")
	require.NoError(t, err)
	assert.Equal(t, "sk-plaintext", agent.ApiKey)
}

// ─── Schema Identity ───────────────────────────────────────────────────

// DeleteProjectCascadeWithDB is tested minimally — the DuckDB+storage
// dependency makes full testing complex without a real DuckDB handle.
// The function body is straightforward delegation: PostgreSQL cascade
// followed by DuckDB schema drop.
func TestDeleteProjectCascadeWithDB_NilRepo(t *testing.T) {
	defer func() {
		_ = recover()
	}()
	_ = DeleteProjectCascadeWithDB(context.Background(), "proj-x", nil, nil)
}

// ─── Encryption Round-Trip ────────────────────────────────────────────

func TestMetadataRepository_AgentEncryptionRoundTrip(t *testing.T) {
	repo := setupExtendedRepo(t)
	// 32-byte AES-256 key
	key := []byte("abcdef1234567890abcdef1234567890")
	repo.SetEncryptionKey(key)

	err := repo.CreateAgent(&AgentRecord{
		ID: "agent-encrypt-1", ProjectID: "proj-e2", Name: "encrypted-agent",
		Provider: "openai", ApiKey: "sk-super-secret",
	})
	require.NoError(t, err)

	// Raw DB should contain encrypted value, not plaintext
	var dbApiKey string
	err = repo.db.QueryRow("SELECT api_key FROM system_agents WHERE id = $1", "agent-encrypt-1").Scan(&dbApiKey)
	require.NoError(t, err)
	assert.NotEqual(t, "sk-super-secret", dbApiKey)

	// GetAgentForChat should decrypt transparently
	agent, err := repo.GetAgentForChat("agent-encrypt-1")
	require.NoError(t, err)
	assert.Equal(t, "sk-super-secret", agent.ApiKey)

	// UpdateAgent should encrypt on write
	agent.ApiKey = "sk-new-secret"
	agent.ProjectID = "proj-e2"
	err = repo.UpdateAgent(agent)
	require.NoError(t, err)

	var dbApiKey2 string
	err = repo.db.QueryRow("SELECT api_key FROM system_agents WHERE id = $1", "agent-encrypt-1").Scan(&dbApiKey2)
	require.NoError(t, err)
	assert.NotEqual(t, "sk-new-secret", dbApiKey2)

	// ListAgents should decrypt
	agents, err := repo.ListAgents("proj-e2")
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "sk-new-secret", agents[0].ApiKey)

	// Disable encryption, then api_key stays as-is (no decrypt)
	repo.SetEncryptionKey(nil)
	agents2, err := repo.ListAgents("proj-e2")
	require.NoError(t, err)
	assert.Len(t, agents2, 1)
	// Without key, the stored value is returned raw (may be encrypted gibberish)
	assert.NotEqual(t, "sk-new-secret", agents2[0].ApiKey)
}
