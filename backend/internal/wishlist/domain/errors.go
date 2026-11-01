package domain

import "errors"

var (
	ErrWishlistNotFound         = errors.New("wishlist: lista de desejos não encontrada")
	ErrEmptyCustomerID          = errors.New("wishlist: id do cliente vazio")
	ErrProductAlreadyInWishlist = errors.New("wishlist: produto já está na lista de desejos")
	ErrProductNotInWishlist     = errors.New("wishlist: produto não está na lista de desejos")
	ErrWishlistAlreadyExists    = errors.New("wishlist: lista de desejos já existe")
)
