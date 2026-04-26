-- W3-08: Audit logging middleware migration
-- Creates audit_log table for tracking mutating operations

CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(255),
  action VARCHAR(50) NOT NULL,  -- create, update, delete
  resource_type VARCHAR(100) NOT NULL,  -- agent, tool, skill, ingestion, task, api_key, notification, etc.
  resource_id VARCHAR(255) NOT NULL,
  timestamp TIMESTAMPTZ DEFAULT NOW(),
  diff JSONB  -- old vs new values (nullable)
);

CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id) WHERE user_id IS NOT NULL;