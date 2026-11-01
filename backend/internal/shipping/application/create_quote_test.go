package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/idempotency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type holdIdem struct{ records map[string]idempotency.Record }

func newHoldIdem() *holdIdem { return &holdIdem{records: map[string]idempotency.Record{}} }

func (h *holdIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	r, ok := h.records[key]
	return r, ok, nil
}

func (h *holdIdem) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if _, ok := h.records[key]; ok {
		return false, nil
	}
	// Simulate another request holding the reservation (in-flight): ResourceID empty.
	h.records[key] = idempotency.Record{Fingerprint: fp}
	return false, nil
}

func (h *holdIdem) Complete(_ context.Context, key, fp, ref string, _ time.Duration) error {
	h.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: ref}
	return nil
}

func (h *holdIdem) Release(_ context.Context, key, _ string) error {
	delete(h.records, key)
	return nil
}

func TestCreateQuote(t *testing.T) {
	t.Run("Blank customer, method or address is rejected", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 1200, IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuote)
	})

	t.Run("Negative subtotal is rejected", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: -1, IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuote)
	})

	t.Run("The method does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: "missing", AddressID: "addr1", IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodNotFound)
	})

	t.Run("The method is inactive", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m, err := domain.NewShippingMethod("Express", "Express", false, 1, 2)
		require.NoError(t, err)
		require.NoError(t, repo.CreateMethod(t.Context(), m))
		z, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
		require.NoError(t, err)
		require.NoError(t, repo.CreateZone(t.Context(), z))
		r, err := domain.NewShippingRate(z.ID, m.ID, 599, 5000, true)
		require.NoError(t, err)
		require.NoError(t, repo.CreateRate(t.Context(), r))
		seedAddressFeature(fm, "cust1", "addr1", "BR")

		_, err = svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodInactive)
	})

	t.Run("The address does not belong to the customer", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust2", MethodID: m.ID, AddressID: "addr1", IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("No zone matches the address", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)
		z2, err := domain.NewShippingZone("US", "US", "", "", "", "")
		require.NoError(t, err)
		require.NoError(t, repo.CreateZone(t.Context(), z2))
		seedAddressFeature(fm, "cust1", "addr2", "US")

		_, err = svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr2", IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNoMatchingRate)
	})

	t.Run("No rate matches the zone and method", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		_ = seedQuoteFixtureFeature(t, repo, fm)
		m3, err := domain.NewShippingMethod("Freight", "Freight", true, 5, 10)
		require.NoError(t, err)
		require.NoError(t, repo.CreateMethod(t.Context(), m3))

		_, err = svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m3.ID, AddressID: "addr1", IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNoMatchingRate)
	})

	t.Run("The rate is inactive", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m, err := domain.NewShippingMethod("Standard", "Standard", true, 3, 5)
		require.NoError(t, err)
		require.NoError(t, repo.CreateMethod(t.Context(), m))
		z, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
		require.NoError(t, err)
		require.NoError(t, repo.CreateZone(t.Context(), z))
		r, err := domain.NewShippingRate(z.ID, m.ID, 299, 0, false)
		require.NoError(t, err)
		require.NoError(t, repo.CreateRate(t.Context(), r))
		seedAddressFeature(fm, "cust1", "addr1", "BR")

		_, err = svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 12000, IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateInactive)
	})

	t.Run("The request carries no idempotency key", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1"})
		require.Error(t, err)
		// The feature expects "cabeçalho Idempotency-Key é obrigatório".
		assert.ErrorIs(t, err, domain.ErrIdempotencyKeyRequired)
	})

	t.Run("The idempotency key is reused with other parameters", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)
		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 1200, IdempotencyKey: "key-1"})
		require.NoError(t, err)

		_, err = svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 2400, IdempotencyKey: "key-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})

	t.Run("The quote is already in progress", func(t *testing.T) {
		store := newHoldIdem()
		svc, repo, fm := newFeatureServiceWithStore(t, store)
		m := seedQuoteFixtureFeature(t, repo, fm)

		_, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 1200, IdempotencyKey: "key-2"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrQuoteInProgress)
	})

	t.Run("A subtotal below the threshold costs the base cost", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		q, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 4999, IdempotencyKey: "key-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(599), q.ShippingCostCents)
	})

	t.Run("A subtotal at the threshold is free", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		q, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 5000, IdempotencyKey: "key-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.ShippingCostCents)
	})

	t.Run("A subtotal above the threshold is free", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		q, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 12000, IdempotencyKey: "key-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.ShippingCostCents)
	})

	t.Run("A rate without a free threshold always costs the base cost", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m, err := domain.NewShippingMethod("Standard", "Standard", true, 3, 5)
		require.NoError(t, err)
		require.NoError(t, repo.CreateMethod(t.Context(), m))
		z, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
		require.NoError(t, err)
		require.NoError(t, repo.CreateZone(t.Context(), z))
		r, err := domain.NewShippingRate(z.ID, m.ID, 299, 0, true)
		require.NoError(t, err)
		require.NoError(t, repo.CreateRate(t.Context(), r))
		seedAddressFeature(fm, "cust1", "addr1", "BR")

		q, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 12000, IdempotencyKey: "key-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(299), q.ShippingCostCents)
	})

	t.Run("The same idempotency key replays the original quote", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)
		req := CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 1200, IdempotencyKey: "key-1"}
		first, err := svc.CreateQuote(t.Context(), req)
		require.NoError(t, err)

		second, err := svc.CreateQuote(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
	})

	t.Run("The quote snapshots the subtotal and the threshold", func(t *testing.T) {
		svc, repo, fm := newFeatureService(t)
		m := seedQuoteFixtureFeature(t, repo, fm)

		q, err := svc.CreateQuote(t.Context(), CreateQuoteRequest{CustomerID: "cust1", MethodID: m.ID, AddressID: "addr1", SubtotalCents: 1200, IdempotencyKey: "key-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(1200), q.SubtotalCents)
		assert.Equal(t, int64(5000), q.FreeThresholdCents)
	})
}
