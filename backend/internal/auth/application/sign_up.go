package application

import (
	"context"
	"log/slog"
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
)

type SignUpRequest struct {
	Username string
	Password string
	Email    string
}

func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (domain.Account, error) {
	acc, err := domain.NewAccount(newID(), req.Username, req.Password, req.Email)
	if err != nil {
		s.m.IncAccountCreated(metrics.StatusError)
		return domain.Account{}, err
	}

	if err := s.repo.Save(ctx, acc); err != nil {
		s.m.IncAccountCreated(metrics.StatusError)
		return domain.Account{}, err
	}

	s.m.IncAccountCreated(metrics.StatusSuccess)

	if s.customers != nil {
		if _, err := s.customers.EnsureCustomer(ctx, acc.ID); err != nil {
			slog.WarnContext(ctx, "failed to provision customer for new account",
				"scope", metricsScope, "account_id", acc.ID, "error", err)
		}
	}

	if err := s.AssignDefaultRole(ctx, acc.ID); err != nil {
		slog.WarnContext(ctx, "failed to assign default role to new account",
			"scope", metricsScope, "account_id", acc.ID, "error", err)
	}

	return acc, nil
}
