-- Remove auth provider fields from admin_settings table
UPDATE admin_settings 
SET value = value::jsonb - 'available_auth_providers' - 'default_auth_provider'
WHERE key = 'system_settings';