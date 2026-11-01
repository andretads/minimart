package outbox

import (
	"crypto/rand"
	"os"
)

// MachineID returns a stable identifier for this process's outbox record locks.
func MachineID() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return rand.Text()
}
