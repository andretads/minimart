package application

import (
	"minimart/internal/auth/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangePassword(t *testing.T) {
	t.Run("Incorrect current password is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "wrong",
			NewPassword:        "newsecret",
			ConfirmNewPassword: "newsecret",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCurrentPassword)
	})

	t.Run("New password and confirmation do not match", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "secret123",
			NewPassword:        "newsecret",
			ConfirmNewPassword: "different",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordMismatch)
	})

	t.Run("New password too short is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "secret123",
			NewPassword:        "abc",
			ConfirmNewPassword: "abc",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})

	t.Run("New password too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		long := strings.Repeat("a", 73)
		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "secret123",
			NewPassword:        long,
			ConfirmNewPassword: long,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooLong)
	})

	t.Run("Password is rehashed and the new password authenticates", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "secret123",
			NewPassword:        "newsecret",
			ConfirmNewPassword: "newsecret",
		})
		require.NoError(t, err)

		got, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "john", Password: "newsecret"})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)

		_, err = svc.Authenticate(t.Context(), AuthenticateRequest{Username: "john", Password: "secret123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Unknown account is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          "unknown",
			CurrentPassword:    "secret123",
			NewPassword:        "newsecret",
			ConfirmNewPassword: "newsecret",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}
