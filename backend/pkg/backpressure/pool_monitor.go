package backpressure

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolMonitor periodically samples the write/read pool utilization and pauses
// the gate when it exceeds the threshold. It runs until ctx is cancelled.
type PoolMonitor struct {
	gate      *Gate
	pools     map[string]*pgxpool.Pool
	threshold float64
	interval  time.Duration
	pauseFor  time.Duration
	now       func() time.Time

	mu          sync.Mutex
	pausedUntil time.Time
}

// NewPoolMonitor builds a monitor. Zero interval/pauseFor fall back to
// one second / five seconds.
func NewPoolMonitor(gate *Gate, pools map[string]*pgxpool.Pool, threshold float64, interval, pauseFor time.Duration) *PoolMonitor {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.9
	}
	if interval <= 0 {
		interval = time.Second
	}
	if pauseFor <= 0 {
		pauseFor = 5 * time.Second
	}
	return &PoolMonitor{
		gate:      gate,
		pools:     pools,
		threshold: threshold,
		interval:  interval,
		pauseFor:  pauseFor,
		now:       time.Now,
	}
}

// Run blocks until ctx is cancelled, sampling pools on every tick and resuming
// the gate once the pause window has elapsed.
func (m *PoolMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		m.sample()
		m.resumeIfDue()
	}
}

func (m *PoolMonitor) sample() {
	for name, pool := range m.pools {
		if pool == nil {
			continue
		}
		stat := pool.Stat()
		maxConns := stat.MaxConns()
		if maxConns == 0 {
			continue
		}
		util := float64(stat.AcquiredConns()) / float64(maxConns)
		if util >= m.threshold {
			m.pause("pool " + name + " saturated")
		}
	}
}

func (m *PoolMonitor) pause(reason string) {
	until := m.now().Add(m.pauseFor)
	m.mu.Lock()
	if until.After(m.pausedUntil) {
		m.pausedUntil = until
	}
	m.mu.Unlock()
	m.gate.Pause(reason)
}

func (m *PoolMonitor) resumeIfDue() {
	m.mu.Lock()
	until := m.pausedUntil
	m.mu.Unlock()
	if until.IsZero() || m.now().Before(until) {
		return
	}
	m.mu.Lock()
	m.pausedUntil = time.Time{}
	m.mu.Unlock()
	m.gate.Resume()
}
