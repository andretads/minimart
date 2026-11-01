package domain

import "errors"

var (
	ErrEmptyUsername          = errors.New("auth: nome de usuário é obrigatório")
	ErrUsernameTooLong        = errors.New("auth: nome de usuário excede o tamanho máximo")
	ErrEmptyPassword          = errors.New("auth: senha é obrigatória")
	ErrPasswordTooShort       = errors.New("auth: a senha deve ter pelo menos 6 caracteres")
	ErrPasswordTooLong        = errors.New("auth: a senha deve ter no máximo 72 caracteres")
	ErrEmptyEmail             = errors.New("auth: e-mail é obrigatório")
	ErrInvalidEmail           = errors.New("auth: e-mail inválido")
	ErrEmailExists            = errors.New("auth: e-mail já existe")
	ErrAccountNotFound        = errors.New("auth: conta não encontrada")
	ErrAccountExists          = errors.New("auth: conta já existe")
	ErrInvalidCredentials     = errors.New("auth: credenciais inválidas")
	ErrInvalidCurrentPassword = errors.New("auth: senha atual incorreta")
	ErrPasswordMismatch       = errors.New("auth: a nova senha e a confirmação não coincidem")
	ErrRoleNotFound           = errors.New("auth: role não encontrada")
	ErrPermissionNotFound     = errors.New("auth: permissão não encontrada")
	ErrRoleAlreadyAssigned    = errors.New("auth: role já atribuída à conta")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrAccountNotFound) || errors.Is(err, ErrRoleNotFound) || errors.Is(err, ErrPermissionNotFound)
}
