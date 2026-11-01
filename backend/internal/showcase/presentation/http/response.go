package http

import (
	"minimart/internal/showcase/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps showcase sentinel errors to HTTP statuses and client
// messages, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrProductNotFound, Status: http.StatusNotFound, Message: "produto não encontrado"},
	{Err: domain.ErrInvalidPage, Status: http.StatusBadRequest, Message: "página deve ser maior que zero"},
	{Err: domain.ErrInvalidPageSize, Status: http.StatusBadRequest, Message: "tamanho da página deve estar entre 1 e 100"},
	{Err: domain.ErrInvalidCursor, Status: http.StatusBadRequest, Message: "cursor inválido"},
}

func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}
