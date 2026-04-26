-- Down migration: removes per-project schemas
-- WARNING: This will destroy per-project data isolation.
-- Data is moved back to main schema.

-- Move data back to main schema before dropping
INSERT INTO main.components SELECT * FROM project_default.components
	WHERE NOT EXISTS (SELECT 1 FROM main.components);

INSERT INTO main.system_features SELECT * FROM project_default.system_features
	WHERE NOT EXISTS (SELECT 1 FROM main.system_features);

DROP TABLE IF EXISTS project_default.components;
DROP TABLE IF EXISTS project_default.system_features;

DROP SCHEMA IF EXISTS project_default;
