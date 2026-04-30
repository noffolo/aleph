-- Add role column to system_api_keys for RBAC support.
-- Default is 'user' for backward compatibility with existing keys.
ALTER TABLE system_api_keys ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

-- Migrate any backend/admin keys: match ALEPH_API_KEY_SECRET_BACKEND
-- by setting the single key whose label contains 'backend' or 'admin' to admin role.
-- This is a no-op if those keys don't exist or use different labels.
UPDATE system_api_keys SET role = 'admin' WHERE label ILIKE '%backend%' OR label ILIKE '%admin%';