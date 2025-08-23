package pg

import (
	"context"
	"go-template/domain/example"
	"go-template/domain/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX represents a database transaction interface
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// Repository aggregates all repositories and provides transaction support
type Repository struct {
	db          *pgxpool.Pool
	ExampleRepo example.Repository
	UserRepo    user.Repository
}

// NewRepository creates a new Repository instance with all sub-repositories
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db:          db,
		ExampleRepo: NewExampleRepository(db),
		UserRepo:    NewUserRepository(db),
	}
}

// WithTx creates repository instances that use the provided transaction
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		db:          r.db,
		ExampleRepo: NewExampleRepository(tx),
		UserRepo:    NewUserRepository(tx),
	}
}

// BeginTx starts a new transaction
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// DB exposes the underlying connection pool as a DBTX for read-only queries
func (r *Repository) DB() DBTX {
	return r.db
}
