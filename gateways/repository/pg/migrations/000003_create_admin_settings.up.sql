-- Create admin settings table
CREATE TABLE admin_settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create index on updated_at for performance
CREATE INDEX idx_admin_settings_updated_at ON admin_settings (updated_at);

-- Insert default settings
INSERT INTO admin_settings (key, value) VALUES 
('maintenance_mode', 'false'),
('registration_enabled', 'true'),
('email_notifications', 'true')
ON CONFLICT (key) DO NOTHING;