-- Initial schema migration for PostgreSQL system tables
-- Contains: system_tasks, system_simulations, system_proposals, system_chat_history,
--           system_agents, system_skills, system_tools, system_api_keys,
--           system_notification_channels (from internal/repository/metadata.go)

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