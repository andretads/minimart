package application

import (
	"context"
	"minimart/pkg/metrics"
)

func (s *Service) PermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	perms, err := s.repo.GetPermissionsForAccount(ctx, accountID)
	if err != nil {
		s.m.IncPermissionsLoaded(metrics.StatusError)
		return nil, err
	}
	s.m.IncPermissionsLoaded(metrics.StatusSuccess)
	return perms, nil
}
