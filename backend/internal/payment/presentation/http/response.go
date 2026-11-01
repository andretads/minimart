package http

import (
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps payment sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrPaymentNotFound, Status: http.StatusNotFound},
	{Err: application.ErrOrderNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrIdempotencyConflict, Status: http.StatusConflict},
	{Err: application.ErrPaymentInProgress, Status: http.StatusConflict},
	{Err: application.ErrOrderNotPayable, Status: http.StatusConflict},
	{Err: domain.ErrPaymentNotRefundable, Status: http.StatusConflict},
	{Err: application.ErrOrderNotCancelled, Status: http.StatusConflict},
	{Err: application.ErrRefundFailed, Status: http.StatusBadGateway},
	{Err: domain.ErrPaymentAmountInvalid, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentCurrencyRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentMethodRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentUserRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentUserMismatch, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentAmountMismatch, Status: http.StatusBadRequest},
	{Err: domain.ErrPaymentCurrencyMismatch, Status: http.StatusBadRequest},
	{Err: application.ErrIdempotencyKeyRequired, Status: http.StatusBadRequest},
}

// respondDomainError maps payment errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
