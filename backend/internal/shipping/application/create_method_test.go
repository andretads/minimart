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

func TestService_CreateMethod(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	method, err := svc.CreateMethod(ctx, CreateMethodRequest{
		Name:             "Standard",
		Description:      "Standard delivery",
		Active:           true,
		EstimatedMinDays: 3,
		EstimatedMaxDays: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, "Standard", method.Name)
	assert.True(t, method.Active)
	assert.Equal(t, 3, method.EstimatedMinDays)
	assert.Equal(t, 5, method.EstimatedMaxDays)
	assert.NotEmpty(t, method.ID)
}

func TestService_CreateMethod_Invalid(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateMethod(ctx, CreateMethodRequest{
		Name:             "",
		Description:      "desc",
		Active:           true,
		EstimatedMinDays: 3,
		EstimatedMaxDays: 5,
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}
