package application

import (
	"context"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
)

type UpdatePreferencesRequest struct {
	UserID           string
	EmailEnabled     bool
	SMSEnabled       bool
	PushEnabled      bool
	MarketingEnabled bool
}

func (s *Service) UpdatePreferences(ctx context.Context, req UpdatePreferencesRequest) (domain.Preferences, error) {
	p, err := domain.NewPreferences(req.UserID, req.EmailEnabled, req.SMSEnabled, req.PushEnabled, req.MarketingEnabled)
	if err != nil {
		s.m.IncNotificationPreferencesUpdated(metrics.StatusError)
		return domain.Preferences{}, err
	}
	if err := s.prefs.Upsert(ctx, p); err != nil {
		s.m.IncNotificationPreferencesUpdated(metrics.StatusError)
		return domain.Preferences{}, err
	}
	s.m.IncNotificationPreferencesUpdated(metrics.StatusSuccess)
	return p, nil
}
