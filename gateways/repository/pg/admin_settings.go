package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"go-template/domain/entities"
	"go-template/gateways/repository/pg/gen"
)

type AdminSettingsRepository struct {
	queries *gen.Queries
	db      DBTX
}

func NewAdminSettingsRepository(db DBTX) *AdminSettingsRepository {
	return &AdminSettingsRepository{
		queries: gen.New(db),
		db:      db,
	}
}

func (r *AdminSettingsRepository) GetSettings(ctx context.Context) (*entities.SystemSettings, error) {
	settings, err := r.queries.GetAllAdminSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin settings: %w", err)
	}

	// Initialize with defaults
	result := &entities.SystemSettings{
		MaintenanceMode:     false,
		RegistrationEnabled: true,
		EmailNotifications:  true,
		SessionTimeout:      1440, // 24 hours in minutes
		MinPasswordLength:   8,
		Require2FA:          false,
		AutoBackup:          true,
		BackupRetentionDays: 30,
	}

	// Override with database values
	for _, setting := range settings {
		switch setting.Key {
		case "maintenance_mode":
			var value bool
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.MaintenanceMode = value
			}
		case "registration_enabled":
			var value bool
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.RegistrationEnabled = value
			}
		case "email_notifications":
			var value bool
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.EmailNotifications = value
			}
		case "session_timeout":
			var value int
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.SessionTimeout = value
			}
		case "min_password_length":
			var value int
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.MinPasswordLength = value
			}
		case "require_2fa":
			var value bool
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.Require2FA = value
			}
		case "auto_backup":
			var value bool
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.AutoBackup = value
			}
		case "backup_retention_days":
			var value int
			if err := json.Unmarshal(setting.Value, &value); err == nil {
				result.BackupRetentionDays = value
			}
		}
	}

	return result, nil
}

func (r *AdminSettingsRepository) UpdateSettings(ctx context.Context, settings *entities.SystemSettings) error {
	// Convert settings to key-value pairs
	settingUpdates := map[string]any{
		"maintenance_mode":      settings.MaintenanceMode,
		"registration_enabled":  settings.RegistrationEnabled,
		"email_notifications":   settings.EmailNotifications,
		"session_timeout":       settings.SessionTimeout,
		"min_password_length":   settings.MinPasswordLength,
		"require_2fa":          settings.Require2FA,
		"auto_backup":          settings.AutoBackup,
		"backup_retention_days": settings.BackupRetentionDays,
	}

	// Update each setting
	for key, value := range settingUpdates {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal setting %s: %w", key, err)
		}

		if err := r.queries.UpsertAdminSetting(ctx, key, valueBytes); err != nil {
			return fmt.Errorf("failed to upsert setting %s: %w", key, err)
		}
	}

	return nil
}

func (r *AdminSettingsRepository) GetSetting(ctx context.Context, key string) (any, error) {
	setting, err := r.queries.GetAdminSetting(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	var value any
	if err := json.Unmarshal(setting.Value, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal setting %s: %w", key, err)
	}

	return value, nil
}

func (r *AdminSettingsRepository) SetSetting(ctx context.Context, key string, value any) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal setting %s: %w", key, err)
	}

	if err := r.queries.UpsertAdminSetting(ctx, key, valueBytes); err != nil {
		return fmt.Errorf("failed to upsert setting %s: %w", key, err)
	}

	return nil
}