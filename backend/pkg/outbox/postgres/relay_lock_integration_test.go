//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	outboxpg "minimart/pkg/outbox/postgres"
	"minimart/pkg/tests"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
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

func TestAdvisoryRelayLockSerializesReplicas(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	first := outboxpg.NewAdvisoryRelayLock(pool)
	second := outboxpg.NewAdvisoryRelayLock(pool)

	acquired, err := first.TryAcquire(t.Context())
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = second.TryAcquire(t.Context())
	require.NoError(t, err)
	require.False(t, acquired)

	require.NoError(t, first.Release(t.Context()))

	acquired, err = second.TryAcquire(t.Context())
	require.NoError(t, err)
	require.True(t, acquired)

	require.NoError(t, second.Release(t.Context()))
}
