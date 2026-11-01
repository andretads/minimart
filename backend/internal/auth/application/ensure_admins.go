package application

import (
	"context"
	"errors"
	"minimart/internal/auth/domain"
)

func (s *Service) EnsureAdmins(ctx context.Context, usernames []string) error {
	if len(usernames) == 0 {
		return nil
	}
	accountIDs, err := s.repo.GetAccountsByUsername(ctx, usernames)
	if err != nil {
		return err
	}
	for _, id := range accountIDs {
		if err := s.AssignRole(ctx, id, "admin"); err != nil && !errors.Is(err, domain.ErrRoleAlreadyAssigned) {
			return err
		}
	}
	return nil
}
