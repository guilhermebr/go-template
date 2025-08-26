package settings

import (
	"context"
	"go-template/domain/entities"
	"log/slog"
	"slices"
)

type UseCase struct {
	repo   Repository
	logger *slog.Logger
}

func NewUseCase(repo Repository, logger *slog.Logger) *UseCase {
	return &UseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *UseCase) GetSettings(ctx context.Context) (*entities.SystemSettings, error) {
	settings, err := uc.repo.GetSettings(ctx)
	if err != nil {
		uc.logger.Error("failed to get settings", "error", err)
		return nil, err
	}

	uc.logger.Debug("retrieved system settings")
	return settings, nil
}

func (uc *UseCase) UpdateSettings(ctx context.Context, settings *entities.SystemSettings) error {
	if err := uc.validateSettings(settings); err != nil {
		uc.logger.Warn("invalid settings provided", "error", err)
		return err
	}

	if err := uc.repo.UpdateSettings(ctx, settings); err != nil {
		uc.logger.Error("failed to update settings", "error", err)
		return err
	}

	uc.logger.Info("system settings updated")
	return nil
}

func (uc *UseCase) GetSetting(ctx context.Context, key string) (any, error) {
	value, err := uc.repo.GetSetting(ctx, key)
	if err != nil {
		uc.logger.Error("failed to get setting", "key", key, "error", err)
		return nil, err
	}

	return value, nil
}

func (uc *UseCase) SetSetting(ctx context.Context, key string, value any) error {
	if err := uc.repo.SetSetting(ctx, key, value); err != nil {
		uc.logger.Error("failed to set setting", "key", key, "error", err)
		return err
	}

	uc.logger.Debug("setting updated", "key", key)
	return nil
}

func (uc *UseCase) validateSettings(settings *entities.SystemSettings) error {
	// Validate session timeout
	if settings.SessionTimeout < 15 || settings.SessionTimeout > 10080 {
		return entities.ErrInvalidSettingValue{Field: "session_timeout", Message: "must be between 15 and 10080 minutes"}
	}

	// Validate minimum password length
	if settings.MinPasswordLength < 6 || settings.MinPasswordLength > 128 {
		return entities.ErrInvalidSettingValue{Field: "min_password_length", Message: "must be between 6 and 128 characters"}
	}

	// Validate backup retention days
	if settings.BackupRetentionDays < 1 || settings.BackupRetentionDays > 365 {
		return entities.ErrInvalidSettingValue{Field: "backup_retention_days", Message: "must be between 1 and 365 days"}
	}

	// Validate auth providers
	if len(settings.AvailableAuthProviders) == 0 {
		return entities.ErrInvalidSettingValue{Field: "available_auth_providers", Message: "at least one auth provider must be available"}
	}

	// Validate supported auth providers
	supportedProviders := map[string]bool{
		"supabase": true,
		// Add more providers here as they're implemented
	}

	for _, provider := range settings.AvailableAuthProviders {
		if !supportedProviders[provider] {
			return entities.ErrInvalidSettingValue{Field: "available_auth_providers", Message: "unsupported provider: " + provider}
		}
	}

	// Validate default auth provider
	if settings.DefaultAuthProvider == "" {
		return entities.ErrInvalidSettingValue{Field: "default_auth_provider", Message: "default auth provider must be specified"}
	}

	// Ensure default provider is in available providers
	if !slices.Contains(settings.AvailableAuthProviders, settings.DefaultAuthProvider) {
		return entities.ErrInvalidSettingValue{Field: "default_auth_provider", Message: "default provider must be in available providers list"}
	}

	return nil
}
