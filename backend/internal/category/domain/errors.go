package domain

import "errors"

var (
	ErrEmptyName              = errors.New("category: nome é obrigatório")
	ErrNameTooLong            = errors.New("category: nome muito longo")
	ErrIconTooLong            = errors.New("category: ícone muito longo")
	ErrCategoryNotFound       = errors.New("category: não encontrada")
	ErrCategoryExists         = errors.New("category: já existe")
	ErrCategoryOptimisticLock = errors.New("category: conflito de bloqueio")
	ErrInvalidVersion         = errors.New("category: versão deve ser maior que zero")
)
