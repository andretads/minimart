package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingPrefsRepo struct{ err error }

func (f failingPrefsRepo) GetByUserID(context.Context, string) (domain.Preferences, error) {
	return domain.Preferences{}, f.err
}

func (f failingPrefsRepo) Upsert(context.Context, domain.Preferences) error { return f.err }

func TestGetPreferences(t *testing.T) {
	t.Run("missing record returns the defaults", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, err := svc.GetPreferences(t.Context(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, domain.DefaultPreferences("u-1"), got)
	})

	t.Run("stored record is returned", func(t *testing.T) {
		repo := memory.New()
		stored, err := domain.NewPreferences("u-1", false, true, false, true)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(t.Context(), stored))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, err := svc.GetPreferences(t.Context(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, stored, got)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.GetPreferences(t.Context(), "")
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("repository error is propagated", func(t *testing.T) {
		boom := errors.New("mongo down")
		svc := NewService(memory.New(), memory.New(), failingPrefsRepo{err: boom}, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.GetPreferences(t.Context(), "u-1")
		assert.ErrorIs(t, err, boom)
	})
}
