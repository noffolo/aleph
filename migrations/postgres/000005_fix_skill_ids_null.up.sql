-- Aleph-v2 migration 000005: Fix skill_ids NULL values
-- system_agents.skill_ids must default to '[]' for JSON array compatibility.
-- Backfill any existing NULL values.
UPDATE system_agents SET skill_ids = '[]' WHERE skill_ids IS NULL;
