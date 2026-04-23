-- Initial schema migration for DuckDB tables
-- Contains: components, system_features, and VSS extension

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