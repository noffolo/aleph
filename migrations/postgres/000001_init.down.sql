-- Down migration: drop all PostgreSQL system tables created in the initial schema

DROP TABLE IF EXISTS system_tasks;
DROP TABLE IF EXISTS system_simulations;
DROP TABLE IF EXISTS system_proposals;
DROP TABLE IF EXISTS system_chat_history;
DROP TABLE IF EXISTS system_agents;
DROP TABLE IF EXISTS system_skills;
DROP TABLE IF EXISTS system_tools;
DROP TABLE IF EXISTS system_api_keys;
DROP TABLE IF EXISTS system_notification_channels;