-- Down migration: drop all DuckDB tables created in the initial schema

DROP TABLE IF EXISTS components;
DROP TABLE IF EXISTS system_features;