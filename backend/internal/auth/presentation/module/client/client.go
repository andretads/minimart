package client

import (
	"context"
	"errors"
)

var (
	ErrAccountNotFound = errors.New("auth: conta não encontrada")
	ErrEmailNotFound   = errors.New("auth: e-mail não encontrado")
)

type AuthClient interface {
	GetAccountEmail(ctx context.Context, userID string) (string, error)
}
