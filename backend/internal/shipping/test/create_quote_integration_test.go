//go:build integration

package test

import (
	"context"
	"fmt"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/idempotency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type holdIdem struct{ records map[string]idempotency.Record }

func (h *holdIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	r, ok := h.records[key]
	return r, ok, nil
}

func (h *holdIdem) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if _, ok := h.records[key]; ok {
		return false, nil
	}
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

func TestCreateQuoteIntegration(t *testing.T) {
	t.Run("Blank customer, method or address is rejected", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: "", MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuote)
	})

	t.Run("Negative subtotal is rejected", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: -1, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuote)
	})

	t.Run("The method does not exist", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: "missing", AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodNotFound)
	})

	t.Run("The method is inactive", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Express", false)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodInactive)
	})

	t.Run("The address does not belong to the customer", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c1 := createTestCustomer(t, custSvc)
		c2 := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a1 := createTestAddressExact(t, addrSvc, c1.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c2.ID, MethodID: m.ID, AddressID: a1.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("No zone matches the address", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		other := []string{"FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG", "DE"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: other, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNoMatchingRate)
	})

	t.Run("No rate matches the zone and method", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m1 := mustCreateMethod(t, svc, "Standard", true)
		m3 := mustCreateMethod(t, svc, "Freight", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m1.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m3.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNoMatchingRate)
	})

	t.Run("The rate is inactive", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 299, 0, false)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 12000, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateInactive)
	})

	t.Run("The request carries no idempotency key", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200})
		require.Error(t, err)
		// The feature expects "cabeçalho Idempotency-Key é obrigatório".
		assert.ErrorIs(t, err, domain.ErrIdempotencyKeyRequired)
	})

	t.Run("The idempotency key is reused with other parameters", func(t *testing.T) {
		key := "key-" + uniqueShippingID()
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: key})
		require.NoError(t, err)
		_, err = svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 2400, IdempotencyKey: key})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})

	t.Run("The quote is already in progress", func(t *testing.T) {
		store := &holdIdem{records: map[string]idempotency.Record{}}
		svc, addrSvc, custSvc, _, _ := newFeatureServicesWithIdem(t, store)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrQuoteInProgress)
	})

	t.Run("A subtotal below the threshold costs the base cost", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		q, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 4999, IdempotencyKey: "key-" + uniqueShippingID()})
		require.NoError(t, err)
		assert.Equal(t, int64(599), q.ShippingCostCents)
	})

	t.Run("A subtotal at the threshold is free", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		q, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 5000, IdempotencyKey: "key-" + uniqueShippingID()})
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.ShippingCostCents)
	})

	t.Run("A subtotal above the threshold is free", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		q, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 12000, IdempotencyKey: "key-" + uniqueShippingID()})
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.ShippingCostCents)
	})

	t.Run("A rate without a free threshold always costs the base cost", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 299, 0, true)
		q, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 12000, IdempotencyKey: "key-" + uniqueShippingID()})
		require.NoError(t, err)
		assert.Equal(t, int64(299), q.ShippingCostCents)
	})

	t.Run("The same idempotency key replays the original quote", func(t *testing.T) {
		key := "key-" + uniqueShippingID()
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		req := application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: key}
		first, err := svc.CreateQuote(t.Context(), req)
		require.NoError(t, err)
		second, err := svc.CreateQuote(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
	})

	t.Run("The quote snapshots the subtotal and the threshold", func(t *testing.T) {
		svc, addrSvc, custSvc, _, _ := newFeatureServices(t)
		c := createTestCustomer(t, custSvc)
		n := uniqueShippingSeq.Add(1)
		loc := fmt.Sprintf("%05d", n)
		country := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "IE", "GR", "HU", "RO", "BG"}[int(n)%20]
		a := createTestAddressExact(t, addrSvc, c.ID, country, "City"+loc, "ST"+loc, loc)
		m := mustCreateMethod(t, svc, "Standard", true)
		z := mustCreateZoneExact(t, svc, application.CreateZoneRequest{Name: "Z-" + uniqueShippingID(), Country: country, State: "ST" + loc, City: "City" + loc, PostalCodeStart: loc, PostalCodeEnd: loc})
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		q, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{CustomerID: c.ID, MethodID: m.ID, AddressID: a.ID, SubtotalCents: 1200, IdempotencyKey: "key-" + uniqueShippingID()})
		require.NoError(t, err)
		assert.Equal(t, int64(1200), q.SubtotalCents)
		assert.Equal(t, int64(5000), q.FreeThresholdCents)
	})
}
