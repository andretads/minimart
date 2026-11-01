package application

import (
	"context"
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
)

// ChangePasswordRequest carries the credentials needed to rotate an account's
// password. AccountID comes from the authenticated access token, never from the
// request body.
type ChangePasswordRequest struct {
	AccountID          string
	CurrentPassword    string
	NewPassword        string
	ConfirmNewPassword string
}

// ChangePassword verifies the current password and persists a hash of the new
// one. The confirmation check runs first so a mismatched request does not touch
// storage.
func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	if req.NewPassword != req.ConfirmNewPassword {
		s.m.IncPasswordChanged(metrics.StatusError)
		return domain.ErrPasswordMismatch
	}

	acc, err := s.repo.GetByID(ctx, req.AccountID)
	if err != nil {
		s.m.IncPasswordChanged(metrics.StatusError)
		return err
	}

	updated, err := acc.ChangePassword(req.CurrentPassword, req.NewPassword)
	if err != nil {
		s.m.IncPasswordChanged(metrics.StatusError)
		return err
	}

	if err := s.repo.UpdatePassword(ctx, updated); err != nil {
		s.m.IncPasswordChanged(metrics.StatusError)
		return err
	}

	s.m.IncPasswordChanged(metrics.StatusSuccess)
	return nil
}
