package entities

import "fmt"

// SystemSettings represents system-wide configuration settings
type SystemSettings struct {
	MaintenanceMode        bool     `json:"maintenance_mode"`
	RegistrationEnabled    bool     `json:"registration_enabled"`
	EmailNotifications     bool     `json:"email_notifications"`
	SessionTimeout         int      `json:"session_timeout"`        // in minutes
	MinPasswordLength      int      `json:"min_password_length"`
	Require2FA             bool     `json:"require_2fa"`
	AutoBackup             bool     `json:"auto_backup"`
	BackupRetentionDays    int      `json:"backup_retention_days"`
	AvailableAuthProviders []string `json:"available_auth_providers"`
	DefaultAuthProvider    string   `json:"default_auth_provider"`
}

// ErrInvalidSettingValue represents a validation error for settings
type ErrInvalidSettingValue struct {
	Field   string
	Message string
}

func (e ErrInvalidSettingValue) Error() string {
	return fmt.Sprintf("invalid value for %s: %s", e.Field, e.Message)
}