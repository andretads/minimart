package http

import (
	"encoding/json"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	"minimart/pkg/metrics"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrderStatusMapping(t *testing.T) {
	mux := newTestMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/orders",
		`{"user_id":"u1","items":[{"product_id":"p1","quantity":1}]}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		Data orderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.ID)

	t.Run("found returns the order", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/orders/"+created.Data.ID, "")
		require.Equal(t, http.StatusOK, got.Code)

		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Equal(t, created.Data.ID, env.Data.ID)
		assert.Equal(t, int64(500), env.Data.TotalCents)
	})

	t.Run("missing returns 404", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/orders/missing", "")
		assert.Equal(t, http.StatusNotFound, got.Code)
	})
}

func TestGetOrderReturnsShippingSnapshot(t *testing.T) {
	repo := memory.New()
	svc := application.NewService(repo, stubModules{}, newStubIdempotency(), metrics.Noop{}, time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)

	t.Run("an order placed with shipping returns its snapshot", func(t *testing.T) {
		o := savedOrder(t, "o1")
		o.ShippingQuoteID = "quote-1"
		o.ShippingMethodID = "method-1"
		o.ShippingMethodName = "Standard"
		o.ShippingCostCents = 250
		o.DestinationSnapshot = domain.DestinationSnapshot{
			Label:    "Home",
			FullName: "Ada Lovelace",
			Street:   "1 Main St",
			City:     "Testville",
			State:    "TS",
			Zip:      "12345",
			Country:  "BR",
			Phone:    "555-0100",
		}
		require.NoError(t, repo.Save(t.Context(), o))

		got := do(t, mux, http.MethodGet, "/v1/orders/o1", "")
		require.Equal(t, http.StatusOK, got.Code)

		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Equal(t, "quote-1", env.Data.ShippingQuoteID)
		assert.Equal(t, "method-1", env.Data.ShippingMethodID)
		assert.Equal(t, "Standard", env.Data.ShippingMethodName)
		assert.Equal(t, int64(250), env.Data.ShippingCostCents)
		require.NotNil(t, env.Data.DestinationSnapshot)
		assert.Equal(t, "Ada Lovelace", env.Data.DestinationSnapshot.FullName)
		assert.Equal(t, "1 Main St", env.Data.DestinationSnapshot.Street)
		assert.Equal(t, "555-0100", env.Data.DestinationSnapshot.Phone)
	})

	t.Run("an order placed without shipping returns no snapshot", func(t *testing.T) {
		require.NoError(t, repo.Save(t.Context(), savedOrder(t, "o2")))

		got := do(t, mux, http.MethodGet, "/v1/orders/o2", "")
		require.Equal(t, http.StatusOK, got.Code)

		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Empty(t, env.Data.ShippingQuoteID)
		assert.Empty(t, env.Data.ShippingMethodID)
		assert.Empty(t, env.Data.ShippingMethodName)
		assert.Zero(t, env.Data.ShippingCostCents)
		assert.Nil(t, env.Data.DestinationSnapshot)
	})
}

// savedOrder builds a placed order for the HTTP tests.
func savedOrder(t *testing.T, id string) domain.Order {
	t.Helper()
	o, err := domain.NewOrder(id, "u1", "c1", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	return o
}

func TestGetOrderAutoCancelAt(t *testing.T) {
	repo := memory.New()
	svc := application.NewService(repo, stubModules{}, newStubIdempotency(), metrics.Noop{}, time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)

	placed := savedOrder(t, "placed-1")
	require.NoError(t, repo.Save(t.Context(), placed))

	paid := savedOrder(t, "paid-1").MarkPaid()
	require.NoError(t, repo.Save(t.Context(), paid))

	t.Run("placed order carries auto_cancel_at", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/orders/placed-1", "")
		require.Equal(t, http.StatusOK, got.Code)
		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		require.NotNil(t, env.Data.AutoCancelAt)
	})

	t.Run("non-placed order omits auto_cancel_at", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/orders/paid-1", "")
		require.Equal(t, http.StatusOK, got.Code)
		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Nil(t, env.Data.AutoCancelAt)
	})
}
