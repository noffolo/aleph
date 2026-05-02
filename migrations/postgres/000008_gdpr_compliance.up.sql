-- Migration 000008: GDPR compliance — data retention policies
--
-- Adds:
--   1. project_id column to audit_log for per-project audit queries
--   2. data_retention_policy table for configurable retention rules
--   3. Indexes for efficient cleanup queries

ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS project_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_audit_log_project_id ON audit_log(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_log_action_timestamp ON audit_log(action, timestamp);

CREATE TABLE IF NOT EXISTS data_retention_policy (
    id TEXT PRIMARY KEY,
    resource_type VARCHAR(100) NOT NULL,  -- agent, tool, skill, task, api_key, chat_history, audit_log, project
    retention_days INTEGER NOT NULL DEFAULT 90,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_retention_policy_resource ON data_retention_policy(resource_type);

-- Default retention policies (GDPR-compliant defaults)
INSERT INTO data_retention_policy (id, resource_type, retention_days, enabled) VALUES
    ('retention_chat_history', 'chat_history', 365, true),
    ('retention_audit_log', 'audit_log', 730, true),  -- 2 years for audit
    ('retention_deleted_project', 'project', 30, true) -- grace period before permanent purge
ON CONFLICT (id) DO NOTHING;
