package http

import (
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/response"
	"net/http"

	catalogclient "minimart/internal/catalog/presentation/module/client"
)

// errorStatuses maps order-domain errors AND the catalog contract's
// ErrProductNotFound / ErrInsufficientStock to HTTP statuses. A product the
// caller named that does not exist, or has too little stock, is a bad request /
// conflict on the caller's input, not a 404 on the order resource.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrOrderNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrOrderAlreadyCancelled, Status: http.StatusConflict},
	{Err: domain.ErrOrderNotCancellable, Status: http.StatusConflict},
	{Err: domain.ErrOrderStateChanged, Status: http.StatusConflict},
	{Err: domain.ErrIdempotencyConflict, Status: http.StatusConflict},
	{Err: application.ErrOrderInProgress, Status: http.StatusConflict},
	{Err: catalogclient.ErrInsufficientStock, Status: http.StatusConflict},
	{Err: domain.ErrNoItems, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusBadRequest},
	{Err: domain.ErrUserIDRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidPage, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidPageSize, Status: http.StatusBadRequest},
	{Err: application.ErrIdempotencyKeyRequired, Status: http.StatusBadRequest},
	{Err: catalogclient.ErrProductNotFound, Status: http.StatusBadRequest},
}

// respondDomainError maps order-domain errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
