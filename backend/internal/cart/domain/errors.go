package domain

import "errors"

var (
	ErrCartNotFound        = errors.New("cart: carrinho não encontrado")
	ErrEmptyCustomerID     = errors.New("cart: id do cliente vazio")
	ErrInvalidQuantity     = errors.New("cart: quantidade inválida")
	ErrCartItemNotFound    = errors.New("cart: item não encontrado")
	ErrCartNotActive       = errors.New("cart: carrinho não ativo")
	ErrCartEmpty           = errors.New("cart: carrinho vazio")
	ErrCartCheckoutPending = errors.New("cart: finalização em andamento")
	ErrCartConsumed        = errors.New("cart: carrinho já consumido")
	ErrCartOptimisticLock  = errors.New("cart: falha no bloqueio otimista")
	ErrCartStateChanged    = errors.New("cart: estado alterado")
	ErrCartAlreadyExists   = errors.New("cart: carrinho já existe")
)
