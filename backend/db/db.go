// Package db applies schema migrations using dbmate as a library (not the CLI),
// so the same binary that serves traffic can also run `minimart migrate`.
package db

import (
	"fmt"
	"minimart/conf"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres" // registers the postgres driver
)

type Migrator struct {
	m *dbmate.DB
}

// New builds a Migrator pointed at the primary database. dbmate looks for
// migration files in ./db/migrations by default.
func New() (*Migrator, error) {
	u, err := url.Parse(conf.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	return NewFromURL(u), nil
}

// NewFromURL builds a Migrator for an explicit database URL. Callers that do not
// want conf's environment lookup (integration tests against an ephemeral
// container, for instance) pass the URL directly.
func NewFromURL(u *url.URL) *Migrator {
	m := dbmate.New(u)
	m.AutoDumpSchema = false // don't write a schema.sql dump; migrations are the source of truth
	return &Migrator{m: m}
}

// WithMigrationsDir overrides where dbmate looks for migration files. It exists
// because dbmate's default (./db/migrations) is relative to the process working
// directory, which is not the repo root when tests run from their package dir.
func (mig *Migrator) WithMigrationsDir(dir string) *Migrator {
	mig.m.MigrationsDir = []string{dir}
	return mig
}

// Migrate creates the database if it does not exist, then applies every pending
// migration in timestamp order. It is safe to run on every deploy.
func (mig *Migrator) Migrate() error {
	if err := mig.m.CreateAndMigrate(); err != nil {
		return fmt.Errorf("create and migrate: %w", err)
	}
	return nil
}
