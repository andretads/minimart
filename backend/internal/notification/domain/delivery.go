package domain

import "time"

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

var AllChannels = []Channel{ChannelEmail, ChannelSMS, ChannelPush}

type DeliveryStatus string

const (
	DeliveryStatusPending      DeliveryStatus = "PENDING"
	DeliveryStatusProcessing   DeliveryStatus = "PROCESSING"
	DeliveryStatusSent         DeliveryStatus = "SENT"
	DeliveryStatusDelivered    DeliveryStatus = "DELIVERED"
	DeliveryStatusRetryPending DeliveryStatus = "RETRY_PENDING"
	DeliveryStatusFailed       DeliveryStatus = "FAILED"
)

type NotificationStatus string

const (
	NotificationStatusAccepted     NotificationStatus = "ACCEPTED"
	NotificationStatusPending      NotificationStatus = "PENDING"
	NotificationStatusProcessing   NotificationStatus = "PROCESSING"
	NotificationStatusSent         NotificationStatus = "SENT"
	NotificationStatusDelivered    NotificationStatus = "DELIVERED"
	NotificationStatusRetryPending NotificationStatus = "RETRY_PENDING"
	NotificationStatusFailed       NotificationStatus = "FAILED"
	NotificationStatusSuppressed   NotificationStatus = "SUPPRESSED"
)

type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

type Category string

const (
	CategoryTransactional Category = "transactional"
	CategoryMarketing     Category = "marketing"
	CategoryCritical      Category = "critical"
)

type Delivery struct {
	ID                string
	NotificationID    string
	RecipientID       string
	Channel           Channel
	Provider          string
	ProviderMessageID string
	Status            DeliveryStatus
	AttemptCount      int
	LastAttemptAt     *time.Time
	NextAttemptAt     *time.Time
	DeliveredAt       *time.Time
	LockedBy          string
	LockedUntil       *time.Time
	Error             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewDelivery(id, notificationID, recipientID string, channel Channel, provider string) (Delivery, error) {
	if id == "" {
		return Delivery{}, ErrEmptyID
	}
	if notificationID == "" {
		return Delivery{}, ErrEmptyNotificationID
	}
	if recipientID == "" {
		return Delivery{}, ErrEmptyUserID
	}
	if !validChannel(channel) {
		return Delivery{}, ErrInvalidChannel
	}
	now := time.Now().UTC()
	return Delivery{
		ID:             id,
		NotificationID: notificationID,
		RecipientID:    recipientID,
		Channel:        channel,
		Provider:       provider,
		Status:         DeliveryStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func validChannel(c Channel) bool {
	switch c {
	case ChannelEmail, ChannelSMS, ChannelPush:
		return true
	}
	return false
}

func (d Delivery) IsTerminal() bool {
	switch d.Status {
	case DeliveryStatusSent, DeliveryStatusDelivered, DeliveryStatusFailed:
		return true
	}
	return false
}
