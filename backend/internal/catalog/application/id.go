package application

import "crypto/rand"

// newID returns a cryptographically random identifier.
func newID() string {
	return rand.Text()
}
