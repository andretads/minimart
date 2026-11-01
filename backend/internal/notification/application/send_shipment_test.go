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

func newShipmentService(m metrics.Metrics, userID string) (*Service, *memory.Repository) {
	repo := memory.New()
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail(userID, "user@example.com")
	return NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{}), repo
}

func TestSendShipmentCreated(t *testing.T) {
	tests := []struct {
		name              string
		orderID           string
		userID            string
		shippingCostCents int64
		wantErr           error
	}{
		{"valid", "o-1", "u-1", 599, nil},
		{"free shipping", "o-1", "u-1", 0, nil},
		{"empty order id", "", "u-1", 599, domain.ErrEmptyOrderID},
		{"empty user id", "o-1", "", 599, domain.ErrEmptyUserID},
		{"negative cost", "o-1", "u-1", -1, domain.ErrNegativeShippingCost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &countingMetrics{}
			svc, repo := newShipmentService(m, "u-1")

			err := svc.SendShipmentCreated(t.Context(), tt.orderID, tt.userID, tt.shippingCostCents)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, []string{metrics.StatusError}, m.sent)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)

			list, err := repo.ListByUserID(t.Context(), tt.userID, 1, 20)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, tt.orderID, list[0].OrderID)
			assert.Equal(t, domain.TypeShipmentCreated, list[0].Type)
		})
	}
}

func TestSendShipmentShipped(t *testing.T) {
	tests := []struct {
		name         string
		orderID      string
		userID       string
		trackingCode string
		carrier      string
		wantErr      error
	}{
		{"valid", "o-1", "u-1", "TRACK123", "Correios", nil},
		{"without tracking details", "o-1", "u-1", "", "", nil},
		{"empty order id", "", "u-1", "TRACK123", "Correios", domain.ErrEmptyOrderID},
		{"empty user id", "o-1", "", "TRACK123", "Correios", domain.ErrEmptyUserID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &countingMetrics{}
			svc, repo := newShipmentService(m, "u-1")

			err := svc.SendShipmentShipped(t.Context(), tt.orderID, tt.userID, tt.trackingCode, tt.carrier)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, []string{metrics.StatusError}, m.sent)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)

			list, err := repo.ListByUserID(t.Context(), tt.userID, 1, 20)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, tt.orderID, list[0].OrderID)
			assert.Equal(t, domain.TypeShipmentShipped, list[0].Type)
			if tt.trackingCode != "" {
				assert.Contains(t, list[0].Body, "TRACK123")
			}
		})
	}
}

func TestSendShipmentDelivered(t *testing.T) {
	tests := []struct {
		name    string
		orderID string
		userID  string
		wantErr error
	}{
		{"valid", "o-1", "u-1", nil},
		{"empty order id", "", "u-1", domain.ErrEmptyOrderID},
		{"empty user id", "o-1", "", domain.ErrEmptyUserID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &countingMetrics{}
			svc, repo := newShipmentService(m, "u-1")

			err := svc.SendShipmentDelivered(t.Context(), tt.orderID, tt.userID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, []string{metrics.StatusError}, m.sent)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)

			list, err := repo.ListByUserID(t.Context(), tt.userID, 1, 20)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, tt.orderID, list[0].OrderID)
			assert.Equal(t, domain.TypeShipmentDelivered, list[0].Type)
		})
	}
}

func TestShipmentNotificationsShareOneOrder(t *testing.T) {
	m := &countingMetrics{}
	svc, repo := newShipmentService(m, "u-1")

	require.NoError(t, svc.SendShipmentCreated(t.Context(), "o-1", "u-1", 599))
	require.NoError(t, svc.SendShipmentShipped(t.Context(), "o-1", "u-1", "TRACK123", "Correios"))
	require.NoError(t, svc.SendShipmentDelivered(t.Context(), "o-1", "u-1"))

	list, err := repo.ListByUserID(t.Context(), "u-1", 1, 20)
	require.NoError(t, err)
	require.Len(t, list, 3)

	types := []domain.Type{list[0].Type, list[1].Type, list[2].Type}
	assert.ElementsMatch(t,
		[]domain.Type{domain.TypeShipmentCreated, domain.TypeShipmentShipped, domain.TypeShipmentDelivered},
		types)
}

func TestShipmentNotificationIsIdempotent(t *testing.T) {
	m := &countingMetrics{}
	svc, repo := newShipmentService(m, "u-1")

	require.NoError(t, svc.SendShipmentCreated(t.Context(), "o-1", "u-1", 599))
	require.NoError(t, svc.SendShipmentCreated(t.Context(), "o-1", "u-1", 599))

	list, err := repo.ListByUserID(t.Context(), "u-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, list, 1, "a redelivered event must not produce a second notification")
}

func TestShipmentNotificationSuppressedByPreference(t *testing.T) {
	m := &countingMetrics{}
	repo := memory.New()
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})

	prefs, err := domain.NewPreferences("u-1", false, true, true, false)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(t.Context(), prefs))

	require.NoError(t, svc.SendShipmentCreated(t.Context(), "o-1", "u-1", 599))

	assert.Equal(t, []string{string(domain.TypeShipmentCreated)}, m.suppressed)
	list, err := repo.ListByUserID(t.Context(), "u-1", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, list, "a suppressed notification must not be recorded")
}

func TestShipmentNotificationStorageFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	m := &countingMetrics{}
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(failingRepo{err: boom}, notificationRepoStub{}, notificationRepoStub{}, sender, nil, nil, resolver, nil, m, RetryConfig{})

	err := svc.SendShipmentCreated(t.Context(), "o-1", "u-1", 599)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.sent)
}
