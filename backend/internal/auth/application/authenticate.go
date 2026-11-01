package application

import (
	"context"
	"log/slog"
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
)

type AuthenticateRequest struct {
	Username string
	Password string
}

func (s *Service) Authenticate(ctx context.Context, req AuthenticateRequest) (domain.Account, error) {
	acc, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if domain.IsNotFound(err) {
			s.m.IncAuthenticated(metrics.StatusError)
			return domain.Account{}, domain.ErrInvalidCredentials
		}
		s.m.IncAuthenticated(metrics.StatusError)
		return domain.Account{}, err
	}

	if !acc.PasswordMatches(req.Password) {
		s.m.IncAuthenticated(metrics.StatusError)
		return domain.Account{}, domain.ErrInvalidCredentials
	}

	s.m.IncAuthenticated(metrics.StatusSuccess)

	if s.customers != nil {
		if _, err := s.customers.EnsureCustomer(ctx, acc.ID); err != nil {
			slog.WarnContext(ctx, "failed to ensure customer on login",
				"scope", metricsScope, "account_id", acc.ID, "error", err)
		}
	}

	return acc, nil
}
