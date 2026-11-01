package http

import (
	"minimart/internal/category/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps category sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrCategoryNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrCategoryExists, Status: http.StatusConflict},
	{Err: domain.ErrCategoryOptimisticLock, Status: http.StatusConflict},
	{Err: domain.ErrEmptyName, Status: http.StatusBadRequest},
	{Err: domain.ErrNameTooLong, Status: http.StatusBadRequest},
	{Err: domain.ErrIconTooLong, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidVersion, Status: http.StatusBadRequest},
}

// respondDomainError maps category errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
