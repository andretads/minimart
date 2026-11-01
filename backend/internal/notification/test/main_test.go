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
	testKafkaBrokers []string
	testMongoURI     string
)

// TestMain provisions all required containers for this package.
// Startup order: Kafka → MongoDB
// Cleanup order: reverse (MongoDB → Kafka)
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start containers in dependency order
	kafka, err := tests.StartKafka(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test kafka:", err)
		os.Exit(1)
	}
	testKafkaBrokers = kafka.Brokers

	mongoDB, err := tests.StartMongo(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test mongo:", err)
		_ = kafka.Close(ctx)
		os.Exit(1)
	}
	testMongoURI = mongoDB.URI

	code := m.Run()

	// Cleanup in reverse order
	if err := mongoDB.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test mongo:", err)
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
	os.Exit(code)
}
