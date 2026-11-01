package application

import (
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateQuote(t *testing.T) {
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
	assert.Equal(t, method.ID, quote.MethodID)
	assert.Equal(t, "cust1", quote.CustomerID)
	assert.Equal(t, domain.QuoteStatusActive, quote.Status)
	assert.Equal(t, int64(599), quote.ShippingCostCents)
}

func TestService_CreateQuote_InvalidMethod(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       "nonexistent",
		AddressID:      "addr1",
		IdempotencyKey: "key-1",
	})
	require.ErrorIs(t, err, domain.ErrMethodNotFound)
}

func TestService_CreateQuote_InactiveMethod(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	method, err := domain.NewShippingMethod("Inactive", "Inactive delivery", false, 3, 5)
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, method)
	require.NoError(t, err)

	_, err = svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		IdempotencyKey: "key-1",
	})
	require.ErrorIs(t, err, domain.ErrMethodInactive)
}

func TestService_CreateQuote_FreeThreshold(t *testing.T) {
	tests := []struct {
		name           string
		subtotalCents  int64
		baseCostCents  int64
		thresholdCents int64
		wantCostCents  int64
	}{
		{
			name:           "below the threshold pays the base cost",
			subtotalCents:  4999,
			baseCostCents:  599,
			thresholdCents: 5000,
			wantCostCents:  599,
		},
		{
			name:           "at the threshold ships free",
			subtotalCents:  5000,
			baseCostCents:  599,
			thresholdCents: 5000,
			wantCostCents:  0,
		},
		{
			name:           "above the threshold ships free",
			subtotalCents:  12000,
			baseCostCents:  599,
			thresholdCents: 5000,
			wantCostCents:  0,
		},
		{
			name:           "a zero threshold never ships free",
			subtotalCents:  12000,
			baseCostCents:  599,
			thresholdCents: 0,
			wantCostCents:  599,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			svc, repo, fakeModules := NewTestServiceWithModules()

			method, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
			require.NoError(t, err)
			require.NoError(t, repo.CreateMethod(ctx, method))

			zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
			require.NoError(t, err)
			require.NoError(t, repo.CreateZone(ctx, zone))

			rate, err := domain.NewShippingRate(zone.ID, method.ID, tt.baseCostCents, tt.thresholdCents, true)
			require.NoError(t, err)
			require.NoError(t, repo.CreateRate(ctx, rate))

			CreateTestAddressForMock(fakeModules, "cust1", "addr1")

			quote, err := svc.CreateQuote(ctx, CreateQuoteRequest{
				CustomerID:     "cust1",
				MethodID:       method.ID,
				AddressID:      "addr1",
				SubtotalCents:  tt.subtotalCents,
				IdempotencyKey: "key-1",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantCostCents, quote.ShippingCostCents)
			assert.Equal(t, tt.subtotalCents, quote.SubtotalCents)
			assert.Equal(t, tt.thresholdCents, quote.FreeThresholdCents)
		})
	}
}

func TestService_CreateQuote_NegativeSubtotal(t *testing.T) {
	ctx := t.Context()
	svc, repo, fakeModules := NewTestServiceWithModules()

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

	_, err = svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  -1,
		IdempotencyKey: "key-1",
	})
	require.ErrorIs(t, err, domain.ErrInvalidQuote)
}

func TestService_CreateQuote_RequiresIdempotencyKey(t *testing.T) {
	ctx := t.Context()
	svc, _, _ := NewTestServiceWithModules()

	_, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID: "cust1",
		MethodID:   "m1",
		AddressID:  "addr1",
	})
	require.ErrorIs(t, err, domain.ErrIdempotencyKeyRequired)
}

func TestService_CreateQuote_IsIdempotent(t *testing.T) {
	ctx := t.Context()
	store := newMemIdempotency()
	svc, repo, fakeModules := NewTestServiceWithStore(store)
	method := seedQuotePricing(t, repo, fakeModules)

	req := CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-1",
	}

	first, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)

	second, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	stored, err := repo.GetQuoteByIdempotencyKey(ctx, quoteKey("key-1"))
	require.NoError(t, err)
	assert.Equal(t, first.ID, stored.ID)
}

func TestService_CreateQuote_ConflictsOnDifferentParameters(t *testing.T) {
	ctx := t.Context()
	store := newMemIdempotency()
	svc, repo, fakeModules := NewTestServiceWithStore(store)
	method := seedQuotePricing(t, repo, fakeModules)

	_, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)

	_, err = svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  2400,
		IdempotencyKey: "key-1",
	})
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

func TestService_CreateQuote_ReplaysFromTheDatabaseWhenTheCacheIsEmpty(t *testing.T) {
	ctx := t.Context()
	store := newMemIdempotency()
	svc, repo, fakeModules := NewTestServiceWithStore(store)
	method := seedQuotePricing(t, repo, fakeModules)

	req := CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-1",
	}

	first, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)

	delete(store.records, quoteKey("key-1"))

	second, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
}

func TestService_CreateQuote_FailsOpenWhenTheStoreIsUnavailable(t *testing.T) {
	ctx := t.Context()
	store := newMemIdempotency()
	store.getErr = errors.New("store unavailable")
	store.reserveErr = errors.New("store unavailable")
	svc, repo, fakeModules := NewTestServiceWithStore(store)
	method := seedQuotePricing(t, repo, fakeModules)

	req := CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-1",
	}

	first, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)

	// The database is the authority: the retry replays the stored quote even
	// though the cache could not be reached.
	second, err := svc.CreateQuote(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
}

func TestService_CreateQuote_ReleasesTheReservationOnFailure(t *testing.T) {
	ctx := t.Context()
	store := newMemIdempotency()
	svc, repo, fakeModules := NewTestServiceWithStore(store)
	method := seedQuotePricing(t, repo, fakeModules)

	_, err := svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       method.ID,
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)

	// A failing request must not hold the key, so a later retry may proceed.
	_, err = svc.CreateQuote(ctx, CreateQuoteRequest{
		CustomerID:     "cust1",
		MethodID:       "missing",
		AddressID:      "addr1",
		SubtotalCents:  1200,
		IdempotencyKey: "key-2",
	})
	require.ErrorIs(t, err, domain.ErrMethodNotFound)

	_, ok := store.records[quoteKey("key-2")]
	assert.False(t, ok)
}

// seedQuotePricing creates the method, zone, rate and address a quote needs
// and returns the method.
func seedQuotePricing(t *testing.T, repo *memory.Repository, fakeModules *FakeModules) domain.ShippingMethod {
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
	return method
}
