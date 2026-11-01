package http

import (
	"minimart/internal/auth/application"
	"minimart/pkg/auth"
	"time"
)

type Handlers struct {
	svc                  *application.Service
	tokens               *auth.TokenService
	cookieEncryption     *auth.CookieEncryption
	cookieConfig         auth.CookieConfig
	rememberMeRefreshTTL time.Duration
}

func NewHandlers(svc *application.Service, tokens *auth.TokenService, cookieEncryption *auth.CookieEncryption, cookieConfig auth.CookieConfig, rememberMeRefreshTTL time.Duration) *Handlers {
	return &Handlers{svc: svc, tokens: tokens, cookieEncryption: cookieEncryption, cookieConfig: cookieConfig, rememberMeRefreshTTL: rememberMeRefreshTTL}
}
