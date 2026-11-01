package http

import (
	"minimart/internal/auth/domain"
	"minimart/pkg/response"
	"net/http"
)

// setNoStore prevents browsers and intermediaries from caching auth responses,
// which bear Set-Cookie headers and user data.
func setNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// errorStatuses maps auth sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrEmptyUsername, Status: http.StatusBadRequest},
	{Err: domain.ErrUsernameTooLong, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyPassword, Status: http.StatusBadRequest},
	{Err: domain.ErrPasswordTooShort, Status: http.StatusBadRequest},
	{Err: domain.ErrPasswordTooLong, Status: http.StatusBadRequest},
	{Err: domain.ErrEmptyEmail, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidEmail, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidCurrentPassword, Status: http.StatusBadRequest},
	{Err: domain.ErrPasswordMismatch, Status: http.StatusBadRequest},
	{Err: domain.ErrAccountExists, Status: http.StatusConflict},
	{Err: domain.ErrEmailExists, Status: http.StatusConflict},
	{Err: domain.ErrAccountNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrInvalidCredentials, Status: http.StatusUnauthorized},
}

// respondDomainError maps auth errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
