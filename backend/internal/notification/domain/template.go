package domain

import (
	"fmt"
	"strings"
)

type RenderedMessage struct {
	Subject string
	Text    string
	HTML    string
	Title   string
	Body    string
}

var templateIDs = map[string]bool{
	"order_confirmation_v1": true,
	"order_cancelled_v1":    true,
	"payment_succeeded_v1":  true,
	"payment_failed_v1":     true,
	"shipment_created_v1":   true,
	"shipment_shipped_v1":   true,
	"shipment_delivered_v1": true,
}

func TemplateIDForType(t Type) string {
	switch t {
	case TypeOrderConfirmation:
		return "order_confirmation_v1"
	case TypeOrderCancellation:
		return "order_cancelled_v1"
	case TypePaymentSucceeded:
		return "payment_succeeded_v1"
	case TypePaymentFailed:
		return "payment_failed_v1"
	case TypeShipmentCreated:
		return "shipment_created_v1"
	case TypeShipmentShipped:
		return "shipment_shipped_v1"
	case TypeShipmentDelivered:
		return "shipment_delivered_v1"
	}
	return ""
}

func PriorityForType(t Type) Priority {
	switch t {
	case TypePaymentFailed:
		return PriorityHigh
	case TypeShipmentShipped, TypeShipmentDelivered:
		return PriorityNormal
	default:
		return PriorityNormal
	}
}

func CategoryForType(t Type) Category {
	return CategoryTransactional
}

func RenderTemplate(templateID string, channel Channel, subject, body string) (RenderedMessage, error) {
	if templateID == "" || !templateIDs[templateID] {
		return RenderedMessage{}, ErrTemplateNotFound
	}
	switch channel {
	case ChannelEmail:
		htmlBody := strings.ReplaceAll(strings.TrimSpace(body), "\n", "<br>")
		return RenderedMessage{
			Subject: subject,
			Text:    body,
			HTML:    fmt.Sprintf("<pre>%s</pre>", htmlBody),
		}, nil
	case ChannelSMS:
		text := body
		if len(text) > 160 {
			text = text[:157] + "..."
		}
		return RenderedMessage{Text: text}, nil
	case ChannelPush:
		title := subject
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		text := body
		if len(text) > 120 {
			text = text[:117] + "..."
		}
		return RenderedMessage{Title: title, Body: text}, nil
	}
	return RenderedMessage{}, ErrInvalidChannel
}
