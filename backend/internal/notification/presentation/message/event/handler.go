// Package event holds the notification domain's message input adapters. A handler
// here is structurally identical to an HTTP controller: it deserializes a transport
// payload (JSON bytes off a Kafka topic) into a request and calls the application
// service. It knows nothing about Kafka.
package event

import "minimart/internal/notification/application"

// Handler adapts notification Kafka messages to application use cases.
type Handler struct {
	svc *application.Service
}

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }
