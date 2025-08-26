package types

import "go-template/domain/entities"

// Admin Dashboard Stats
type DashboardStats struct {
	TotalUsers     int64 `json:"total_users"`
	AdminUsers     int64 `json:"admin_users"`
	ActiveSessions int64 `json:"active_sessions"`
	SystemAlerts   int64 `json:"system_alerts"`
}

// User List Response
type UserListResponse struct {
	Users      []entities.User `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// System Settings
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