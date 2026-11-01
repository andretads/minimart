package backpressure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestGateStartsRunning(t *testing.T) {
	g := NewGate()
	assert.False(t, g.Paused())
	require.NoError(t, g.Wait(t.Context()))
}

func TestGatePauseBlocksWaitUntilResume(t *testing.T) {
	g := NewGate()
	g.Pause("test")
	assert.True(t, g.Paused())

	done := make(chan error, 1)
	go func() { done <- g.Wait(t.Context()) }()

	select {
	case <-done:
		t.Fatal("Wait returned while paused")
	case <-time.After(50 * time.Millisecond):
	}

	g.Resume()
	assert.False(t, g.Paused())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after Resume")
	}
}

func TestGateWaitReturnsOnContextCancel(t *testing.T) {
	g := NewGate()
	g.Pause("test")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, g.Wait(ctx), context.Canceled)
}

func TestGateIdempotentPauseAndResume(t *testing.T) {
	g := NewGate()
	g.Pause("first")
	g.Pause("second")
	assert.True(t, g.Paused())

	g.Resume()
	g.Resume()
	assert.False(t, g.Paused())
	require.NoError(t, g.Wait(t.Context()))
}
