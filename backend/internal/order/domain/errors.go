package domain

import "errors"

var (
	ErrOrderNotFound         = errors.New("pedido não encontrado")
	ErrOrderAlreadyCancelled = errors.New("pedido já está cancelado")
	// ErrOrderNotCancellable reports that the order has reached a state from which
	// it can no longer be cancelled - currently a paid order. Reversing a paid
	// order is a refund/return, a separate use case.
	ErrOrderNotCancellable = errors.New("pedido não pode ser cancelado")
	// ErrOrderStateChanged reports that the order was changed by someone else after
	// it was read: the optimistic write lost a race and wrote nothing.
	ErrOrderStateChanged = errors.New("estado do pedido alterado")
	// ErrIdempotencyConflict reports that the idempotency key already belongs to an
	// order with different parameters; the caller must not create a second order.
	ErrIdempotencyConflict = errors.New("chave de idempotência reutilizada com parâmetros diferentes")
	ErrNoItems             = errors.New("pedido deve conter pelo menos um item")
	ErrInvalidQuantity     = errors.New("quantidade do item deve ser maior que zero")
	// ErrUserIDRequired reports that an order list request omitted its user id.
	ErrUserIDRequired = errors.New("id do usuário é obrigatório")
	// ErrInvalidPage reports that the requested page is less than 1.
	ErrInvalidPage = errors.New("página deve ser maior que zero")
	// ErrInvalidPageSize reports that page_size is outside 1..100.
	ErrInvalidPageSize = errors.New("tamanho da página deve estar entre 1 e 100")
)
