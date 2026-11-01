package application

import "errors"

var (
	ErrUserIDRequired    = errors.New("wishlist: user_id é obrigatório")
	ErrProductIDRequired = errors.New("wishlist: product_id é obrigatório")
)
