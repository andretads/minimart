package http

import (
	"minimart/internal/address/domain"
	"minimart/pkg/response"
	"net/http"
)

// errorStatuses maps address sentinel errors to HTTP statuses with fixed
// client messages, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrAddressNotFound, Status: http.StatusNotFound, Message: "endereço não encontrado"},
	{Err: domain.ErrAddressExists, Status: http.StatusConflict, Message: "endereço já existe"},
	{Err: domain.ErrInvalidVersion, Status: http.StatusConflict, Message: "endereço foi modificado por outra requisição; recarregue e tente novamente"},
	{Err: domain.ErrDefaultAddressExists, Status: http.StatusConflict, Message: "cliente já possui um endereço padrão"},
	{Err: domain.ErrEmptyCustomerID, Status: http.StatusBadRequest, Message: "id do cliente é obrigatório"},
	{Err: domain.ErrEmptyLabel, Status: http.StatusBadRequest, Message: "rótulo é obrigatório"},
	{Err: domain.ErrLabelTooLong, Status: http.StatusBadRequest, Message: "rótulo excede o tamanho máximo"},
	{Err: domain.ErrEmptyFullName, Status: http.StatusBadRequest, Message: "nome completo é obrigatório"},
	{Err: domain.ErrFullNameTooLong, Status: http.StatusBadRequest, Message: "nome completo excede o tamanho máximo"},
	{Err: domain.ErrEmptyStreet, Status: http.StatusBadRequest, Message: "logradouro é obrigatório"},
	{Err: domain.ErrStreetTooLong, Status: http.StatusBadRequest, Message: "logradouro excede o tamanho máximo"},
	{Err: domain.ErrEmptyCity, Status: http.StatusBadRequest, Message: "cidade é obrigatória"},
	{Err: domain.ErrCityTooLong, Status: http.StatusBadRequest, Message: "cidade excede o tamanho máximo"},
	{Err: domain.ErrEmptyState, Status: http.StatusBadRequest, Message: "estado é obrigatório"},
	{Err: domain.ErrStateTooLong, Status: http.StatusBadRequest, Message: "estado excede o tamanho máximo"},
	{Err: domain.ErrEmptyZip, Status: http.StatusBadRequest, Message: "CEP é obrigatório"},
	{Err: domain.ErrZipTooLong, Status: http.StatusBadRequest, Message: "CEP excede o tamanho máximo"},
	{Err: domain.ErrEmptyCountry, Status: http.StatusBadRequest, Message: "país é obrigatório"},
	{Err: domain.ErrCountryTooLong, Status: http.StatusBadRequest, Message: "país excede o tamanho máximo"},
	{Err: domain.ErrPhoneTooLong, Status: http.StatusBadRequest, Message: "telefone excede o tamanho máximo"},
}

func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno do servidor")
}
