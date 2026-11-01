package domain

import "errors"

// Domain errors are exported sentinels. Callers compare with errors.Is; the HTTP
// edge maps them to status codes. The domain itself knows nothing about HTTP.
var (
	ErrProductNotFound       = errors.New("produto não encontrado")
	ErrEmptyName             = errors.New("nome do produto não pode ser vazio")
	ErrNonPositivePrice      = errors.New("preço do produto deve ser maior que zero")
	ErrNegativeStock         = errors.New("estoque do produto não pode ser negativo")
	ErrInvalidVersion        = errors.New("versão do produto deve ser maior que zero")
	ErrProductOptimisticLock = errors.New("conflito de bloqueio do produto")

	// ErrInsufficientStock reports that a reservation asked for more units than
	// are available. It is the domain's signal for a sold-out product.
	ErrInsufficientStock = errors.New("estoque insuficiente")
	// ErrInvalidReservation reports a reservation item with a non-positive
	// quantity.
	ErrInvalidReservation = errors.New("quantidade da reserva deve ser maior que zero")

	// Image errors
	ErrProductImageNotFound = errors.New("imagem do produto não encontrada")
	ErrImageRequired        = errors.New("imagem do produto é obrigatória")
	ErrEmptyProductID       = errors.New("id do produto da imagem é obrigatório")
	ErrEmptyObjectKey       = errors.New("chave do objeto da imagem é obrigatória")
	ErrEmptyBucket          = errors.New("bucket da imagem é obrigatório")
	ErrInvalidImageSize     = errors.New("tamanho da imagem do produto deve ser maior que zero")
	ErrInvalidImageFormat   = errors.New("formato de imagem inválido, deve ser uma imagem")
	ErrImageTooLarge        = errors.New("imagem excede o tamanho máximo permitido")
	ErrImageUploadFailed    = errors.New("falha no upload da imagem")

	// Category errors
	ErrCategoryNotFound        = errors.New("categoria não encontrada")
	ErrProductCategoryRequired = errors.New("produto deve ter uma categoria")

	// SKU errors
	ErrEmptySKU     = errors.New("SKU do produto não pode ser vazio")
	ErrDuplicateSKU = errors.New("SKU do produto já existe")

	// Review errors
	ErrInvalidRating          = errors.New("avaliação deve ser entre 1 e 5")
	ErrProductReviewNotFound  = errors.New("avaliação do produto não encontrada")
	ErrReviewAlreadyModerated = errors.New("avaliação já moderada")
	ErrModerationFailed       = errors.New("falha na moderação")
	ErrEmptyCustomerID        = errors.New("id do cliente é obrigatório")
	ErrInvalidPage            = errors.New("página inválida")
	ErrInvalidPageSize        = errors.New("tamanho de página inválido")
	ErrDuplicateReview        = errors.New("você já avaliou este produto")
)
