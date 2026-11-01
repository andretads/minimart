package domain

import "errors"

var (
	ErrNotificationNotFound    = errors.New("notificação não encontrada")
	ErrPreferencesNotFound     = errors.New("notification preferences not found")
	ErrEmptyID                 = errors.New("id da notificação não pode ser vazio")
	ErrEmptyUserID             = errors.New("id do usuário da notificação não pode ser vazio")
	ErrEmptyOrderID            = errors.New("id do pedido da notificação não pode ser vazio")
	ErrEmptyNotificationID     = errors.New("notification id cannot be empty")
	ErrNonPositiveTotal        = errors.New("total da notificação deve ser maior que zero")
	ErrNegativeShippingCost    = errors.New("notification shipping cost must not be negative")
	ErrRecipientNotFound       = errors.New("notification: recipient not found")
	ErrEmptyRecipient          = errors.New("notification: recipient email is empty")
	ErrInvalidChannel          = errors.New("notification: invalid channel")
	ErrInvalidDeliveryStatus   = errors.New("notification: invalid delivery status")
	ErrTemplateNotFound        = errors.New("notification: template not found")
	ErrDeliveryNotFound        = errors.New("notification: delivery not found")
	ErrDeliveryAlreadyTerminal = errors.New("notification: delivery already terminal")
)
