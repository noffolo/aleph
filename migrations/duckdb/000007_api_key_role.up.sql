ALTER TABLE system_api_keys ADD COLUMN role VARCHAR DEFAULT 'user';
UPDATE system_api_keys SET role = 'user' WHERE role IS NULL;
