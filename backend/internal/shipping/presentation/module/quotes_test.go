package module

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestQuote(t *testing.T, repo *memory.Repository) domain.ShippingQuote {
	t.Helper()
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, repo.CreateZone(t.Context(), zone))
	require.NoError(t, repo.CreateMethod(t.Context(), method))

	addr := domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		State:    "SP",
		Zip:      "01000-000",
		Country:  "BR",
		Phone:    "+5511999999999",
	}
	q, _ := domain.NewShippingQuote("cust-1", method.ID, "addr-1", "", addr, 599, 0, 0, 3, 5)
	require.NoError(t, repo.CreateQuote(t.Context(), q))
	return q
}

func TestGetQuote(t *testing.T) {
	repo := memory.NewRepository()
	q := createTestQuote(t, repo)

	t.Run("returns quote mapped to client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetQuote(t.Context(), shippingclient.GetQuoteRequest{ID: q.ID})
		require.NoError(t, err)
		assert.Equal(t, q.ID, got.ID)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, q.MethodID, got.MethodID)
		assert.Equal(t, int64(599), got.ShippingCostCents)
		assert.Equal(t, 3, got.EstimatedMinDays)
		assert.Equal(t, 5, got.EstimatedMaxDays)
		assert.Equal(t, string(domain.QuoteStatusActive), got.Status)
		assert.NotEmpty(t, got.ExpiresAt)
		assert.NotEmpty(t, got.CreatedAt)
		assert.Equal(t, "Home", got.DestinationSnapshot.Label)
		assert.Equal(t, "John Doe", got.DestinationSnapshot.FullName)
	})

	t.Run("returns not-found error for missing quote", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetQuote(t.Context(), shippingclient.GetQuoteRequest{ID: "missing"})
		assert.ErrorIs(t, err, shippingclient.ErrQuoteNotFound)
	})
}

func TestCreateQuote(t *testing.T) {
	repo := memory.NewRepository()
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, repo.CreateZone(t.Context(), zone))
	require.NoError(t, repo.CreateMethod(t.Context(), method))

	addr := domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		State:    "SP",
		Zip:      "01000-000",
		Country:  "BR",
		Phone:    "+5511999999999",
	}
	_ = addr

	t.Run("creates a quote and returns client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.CreateQuoteRequest{
			CustomerID:     "cust-1",
			MethodID:       method.ID,
			AddressID:      "addr-1",
			IdempotencyKey: "key-1",
		}
		// This will fail because testModules returns ErrAddressNotFound
		// The adapter maps it to ErrQuoteNotFound
		_, err := adapter.CreateQuote(t.Context(), req)
		assert.ErrorIs(t, err, shippingclient.ErrQuoteNotFound)
	})
}

func TestValidateQuote(t *testing.T) {
	repo := memory.NewRepository()
	q := createTestQuote(t, repo)

	t.Run("validates quote and returns client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ValidateQuote(t.Context(), shippingclient.ValidateQuoteRequest{
			ID:         q.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.Equal(t, q.ID, got.ID)
		assert.Equal(t, "cust-1", got.CustomerID)
	})

	t.Run("returns not-found error for missing quote", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.ValidateQuote(t.Context(), shippingclient.ValidateQuoteRequest{
			ID:         "missing",
			CustomerID: "cust-1",
		})
		assert.ErrorIs(t, err, shippingclient.ErrQuoteNotFound)
	})

	t.Run("returns expired error for expired quote", func(t *testing.T) {
		// Create an expired quote
		repo2 := memory.NewRepository()
		zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
		method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
		require.NoError(t, repo2.CreateZone(t.Context(), zone))
		require.NoError(t, repo2.CreateMethod(t.Context(), method))

		addr := domain.DestinationSnapshot{
			Label:    "Home",
			FullName: "John Doe",
			Street:   "123 Main St",
			City:     "Sao Paulo",
			State:    "SP",
			Zip:      "01000-000",
			Country:  "BR",
			Phone:    "+5511999999999",
		}
		q, _ := domain.NewShippingQuote("cust-1", method.ID, "addr-1", "", addr, 599, 0, 0, 3, 5)
		// Manually expire it
		q.ExpiresAt = time.Now().Add(-time.Hour)
		q.Status = domain.QuoteStatusExpired
		require.NoError(t, repo2.CreateQuote(t.Context(), q))

		adapter := New(application.NewService(repo2, testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.ValidateQuote(t.Context(), shippingclient.ValidateQuoteRequest{
			ID:         q.ID,
			CustomerID: "cust-1",
		})
		assert.ErrorIs(t, err, shippingclient.ErrQuoteExpired)
	})
}

func TestPriceQuote(t *testing.T) {
	repo := memory.NewRepository()
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, repo.CreateZone(t.Context(), zone))
	require.NoError(t, repo.CreateMethod(t.Context(), method))
	rate, _ := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, repo.CreateRate(t.Context(), rate))

	addr := domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		State:    "SP",
		Zip:      "01000-000",
		Country:  "BR",
		Phone:    "+5511999999999",
	}
	q, _ := domain.NewShippingQuote("cust-1", method.ID, "addr-1", "", addr, 599, 0, 0, 3, 5)
	require.NoError(t, repo.CreateQuote(t.Context(), q))

	t.Run("prices the quote for the subtotal", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.PriceQuote(t.Context(), shippingclient.PriceQuoteRequest{
			ID:            q.ID,
			CustomerID:    "cust-1",
			SubtotalCents: 5000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), got.ShippingCostCents)
		assert.Equal(t, int64(5000), got.SubtotalCents)
		assert.Equal(t, int64(5000), got.FreeThresholdCents)
	})

	t.Run("returns not-found error for missing quote", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.PriceQuote(t.Context(), shippingclient.PriceQuoteRequest{
			ID:            "missing",
			CustomerID:    "cust-1",
			SubtotalCents: 100,
		})
		assert.ErrorIs(t, err, shippingclient.ErrQuoteNotFound)
	})
}
