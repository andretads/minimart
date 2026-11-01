package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPreferences(t *testing.T) {
	p := DefaultPreferences("u-1")

	assert.Equal(t, "u-1", p.UserID)
	assert.True(t, p.EmailEnabled)
	assert.False(t, p.SMSEnabled)
	assert.False(t, p.PushEnabled)
	assert.False(t, p.MarketingEnabled)
	assert.True(t, p.UpdatedAt.IsZero(), "defaults have not been saved yet")
}

func TestNewPreferences(t *testing.T) {
	t.Run("stamps the user and update time", func(t *testing.T) {
		p, err := NewPreferences("u-1", true, true, true, true)
		require.NoError(t, err)

		assert.Equal(t, "u-1", p.UserID)
		assert.True(t, p.EmailEnabled)
		assert.True(t, p.SMSEnabled)
		assert.True(t, p.PushEnabled)
		assert.True(t, p.MarketingEnabled)
		assert.False(t, p.UpdatedAt.IsZero())
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		_, err := NewPreferences("", true, true, true, false)
		assert.ErrorIs(t, err, ErrEmptyUserID)
	})
}

func TestPreferencesAllows(t *testing.T) {
	enabled, err := NewPreferences("u-1", true, true, true, true)
	require.NoError(t, err)
	disabled, err := NewPreferences("u-1", false, false, false, false)
	require.NoError(t, err)

	tests := []struct {
		name     string
		prefs    Preferences
		channel  Channel
		category Category
		want     bool
	}{
		{"email transactional enabled", enabled, ChannelEmail, CategoryTransactional, true},
		{"sms transactional enabled", enabled, ChannelSMS, CategoryTransactional, true},
		{"push transactional enabled", enabled, ChannelPush, CategoryTransactional, true},
		{"marketing requires marketing enabled", disabled, ChannelEmail, CategoryMarketing, false},
		{"marketing allowed when enabled", enabled, ChannelEmail, CategoryMarketing, true},
		{"critical bypasses channel", disabled, ChannelEmail, CategoryCritical, true},
		{"disabled channel blocks transactional", disabled, ChannelEmail, CategoryTransactional, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.prefs.Allows(tt.channel, tt.category))
		})
	}
}
