//go:build integration

package test

import (
	"context"
	"fmt"
	"minimart/pkg/tests"
	"os"
	"testing"
)

var testDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	pg, err := tests.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test postgres:", err)
		os.Exit(1)
	}
	testDSN = pg.DSN

	code := m.Run()

	if err := pg.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "terminate test postgres:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
