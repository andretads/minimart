package wishlisthttp

import (
	catalogclient "minimart/internal/catalog/presentation/module/client"
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/response"
	"net/http"
	"time"
)

// errorStatuses maps wishlist errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrWishlistNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrEmptyCustomerID, Status: http.StatusBadRequest},
	{Err: domain.ErrProductAlreadyInWishlist, Status: http.StatusConflict},
	{Err: domain.ErrProductNotInWishlist, Status: http.StatusNotFound},
	{Err: domain.ErrWishlistAlreadyExists, Status: http.StatusConflict},
	{Err: application.ErrUserIDRequired, Status: http.StatusBadRequest},
	{Err: application.ErrProductIDRequired, Status: http.StatusBadRequest},
	{Err: catalogclient.ErrProductNotFound, Status: http.StatusNotFound},
}

func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}

func toWishlistResponse(w domain.Wishlist) wishlistResponse {
	items := make([]wishlistItemResponse, len(w.Items))
	for i, item := range w.Items {
		items[i] = wishlistItemResponse{
			ProductID: item.ProductID,
			AddedAt:   item.AddedAt.UTC().Format(time.RFC3339),
		}
	}
	return wishlistResponse{
		ID:         w.ID,
		CustomerID: w.CustomerID,
		Items:      items,
		CreatedAt:  w.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  w.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
