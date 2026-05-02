-- Migration 000009: Add constraints, foreign keys, and indexes
--
-- 1. Create system_chat_sessions table (referenced in metadata.go but missing)
-- 2. Set NOT NULL constraints on project_id columns
-- 3. Add FOREIGN KEY constraints with ON DELETE CASCADE
-- 4. Add performance indexes

-- ---------------------------------------------------------------
-- 1. Create system_chat_sessions (must exist before FK constraints)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS system_chat_sessions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    agent_id TEXT,
    title TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------
-- 2. NOT NULL constraints on project_id
-- ---------------------------------------------------------------
ALTER TABLE system_agents ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_skills ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_tasks ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_chat_history ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE system_chat_sessions ALTER COLUMN project_id SET NOT NULL;

-- ---------------------------------------------------------------
-- 3. Foreign key constraints (ON DELETE CASCADE)
-- ---------------------------------------------------------------
ALTER TABLE system_agents
    ADD CONSTRAINT fk_agents_project
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

ALTER TABLE system_skills
    ADD CONSTRAINT fk_skills_project
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

ALTER TABLE system_tasks
    ADD CONSTRAINT fk_tasks_project
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

ALTER TABLE system_chat_history
    ADD CONSTRAINT fk_chat_agent
    FOREIGN KEY (agent_id) REFERENCES system_agents(id) ON DELETE CASCADE;

ALTER TABLE system_chat_history
    ADD CONSTRAINT fk_chat_history_project
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

ALTER TABLE system_chat_sessions
    ADD CONSTRAINT fk_sessions_project
    FOREIGN KEY (project_id) REFERENCES system_projects(id) ON DELETE CASCADE;

-- ---------------------------------------------------------------
-- 4. Performance indexes
-- ---------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_agents_project_status ON system_agents(project_id, status);
CREATE INDEX IF NOT EXISTS idx_skills_project ON system_skills(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project_status ON system_tasks(project_id, status);
CREATE INDEX IF NOT EXISTS idx_chat_history_agent_created ON system_chat_history(agent_id, created_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_project ON system_api_keys(project_id);