package domain

import (
	"strings"
	"time"
)

const (
	maxUsernameLength = 200
	maxPasswordLength = 72
	minPasswordLength = 6
	maxEmailLength    = 254
)

type Account struct {
	ID           string
	Username     string
	Email        string
	PasswordHash PasswordHash
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewAccount(id, username, password, email string) (Account, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return Account{}, ErrEmptyUsername
	}
	if len(username) > maxUsernameLength {
		return Account{}, ErrUsernameTooLong
	}

	if err := validatePassword(password); err != nil {
		return Account{}, err
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return Account{}, ErrEmptyEmail
	}
	if len(email) > maxEmailLength {
		return Account{}, ErrInvalidEmail
	}
	if !isValidEmail(email) {
		return Account{}, ErrInvalidEmail
	}

	hash, err := newPasswordHash(password)
	if err != nil {
		return Account{}, err
	}

	now := time.Now().UTC()
	return Account{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func LoadAccount(id, username, email string, passwordHash []byte, createdAt, updatedAt time.Time) Account {
	return Account{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: loadPasswordHash(passwordHash),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func (a *Account) PasswordMatches(password string) bool {
	return a.PasswordHash.Matches(password)
}

func (a Account) ChangePassword(currentPassword, newPassword string) (Account, error) {
	if !a.PasswordMatches(currentPassword) {
		return Account{}, ErrInvalidCurrentPassword
	}
	if err := validatePassword(newPassword); err != nil {
		return Account{}, err
	}

	hash, err := newPasswordHash(newPassword)
	if err != nil {
		return Account{}, err
	}

	a.PasswordHash = hash
	a.UpdatedAt = time.Now().UTC()
	return a, nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrEmptyPassword
	}
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	if len(email) > maxEmailLength {
		return false
	}
	local, domainPart, found := strings.Cut(email, "@")
	if !found || local == "" || domainPart == "" {
		return false
	}
	if strings.Contains(local, " ") || strings.Contains(domainPart, " ") {
		return false
	}
	if !strings.Contains(domainPart, ".") {
		return false
	}
	return true
}
