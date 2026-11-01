package http

import (
	"minimart/internal/notification/domain"
	"time"
)

type preferencesResponse struct {
	EmailEnabled     bool       `json:"email_enabled"`
	SMSEnabled       bool       `json:"sms_enabled"`
	PushEnabled      bool       `json:"push_enabled"`
	MarketingEnabled bool       `json:"marketing_enabled"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
} //	@name	NotificationPreferences

func toPreferencesResponse(p domain.Preferences) preferencesResponse {
	out := preferencesResponse{
		EmailEnabled:     p.EmailEnabled,
		SMSEnabled:       p.SMSEnabled,
		PushEnabled:      p.PushEnabled,
		MarketingEnabled: p.MarketingEnabled,
	}
	if !p.UpdatedAt.IsZero() {
		updatedAt := p.UpdatedAt
		out.UpdatedAt = &updatedAt
	}
	return out
}

type updatePreferencesRequest struct {
	EmailEnabled     *bool `json:"email_enabled"`
	SMSEnabled       *bool `json:"sms_enabled"`
	PushEnabled      *bool `json:"push_enabled"`
	MarketingEnabled *bool `json:"marketing_enabled"`
}

func (r updatePreferencesRequest) complete() bool {
	return r.EmailEnabled != nil && r.SMSEnabled != nil &&
		r.PushEnabled != nil && r.MarketingEnabled != nil
}

type notificationResponse struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	UserID    string     `json:"user_id"`
	OrderID   string     `json:"order_id"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	SentAt    time.Time  `json:"sent_at"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
} //	@name	Notification

func toNotificationResponse(n domain.Notification) notificationResponse {
	return notificationResponse{
		ID:        n.ID,
		Type:      string(n.Type),
		UserID:    n.UserID,
		OrderID:   n.OrderID,
		Subject:   n.Subject,
		Body:      n.Body,
		SentAt:    n.SentAt,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	}
}
