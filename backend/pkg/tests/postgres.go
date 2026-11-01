// Package tests spins up the real dependencies integration tests need.
//
// It exists so integration tests are self-contained: they start their own
// Postgres with testcontainers-go and apply the same db/migrations the
// production binary runs, instead of depending on a pre-provisioned
// docker-compose database. Nothing here imports "testing"; TestMain drives it,
// which keeps the testing flag registration out of a non-test package.
package tests

import (
	"context"
	"fmt"
	"minimart/db"
	"net/url"
	"path/filepath"
	"runtime"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// postgresImage matches the version docker-compose serves locally, so tests and
// local development exercise the same Postgres.
const postgresImage = "postgres:16-alpine"

// Postgres is a running Postgres container with the schema already migrated.
type Postgres struct {
	Container *postgres.PostgresContainer
	DSN       string
}

// StartPostgres starts an ephemeral Postgres container, waits until it accepts
// connections, applies every migration under db/migrations, and returns the
// container plus a DSN for it. Call Close to stop it.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	container, err := postgres.Run(ctx, postgresImage,
		postgres.WithDatabase("minimart"),
		postgres.WithUsername("minimart"),
		postgres.WithPassword("minimart"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("postgres connection string: %w", err)
	}

	if err := Migrate(dsn); err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	return &Postgres{Container: container, DSN: dsn}, nil
}

// Close stops the container. The testcontainers Ryuk reaper also removes
// containers left behind by a crashed test run.
func (p *Postgres) Close(ctx context.Context) error {
	return p.Container.Terminate(ctx)
}

// Migrate applies the repo's db/migrations to the database at dsn.
func Migrate(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("parse test database URL: %w", err)
	}
	if err := db.NewFromURL(u).WithMigrationsDir(migrationsDir()).Migrate(); err != nil {
		return fmt.Errorf("migrate test database: %w", err)
	}
	return nil
}

// migrationsDir resolves <repoRoot>/db/migrations from this file's location, so
// it is correct no matter which package's working directory the test runs in.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}
