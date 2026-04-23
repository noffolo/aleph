-- Initial schema migration for DuckDB tables
-- This captures all existing tables from the codebase

-- Table: components (from internal/registry/duckdb_registry.go)
CREATE TABLE IF NOT EXISTS components (
	id VARCHAR PRIMARY KEY,
	name VARCHAR,
	description VARCHAR,
	version VARCHAR,
	type VARCHAR,
	category VARCHAR,
	source VARCHAR,
	status VARCHAR,
	approval_status VARCHAR,
	config_schema_json VARCHAR,
	execution_command VARCHAR,
	dependencies_json VARCHAR,
	input_schema_json VARCHAR,
	output_schema_json VARCHAR,
	prompt_template VARCHAR,
	tool_ids_json VARCHAR,
	avg_cpu_usage DOUBLE DEFAULT 0,
	avg_memory_mb DOUBLE DEFAULT 0,
	avg_exec_time_ms DOUBLE DEFAULT 0,
	avg_brier_score DOUBLE DEFAULT 0,
	avg_latency_ms DOUBLE DEFAULT 0,
	trust_score DOUBLE DEFAULT 0,
	created_by_agent_id VARCHAR,
	creation_timestamp TIMESTAMP,
	last_updated_timestamp TIMESTAMP
);

-- Table: system_tasks (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_tasks (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	name TEXT, 
	source_type TEXT, 
	config_json TEXT, 
	status TEXT, 
	progress INTEGER, 
	schedule TEXT DEFAULT '', 
	is_predictive INTEGER DEFAULT 0, 
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_simulations (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_simulations (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	task_id TEXT, 
	scenario_name TEXT, 
	results_json TEXT, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_proposals (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_proposals (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	action_name TEXT, 
	object_id TEXT, 
	params_json TEXT, 
	status TEXT DEFAULT 'pending', 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_chat_history (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_chat_history (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	agent_id TEXT, 
	role TEXT, 
	content TEXT, 
	tool_call TEXT, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_agents (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_agents (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	name TEXT, 
	provider TEXT, 
	model TEXT, 
	api_key TEXT, 
	system_prompt TEXT, 
	skill_ids TEXT, 
	base_url TEXT DEFAULT ''
);

-- Table: system_skills (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_skills (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	name TEXT, 
	description TEXT, 
	tool_ids TEXT
);

-- Table: system_tools (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_tools (
	id TEXT PRIMARY KEY, 
	name TEXT, 
	description TEXT, 
	code TEXT
);

-- Table: system_api_keys (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_api_keys (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	label TEXT, 
	key TEXT, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_notification_channels (from internal/repository/metadata.go)
CREATE TABLE IF NOT EXISTS system_notification_channels (
	id TEXT PRIMARY KEY, 
	project_id TEXT, 
	name TEXT, 
	type TEXT, 
	config_json TEXT, 
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table: system_features (from internal/ingestion/engine.go)
CREATE TABLE IF NOT EXISTS system_features (
	project_id VARCHAR,
	task_id VARCHAR,
	entity_id VARCHAR,
	feature_type VARCHAR,
	feature_value FLOAT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Install and load VSS extension for vector similarity search (from internal/storage/duckdb.go)
INSTALL vss;
LOAD vss;