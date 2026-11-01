package application

import (
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendPaymentSucceeded(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		userID      string
		amountCents int64
		wantErr     error
	}{
		{"valid", "o-1", "u-1", 1250, nil},
		{"empty order id", "", "u-1", 1250, domain.ErrEmptyOrderID},
		{"empty user id", "o-1", "", 1250, domain.ErrEmptyUserID},
		{"zero amount", "o-1", "u-1", 0, domain.ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &countingMetrics{}
			repo := memory.New()
			sender := newFakeEmailSender()
			resolver := newFakeRecipientResolver()
			resolver.SetEmail("u-1", "user@example.com")
			svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})
			err := svc.SendPaymentSucceeded(t.Context(), tt.orderID, tt.userID, tt.amountCents)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, []string{metrics.StatusError}, m.sent)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)

			list, err := svc.repo.ListByUserID(t.Context(), tt.userID, 1, 20)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, tt.orderID, list[0].OrderID)
			assert.Equal(t, domain.TypePaymentSucceeded, list[0].Type)
		})
	}
}

func TestSendPaymentSucceededPersistenceFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	m := &countingMetrics{}
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(failingRepo{err: boom}, notificationRepoStub{}, notificationRepoStub{}, sender, nil, nil, resolver, nil, m, RetryConfig{})

	err := svc.SendPaymentSucceeded(t.Context(), "o-1", "u-1", 1250)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.sent)
}
