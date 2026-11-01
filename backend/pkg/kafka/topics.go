package kafka

// Subscription identifies a consumer group subscribed to a topic.
type Subscription struct {
	Topic Topic
	Group string
}

type Topic string

func (t Topic) String() string { return string(t) }

func DLQTopic(source Topic) Topic {
	return Topic(source.String() + ".dlq")
}

const (
	OrderPlacedTopic    Topic = "order-placed"
	OrderCancelledTopic Topic = "order-cancelled"

	OrderStockReleaseTopic Topic = "order-stock-release"

	OrderPaidTopic Topic = "order-paid"

	PaymentInitiatedTopic Topic = "payment-initiated"
	PaymentSucceededTopic Topic = "payment-succeeded"
	PaymentFailedTopic    Topic = "payment-failed"
	PaymentUnknownTopic   Topic = "payment-unknown"
	PaymentRefundedTopic  Topic = "payment-refunded"

	ReviewCreatedTopic Topic = "review-created"

	ReviewModeratedTopic Topic = "review-moderated"

	ShipmentCreatedTopic Topic = "shipment-created"

	ShipmentShippedTopic Topic = "shipment-shipped"

	ShipmentDeliveredTopic Topic = "shipment-delivered"

	NotificationEmailTopic Topic = "notification-email"

	NotificationSMSTopic Topic = "notification-sms"

	NotificationPushTopic Topic = "notification-push"
)

func AllTopics() []Topic {
	return []Topic{
		OrderPlacedTopic,
		OrderCancelledTopic,
		OrderStockReleaseTopic,
		OrderPaidTopic,
		PaymentInitiatedTopic,
		PaymentSucceededTopic,
		PaymentFailedTopic,
		PaymentUnknownTopic,
		PaymentRefundedTopic,
		ReviewCreatedTopic,
		ReviewModeratedTopic,
		ShipmentCreatedTopic,
		ShipmentShippedTopic,
		ShipmentDeliveredTopic,
		NotificationEmailTopic,
		NotificationSMSTopic,
		NotificationPushTopic,
	}
}

const (
	OrderConfirmationConsumer = "order-confirmation"

	NotificationConsumer = "notification"

	OrderPaymentConsumer = "order-payment"

	PaymentRefundConsumer = "payment-refund"

	CatalogStockConsumer = "catalog-stock"

	ReviewModerationConsumer = "review-moderation"

	ReviewRatingAggregatorConsumer = "review-rating-aggregator"

	ShippingShipmentConsumer = "shipping-shipment-creation"

	NotificationShipmentConsumer = "notification-shipment"

	NotificationEmailConsumer = "notification-email"

	NotificationSMSConsumer = "notification-sms"

	NotificationPushConsumer = "notification-push"
)
