ALTER TABLE system_tools DROP COLUMN IF EXISTS category;
ALTER TABLE system_tools DROP COLUMN IF EXISTS version;
ALTER TABLE system_tools DROP COLUMN IF EXISTS health_status;
ALTER TABLE system_tools DROP COLUMN IF EXISTS last_checked_at;
ALTER TABLE system_tools DROP COLUMN IF EXISTS source_type;