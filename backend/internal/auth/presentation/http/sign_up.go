package http

import (
	"encoding/json"
	"minimart/internal/auth/application"
	"minimart/pkg/response"
	"net/http"
)

// SignUp creates a new account.
//
//	@Summary		Create a new account
//	@Description	Creates a new user account with username, email, and password. Returns the created user info.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			signup	body		signUpRequest	true	"Account credentials"
//	@Success		201		{object}	response.Response{data=userInfo}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/auth/signup [post]
func (h *Handlers) SignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}

	acc, err := h.svc.SignUp(r.Context(), application.SignUpRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, response.Created(toUserInfo(acc.ID, acc.Username, acc.Email, acc.CreatedAt)))
}
