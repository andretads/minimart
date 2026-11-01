package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrder(t *testing.T) {
	tests := []struct {
		name    string
		items   []Item
		wantErr error
	}{
		{"empty items", []Item{}, ErrNoItems},
		{"quantity zero", []Item{{ProductID: "p1", Name: "A", UnitPriceCents: 100, Quantity: 0}}, ErrInvalidQuantity},
		{"quantity negative", []Item{{ProductID: "p1", Name: "A", UnitPriceCents: 100, Quantity: -1}}, ErrInvalidQuantity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOrder("id-1", "user-1", "cust-1", "k1", tt.items)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewOrderComputesTotals(t *testing.T) {
	before := time.Now().UTC()
	items := []Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 100, Quantity: 2},
		{ProductID: "p2", Name: "Gadget", UnitPriceCents: 50, Quantity: 3},
	}
	o, err := NewOrder("id-1", "user-1", "cust-1", "k1", items)
	require.NoError(t, err)

	assert.Equal(t, "id-1", o.ID)
	assert.Equal(t, "user-1", o.UserID)
	assert.Equal(t, StatusPlaced, o.Status)
	assert.Nil(t, o.CancelledAt)
	require.Len(t, o.Items, 2)
	assert.Equal(t, int64(200), o.Items[0].LineTotalCents)
	assert.Equal(t, int64(150), o.Items[1].LineTotalCents)
	assert.Equal(t, int64(350), o.TotalCents)
	assert.False(t, o.CreatedAt.Before(before))
	assert.False(t, o.CreatedAt.After(time.Now().UTC()))
}

func TestOrderCancel(t *testing.T) {
	t.Run("marks a placed order cancelled", func(t *testing.T) {
		o, err := NewOrder("id-1", "user-1", "cust-1", "k1", []Item{{ProductID: "p1", Name: "A", UnitPriceCents: 100, Quantity: 1}})
		require.NoError(t, err)

		cancelled, err := o.Cancel()
		require.NoError(t, err)
		assert.Equal(t, StatusCancelled, cancelled.Status)
		require.NotNil(t, cancelled.CancelledAt)
		assert.Equal(t, StatusPlaced, o.Status, "Cancel does not mutate the receiver")
		assertHasStockRelease(t, cancelled)
	})

	t.Run("rejects cancelling twice", func(t *testing.T) {
		o, err := NewOrder("id-1", "user-1", "cust-1", "k1", []Item{{ProductID: "p1", Name: "A", UnitPriceCents: 100, Quantity: 1}})
		require.NoError(t, err)
		cancelled, err := o.Cancel()
		require.NoError(t, err)

		_, err = cancelled.Cancel()
		assert.ErrorIs(t, err, ErrOrderAlreadyCancelled)
	})

	t.Run("a paid order cannot be cancelled", func(t *testing.T) {
		paid := orderIn(t, StatusPaid)
		_, err := paid.Cancel()
		assert.ErrorIs(t, err, ErrOrderNotCancellable)
	})
}

// assertHasStockRelease asserts the order recorded exactly one OrderStockRelease
// event naming its own id.
func assertHasStockRelease(t *testing.T, o Order) {
	t.Helper()
	var found int
	for _, ev := range o.Events() {
		if e, ok := ev.(OrderStockReleaseEvent); ok {
			found++
			assert.Equal(t, o.ID, e.OrderID)
		}
	}
	assert.Equal(t, 1, found, "exactly one stock-release event is recorded")
}

// orderIn returns a placed order forced into the given status. Tests set status
// directly because the payment transitions are the subject under test.
func orderIn(t *testing.T, status Status) Order {
	t.Helper()
	o, err := NewOrder("id-1", "user-1", "cust-1", "k1", []Item{{ProductID: "p1", Name: "A", UnitPriceCents: 100, Quantity: 1}})
	require.NoError(t, err)
	o.Status = status
	return o
}

func TestMarkPaid(t *testing.T) {
	for _, status := range []Status{StatusPlaced, StatusPaymentUnknown} {
		t.Run("moves "+string(status)+" to paid", func(t *testing.T) {
			got := orderIn(t, status).MarkPaid()
			assert.Equal(t, StatusPaid, got.Status)
			assert.Len(t, got.Events(), 1, "a payment outcome records no order event")
		})
	}

	// payment_failed is terminal for fulfilment: a successful retry is a new
	// order, so a late paid event must not resurrect it. paid and cancelled are
	// likewise absorbing.
	for _, status := range []Status{StatusPaid, StatusPaymentFailed, StatusCancelled} {
		t.Run("no-op when "+string(status), func(t *testing.T) {
			o := orderIn(t, status)
			got := o.MarkPaid()
			assert.Equal(t, status, got.Status)
			assert.Equal(t, o, got, "a no-op returns the order unchanged")
		})
	}
}

func TestMarkPaymentFailed(t *testing.T) {
	for _, status := range []Status{StatusPlaced, StatusPaymentUnknown} {
		t.Run("moves "+string(status)+" to payment_failed", func(t *testing.T) {
			got := orderIn(t, status).MarkPaymentFailed()
			assert.Equal(t, StatusPaymentFailed, got.Status)
			assertHasStockRelease(t, got)
		})
	}

	for _, status := range []Status{StatusPaid, StatusPaymentFailed, StatusCancelled} {
		t.Run("no-op when "+string(status), func(t *testing.T) {
			o := orderIn(t, status)
			got := o.MarkPaymentFailed()
			assert.Equal(t, status, got.Status)
			assert.Equal(t, o, got)
		})
	}
}

func TestMarkPaymentUnknown(t *testing.T) {
	t.Run("moves placed to payment_unknown", func(t *testing.T) {
		got := orderIn(t, StatusPlaced).MarkPaymentUnknown()
		assert.Equal(t, StatusPaymentUnknown, got.Status)
		assert.Len(t, got.Events(), 1)
	})

	for _, status := range []Status{StatusPaid, StatusPaymentFailed, StatusPaymentUnknown, StatusCancelled} {
		t.Run("no-op when "+string(status), func(t *testing.T) {
			o := orderIn(t, status)
			got := o.MarkPaymentUnknown()
			assert.Equal(t, status, got.Status)
			assert.Equal(t, o, got)
		})
	}
}

func TestMarkPaidDoesNotMutateReceiver(t *testing.T) {
	o := orderIn(t, StatusPlaced)
	_ = o.MarkPaid()
	assert.Equal(t, StatusPlaced, o.Status, "MarkPaid does not mutate the receiver")
}

func TestMarkPaidWithShipping(t *testing.T) {
	dest := DestinationSnapshot{
		Label:    "Home",
		FullName: "Ada Lovelace",
		Street:   "1 Main St",
		City:     "Testville",
		State:    "TS",
		Zip:      "12345",
		Country:  "BR",
		Phone:    "555-0100",
	}

	t.Run("moves placed to paid and records the order-paid event", func(t *testing.T) {
		o := orderIn(t, StatusPlaced)
		o.ShippingQuoteID = "quote-1"
		o.ShippingMethodID = "method-1"
		o.ShippingMethodName = "Standard"
		o.ShippingCostCents = 250
		o.DestinationSnapshot = dest

		got := o.MarkPaidWithShipping()
		assert.Equal(t, StatusPaid, got.Status)

		var paid *OrderPaidEvent
		for _, ev := range got.Events() {
			if e, ok := ev.(OrderPaidEvent); ok {
				paid = &e
			}
		}
		require.NotNil(t, paid, "the order-paid event is recorded for the shipping handoff")
		assert.Equal(t, o.ID, paid.OrderID)
		assert.Equal(t, o.UserID, paid.UserID)
		assert.Equal(t, o.CustomerID, paid.CustomerID)
		assert.Equal(t, o.TotalCents, paid.TotalCents)
		assert.Equal(t, "quote-1", paid.ShippingQuoteID)
		assert.Equal(t, "method-1", paid.ShippingMethodID)
		assert.Equal(t, dest, paid.DestinationSnapshot)
	})

	for _, status := range []Status{StatusPaid, StatusPaymentFailed, StatusCancelled} {
		t.Run("no-op when "+string(status), func(t *testing.T) {
			o := orderIn(t, status)
			got := o.MarkPaidWithShipping()
			assert.Equal(t, status, got.Status)
			assert.Equal(t, o, got, "a no-op returns the order unchanged")
		})
	}
}
