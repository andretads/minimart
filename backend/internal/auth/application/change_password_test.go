package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangePassword(t *testing.T) {
	t.Run("changes the password and authenticates with the new one", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})
		require.NoError(t, err)

		err = svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "new-password",
			ConfirmNewPassword: "new-password",
		})
		require.NoError(t, err)

		got, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "testuser", Password: "new-password"})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
	})

	t.Run("rejects mismatched confirmation", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "new-password",
			ConfirmNewPassword: "different",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordMismatch)
	})

	t.Run("rejects an incorrect current password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "wrong",
			NewPassword:        "new-password",
			ConfirmNewPassword: "new-password",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCurrentPassword)
	})

	t.Run("rejects an empty new password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "",
			ConfirmNewPassword: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyPassword)
	})

	t.Run("rejects a short new password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "12345",
			ConfirmNewPassword: "12345",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})

	t.Run("rejects a long new password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "a" + string(make([]byte, 72)),
			ConfirmNewPassword: "a" + string(make([]byte, 72)),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooLong)
	})

	t.Run("rejects an unknown account", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		err := svc.ChangePassword(t.Context(), ChangePasswordRequest{
			AccountID:          "unknown",
			CurrentPassword:    "password123",
			NewPassword:        "new-password",
			ConfirmNewPassword: "new-password",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}
