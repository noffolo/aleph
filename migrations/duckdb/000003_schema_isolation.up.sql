-- Migration: 000003_schema_isolation
-- Creates per-project DuckDB schemas for cross-project data isolation.
-- Each project's data is stored in project_{id} schema.
--
-- SECURITY: This is CRITICAL for multi-tenant isolation.
-- A project can only see its own tables and data.

-- Create the default schema for backward compatibility
CREATE SCHEMA IF NOT EXISTS project_default;

-- Components table in project_default schema (replica of main.components)
CREATE TABLE IF NOT EXISTS project_default.components (
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

-- System features table in project_default schema
CREATE TABLE IF NOT EXISTS project_default.system_features (
	project_id VARCHAR,
	task_id VARCHAR,
	entity_id VARCHAR,
	feature_type VARCHAR,
	feature_value FLOAT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Migrate existing data from main schema if tables exist and data hasn't been migrated yet
INSERT INTO project_default.components
	SELECT * FROM main.components
	WHERE NOT EXISTS (SELECT 1 FROM project_default.components);

INSERT INTO project_default.system_features
	SELECT * FROM main.system_features
	WHERE NOT EXISTS (SELECT 1 FROM project_default.system_features);
