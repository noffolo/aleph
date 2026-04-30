-- Migration: 000006_ontology_versions
-- Adds ontology version history table for accepting/rejecting ontology diffs.

CREATE TABLE IF NOT EXISTS ontology_versions (
    version_id        VARCHAR PRIMARY KEY,
    project_id        VARCHAR NOT NULL,
    parent_version_id VARCHAR,
    diff_json         VARCHAR NOT NULL,
    core_aleph_snapshot VARCHAR NOT NULL,
    status            VARCHAR DEFAULT 'pending',
    source_description VARCHAR,
    rationale         TEXT,
    confidence        FLOAT,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at       TIMESTAMP
);
