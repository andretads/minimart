package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
)

func (s *Service) GetPreferences(ctx context.Context, userID string) (domain.Preferences, error) {
	if userID == "" {
		return domain.Preferences{}, domain.ErrEmptyUserID
	}
	p, err := s.prefs.GetByUserID(ctx, userID)
	switch {
	case errors.Is(err, domain.ErrPreferencesNotFound):
		return domain.DefaultPreferences(userID), nil
	case err != nil:
		return domain.Preferences{}, err
	}
	return p, nil
}
