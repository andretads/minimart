package carthttp

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/response"
	"net/http"
	"time"
)

// errorStatuses maps cart errors and the contract errors checkout can surface
// to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrCartNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrEmptyCustomerID, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusBadRequest},
	{Err: domain.ErrCartItemNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrCartNotActive, Status: http.StatusConflict},
	{Err: domain.ErrCartEmpty, Status: http.StatusBadRequest},
	{Err: domain.ErrCartCheckoutPending, Status: http.StatusConflict},
	{Err: domain.ErrCartConsumed, Status: http.StatusConflict},
	{Err: domain.ErrCartOptimisticLock, Status: http.StatusConflict},
	{Err: domain.ErrCartStateChanged, Status: http.StatusConflict},
	{Err: application.ErrCartCheckoutInProgress, Status: http.StatusConflict},
	{Err: application.ErrUserIdRequired, Status: http.StatusBadRequest},
	{Err: application.ErrIdempotencyKeyRequired, Status: http.StatusBadRequest},
	{Err: application.ErrShippingQuoteRequired, Status: http.StatusBadRequest},
	{Err: application.ErrAddressRequired, Status: http.StatusBadRequest},
	{Err: shippingclient.ErrInvalidQuote, Status: http.StatusBadRequest},
	{Err: shippingclient.ErrQuoteNotFound, Status: http.StatusNotFound},
	{Err: shippingclient.ErrQuoteExpired, Status: http.StatusConflict},
	{Err: shippingclient.ErrQuoteConsumed, Status: http.StatusConflict},
	{Err: application.ErrQuoteOwnershipMismatch, Status: http.StatusForbidden},
	{Err: application.ErrQuoteDestinationMismatch, Status: http.StatusForbidden},
	{Err: orderclient.ErrInsufficientStock, Status: http.StatusConflict},
	{Err: catalogclient.ErrProductNotFound, Status: http.StatusBadRequest},
	{Err: orderclient.ErrProductNotFound, Status: http.StatusBadRequest},
	{Err: orderclient.ErrIdempotencyConflict, Status: http.StatusConflict},
	{Err: orderclient.ErrOrderInProgress, Status: http.StatusConflict},
	{Err: orderclient.ErrNoItems, Status: http.StatusBadRequest},
	{Err: orderclient.ErrInvalidQuantity, Status: http.StatusBadRequest},
	{Err: orderclient.ErrIdempotencyKeyRequired, Status: http.StatusBadRequest},
}

func respondDomainError(w http.ResponseWriter, err error) {
	response.DomainError(w, err, errorStatuses, "erro interno")
}

func toCartResponse(cart domain.Cart) cartResponse {
	items := make([]cartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = cartItemResponse{
			ProductID:              item.ProductID,
			ProductNameSnapshot:    item.ProductNameSnapshot,
			UnitPriceSnapshotCents: item.UnitPriceSnapshotCents,
			Quantity:               item.Quantity,
			LineTotalCents:         item.UnitPriceSnapshotCents * int64(item.Quantity),
		}
	}
	return cartResponse{
		ID:                  cart.ID,
		CustomerID:          cart.CustomerID,
		Status:              string(cart.Status),
		Items:               items,
		EstimatedTotalCents: cart.EstimatedTotalCents(),
		CreatedAt:           cart.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           cart.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toCheckoutResponse(ord orderclient.Order) checkoutResponse {
	return checkoutResponse{
		OrderID:    ord.ID,
		UserID:     ord.UserID,
		CustomerID: ord.CustomerID,
		Status:     ord.Status,
		TotalCents: ord.TotalCents,
		CreatedAt:  ord.CreatedAt.UTC().Format(time.RFC3339),
	}
}
