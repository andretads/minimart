package client

import (
	"context"
	"errors"
	"minimart/internal/auth/presentation/module/client"
	"minimart/internal/notification/domain"
	"minimart/pkg/module/contracts"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAccountReader struct {
	email string
	err   error
}

func (f fakeAccountReader) GetAccountEmail(ctx context.Context, userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.email, nil
}

func TestAccountReaderEmailForUser(t *testing.T) {
	t.Run("returns email from auth contract", func(t *testing.T) {
		fakeReader := fakeAccountReader{email: "user@example.com"}
		ctr := &contracts.Contracts{AuthClient: fakeReader}
		adapter := NewAccountReader(ctr)

		got, err := adapter.EmailForUser(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", got)
	})

	t.Run("translates account not found into domain sentinel", func(t *testing.T) {
		fakeReader := fakeAccountReader{err: client.ErrAccountNotFound}
		ctr := &contracts.Contracts{AuthClient: fakeReader}
		adapter := NewAccountReader(ctr)

		_, err := adapter.EmailForUser(t.Context(), "user-1")
		assert.ErrorIs(t, err, domain.ErrRecipientNotFound)
	})

	t.Run("translates email not found into domain sentinel", func(t *testing.T) {
		fakeReader := fakeAccountReader{err: client.ErrEmailNotFound}
		ctr := &contracts.Contracts{AuthClient: fakeReader}
		adapter := NewAccountReader(ctr)

		_, err := adapter.EmailForUser(t.Context(), "user-1")
		assert.ErrorIs(t, err, domain.ErrRecipientNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		fakeReader := fakeAccountReader{err: boom}
		ctr := &contracts.Contracts{AuthClient: fakeReader}
		adapter := NewAccountReader(ctr)

		_, err := adapter.EmailForUser(t.Context(), "user-1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrRecipientNotFound)
	})
}
