package application

import (
	"context"
	"errors"
	"log/slog"
	"minimart/internal/cart/domain"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"

	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// CheckoutCartRequest is the input to CheckoutCart.
type CheckoutCartRequest struct {
	UserID          string
	IdempotencyKey  string
	ShippingQuoteID string
	AddressID       string
}

// checkoutPlan is the resolved checkout input: the quote the customer chose,
// the name of the method it names, and the identifiers the order placement
// needs. It is resolved once and reused by both the first attempt and a retry,
// so the order always gets the same shipping snapshot.
type checkoutPlan struct {
	userID          string
	idempotencyKey  string
	shippingQuoteID string
	quote           shippingclient.ShippingQuote
	methodName      string
}

// placeOrderRequest builds the order placement request for the cart's items.
func (p checkoutPlan) placeOrderRequest(items []orderclient.LineItem) orderclient.PlaceOrderRequest {
	return orderclient.PlaceOrderRequest{
		UserID:              p.userID,
		IdempotencyKey:      p.idempotencyKey,
		Items:               items,
		ShippingQuoteID:     p.shippingQuoteID,
		ShippingMethodID:    p.quote.MethodID,
		ShippingMethodName:  p.methodName,
		DestinationSnapshot: p.quote.DestinationSnapshot,
		ShippingCostCents:   p.quote.ShippingCostCents,
	}
}

// shippingMethodName resolves the display name of the quote's method for the
// order's shipping snapshot. The name is cosmetic - the order only needs it to
// show the customer how their parcel travels - so a lookup failure degrades to
// an empty name instead of failing the checkout.
func (s *Service) shippingMethodName(ctx context.Context, methodID string) string {
	if methodID == "" {
		return ""
	}
	method, err := s.modules.GetMethod(ctx, methodID)
	if err != nil {
		slog.WarnContext(ctx, "failed to resolve shipping method name", "method_id", methodID, "error", err)
		return ""
	}
	return method.Name
}

// CheckoutCart creates an order from the cart items.
func (s *Service) CheckoutCart(ctx context.Context, req CheckoutCartRequest) (orderclient.Order, error) {
	ord, err := s.checkoutCart(ctx, req)
	if err != nil {
		s.m.IncCartCheckout(metrics.StatusError)
		return orderclient.Order{}, err
	}
	s.m.IncCartCheckout(metrics.StatusSuccess)
	return ord, nil
}

func (s *Service) checkoutCart(ctx context.Context, req CheckoutCartRequest) (orderclient.Order, error) {
	if req.UserID == "" {
		return orderclient.Order{}, ErrUserIdRequired
	}
	if req.IdempotencyKey == "" {
		return orderclient.Order{}, ErrIdempotencyKeyRequired
	}
	if req.ShippingQuoteID == "" {
		return orderclient.Order{}, ErrShippingQuoteRequired
	}
	if req.AddressID == "" {
		return orderclient.Order{}, ErrAddressRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return orderclient.Order{}, err
	}

	// Validate the shipping quote belongs to this customer and is valid
	quote, err := s.modules.GetQuote(ctx, shippingclient.GetQuoteRequest{ID: req.ShippingQuoteID})
	if err != nil {
		return orderclient.Order{}, err
	}
	if quote.CustomerID != cust.ID {
		return orderclient.Order{}, ErrQuoteOwnershipMismatch
	}
	_, err = s.modules.ValidateQuote(ctx, shippingclient.ValidateQuoteRequest{ID: req.ShippingQuoteID, CustomerID: cust.ID})
	if err != nil {
		return orderclient.Order{}, err
	}

	plan := checkoutPlan{
		userID:          req.UserID,
		idempotencyKey:  req.IdempotencyKey,
		shippingQuoteID: req.ShippingQuoteID,
		quote:           quote,
		methodName:      s.shippingMethodName(ctx, quote.MethodID),
	}

	// First try to get the cart (active or checkout_pending)
	cart, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		if errors.Is(err, domain.ErrCartNotFound) {
			// Cart not found as active/checkout_pending - check if it exists as consumed
			cart, err = s.repo.GetByCustomerIDIncludingConsumed(ctx, cust.ID)
			if err != nil {
				if errors.Is(err, domain.ErrCartNotFound) {
					return orderclient.Order{}, domain.ErrCartEmpty
				}
				return orderclient.Order{}, err
			}
			// Cart exists but is consumed - retry checkout
			return s.retryCheckout(ctx, cart, plan)
		}
		return orderclient.Order{}, err
	}
	if cart.Status == domain.StatusConsumed {
		// Cart already consumed, try to get existing order by retrying PlaceOrder
		return s.retryCheckout(ctx, cart, plan)
	}
	if cart.Status == domain.StatusCheckoutPending {
		if cart.CheckoutKey == req.IdempotencyKey {
			return s.retryCheckout(ctx, cart, plan)
		}
		return orderclient.Order{}, ErrCartCheckoutInProgress
	}
	if len(cart.Items) == 0 {
		return orderclient.Order{}, domain.ErrCartEmpty
	}
	beginErr := cart.BeginCheckout(req.IdempotencyKey)
	if beginErr != nil {
		return orderclient.Order{}, beginErr
	}
	updateErr := s.repo.Update(ctx, cart, cart.Version)
	if updateErr != nil {
		return orderclient.Order{}, updateErr
	}
	// Re-fetch cart to get updated version after checkout began
	cart, err = s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		return orderclient.Order{}, err
	}

	// The quote was priced against a subtotal the customer supplied. The cart
	// knows the real subtotal, so the quote is priced again against it before
	// the order is placed.
	priced, err := s.modules.PriceQuote(ctx, shippingclient.PriceQuoteRequest{
		ID:            req.ShippingQuoteID,
		CustomerID:    cust.ID,
		SubtotalCents: cart.EstimatedTotalCents(),
	})
	if err != nil {
		s.releaseCheckout(ctx, &cart)
		return orderclient.Order{}, err
	}
	plan.quote = priced

	return s.executeCheckout(ctx, cart, plan)
}

// executeCheckout places the order for a cart that just began checkout and
// marks the cart consumed. Placement failures that the customer can fix
// release the checkout lock, returning the cart to active.
func (s *Service) executeCheckout(ctx context.Context, cart domain.Cart, plan checkoutPlan) (orderclient.Order, error) {
	return s.placeOrderAndConsume(ctx, cart, plan, func(err error) {
		s.handleCheckoutFailure(ctx, cart, err)
	})
}

// retryCheckout replays the order placement for a cart that is already
// checkout_pending or consumed.
//
// The same idempotency key is used, so the order's own idempotency layer
// collapses the retry to the original order and no duplicate is placed. There
// is no checkout lock to release, so placement failures are returned as-is.
func (s *Service) retryCheckout(ctx context.Context, cart domain.Cart, plan checkoutPlan) (orderclient.Order, error) {
	return s.placeOrderAndConsume(ctx, cart, plan, nil)
}

// placeOrderAndConsume places the order for the cart's items and, unless the
// cart was already consumed, marks it consumed.
//
// onPlaceError handles a failed placement; the first attempt passes a
// releaser for business errors (see handleCheckoutFailure), while a retry
// passes nil.
func (s *Service) placeOrderAndConsume(ctx context.Context, cart domain.Cart, plan checkoutPlan, onPlaceError func(err error)) (orderclient.Order, error) {
	items := make([]orderclient.LineItem, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = orderclient.LineItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	ord, err := s.modules.PlaceOrder(ctx, plan.placeOrderRequest(items))
	if err != nil {
		if onPlaceError != nil {
			onPlaceError(err)
		}
		return orderclient.Order{}, err
	}
	// A consumed cart is a replay of an earlier checkout: the order above was
	// deduplicated by the idempotency key and the cart needs no update.
	if cart.Status == domain.StatusConsumed {
		return ord, nil
	}
	if err := cart.MarkConsumed(ord.ID); err != nil {
		return orderclient.Order{}, err
	}
	if err := s.repo.Update(ctx, cart, cart.Version); err != nil {
		s.m.IncCartConsumed(metrics.StatusError)
		//nolint:nilerr // Order was placed successfully; cart mark-consumed failure is non-fatal
		return ord, nil
	}
	s.m.IncCartConsumed(metrics.StatusSuccess)
	return ord, nil
}

// handleCheckoutFailure releases the checkout lock when the placement
// failure is a business error the customer can fix - stale stock, bad
// quantities, a conflicting idempotency key - so the cart returns to active
// instead of staying stuck checkout_pending. Infrastructure failures keep the
// lock: the customer can retry checkout with the same idempotency key.
func (s *Service) handleCheckoutFailure(ctx context.Context, cart domain.Cart, err error) {
	if errors.Is(err, orderclient.ErrInsufficientStock) ||
		errors.Is(err, orderclient.ErrProductNotFound) ||
		errors.Is(err, orderclient.ErrNoItems) ||
		errors.Is(err, orderclient.ErrInvalidQuantity) ||
		errors.Is(err, orderclient.ErrIdempotencyConflict) {
		s.releaseCheckout(ctx, &cart)
	}
}

func (s *Service) releaseCheckout(ctx context.Context, cart *domain.Cart) {
	if err := cart.ReleaseCheckout(); err != nil {
		logger.FromContext(ctx).Warn("failed to release cart checkout",
			"error", err, "cart_id", cart.ID, "status", string(cart.Status))
		return
	}
	if err := s.repo.Update(ctx, *cart, cart.Version); err != nil {
		logger.FromContext(ctx).Warn("failed to persist cart checkout release",
			"error", err, "cart_id", cart.ID)
	}
}
