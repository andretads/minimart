package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelivery(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		notificationID string
		recipientID    string
		channel        Channel
		provider       string
		wantErr        error
	}{
		{"valid email", "d-1", "n-1", "u-1", ChannelEmail, "resend", nil},
		{"valid sms", "d-2", "n-1", "u-1", ChannelSMS, "log", nil},
		{"valid push", "d-3", "n-1", "u-1", ChannelPush, "log", nil},
		{"empty id", "", "n-1", "u-1", ChannelEmail, "resend", ErrEmptyID},
		{"empty notification id", "d-1", "", "u-1", ChannelEmail, "resend", ErrEmptyNotificationID},
		{"empty recipient", "d-1", "n-1", "", ChannelEmail, "resend", ErrEmptyUserID},
		{"invalid channel", "d-1", "n-1", "u-1", Channel("fax"), "resend", ErrInvalidChannel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDelivery(tt.id, tt.notificationID, tt.recipientID, tt.channel, tt.provider)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, d.ID)
			assert.Equal(t, tt.notificationID, d.NotificationID)
			assert.Equal(t, tt.recipientID, d.RecipientID)
			assert.Equal(t, tt.channel, d.Channel)
			assert.Equal(t, tt.provider, d.Provider)
			assert.Equal(t, DeliveryStatusPending, d.Status)
			assert.False(t, d.CreatedAt.IsZero())
			assert.False(t, d.UpdatedAt.IsZero())
		})
	}
}

func TestDeliveryIsTerminal(t *testing.T) {
	tests := []struct {
		status   DeliveryStatus
		terminal bool
	}{
		{DeliveryStatusPending, false},
		{DeliveryStatusProcessing, false},
		{DeliveryStatusSent, true},
		{DeliveryStatusDelivered, true},
		{DeliveryStatusRetryPending, false},
		{DeliveryStatusFailed, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			d := Delivery{Status: tt.status}
			assert.Equal(t, tt.terminal, d.IsTerminal())
		})
	}
}

func TestValidChannel(t *testing.T) {
	assert.True(t, validChannel(ChannelEmail))
	assert.True(t, validChannel(ChannelSMS))
	assert.True(t, validChannel(ChannelPush))
	assert.False(t, validChannel(Channel("fax")))
}

func TestDeliveryTimestamps(t *testing.T) {
	d, err := NewDelivery("d-1", "n-1", "u-1", ChannelEmail, "resend")
	require.NoError(t, err)
	now := time.Now().UTC()
	d.DeliveredAt = &now
	assert.NotNil(t, d.DeliveredAt)
}
