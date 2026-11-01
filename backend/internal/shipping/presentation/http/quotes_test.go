package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateQuote(t *testing.T) {
	quoteBody := `{"method_id":"%s","address_id":"%s"}`

	setup := func(t *testing.T) (*testEnv, string, string) {
		t.Helper()
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		zone := env.seedZone(t)
		env.seedRate(t, zone.ID, method.ID, true)
		addr := env.addAddress("cust-user-1")
		return env, method.ID, addr.ID
	}

	t.Run("valid request returns 201", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, methodID, addressID), "test-key")
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var quote ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(envl.Data, &quote))
		assert.NotEmpty(t, quote.ID)
		assert.Equal(t, "cust-user-1", quote.CustomerID)
		assert.Equal(t, methodID, quote.MethodID)
		assert.Equal(t, int64(599), quote.ShippingCostCents)
		assert.Equal(t, "active", quote.Status)
	})

	t.Run("missing authentication returns 401", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doWithKey(t, http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, methodID, addressID), "test-key")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes", `{`, "test-key")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("inactive method returns 409", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, false)
		addr := env.addAddress("cust-user-1")

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, method.ID, addr.ID), "test-key")
		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("unknown method returns 404", func(t *testing.T) {
		env, _, addressID := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, "missing", addressID), "test-key")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unknown address returns 404", func(t *testing.T) {
		env, methodID, _ := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, methodID, "missing"), "test-key")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("address owned by another customer returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		env.seedZone(t)
		env.seedRate(t, "zone-1", method.ID, true)
		addr := env.addAddress("cust-user-2")

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, method.ID, addr.ID), "test-key")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("no matching zone returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		addr := env.addAddress("cust-user-1")

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, method.ID, addr.ID), "test-key")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("no matching rate returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		env.seedZone(t)
		addr := env.addAddress("cust-user-1")

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, method.ID, addr.ID), "test-key")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a subtotal at the free threshold ships free", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(`{"method_id":"%s","address_id":"%s","subtotal_cents":5000}`, methodID, addressID), "test-key")
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		var quote ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(envl.Data, &quote))
		assert.Equal(t, int64(0), quote.ShippingCostCents)
		assert.Equal(t, int64(5000), quote.SubtotalCents)
		assert.Equal(t, int64(5000), quote.FreeThresholdCents)
	})

	t.Run("a subtotal below the free threshold pays the base cost", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(`{"method_id":"%s","address_id":"%s","subtotal_cents":4999}`, methodID, addressID), "test-key")
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		var quote ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(envl.Data, &quote))
		assert.Equal(t, int64(599), quote.ShippingCostCents)
		assert.Equal(t, int64(4999), quote.SubtotalCents)
	})

	t.Run("a missing idempotency key returns 400", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doAs(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, methodID, addressID))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("the same key replays the original quote", func(t *testing.T) {
		env, methodID, addressID := setup(t)
		body := fmt.Sprintf(quoteBody, methodID, addressID)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes", body, "key-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		first := decodeEnvelope(t, rec)
		var firstQuote ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(first.Data, &firstQuote))

		rec = env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes", body, "key-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		second := decodeEnvelope(t, rec)
		var secondQuote ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(second.Data, &secondQuote))

		assert.Equal(t, firstQuote.ID, secondQuote.ID)
	})

	t.Run("the same key with other parameters returns 409", func(t *testing.T) {
		env, methodID, addressID := setup(t)

		rec := env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(quoteBody, methodID, addressID), "key-1")
		require.Equal(t, http.StatusCreated, rec.Code)

		rec = env.doAsWithKey(t, "user-1", http.MethodPost, "/v1/shipping/quotes",
			fmt.Sprintf(`{"method_id":"%s","address_id":"%s","subtotal_cents":2400}`, methodID, addressID), "key-1")
		require.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestGetQuote(t *testing.T) {
	t.Run("found returns the quote", func(t *testing.T) {
		env := newTestEnv(t)
		quote := env.seedQuote(t, "cust-user-1")

		rec := env.doAs(t, "cust-user-1", http.MethodGet, "/v1/shipping/quotes/"+quote.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var got ShippingQuoteResponse
		require.NoError(t, json.Unmarshal(envl.Data, &got))
		assert.Equal(t, quote.ID, got.ID)
		assert.Equal(t, "cust-user-1", got.CustomerID)
		assert.Equal(t, int64(599), got.ShippingCostCents)
	})

	t.Run("another user's quote returns 403", func(t *testing.T) {
		env := newTestEnv(t)
		quote := env.seedQuote(t, "cust-user-1")

		rec := env.doAs(t, "cust-user-2", http.MethodGet, "/v1/shipping/quotes/"+quote.ID, "")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing quote returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAs(t, "cust-user-1", http.MethodGet, "/v1/shipping/quotes/missing", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("without a user returns 401", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodGet, "/v1/shipping/quotes/missing", "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
