-- Down migration: drops all system_* tables added in 000004
-- WARNING: This will permanently delete all application metadata.

DROP TABLE IF EXISTS system_tasks;
DROP TABLE IF EXISTS system_simulations;
DROP TABLE IF EXISTS system_proposals;
DROP TABLE IF EXISTS system_chat_history;
DROP TABLE IF EXISTS system_agents;
DROP TABLE IF EXISTS system_skills;
DROP TABLE IF EXISTS system_tools;
DROP TABLE IF EXISTS system_api_keys;
DROP TABLE IF EXISTS system_notification_channels;
