package application

import "errors"

// ErrIdempotencyKeyRequired is returned when PlaceOrder is called without an
// idempotency key. The HTTP layer requires the Idempotency-Key header, so this
// only surfaces if a caller bypasses it.
var ErrIdempotencyKeyRequired = errors.New("chave de idempotência é obrigatória")

// ErrOrderInProgress is returned when another request with the same idempotency
// key is still being processed. Callers should retry shortly.
var ErrOrderInProgress = errors.New("já existe um pedido com esta chave de idempotência em processamento")
