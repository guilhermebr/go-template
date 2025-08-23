package pg

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

var (
	pool     *dockertest.Pool
	resource *dockertest.Resource
)

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		panic(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=go_app_template_test",
		},
		ExposedPorts: []string{"5432/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {{HostIP: "localhost", HostPort: ""}},
		},
	}

	resource, err = pool.RunWithOptions(&opts)
	if err != nil {
		panic(fmt.Sprintf("Could not start resource: %s", err))
	}

	// Wait for the container to be ready
	if err := pool.Retry(func() error {
		conn, err := pgxpool.New(context.Background(), getTestDSN())
		if err != nil {
			return err
		}
		defer conn.Close()
		return conn.Ping(context.Background())
	}); err != nil {
		panic(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	// Run migrations
	mig, err := migrate.New(
		"file://migrations",
		getTestDSN(),
	)
	if err != nil {
		panic(fmt.Sprintf("Could not create migration: %s", err))
	}

	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		panic(fmt.Sprintf("Could not run migrations: %s", err))
	}

	code := m.Run()

	// Cleanup
	if err := pool.Purge(resource); err != nil {
		panic(fmt.Sprintf("Could not purge resource: %s", err))
	}

	os.Exit(code)
}

func getTestDSN() string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:%s/go_app_template_test?sslmode=disable", resource.GetPort("5432/tcp"))
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), getTestDSN())
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	return pool
}
