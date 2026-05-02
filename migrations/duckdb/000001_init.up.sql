-- Initial schema migration for DuckDB tables
-- Contains: components, system_features, and VSS extension

-- Table: components (from internal/registry/duckdb_registry.go)
CREATE TABLE IF NOT EXISTS components (
	id TEXT PRIMARY KEY,
	name TEXT,
	description TEXT,
	version TEXT,
	type TEXT,
	category TEXT,
	source TEXT,
	status TEXT,
	approval_status TEXT,
	config_schema_json TEXT,
	execution_command TEXT,
	dependencies_json TEXT,
	input_schema_json TEXT,
	output_schema_json TEXT,
	prompt_template TEXT,
	tool_ids_json TEXT,
	avg_cpu_usage DOUBLE DEFAULT 0,
	avg_memory_mb DOUBLE DEFAULT 0,
	avg_exec_time_ms DOUBLE DEFAULT 0,
	avg_brier_score DOUBLE DEFAULT 0,
	avg_latency_ms DOUBLE DEFAULT 0,
	trust_score DOUBLE DEFAULT 0,
	created_by_agent_id TEXT,
	creation_timestamp TIMESTAMP,
	last_updated_timestamp TIMESTAMP
);

CREATE TABLE IF NOT EXISTS system_features (
	project_id TEXT,
	task_id TEXT,
	entity_id TEXT,
	feature_type TEXT,
	feature_value FLOAT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Install and load VSS extension for vector similarity search (from internal/storage/duckdb.go)
INSTALL vss;
LOAD vss;