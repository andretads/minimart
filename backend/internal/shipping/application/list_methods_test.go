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

func TestService_ListMethods(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	err := repo.CreateMethod(ctx, domain.ShippingMethod{ID: "m1", Name: "Standard", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, domain.ShippingMethod{ID: "m2", Name: "Express", Active: true, EstimatedMinDays: 1, EstimatedMaxDays: 2})
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, domain.ShippingMethod{ID: "m3", Name: "Inactive", Active: false, EstimatedMinDays: 5, EstimatedMaxDays: 7})
	require.NoError(t, err)

	methods, err := svc.ListMethods(ctx)
	require.NoError(t, err)
	assert.Len(t, methods, 2)
	assert.Equal(t, "Express", methods[0].Name)
	assert.Equal(t, "Standard", methods[1].Name)
}
