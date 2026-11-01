package application

import (
	"context"
	"errors"
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
)

func (s *Service) AssignRole(ctx context.Context, accountID, roleID string) error {
	if _, err := s.repo.GetRole(ctx, roleID); err != nil {
		s.m.IncRoleAssigned(metrics.StatusError)
		return err
	}

	if err := s.repo.AssignRole(ctx, accountID, roleID); err != nil {
		if errors.Is(err, domain.ErrRoleAlreadyAssigned) {
			return nil
		}
		s.m.IncRoleAssigned(metrics.StatusError)
		return err
	}
	s.m.IncRoleAssigned(metrics.StatusSuccess)
	return nil
}

func (s *Service) AssignDefaultRole(ctx context.Context, accountID string) error {
	return s.AssignRole(ctx, accountID, "customer")
}
