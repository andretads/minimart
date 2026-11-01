// Package event holds the catalog domain's message input adapters. A handler here
// is structurally identical to an HTTP controller: it deserializes a transport
// payload (JSON bytes off a Kafka topic) into a call on the application service.
// It knows nothing about Kafka.
package event

import "minimart/internal/catalog/application"

// Handler adapts catalog Kafka messages to application use cases.
type Handler struct {
	svc *application.Service
}

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }
