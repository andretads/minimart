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

func TestService_ExpireQuotes(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()

	method, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, method)
	require.NoError(t, err)

	zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	err = repo.CreateZone(ctx, zone)
	require.NoError(t, err)

	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	err = repo.CreateRate(ctx, rate)
	require.NoError(t, err)

	// Setup mock address
	addressID := "addr1"
	CreateTestAddressForMock(fakeModules, "cust1", addressID)

	// Create active quote
	quote, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      addressID,
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)

	// Expire it
	quote.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	err = repo.UpdateQuote(ctx, quote)
	require.NoError(t, err)

	// Create another active quote
	CreateTestAddressForMock(fakeModules, "cust2", "addr2")
	quote2, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust2",
		MethodID:       method.ID,
		AddressID:      "addr2",
		IdempotencyKey: "key-2",
	})
	require.NoError(t, err)

	// Run expire
	err = svc.ExpireQuotes(ctx)
	require.NoError(t, err)

	// First quote should be expired
	expired, err := repo.GetQuote(ctx, quote.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusExpired, expired.Status)

	// Second quote should still be active
	active, err := repo.GetQuote(ctx, quote2.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusActive, active.Status)
}

func TestService_ExpireQuotes_NoQuotes(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	err := svc.ExpireQuotes(ctx)
	require.NoError(t, err)
}
