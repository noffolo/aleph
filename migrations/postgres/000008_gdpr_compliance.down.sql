DROP TABLE IF EXISTS data_retention_policy;
DROP INDEX IF EXISTS idx_audit_log_project_id;
DROP INDEX IF EXISTS idx_audit_log_action_timestamp;
ALTER TABLE audit_log DROP COLUMN IF EXISTS project_id;
