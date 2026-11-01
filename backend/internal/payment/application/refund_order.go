package application

import (
	"context"
	"errors"
	"minimart/internal/payment/domain"
)

// RefundOrderPayments refunds every captured payment for an order. It is the
// handler for the order-cancelled event. Delivery is at-least-once, so a
// payment that was already refunded - or whose refund lost a race - is a
// no-op, not an error.
func (s *Service) RefundOrderPayments(ctx context.Context, orderID string) error {
	payments, err := s.repo.ListSucceededByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	for _, p := range payments {
		if _, err := s.refund(ctx, p, refundReasonOrderCancelled); err != nil {
			if errors.Is(err, domain.ErrPaymentNotRefundable) {
				continue
			}
			return err
		}
	}
	return nil
}
