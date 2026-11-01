package application

import (
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPricableQuote(t *testing.T, svc *Service, repo *memory.Repository, fakeModules *FakeModules) (domain.ShippingQuote, domain.ShippingRate) {
	t.Helper()
	ctx := t.Context()

	method, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateMethod(ctx, method))

	zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateZone(ctx, zone))

	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRate(ctx, rate))

	CreateTestAddressForMock(fakeModules, "cust1", "addr1")

	quote, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)
	return quote, rate
}

func TestService_PriceQuote(t *testing.T) {
	tests := []struct {
		name          string
		subtotalCents int64
		wantCostCents int64
	}{
		{name: "below the threshold pays the base cost", subtotalCents: 1200, wantCostCents: 599},
		{name: "at the threshold ships free", subtotalCents: 5000, wantCostCents: 0},
		{name: "above the threshold ships free", subtotalCents: 9000, wantCostCents: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			svc, repo, fakeModules := NewTestServiceWithModules()
			quote, rate := seedPricableQuote(t, svc, repo, fakeModules)

			priced, err := svc.PriceQuote(ctx, quote.ID, "cust1", tt.subtotalCents)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCostCents, priced.ShippingCostCents)
			assert.Equal(t, tt.subtotalCents, priced.SubtotalCents)
			assert.Equal(t, rate.FreeThresholdCents, priced.FreeThresholdCents)

			stored, err := repo.GetQuote(ctx, quote.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCostCents, stored.ShippingCostCents)
			assert.Equal(t, tt.subtotalCents, stored.SubtotalCents)
			assert.Equal(t, rate.FreeThresholdCents, stored.FreeThresholdCents)
		})
	}
}

func TestService_PriceQuote_NotFound(t *testing.T) {
	ctx := t.Context()
	svc, _, _ := NewTestServiceWithModules()

	_, err := svc.PriceQuote(ctx, "missing", "cust1", 1000)
	require.ErrorIs(t, err, domain.ErrQuoteNotFound)
}

func TestService_PriceQuote_OwnershipMismatch(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()
	quote, _ := seedPricableQuote(t, svc, repo, fakeModules)

	_, err := svc.PriceQuote(ctx, quote.ID, "cust2", 1000)
	require.ErrorIs(t, err, domain.ErrOwnershipMismatch)
}

func TestService_PriceQuote_Expired(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()
	quote, _ := seedPricableQuote(t, svc, repo, fakeModules)

	quote.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	require.NoError(t, repo.UpdateQuote(ctx, quote))

	_, err := svc.PriceQuote(ctx, quote.ID, "cust1", 1000)
	require.ErrorIs(t, err, domain.ErrQuoteExpired)
}

func TestService_PriceQuote_NegativeSubtotal(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()
	quote, _ := seedPricableQuote(t, svc, repo, fakeModules)

	_, err := svc.PriceQuote(ctx, quote.ID, "cust1", -1)
	require.ErrorIs(t, err, domain.ErrInvalidQuote)
}

func TestService_PriceQuote_InactiveRate(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()
	quote, rate := seedPricableQuote(t, svc, repo, fakeModules)

	rate.Active = false
	require.NoError(t, repo.UpdateRate(ctx, rate))

	_, err := svc.PriceQuote(ctx, quote.ID, "cust1", 1000)
	require.ErrorIs(t, err, domain.ErrRateInactive)
}
