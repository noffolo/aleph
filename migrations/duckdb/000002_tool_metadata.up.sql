-- Ensure system_tools table exists (this migration depends on it)
-- Full definition in 000004; this is a safety net for ordering
CREATE TABLE IF NOT EXISTS system_tools (
	id VARCHAR PRIMARY KEY,
	name VARCHAR,
	description VARCHAR,
	code VARCHAR
);

ALTER TABLE system_tools ADD COLUMN IF NOT EXISTS category VARCHAR DEFAULT '';
ALTER TABLE system_tools ADD COLUMN IF NOT EXISTS version VARCHAR DEFAULT '1.0.0';
ALTER TABLE system_tools ADD COLUMN IF NOT EXISTS health_status VARCHAR DEFAULT 'unknown';
ALTER TABLE system_tools ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE system_tools ADD COLUMN IF NOT EXISTS source_type VARCHAR DEFAULT 'builtin';