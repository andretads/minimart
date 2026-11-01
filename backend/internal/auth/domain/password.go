package domain

import (
	"golang.org/x/crypto/bcrypt"
)

type PasswordHash struct {
	value []byte
}

func (h *PasswordHash) Matches(secret string) bool {
	if h == nil || len(h.value) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(h.value, []byte(secret)) == nil
}

func newPasswordHash(secret string) (PasswordHash, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return PasswordHash{}, err
	}
	return PasswordHash{value: hashed}, nil
}

func loadPasswordHash(hash []byte) PasswordHash {
	return PasswordHash{value: hash}
}

func (h *PasswordHash) Bytes() []byte {
	if h == nil {
		return nil
	}
	return h.value
}
