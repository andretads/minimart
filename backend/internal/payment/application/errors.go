package application

import (
	"errors"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
)

// ErrIdempotencyKeyRequired is returned when CreatePayment is called without an
// idempotency key. The HTTP layer requires the Idempotency-Key header, so this
// only surfaces if a caller bypasses it.
var ErrIdempotencyKeyRequired = errors.New("chave de idempotência é obrigatória")

// ErrPaymentInProgress is returned when another request with the same
// idempotency key is still being processed. Callers should retry shortly.
var ErrPaymentInProgress = errors.New("já existe um pagamento com esta chave de idempotência em processamento")

// ErrOrderNotFound is returned when the payment references an unknown order.
var ErrOrderNotFound = orderclient.ErrOrderNotFound

// ErrPaymentUserMismatch is returned when the payment user does not match the order's owner.
var ErrPaymentUserMismatch = domain.ErrPaymentUserMismatch

// ErrPaymentAmountMismatch is returned when the payment amount does not match the order total.
var ErrPaymentAmountMismatch = domain.ErrPaymentAmountMismatch

// ErrPaymentCurrencyMismatch is returned when the payment currency does not match the order.
var ErrPaymentCurrencyMismatch = domain.ErrPaymentCurrencyMismatch

// ErrOrderNotPayable is returned when a payment is attempted for an order that
// can no longer be paid (cancelled or already failed).
var ErrOrderNotPayable = errors.New("pedido não pode mais ser pago")

// ErrOrderNotCancelled is returned when a refund is attempted for a payment
// whose order has not been cancelled.
var ErrOrderNotCancelled = errors.New("pedido não está cancelado")

// ErrRefundFailed is returned when the payment provider refuses or fails a
// refund; the payment keeps its previous state.
var ErrRefundFailed = errors.New("reembolso falhou no provedor de pagamentos")
