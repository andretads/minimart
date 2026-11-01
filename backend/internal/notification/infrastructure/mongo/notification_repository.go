package mongo

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/notification/domain"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const notificationsCollectionName = "notifications"

type NotificationRepository struct {
	coll *mongo.Collection
}

var _ domain.NotificationRepository = (*NotificationRepository)(nil)

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{coll: db.Collection(notificationsCollectionName)}
}

func (r *NotificationRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "order_id", Value: 1}, {Key: "type", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "dedupe_key", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{{Key: "dedupe_key", Value: bson.D{{Key: "$type", Value: "string"}}}}),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return fmt.Errorf("ensure notification indexes: %w", err)
	}
	return nil
}

func (r *NotificationRepository) Save(ctx context.Context, n domain.Notification) error {
	if n.DedupeKey == "" {
		n.DedupeKey = n.OrderID + "/" + string(n.Type)
	}
	filter := bson.D{{Key: "dedupe_key", Value: n.DedupeKey}}
	update := bson.D{{Key: "$setOnInsert", Value: toNotificationDoc(n)}}
	if _, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("save notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (domain.Notification, error) {
	var doc notificationDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Notification{}, domain.ErrNotificationNotFound
	case err != nil:
		return domain.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *NotificationRepository) GetByIDForUser(ctx context.Context, id, userID string) (domain.Notification, error) {
	var doc notificationDoc
	err := r.coll.FindOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "user_id", Value: userID},
	}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Notification{}, domain.ErrNotificationNotFound
	case err != nil:
		return domain.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *NotificationRepository) GetByOrderIDAndType(ctx context.Context, orderID string, nType domain.Type) (domain.Notification, error) {
	var doc notificationDoc
	filter := bson.D{
		{Key: "order_id", Value: orderID},
		{Key: "type", Value: string(nType)},
	}
	err := r.coll.FindOne(ctx, filter).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Notification{}, domain.ErrNotificationNotFound
	case err != nil:
		return domain.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *NotificationRepository) GetByDedupeKey(ctx context.Context, key string) (domain.Notification, error) {
	var doc notificationDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "dedupe_key", Value: key}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Notification{}, domain.ErrNotificationNotFound
	case err != nil:
		return domain.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *NotificationRepository) ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]domain.Notification, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	cur, err := r.coll.Find(ctx,
		bson.D{{Key: "user_id", Value: userID}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetSkip(int64((page-1)*pageSize)).SetLimit(int64(pageSize)),
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []notificationDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	out := make([]domain.Notification, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
}

func (r *NotificationRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "user_id", Value: userID}})
	if err != nil {
		return 0, fmt.Errorf("count notifications: %w", err)
	}
	return int(n), nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	now := time.Now().UTC()
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "read_at", Value: now}}}},
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkUnread(ctx context.Context, id, userID string) error {
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "user_id", Value: userID}},
		bson.D{{Key: "$unset", Value: bson.D{{Key: "read_at", Value: ""}}}},
	)
	if err != nil {
		return fmt.Errorf("mark notification unread: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	_, err := r.coll.UpdateMany(ctx,
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "read_at", Value: now}}}},
	)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.coll.DeleteOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "user_id", Value: userID},
	})
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status domain.NotificationStatus, sentAt, deliveredAt *time.Time) error {
	set := bson.D{{Key: "status", Value: string(status)}}
	if sentAt != nil {
		set = append(set, bson.E{Key: "sent_at", Value: *sentAt})
	}
	if deliveredAt != nil {
		set = append(set, bson.E{Key: "delivered_at", Value: *deliveredAt})
	}
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: set}},
	)
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

type notificationDoc struct {
	ID               string         `bson:"_id"`
	Type             string         `bson:"type"`
	RecipientID      string         `bson:"recipient_id"`
	CustomerID       string         `bson:"customer_id,omitempty"`
	UserID           string         `bson:"user_id"`
	OrderID          string         `bson:"order_id"`
	DedupeKey        string         `bson:"dedupe_key"`
	TemplateID       string         `bson:"template_id"`
	Payload          map[string]any `bson:"payload,omitempty"`
	Priority         string         `bson:"priority"`
	Status           string         `bson:"status"`
	Subject          string         `bson:"subject"`
	Body             string         `bson:"body"`
	SentAt           time.Time      `bson:"sent_at"`
	CreatedAt        time.Time      `bson:"created_at"`
	DeliveredAt      *time.Time     `bson:"delivered_at,omitempty"`
	ReadAt           *time.Time     `bson:"read_at,omitempty"`
	SuppressedReason string         `bson:"suppressed_reason,omitempty"`
}

func toNotificationDoc(n domain.Notification) notificationDoc {
	status := n.Status
	if status == "" {
		status = domain.NotificationStatusAccepted
	}
	return notificationDoc{
		ID:               n.ID,
		Type:             string(n.Type),
		RecipientID:      n.RecipientID,
		CustomerID:       n.CustomerID,
		UserID:           n.UserID,
		OrderID:          n.OrderID,
		DedupeKey:        n.DedupeKey,
		TemplateID:       n.TemplateID,
		Payload:          n.Payload,
		Priority:         string(n.Priority),
		Status:           string(status),
		Subject:          n.Subject,
		Body:             n.Body,
		SentAt:           n.SentAt,
		CreatedAt:        n.CreatedAt,
		DeliveredAt:      n.DeliveredAt,
		ReadAt:           n.ReadAt,
		SuppressedReason: n.SuppressedReason,
	}
}

func (d notificationDoc) toDomain() domain.Notification {
	status := domain.NotificationStatus(d.Status)
	if status == "" {
		status = domain.NotificationStatusSent
	}
	priority := domain.Priority(d.Priority)
	if priority == "" {
		priority = domain.PriorityNormal
	}
	templateID := d.TemplateID
	if templateID == "" {
		templateID = domain.TemplateIDForType(domain.Type(d.Type))
	}
	recipientID := d.RecipientID
	if recipientID == "" {
		recipientID = d.UserID
	}
	return domain.Notification{
		ID:               d.ID,
		Type:             domain.Type(d.Type),
		RecipientID:      recipientID,
		CustomerID:       d.CustomerID,
		UserID:           d.UserID,
		OrderID:          d.OrderID,
		DedupeKey:        d.DedupeKey,
		TemplateID:       templateID,
		Payload:          d.Payload,
		Priority:         priority,
		Status:           status,
		Subject:          d.Subject,
		Body:             d.Body,
		SentAt:           d.SentAt,
		CreatedAt:        d.CreatedAt,
		DeliveredAt:      d.DeliveredAt,
		ReadAt:           d.ReadAt,
		SuppressedReason: d.SuppressedReason,
	}
}
