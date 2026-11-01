package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Type string

const (
	TypeOrderConfirmation Type = "order_confirmation"
	TypeOrderCancellation Type = "order_cancelled"
	TypePaymentSucceeded  Type = "payment_succeeded"
	TypePaymentFailed     Type = "payment_failed"
	TypeShipmentCreated   Type = "shipment_created"
	TypeShipmentShipped   Type = "shipment_shipped"
	TypeShipmentDelivered Type = "shipment_delivered"
)

type Notification struct {
	ID               string
	Type             Type
	RecipientID      string
	CustomerID       string
	UserID           string
	OrderID          string
	DedupeKey        string
	TemplateID       string
	Payload          map[string]any
	Priority         Priority
	Status           NotificationStatus
	Subject          string
	Body             string
	SentAt           time.Time
	CreatedAt        time.Time
	DeliveredAt      *time.Time
	ReadAt           *time.Time
	SuppressedReason string
}

func newNotification(id string, typ Type, userID, orderID, subject, body string) (Notification, error) {
	if id == "" {
		return Notification{}, ErrEmptyID
	}
	if orderID == "" {
		return Notification{}, ErrEmptyOrderID
	}
	if userID == "" {
		return Notification{}, ErrEmptyUserID
	}
	now := time.Now().UTC()
	return Notification{
		ID:          id,
		Type:        typ,
		RecipientID: userID,
		UserID:      userID,
		OrderID:     orderID,
		DedupeKey:   orderID + "/" + string(typ),
		TemplateID:  TemplateIDForType(typ),
		Priority:    PriorityForType(typ),
		Status:      NotificationStatusAccepted,
		Subject:     subject,
		Body:        body,
		SentAt:      now,
		CreatedAt:   now,
	}, nil
}

func NewOrderConfirmation(id, orderID, userID string, totalCents int64) (Notification, error) {
	if totalCents <= 0 {
		return Notification{}, ErrNonPositiveTotal
	}
	return newNotification(id, TypeOrderConfirmation, userID, orderID,
		fmt.Sprintf("Pedido %s confirmado", orderID),
		fmt.Sprintf("Obrigado pela sua compra! O pedido %s no valor de %s foi confirmado.",
			orderID, formatMoney(totalCents)))
}

func NewOrderCancellation(id, orderID, userID string, totalCents int64) (Notification, error) {
	if totalCents <= 0 {
		return Notification{}, ErrNonPositiveTotal
	}
	return newNotification(id, TypeOrderCancellation, userID, orderID,
		fmt.Sprintf("Pedido %s cancelado", orderID),
		fmt.Sprintf("O pedido %s no valor de %s foi cancelado.", orderID, formatMoney(totalCents)))
}

func NewPaymentSucceeded(id, orderID, userID string, amountCents int64) (Notification, error) {
	if amountCents <= 0 {
		return Notification{}, ErrNonPositiveTotal
	}
	return newNotification(id, TypePaymentSucceeded, userID, orderID,
		fmt.Sprintf("Pagamento do pedido %s recebido", orderID),
		fmt.Sprintf("Recebemos o seu pagamento de %s referente ao pedido %s.", formatMoney(amountCents), orderID))
}

func NewPaymentFailed(id, orderID, userID string, amountCents int64, reason string) (Notification, error) {
	if amountCents <= 0 {
		return Notification{}, ErrNonPositiveTotal
	}
	body := fmt.Sprintf("O seu pagamento de %s referente ao pedido %s foi recusado.", formatMoney(amountCents), orderID)
	if reason != "" {
		body = fmt.Sprintf("%s Motivo: %s.", body, reason)
	}
	return newNotification(id, TypePaymentFailed, userID, orderID,
		fmt.Sprintf("Pagamento do pedido %s recusado", orderID), body)
}

func NewShipmentCreated(id, orderID, userID string, shippingCostCents int64) (Notification, error) {
	if shippingCostCents < 0 {
		return Notification{}, ErrNegativeShippingCost
	}
	return newNotification(id, TypeShipmentCreated, userID, orderID,
		"Envio criado para o pedido "+orderID,
		fmt.Sprintf("Criamos um envio para o pedido %s. Custo de envio: %s.", orderID, formatMoney(shippingCostCents)))
}

func NewShipmentShipped(id, orderID, userID, trackingCode, carrier string) (Notification, error) {
	body := fmt.Sprintf("O seu envio do pedido %s está a caminho.", orderID)
	if carrier != "" || trackingCode != "" {
		body = fmt.Sprintf("%s Transportadora: %s. Código de rastreio: %s.", body, carrier, trackingCode)
	}
	return newNotification(id, TypeShipmentShipped, userID, orderID,
		fmt.Sprintf("Envio do pedido %s despachado", orderID), body)
}

func NewShipmentDelivered(id, orderID, userID string) (Notification, error) {
	return newNotification(id, TypeShipmentDelivered, userID, orderID,
		fmt.Sprintf("Envio do pedido %s entregue", orderID),
		fmt.Sprintf("O seu envio do pedido %s foi entregue.", orderID))
}

func formatMoney(cents int64) string {
	reais := cents / 100
	centavos := cents % 100
	if centavos < 0 {
		centavos = -centavos
	}
	// pt-BR grouping: thousands separated by '.', decimal by ','.
	sign := ""
	if cents < 0 {
		sign = "-"
	}
	integerStr := formatThousands(reais)
	return fmt.Sprintf("%sR$ %s,%02d", sign, integerStr, centavos)
}

func formatThousands(n int64) string {
	if n < 0 {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte('.')
		}
		result.WriteRune(c)
	}
	return result.String()
}
