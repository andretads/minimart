package http

import (
	"minimart/internal/payment/domain"
	"time"
)

// createPaymentRequest is the body of POST /v1/payments. The idempotency key is
// NOT part of the body: it travels in the Idempotency-Key header.
type createPaymentRequest struct {
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Method      string `json:"method"`
} //	@name	PaymentCreateRequest

type paymentResponse struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	UserID        string  `json:"user_id"`
	AmountCents   int64   `json:"amount_cents"`
	Currency      string  `json:"currency"`
	Method        string  `json:"method"`
	Status        string  `json:"status"`
	Provider      string  `json:"provider"`
	ProviderRef   *string `json:"provider_ref,omitempty"`
	FailureReason *string `json:"failure_reason,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
} //	@name	Payment

func toPaymentResponse(p domain.Payment) paymentResponse {
	return paymentResponse{
		ID:            p.ID,
		OrderID:       p.OrderID,
		UserID:        p.UserID,
		AmountCents:   p.AmountCents,
		Currency:      p.Currency,
		Method:        p.Method,
		Status:        string(p.Status),
		Provider:      p.Provider,
		ProviderRef:   p.ProviderRef,
		FailureReason: p.FailureReason,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.Format(time.RFC3339),
	}
}
