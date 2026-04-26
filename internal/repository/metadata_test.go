package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

var testCtx = context.Background()

func setupMetadataRepo(t *testing.T) *MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Create the system tables that the migrator would normally create
	queries := []string{
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
			id TEXT PRIMARY KEY, project_id TEXT, label TEXT, key TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_chat_history (
			id TEXT PRIMARY KEY, project_id TEXT, agent_id TEXT, role TEXT,
			content TEXT, tool_call TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_notification_channels (
			id TEXT PRIMARY KEY, project_id TEXT, name TEXT, type TEXT, config_json TEXT
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

func TestNewMetadataRepository(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo, err := NewMetadataRepository(db)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
}

// TestNewMetadataRepository_NilDB is intentionally omitted because
// NewMetadataRepository with a nil *sql.DB panics in init().

func TestMetadataRepository_ListNotificationChannels_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	channels, err := repo.ListNotificationChannels("project1")
	assert.NoError(t, err)
	assert.Empty(t, channels)
}

func TestMetadataRepository_UpdateTaskProgress_NoRows(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.UpdateTaskProgress("nonexistent", 50, "running")
	assert.NoError(t, err)
}

func TestMetadataRepository_Lifecycle_Task(t *testing.T) {
	repo := setupMetadataRepo(t)

	// Create
	task := &IngestionTaskRecord{
		ID:         "task-1",
		ProjectID:  "project1",
		Name:       "test-task",
		SourceType: "csv",
		ConfigJSON: `{"path": "/tmp/test.csv"}`,
		Status:     "pending",
		Progress:   0,
	}
	err := repo.CreateTask(task)
	require.NoError(t, err)

	// Get by ID
	got, err := repo.GetTaskByID("task-1")
	require.NoError(t, err)
	assert.Equal(t, "task-1", got.ID)
	assert.Equal(t, "test-task", got.Name)

	// Update progress
	err = repo.UpdateTaskProgress("task-1", 50, "running")
	require.NoError(t, err)

	// Get progress
	progress, err := repo.GetTaskProgress("task-1")
	require.NoError(t, err)
	assert.Equal(t, int32(50), progress)

	// List
	tasks, err := repo.ListTasks("project1")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "csv", tasks[0].SourceType)

	// Delete
	err = repo.DeleteTask("task-1", "project1")
	require.NoError(t, err)

	tasks, err = repo.ListTasks("project1")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestMetadataRepository_GetTaskByID_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	task, err := repo.GetTaskByID("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, task)
}

func TestMetadataRepository_GetTaskProgress_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	_, err := repo.GetTaskProgress("nonexistent")
	assert.Error(t, err)
}

func TestMetadataRepository_Lifecycle_Agent(t *testing.T) {
	repo := setupMetadataRepo(t)

	agent := &AgentRecord{
		ID:           "agent-1",
		ProjectID:    "project1",
		Name:         "test-agent",
		Provider:     "openai",
		Model:        "gpt-4",
		ApiKey:       "sk-test",
		SystemPrompt: "You are a test agent",
		SkillIDsJSON: `["skill-1"]`,
		BaseURL:      "",
	}

	err := repo.CreateAgent(agent)
	require.NoError(t, err)

	// Confirm in project
	exists, err := repo.ConfirmAgentInProject("agent-1", "project1")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ConfirmAgentInProject("agent-1", "wrong-project")
	require.NoError(t, err)
	assert.False(t, exists)

	// Get for chat
	got, err := repo.GetAgentForChat("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", got.ID)
	assert.Equal(t, "openai", got.Provider)

	// List
	agents, err := repo.ListAgents("project1")
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "test-agent", agents[0].Name)

	// List cursor
	cursorAgents, err := repo.ListAgentsCursor("project1", "", 10)
	require.NoError(t, err)
	assert.Len(t, cursorAgents, 1)

	// Update
	agent.Name = "updated-agent"
	err = repo.UpdateAgent(agent)
	require.NoError(t, err)

	agents, err = repo.ListAgents("project1")
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "updated-agent", agents[0].Name)

	// Delete
	err = repo.DeleteAgent("agent-1", "project1")
	require.NoError(t, err)

	exists, err = repo.ConfirmAgentInProject("agent-1", "project1")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMetadataRepository_ConfirmAgentInProject_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	exists, err := repo.ConfirmAgentInProject("nonexistent", "project1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMetadataRepository_GetAgentForChat_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	_, err := repo.GetAgentForChat("nonexistent")
	assert.Error(t, err)
}

func TestMetadataRepository_Lifecycle_Tool(t *testing.T) {
	repo := setupMetadataRepo(t)

	tool := &ToolRecord{
		ID:           "tool-1",
		Name:         "test-tool",
		Description:  "A test tool",
		Code:         `{"type": "object"}`,
		Category:     "analysis",
		Version:      "1.0.0",
		HealthStatus: "healthy",
		SourceType:   "builtin",
	}

	err := repo.CreateTool(tool)
	require.NoError(t, err)

	// List
	tools, err := repo.ListTools()
	require.NoError(t, err)
	assert.Len(t, tools, 1)
	assert.Equal(t, "test-tool", tools[0].Name)

	// List cursor
	cursorTools, err := repo.ListToolsCursor("", 10)
	require.NoError(t, err)
	assert.Len(t, cursorTools, 1)

	// Get by category
	categoryTools, err := repo.GetToolByCategory("analysis")
	require.NoError(t, err)
	assert.Len(t, categoryTools, 1)
	assert.Equal(t, "test-tool", categoryTools[0].Name)

	// Update code
	err = repo.UpdateToolCode(context.Background(), "tool-1", `{"type": "updated"}`)
	require.NoError(t, err)

	// Update health
	err = repo.UpdateHealthStatus("tool-1", "degraded")
	require.NoError(t, err)

	tools, err = repo.ListTools()
	require.NoError(t, err)
	assert.Len(t, tools, 1)

	// Delete
	err = repo.DeleteTool("tool-1")
	require.NoError(t, err)

	tools, err = repo.ListTools()
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestMetadataRepository_Lifecycle_Skill(t *testing.T) {
	repo := setupMetadataRepo(t)

	skill := &SkillRecord{
		ID:          "skill-1",
		ProjectID:   "project1",
		Name:        "test-skill",
		Description: "A test skill",
		ToolIDsJSON: `["tool-1", "tool-2"]`,
	}

	err := repo.CreateSkill(skill)
	require.NoError(t, err)

	// List
	skills, err := repo.ListSkills("project1")
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "test-skill", skills[0].Name)

	// Get tool IDs
	toolIDs, err := repo.GetSkillToolIDs("skill-1")
	require.NoError(t, err)
	assert.Equal(t, `["tool-1", "tool-2"]`, toolIDs)

	// List cursor
	cursorSkills, err := repo.ListSkillsCursor("project1", "", 10)
	require.NoError(t, err)
	assert.Len(t, cursorSkills, 1)

	// Delete
	err = repo.DeleteSkill("skill-1", "project1")
	require.NoError(t, err)

	skills, err = repo.ListSkills("project1")
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestMetadataRepository_GetSkillToolIDs_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	_, err := repo.GetSkillToolIDs("nonexistent")
	assert.Error(t, err)
}

func TestMetadataRepository_Lifecycle_APIKey(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.CreateAPIKey("key-1", "project1", "test-key", "hashed-value")
	require.NoError(t, err)

	// Validate
	pid, err := repo.ValidateAPIKey("hashed-value")
	assert.NoError(t, err)
	assert.Equal(t, "project1", pid)

	// List
	keys, err := repo.ListAPIKeys("project1")
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "test-key", keys[0].Label)

	// Delete
	err = repo.DeleteAPIKey("key-1", "project1")
	require.NoError(t, err)

	keys, err = repo.ListAPIKeys("project1")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestMetadataRepository_ValidateAPIKey_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	_, err := repo.ValidateAPIKey("nonexistent")
	assert.Error(t, err)
}

func TestMetadataRepository_Lifecycle_ChatMessage(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.SaveChatMessage(testCtx, "project1", "agent-1", "user", "Hello", "")
	require.NoError(t, err)

	err = repo.SaveChatMessage(testCtx, "project1", "agent-1", "assistant", "Hi there!", "")
	require.NoError(t, err)

	// Get messages
	messages, err := repo.GetChatMessages(testCtx, "project1", "agent-1")
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)

	// Get chat history (alias)
	history, err := repo.GetChatHistory(testCtx, "project1", "agent-1")
	require.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestMetadataRepository_GetChatMessages_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	messages, err := repo.GetChatMessages(testCtx, "project1", "agent-1")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestMetadataRepository_ListToolsCursor_WithAfter(t *testing.T) {
	repo := setupMetadataRepo(t)

	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		err := repo.CreateTool(&ToolRecord{ID: id + "-tool", Name: id + "-tool", SourceType: "builtin"})
		require.NoError(t, err)
	}

	tools, err := repo.ListToolsCursor("a-tool", 2)
	require.NoError(t, err)
	// Should return tools after 'a-tool'
	assert.NotEmpty(t, tools)
}

func TestMetadataRepository_ListSkillsCursor_WithAfter(t *testing.T) {
	repo := setupMetadataRepo(t)

	for i := 0; i < 3; i++ {
		id := string(rune('a' + i))
		err := repo.CreateSkill(&SkillRecord{ID: id + "-skill", ProjectID: "project1", Name: id + "-skill"})
		require.NoError(t, err)
	}

	skills, err := repo.ListSkillsCursor("project1", "a-skill", 2)
	require.NoError(t, err)
	assert.NotEmpty(t, skills)
}

func TestMetadataRepository_GetToolCode(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.CreateTool(&ToolRecord{
		ID:   "tool-code",
		Name: "tool-code",
		Code: `{"type": "object", "properties": {}}`,
	})
	require.NoError(t, err)

	code, err := repo.GetToolCode(context.Background(), "tool-code")
	require.NoError(t, err)
	assert.Equal(t, `{"type": "object", "properties": {}}`, code)
}

func TestMetadataRepository_GetToolCode_NotFound(t *testing.T) {
	repo := setupMetadataRepo(t)

	_, err := repo.GetToolCode(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestMetadataRepository_DeleteAgent_NotInProject(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.DeleteAgent("nonexistent", "project1")
	assert.NoError(t, err)
}

func TestMetadataRepository_DeleteSkill_NotInProject(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.DeleteSkill("nonexistent", "project1")
	assert.NoError(t, err)
}

func TestMetadataRepository_DeleteAPIKey_NotInProject(t *testing.T) {
	repo := setupMetadataRepo(t)

	err := repo.DeleteAPIKey("nonexistent", "project1")
	assert.NoError(t, err)
}

func TestMetadataRepository_ListAPIKeys_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	keys, err := repo.ListAPIKeys("project1")
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestMetadataRepository_ListAgents_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	agents, err := repo.ListAgents("project1")
	assert.NoError(t, err)
	assert.Empty(t, agents)
}

func TestMetadataRepository_ListTools_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	tools, err := repo.ListTools()
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

func TestMetadataRepository_ListSkills_Empty(t *testing.T) {
	repo := setupMetadataRepo(t)

	skills, err := repo.ListSkills("project1")
	assert.NoError(t, err)
	assert.Empty(t, skills)
}
