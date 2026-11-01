package http

import (
	"minimart/internal/notification/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps notification sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrNotificationNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrEmptyID, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyUserID, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyOrderID, Status: http.StatusBadRequest},
	{Err: domain.ErrNonPositiveTotal, Status: http.StatusBadRequest},
}

// respondDomainError is the single place that maps a domain error to an HTTP
// status. A message we don't recognise becomes a generic 500, so internal
// details never leak to clients.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
