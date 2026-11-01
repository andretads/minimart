package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrderConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		orderID    string
		userID     string
		totalCents int64
		wantErr    error
	}{
		{"valid", "n-1", "o-1", "u-1", 1250, nil},
		{"empty id", "", "o-1", "u-1", 1250, ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", 1250, ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", 1250, ErrEmptyUserID},
		{"zero total", "n-1", "o-1", "u-1", 0, ErrNonPositiveTotal},
		{"negative total", "n-1", "o-1", "u-1", -100, ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewOrderConfirmation(tt.id, tt.orderID, tt.userID, tt.totalCents)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, n.ID)
			assert.Equal(t, TypeOrderConfirmation, n.Type)
			assert.Equal(t, tt.userID, n.UserID)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Subject, tt.orderID)
			assert.Contains(t, n.Body, tt.orderID)
			assert.False(t, n.SentAt.IsZero(), "SentAt is stamped on creation")
			assert.False(t, n.CreatedAt.IsZero(), "CreatedAt is stamped on creation")
			assert.True(t, n.SentAt.Equal(n.CreatedAt), "SentAt and CreatedAt start equal")
		})
	}
}

// TestNewOrderConfirmation_Body renders money as exact BRL amounts - the
// derived body is the visible contract of the integer-cents rule.
func TestNewOrderConfirmation_Body(t *testing.T) {
	tests := []struct {
		name       string
		totalCents int64
		want       string
	}{
		{"dollars and cents", 1250, "R$ 12,50"},
		{"cents only", 5, "R$ 0,05"},
		{"whole dollars", 100, "R$ 1,00"},
		{"thousands grouping", 1200000, "R$ 12.000,00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewOrderConfirmation("n-1", "o-1", "u-1", tt.totalCents)
			require.NoError(t, err)
			assert.Contains(t, n.Body, tt.want)
		})
	}
}

// TestNewOrderCancellation covers the cancelled-order notification, sharing the
// validation and timestamps of every constructor.
func TestNewOrderCancellation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		orderID    string
		userID     string
		totalCents int64
		wantErr    error
	}{
		{"valid", "n-1", "o-1", "u-1", 1250, nil},
		{"empty id", "", "o-1", "u-1", 1250, ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", 1250, ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", 1250, ErrEmptyUserID},
		{"zero total", "n-1", "o-1", "u-1", 0, ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewOrderCancellation(tt.id, tt.orderID, tt.userID, tt.totalCents)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypeOrderCancellation, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Body, tt.orderID)
			assert.False(t, n.SentAt.IsZero())
		})
	}
}

// TestNewPaymentSucceeded covers the captured-payment notification.
func TestNewPaymentSucceeded(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		orderID     string
		userID      string
		amountCents int64
		wantErr     error
	}{
		{"valid", "n-1", "o-1", "u-1", 1250, nil},
		{"empty id", "", "o-1", "u-1", 1250, ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", 1250, ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", 1250, ErrEmptyUserID},
		{"zero amount", "n-1", "o-1", "u-1", 0, ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewPaymentSucceeded(tt.id, tt.orderID, tt.userID, tt.amountCents)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypePaymentSucceeded, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Body, "R$ 12,50")
			assert.False(t, n.SentAt.IsZero())
		})
	}
}

// TestNewPaymentFailed covers the declined-payment notification and that the
// provider's reason is carried into the body when present.
func TestNewPaymentFailed(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		orderID     string
		userID      string
		amountCents int64
		reason      string
		wantErr     error
	}{
		{"valid", "n-1", "o-1", "u-1", 1250, "insufficient funds", nil},
		{"valid without reason", "n-1", "o-1", "u-1", 1250, "", nil},
		{"empty id", "", "o-1", "u-1", 1250, "", ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", 1250, "", ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", 1250, "", ErrEmptyUserID},
		{"zero amount", "n-1", "o-1", "u-1", 0, "", ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewPaymentFailed(tt.id, tt.orderID, tt.userID, tt.amountCents, tt.reason)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypePaymentFailed, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Body, "R$ 12,50")
			if tt.reason != "" {
				assert.Contains(t, n.Body, tt.reason)
			}
			assert.False(t, n.SentAt.IsZero())
		})
	}
}
