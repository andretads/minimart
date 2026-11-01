package domain

import "crypto/rand"

func newID() string {
	return rand.Text()
}
