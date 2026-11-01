package background

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunnerRun(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(t.Context())
	r := New(10*time.Millisecond, nil, func(context.Context) { calls.Add(1) })

	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool { return calls.Load() >= 2 }, time.Second, 10*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runner did not stop")
	}
	require.GreaterOrEqual(t, int(calls.Load()), 2)
}

func TestRunnerRunNilFunction(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	New(10*time.Millisecond, nil, nil).Run(ctx)
}

func TestRunnerAliases(t *testing.T) {
	fn := func(context.Context) {}
	r1 := NewRunner(time.Second, nil, fn)
	r2 := NewTicker(time.Second, nil, fn)
	require.NotNil(t, r1)
	require.NotNil(t, r2)
	var wg sync.WaitGroup
	wg.Add(0)
}
