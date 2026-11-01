//go:build integration

package test

import (
	"context"
	"fmt"
	"minimart/pkg/tests"
	"os"
	"testing"
)

var (
	testDSN string
	testS3  *tests.S3
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pg, err := tests.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test postgres:", err)
		os.Exit(1)
	}
	testDSN = pg.DSN

	s3c, err := tests.StartS3(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start test s3:", err)
		_ = pg.Close(ctx)
		os.Exit(1)
	}
	testS3 = s3c

	code := m.Run()

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
