package http

import (
	"minimart/internal/catalog/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps catalog sentinel errors to HTTP statuses, in matching
// order. Translation happens only at the edge; deeper layers never import
// net/http.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrProductNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrProductImageNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrProductReviewNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrProductOptimisticLock, Status: http.StatusConflict},
	{Err: domain.ErrDuplicateReview, Status: http.StatusConflict},
	{Err: domain.ErrEmptyName, Status: http.StatusBadRequest},
	{Err: domain.ErrNonPositivePrice, Status: http.StatusBadRequest},
	{Err: domain.ErrNegativeStock, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidVersion, Status: http.StatusBadRequest},
	{Err: domain.ErrProductCategoryRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptySKU, Status: http.StatusBadRequest},
	{Err: domain.ErrDuplicateSKU, Status: http.StatusBadRequest},
	{Err: domain.ErrImageRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidImageSize, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyProductID, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyObjectKey, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyBucket, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidRating, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidPage, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidPageSize, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidImageFormat, Status: http.StatusBadRequest},
	{Err: domain.ErrImageTooLarge, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyCustomerID, Status: http.StatusBadRequest},
	{Err: domain.ErrImageUploadFailed, Status: http.StatusInternalServerError, Message: "armazenamento de imagens indisponível"},
}

// respondDomainError is the single place that maps a domain error to an HTTP
// status. A message we don't recognise becomes a generic 500, so internal
// details never leak to clients.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
