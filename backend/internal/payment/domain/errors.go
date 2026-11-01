package domain

import "errors"

var (
	ErrPaymentNotFound         = errors.New("pagamento não encontrado")
	ErrPaymentNotPending       = errors.New("payment is not pending")
	ErrPaymentNotRefundable    = errors.New("payment cannot be refunded")
	ErrPaymentAmountInvalid    = errors.New("valor do pagamento deve ser maior que zero")
	ErrPaymentCurrencyRequired = errors.New("moeda do pagamento é obrigatória")
	ErrPaymentMethodRequired   = errors.New("método de pagamento é obrigatório")
	ErrPaymentUserRequired     = errors.New("id do usuário do pagamento é obrigatório")
	ErrPaymentUserMismatch     = errors.New("usuário do pagamento não corresponde ao pedido")
	ErrPaymentAmountMismatch   = errors.New("valor do pagamento não corresponde ao total do pedido")
	ErrPaymentCurrencyMismatch = errors.New("moeda do pagamento não corresponde ao pedido")
	ErrIdempotencyConflict     = errors.New("chave de idempotência reutilizada com parâmetros diferentes")
	ErrInvalidTransition       = errors.New("transição de pagamento inválida")
)
