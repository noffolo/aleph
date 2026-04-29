-- Migration: 000005_tool_usage
-- Creates tool_usage table for tracking tool call executions.
-- Used by the Usage Tracking Subsystem (W1.5-04).

CREATE TABLE IF NOT EXISTS tool_usage (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    project_id  VARCHAR NOT NULL,
    tool_name   VARCHAR NOT NULL,
    input_hash  VARCHAR,
    duration_ms BIGINT,
    success     BOOLEAN DEFAULT TRUE,
    error_msg   VARCHAR DEFAULT '',
    timestamp   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tool_usage_user_time ON tool_usage(user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_tool_usage_tool ON tool_usage(tool_name);
