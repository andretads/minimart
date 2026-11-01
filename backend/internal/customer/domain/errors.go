package domain

import "errors"

var (
	ErrEmptyUserID            = errors.New("customer: user_id é obrigatório")
	ErrCustomerNotFound       = errors.New("customer: não encontrado")
	ErrCustomerExists         = errors.New("customer: já existe para o usuário")
	ErrDisplayNameTooLong     = errors.New("customer: nome de exibição muito longo")
	ErrCustomerOptimisticLock = errors.New("customer: conflito de bloqueio")
	ErrInvalidVersion         = errors.New("customer: versão deve ser maior que zero")
)
