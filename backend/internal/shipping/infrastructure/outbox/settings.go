package outbox

import "time"

type Settings struct {
	ProcessInterval           time.Duration
	LockCheckerInterval       time.Duration
	CleanupWorkerInterval     time.Duration
	MaxLockTimeDuration       time.Duration
	MessagesRetentionDuration time.Duration
	MaxSendAttempts           int
}
