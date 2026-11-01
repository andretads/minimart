package application

import (
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_UpdateMethod(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	err := repo.CreateMethod(ctx, domain.ShippingMethod{ID: "m1", Name: "Standard", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
	require.NoError(t, err)

	updated, err := svc.UpdateMethod(ctx, "m1", UpdateMethodRequest{
		Name:             "Express",
		Description:      "Fast delivery",
		Active:           false,
		EstimatedMinDays: 1,
		EstimatedMaxDays: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, "Express", updated.Name)
	assert.False(t, updated.Active)
	assert.Equal(t, 1, updated.EstimatedMinDays)
}

func TestService_UpdateMethod_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.UpdateMethod(ctx, "nonexistent", UpdateMethodRequest{})
	require.ErrorIs(t, err, domain.ErrMethodNotFound)
}
