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
	testDSN          string
	testKafkaBrokers []string
	testRedisURL     string
)

// TestMain provisions all required containers for this package.
// Startup order: Postgres → Kafka → Redis
// Cleanup order: reverse (Redis → Kafka → Postgres)
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start containers in dependency order
	pg, err := tests.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test postgres:", err)
		os.Exit(1)
	}
	testDSN = pg.DSN

	kafka, err := tests.StartKafka(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test kafka:", err)
		_ = pg.Close(ctx)
		os.Exit(1)
	}
	testKafkaBrokers = kafka.Brokers

	rdb, err := tests.StartRedis(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test redis:", err)
		_ = kafka.Close(ctx)
		_ = pg.Close(ctx)
		os.Exit(1)
	}
	testRedisURL = rdb.URL

	code := m.Run()

	// Cleanup in reverse order
	if err := rdb.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test redis:", err)
		if code == 0 {
			code = 1
		}
	}
	if err := kafka.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test kafka:", err)
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
