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
	testS3  *tests.S3
)

// TestMain provisions all required containers for this package.
// Startup order: Postgres → S3 (MinIO)
// Cleanup order: reverse (S3 → Postgres)
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start containers in dependency order
	pg, err := tests.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test postgres:", err)
		os.Exit(1)
	}
	testDSN = pg.DSN

	s3, err := tests.StartS3(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test s3:", err)
		_ = pg.Close(ctx)
		os.Exit(1)
	}
	testS3 = s3

	code := m.Run()

	// Cleanup in reverse order
	if err := testS3.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test s3:", err)
		if code == 0 {
			code = 1
		}
	}
	if err := pg.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test postgres:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
