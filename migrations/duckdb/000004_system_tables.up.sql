-- Migration: 000004_system_tables
-- Adds system_* tables (excluding system_features which exists in 000001)
-- These tables support integration tests and general application metadata storage.
-- Source: /migrations/000001_init_schema.up.sql (lines 34-126)

-- Table: system_tasks (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_tasks (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	name VARCHAR,
	source_type VARCHAR,
	config_json VARCHAR,
	status VARCHAR,
	progress INTEGER,
	schedule VARCHAR DEFAULT '',
	is_predictive INTEGER DEFAULT 0,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_simulations (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_simulations (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	task_id VARCHAR,
	scenario_name VARCHAR,
	results_json VARCHAR,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_proposals (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_proposals (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	action_name VARCHAR,
	object_id VARCHAR,
	params_json VARCHAR,
	status VARCHAR DEFAULT 'pending',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_chat_history (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_chat_history (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	agent_id VARCHAR,
	role VARCHAR,
	content VARCHAR,
	tool_call VARCHAR,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_agents (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_agents (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	name VARCHAR,
	provider VARCHAR,
	model VARCHAR,
	api_key VARCHAR,
	system_prompt VARCHAR,
	skill_ids VARCHAR,
	base_url VARCHAR DEFAULT ''
);

-- Table: system_skills (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_skills (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	name VARCHAR,
	description VARCHAR,
	tool_ids VARCHAR
);

-- Table: system_tools (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_tools (
	id VARCHAR PRIMARY KEY,
	name VARCHAR,
	description VARCHAR,
	code VARCHAR
);

-- Table: system_api_keys (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_api_keys (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	label VARCHAR,
	key VARCHAR,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_notification_channels (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_notification_channels (
	id VARCHAR PRIMARY KEY,
	project_id VARCHAR,
	name VARCHAR,
	type VARCHAR,
	config_json VARCHAR,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
