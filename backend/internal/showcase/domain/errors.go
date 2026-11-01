package domain

import "errors"

// ErrProductNotFound is returned when no product has the given ID.
var ErrProductNotFound = errors.New("produto não encontrado")

// ErrInvalidPage is returned when the pagination page number is less than 1.
var ErrInvalidPage = errors.New("página deve ser maior que zero")

// ErrInvalidPageSize is returned when page_size is less than 1 or greater than 100.
var ErrInvalidPageSize = errors.New("tamanho da página deve estar entre 1 e 100")

// ErrInvalidCursor is returned when the cursor token is malformed or expired.
var ErrInvalidCursor = errors.New("cursor inválido")
