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
	testMongoURI string
)

// TestMain provisions all required containers for this package.
// Startup order: MongoDB
// Cleanup order: reverse (MongoDB)
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start containers in dependency order
	mongoDB, err := tests.StartMongo(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test mongo:", err)
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
	os.Exit(code)
}
