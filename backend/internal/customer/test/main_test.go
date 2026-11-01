//go:build integration

package test

import (
	"context"
	"fmt"
	"minimart/pkg/tests"
	"os"
	"testing"
)

// Package-level variables for container handles. Use var block for consistency.
var (
	testDSN string
)

// TestMain provisions all required containers for this package.
// Startup order: Postgres
// Cleanup order: reverse (Postgres)
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start containers in dependency order
	pg, err := tests.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test postgres:", err)
		os.Exit(1)
	}
	testDSN = pg.DSN

	code := m.Run()

	// Cleanup in reverse order
	if err := pg.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test postgres:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
