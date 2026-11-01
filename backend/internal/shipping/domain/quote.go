package domain

import (
	"errors"
	"minimart/pkg/message"
	"time"

	"github.com/google/uuid"
)

type QuoteStatus string

const (
	QuoteStatusActive   QuoteStatus = "active"
	QuoteStatusExpired  QuoteStatus = "expired"
	QuoteStatusConsumed QuoteStatus = "consumed"
)

// DestinationSnapshot is an immutable copy of the delivery address. It is the
// shared cross-domain value type: the event payloads and the order/shipping
// contract clients alias it.
type DestinationSnapshot = message.DestinationSnapshot

// validateDestination reports whether the snapshot carries the minimum
// address data a quote or shipment needs.
func validateDestination(d DestinationSnapshot) error {
	if d.FullName == "" || d.Street == "" || d.City == "" || d.Zip == "" || d.Country == "" {
		return errors.New("incomplete destination")
	}
	return nil
}

type ShippingQuote struct {
	ID                  string
	CustomerID          string
	MethodID            string
	AddressID           string
	IdempotencyKey      string
	DestinationSnapshot DestinationSnapshot
	ShippingCostCents   int64
	SubtotalCents       int64
	FreeThresholdCents  int64
	EstimatedMinDays    int
	EstimatedMaxDays    int
	Status              QuoteStatus
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

const QuoteTTLHours = 24

// NewShippingQuote builds an active quote for one address and method. The
// address id and the idempotency key are kept with the quote: the key makes a
// repeated request replay this quote instead of creating a second one, and the
// address id lets the request fingerprint be rebuilt from the stored quote.
func NewShippingQuote(customerID, methodID, addressID, idempotencyKey string, destination DestinationSnapshot, shippingCostCents, subtotalCents, freeThresholdCents int64, estimatedMinDays, estimatedMaxDays int) (ShippingQuote, error) {
	if customerID == "" || methodID == "" {
		return ShippingQuote{}, ErrInvalidInput
	}
	if err := validateDestination(destination); err != nil {
		return ShippingQuote{}, ErrInvalidQuoteDestination
	}
	if shippingCostCents < 0 || subtotalCents < 0 || freeThresholdCents < 0 {
		return ShippingQuote{}, ErrInvalidTransition
	}
	now := time.Now().UTC()
	return ShippingQuote{
		ID:                  uuid.New().String(),
		CustomerID:          customerID,
		MethodID:            methodID,
		AddressID:           addressID,
		IdempotencyKey:      idempotencyKey,
		DestinationSnapshot: destination,
		ShippingCostCents:   shippingCostCents,
		SubtotalCents:       subtotalCents,
		FreeThresholdCents:  freeThresholdCents,
		EstimatedMinDays:    estimatedMinDays,
		EstimatedMaxDays:    estimatedMaxDays,
		Status:              QuoteStatusActive,
		ExpiresAt:           now.Add(QuoteTTLHours * time.Hour),
		CreatedAt:           now,
	}, nil
}

func (q *ShippingQuote) ApplyPricing(shippingCostCents, subtotalCents, freeThresholdCents int64) error {
	if shippingCostCents < 0 || subtotalCents < 0 || freeThresholdCents < 0 {
		return ErrInvalidTransition
	}
	q.ShippingCostCents = shippingCostCents
	q.SubtotalCents = subtotalCents
	q.FreeThresholdCents = freeThresholdCents
	return nil
}

func (q *ShippingQuote) Validate(destination DestinationSnapshot) error {
	if q.Status != QuoteStatusActive {
		if q.Status == QuoteStatusExpired {
			return ErrQuoteExpired
		}
		return ErrQuoteConsumed
	}
	if time.Now().UTC().After(q.ExpiresAt) {
		q.Status = QuoteStatusExpired
		return ErrQuoteExpired
	}
	if q.DestinationSnapshot.Country != destination.Country ||
		q.DestinationSnapshot.State != destination.State ||
		q.DestinationSnapshot.City != destination.City ||
		q.DestinationSnapshot.Zip != destination.Zip {
		return ErrInvalidQuoteDestination
	}
	return nil
}

func (q *ShippingQuote) Consume() error {
	if q.Status != QuoteStatusActive {
		if q.Status == QuoteStatusExpired {
			return ErrQuoteExpired
		}
		return ErrQuoteConsumed
	}
	q.Status = QuoteStatusConsumed
	return nil
}

func (q *ShippingQuote) IsExpired() bool {
	return time.Now().UTC().After(q.ExpiresAt)
}

func (q *ShippingQuote) Expire() {
	if q.Status == QuoteStatusActive {
		q.Status = QuoteStatusExpired
	}
}
