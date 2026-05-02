-- Migration 000007: system_projects table for multi-tenancy foundations
-- Tracks project records and enables soft limits (MaxProjects/MaxAgents)

CREATE TABLE IF NOT EXISTS system_projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
