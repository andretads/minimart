package domain

import (
	"context"
	"time"
)

type NotificationRepository interface {
	Save(ctx context.Context, n Notification) error
	GetByID(ctx context.Context, id string) (Notification, error)
	GetByIDForUser(ctx context.Context, id, userID string) (Notification, error)
	GetByOrderIDAndType(ctx context.Context, orderID string, nType Type) (Notification, error)
	GetByDedupeKey(ctx context.Context, key string) (Notification, error)
	ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]Notification, error)
	CountByUserID(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkUnread(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	Delete(ctx context.Context, id, userID string) error
	UpdateStatus(ctx context.Context, id string, status NotificationStatus, sentAt, deliveredAt *time.Time) error
}

type DeliveryRepository interface {
	SaveDelivery(ctx context.Context, d Delivery) error
	GetDeliveryByID(ctx context.Context, id string) (Delivery, error)
	GetByNotificationAndChannel(ctx context.Context, notificationID string, channel Channel) (Delivery, error)
	ListByNotification(ctx context.Context, notificationID string) ([]Delivery, error)
	ClaimDue(ctx context.Context, workerID string, limit int) ([]Delivery, error)
	MarkSent(ctx context.Context, id, provider, providerMessageID string, attemptCount int, delivered bool, deliveredAt *time.Time) error
	MarkRetry(ctx context.Context, id string, attemptCount int, nextAttemptAt time.Time, err error) error
	MarkFailed(ctx context.Context, id string, attemptCount int, err error) error
	ReleaseLease(ctx context.Context, id, workerID string) error
	UpdateDeliveryStatus(ctx context.Context, id string, status DeliveryStatus) error
}

type PreferencesRepository interface {
	GetByUserID(ctx context.Context, userID string) (Preferences, error)
	Upsert(ctx context.Context, p Preferences) error
}
