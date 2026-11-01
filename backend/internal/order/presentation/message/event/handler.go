// Package event holds the order domain's own message input adapters. Consuming an
// event your own domain published is a normal pattern - here it stands in for a
// follow-up step like starting fulfilment.
package event

import "minimart/internal/order/application"

// Handler adapts order Kafka messages to application use cases.
type Handler struct {
	svc *application.Service
}

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }
