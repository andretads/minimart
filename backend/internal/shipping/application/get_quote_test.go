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

func TestService_GetQuote(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()

	// Create method
	method, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, method)
	require.NoError(t, err)

	// Create zone
	zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	err = repo.CreateZone(ctx, zone)
	require.NoError(t, err)

	// Create rate
	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	err = repo.CreateRate(ctx, rate)
	require.NoError(t, err)

	// Setup mock address
	addressID := "addr1"
	CreateTestAddressForMock(fakeModules, "cust1", addressID)

	// Create quote
	quote, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      addressID,
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)

	got, err := svc.GetQuote(ctx, quote.ID)
	require.NoError(t, err)
	assert.Equal(t, quote.ID, got.ID)
	assert.Equal(t, quote.CustomerID, got.CustomerID)
}

func TestService_GetQuote_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.GetQuote(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrQuoteNotFound)
}
