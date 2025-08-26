package pg

import (
	"context"
	"database/sql"
	"fmt"
	"go-template/domain"
	"go-template/domain/entities"
	"go-template/gateways/repository/pg/gen"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct {
	queries *gen.Queries
	db      DBTX
}

func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{
		queries: gen.New(db),
		db:      db,
	}
}

func (r *UserRepository) Create(ctx context.Context, user entities.User) error {
	err := r.queries.CreateUser(ctx, gen.CreateUserParams{
		ID:             user.ID,
		Email:          user.Email,
		AuthProvider:   user.AuthProvider,
		AuthProviderID: &user.AuthProviderID,
		AccountType:    gen.AccountType(user.AccountType),
		CreatedAt:      &user.CreatedAt,
		UpdatedAt:      &user.UpdatedAt,
	})
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return fmt.Errorf("user with email '%s' already exists: %w", user.Email, domain.ErrDuplicateKey)
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (entities.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return entities.User{}, domain.ErrNotFound
		}
		return entities.User{}, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return entities.User{
		ID:             user.ID,
		Email:          user.Email,
		AuthProvider:   user.AuthProvider,
		AuthProviderID: *user.AuthProviderID,
		AccountType:    entities.AccountType(user.AccountType),
		CreatedAt:      *user.CreatedAt,
		UpdatedAt:      *user.UpdatedAt,
	}, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (entities.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return entities.User{}, domain.ErrNotFound
		}
		return entities.User{}, fmt.Errorf("failed to get user by email: %w", err)
	}

	return entities.User{
		ID:             user.ID,
		Email:          user.Email,
		AuthProvider:   user.AuthProvider,
		AuthProviderID: *user.AuthProviderID,
		AccountType:    entities.AccountType(user.AccountType),
		CreatedAt:      *user.CreatedAt,
		UpdatedAt:      *user.UpdatedAt,
	}, nil
}

func (r *UserRepository) Update(ctx context.Context, user entities.User) error {
	err := r.queries.UpdateUser(ctx, gen.UpdateUserParams{
		ID:             user.ID,
		Email:          user.Email,
		AuthProvider:   user.AuthProvider,
		AuthProviderID: &user.AuthProviderID,
		AccountType:    gen.AccountType(user.AccountType),
		UpdatedAt:      &user.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByAuthProviderID(ctx context.Context, provider, providerID string) (entities.User, error) {
	user, err := r.queries.GetUserByAuthProviderID(ctx, provider, &providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return entities.User{}, domain.ErrNotFound
		}
		return entities.User{}, fmt.Errorf("failed to get user by auth provider ID: %w", err)
	}

	return entities.User{
		ID:             user.ID,
		Email:          user.Email,
		AuthProvider:   user.AuthProvider,
		AuthProviderID: *user.AuthProviderID,
		AccountType:    entities.AccountType(user.AccountType),
		CreatedAt:      *user.CreatedAt,
		UpdatedAt:      *user.UpdatedAt,
	}, nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (r *UserRepository) ListUsers(ctx context.Context, params entities.ListUsersParams) ([]entities.User, error) {
	rows, err := r.queries.ListUsers(ctx, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]entities.User, len(rows))
	for i, row := range rows {
		users[i] = entities.User{
			ID:             row.ID,
			Email:          row.Email,
			AuthProvider:   row.AuthProvider,
			AuthProviderID: *row.AuthProviderID,
			AccountType:    entities.AccountType(row.AccountType),
			CreatedAt:      *row.CreatedAt,
			UpdatedAt:      *row.UpdatedAt,
		}
	}

	return users, nil
}

func (r *UserRepository) CountUsers(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) CountUsersByAccountType(ctx context.Context, accountType entities.AccountType) (int64, error) {
	count, err := r.queries.CountUsersByAccountType(ctx, gen.AccountType(accountType))
	if err != nil {
		return 0, fmt.Errorf("failed to count users by account type: %w", err)
	}
	return count, nil
}

func (r *UserRepository) GetUserStats(ctx context.Context) (entities.UserStats, error) {
	stats, err := r.queries.GetUserStats(ctx)
	if err != nil {
		return entities.UserStats{}, fmt.Errorf("failed to get user stats: %w", err)
	}

	return entities.UserStats{
		TotalUsers:      stats.TotalUsers,
		AdminUsers:      stats.AdminUsers,
		SuperAdminUsers: stats.SuperAdminUsers,
		RegularUsers:    stats.RegularUsers,
		RecentSignups:   stats.RecentSignups,
	}, nil
}
