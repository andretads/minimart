package http

import (
	"minimart/internal/customer/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps customer sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrCustomerNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrCustomerExists, Status: http.StatusConflict},
	{Err: domain.ErrCustomerOptimisticLock, Status: http.StatusConflict},
	{Err: domain.ErrEmptyUserID, Status: http.StatusBadRequest},
	{Err: domain.ErrDisplayNameTooLong, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidVersion, Status: http.StatusBadRequest},
}

// respondDomainError maps customer errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
