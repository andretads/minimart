package wishlisthttp

import "minimart/internal/wishlist/application"

type Handlers struct {
	svc *application.Service
}

func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
