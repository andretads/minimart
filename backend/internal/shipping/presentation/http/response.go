package http

import (
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps shipping-domain errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrMethodNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrZoneNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrRateNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrQuoteNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrShipmentNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrAddressNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrQuoteExpired, Status: http.StatusConflict},
	{Err: domain.ErrQuoteConsumed, Status: http.StatusConflict},
	{Err: domain.ErrShipmentExists, Status: http.StatusConflict},
	{Err: domain.ErrMethodInactive, Status: http.StatusConflict},
	{Err: domain.ErrZoneInactive, Status: http.StatusConflict},
	{Err: domain.ErrRateInactive, Status: http.StatusConflict},
	{Err: domain.ErrQuoteDestinationMismatch, Status: http.StatusConflict},
	{Err: domain.ErrIdempotencyConflict, Status: http.StatusConflict},
	{Err: domain.ErrQuoteInProgress, Status: http.StatusConflict},
	{Err: domain.ErrShipmentInProgress, Status: http.StatusConflict},
	{Err: domain.ErrOwnershipMismatch, Status: http.StatusForbidden},
	{Err: domain.ErrQuoteOwnershipMismatch, Status: http.StatusForbidden},
	{Err: domain.ErrInvalidQuoteDestination, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidTransition, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidInput, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidQuote, Status: http.StatusBadRequest},
	{Err: domain.ErrNoMatchingRate, Status: http.StatusBadRequest},
	{Err: domain.ErrShippingQuoteRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrAddressRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidAddress, Status: http.StatusBadRequest},
	{Err: domain.ErrUserIDRequired, Status: http.StatusBadRequest},
	{Err: domain.ErrIdempotencyKeyRequired, Status: http.StatusBadRequest},
}

// respondDomainError maps shipping-domain errors to HTTP status codes.
func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
