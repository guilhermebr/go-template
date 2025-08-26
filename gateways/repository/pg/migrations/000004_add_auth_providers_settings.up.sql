-- Add auth provider fields to admin_settings table
UPDATE admin_settings 
SET value = jsonb_set(
    jsonb_set(
        value::jsonb,
        '{available_auth_providers}',
        '["supabase"]'::jsonb
    ),
    '{default_auth_provider}',
    '"supabase"'::jsonb
)
WHERE key = 'system_settings';

-- Insert default settings if they don't exist
INSERT INTO admin_settings (key, value, created_at, updated_at)
SELECT 'system_settings', 
       '{
         "maintenance_mode": false,
         "registration_enabled": true,
         "email_notifications": true,
         "session_timeout": 1440,
         "min_password_length": 8,
         "require_2fa": false,
         "auto_backup": true,
         "backup_retention_days": 30,
         "available_auth_providers": ["supabase"],
         "default_auth_provider": "supabase"
       }'::jsonb,
       now(),
       now()
WHERE NOT EXISTS (SELECT 1 FROM admin_settings WHERE key = 'system_settings');