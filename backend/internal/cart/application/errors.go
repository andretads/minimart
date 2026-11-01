package application

import "errors"

var (
	ErrCartCheckoutInProgress   = errors.New("cart: finalização em andamento")
	ErrUserIdRequired           = errors.New("cart: parâmetro de consulta user_id é obrigatório")
	ErrIdempotencyKeyRequired   = errors.New("cart: cabeçalho Idempotency-Key é obrigatório")
	ErrShippingQuoteRequired    = errors.New("cart: ID da cotação de frete é obrigatório")
	ErrAddressRequired          = errors.New("cart: ID do endereço é obrigatório")
	ErrQuoteOwnershipMismatch   = errors.New("cart: a cotação de frete não pertence ao cliente")
	ErrQuoteDestinationMismatch = errors.New("cart: o destino da cotação de frete não corresponde ao endereço")
	ErrAddressNotFound          = errors.New("cart: endereço não encontrado")
)
