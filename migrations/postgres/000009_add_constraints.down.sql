-- Reverse migration 000009: Drop indexes, foreign keys, NOT NULL constraints, and chat_sessions table

DROP INDEX IF EXISTS idx_api_keys_project;
DROP INDEX IF EXISTS idx_chat_history_agent_created;
DROP INDEX IF EXISTS idx_tasks_project_status;
DROP INDEX IF EXISTS idx_skills_project;
DROP INDEX IF EXISTS idx_agents_project_status;

ALTER TABLE system_chat_sessions DROP CONSTRAINT IF EXISTS fk_sessions_project;
ALTER TABLE system_chat_history DROP CONSTRAINT IF EXISTS fk_chat_history_project;
ALTER TABLE system_chat_history DROP CONSTRAINT IF EXISTS fk_chat_agent;
ALTER TABLE system_tasks DROP CONSTRAINT IF EXISTS fk_tasks_project;
ALTER TABLE system_skills DROP CONSTRAINT IF EXISTS fk_skills_project;
ALTER TABLE system_agents DROP CONSTRAINT IF EXISTS fk_agents_project;

ALTER TABLE system_chat_sessions ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_chat_history ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_tasks ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_skills ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE system_agents ALTER COLUMN project_id DROP NOT NULL;

DROP TABLE IF EXISTS system_chat_sessions;