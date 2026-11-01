package domain

import "time"

type Preferences struct {
	UserID           string
	EmailEnabled     bool
	SMSEnabled       bool
	PushEnabled      bool
	MarketingEnabled bool
	UpdatedAt        time.Time
}

func DefaultPreferences(userID string) Preferences {
	return Preferences{
		UserID:           userID,
		EmailEnabled:     true,
		SMSEnabled:       false,
		PushEnabled:      false,
		MarketingEnabled: false,
		UpdatedAt:        time.Time{},
	}
}

func NewPreferences(userID string, emailEnabled, smsEnabled, pushEnabled, marketingEnabled bool) (Preferences, error) {
	if userID == "" {
		return Preferences{}, ErrEmptyUserID
	}
	return Preferences{
		UserID:           userID,
		EmailEnabled:     emailEnabled,
		SMSEnabled:       smsEnabled,
		PushEnabled:      pushEnabled,
		MarketingEnabled: marketingEnabled,
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

func (p Preferences) Allows(channel Channel, category Category) bool {
	if category == CategoryCritical {
		return true
	}
	switch channel {
	case ChannelEmail:
		if !p.EmailEnabled {
			return false
		}
	case ChannelSMS:
		if !p.SMSEnabled {
			return false
		}
	case ChannelPush:
		if !p.PushEnabled {
			return false
		}
	default:
		return false
	}
	if category == CategoryMarketing {
		return p.MarketingEnabled
	}
	return true
}
