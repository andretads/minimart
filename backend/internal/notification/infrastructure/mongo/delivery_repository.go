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

const deliveriesCollectionName = "notification_deliveries"

type DeliveryRepository struct {
	coll *mongo.Collection
}

var _ domain.DeliveryRepository = (*DeliveryRepository)(nil)

func NewDeliveryRepository(db *mongo.Database) *DeliveryRepository {
	return &DeliveryRepository{coll: db.Collection(deliveriesCollectionName)}
}

func (r *DeliveryRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "notification_id", Value: 1}, {Key: "channel", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_attempt_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "recipient_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return fmt.Errorf("ensure delivery indexes: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) SaveDelivery(ctx context.Context, d domain.Delivery) error {
	filter := bson.D{{Key: "notification_id", Value: d.NotificationID}, {Key: "channel", Value: string(d.Channel)}}
	update := bson.D{{Key: "$setOnInsert", Value: toDeliveryDoc(d)}}
	if _, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("save delivery: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) GetDeliveryByID(ctx context.Context, id string) (domain.Delivery, error) {
	var doc deliveryDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Delivery{}, domain.ErrDeliveryNotFound
	case err != nil:
		return domain.Delivery{}, fmt.Errorf("get delivery: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *DeliveryRepository) GetByNotificationAndChannel(ctx context.Context, notificationID string, channel domain.Channel) (domain.Delivery, error) {
	var doc deliveryDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "notification_id", Value: notificationID}, {Key: "channel", Value: string(channel)}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Delivery{}, domain.ErrDeliveryNotFound
	case err != nil:
		return domain.Delivery{}, fmt.Errorf("get delivery: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *DeliveryRepository) ListByNotification(ctx context.Context, notificationID string) ([]domain.Delivery, error) {
	cur, err := r.coll.Find(ctx, bson.D{{Key: "notification_id", Value: notificationID}})
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []deliveryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode deliveries: %w", err)
	}
	out := make([]domain.Delivery, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
}

func (r *DeliveryRepository) ClaimDue(ctx context.Context, workerID string, limit int) ([]domain.Delivery, error) {
	if limit <= 0 {
		limit = 50
	}
	now := time.Now().UTC()
	filter := bson.D{
		{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{string(domain.DeliveryStatusPending), string(domain.DeliveryStatusRetryPending)}}}},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "next_attempt_at", Value: nil}},
			bson.D{{Key: "next_attempt_at", Value: bson.D{{Key: "$lte", Value: now}}}},
		}},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "locked_until", Value: nil}},
			bson.D{{Key: "locked_until", Value: bson.D{{Key: "$lte", Value: now}}}},
		}},
	}
	opts := options.Find().SetLimit(int64(limit))
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("claim deliveries: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []deliveryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode deliveries: %w", err)
	}
	out := make([]domain.Delivery, 0, len(docs))
	for _, doc := range docs {
		d := doc.toDomain()
		until := now.Add(30 * time.Second)
		d.Status = domain.DeliveryStatusProcessing
		d.LockedBy = workerID
		d.LockedUntil = &until
		d.UpdatedAt = now
		if _, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: d.ID}, {Key: "locked_by", Value: bson.D{{Key: "$in", Value: bson.A{nil, ""}}}}}, bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: string(d.Status)},
			{Key: "locked_by", Value: workerID},
			{Key: "locked_until", Value: until},
			{Key: "updated_at", Value: now},
		}}}); err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *DeliveryRepository) MarkSent(ctx context.Context, id, provider, providerMessageID string, attemptCount int, delivered bool, deliveredAt *time.Time) error {
	now := time.Now().UTC()
	status := string(domain.DeliveryStatusSent)
	set := bson.D{
		{Key: "attempt_count", Value: attemptCount},
		{Key: "provider", Value: provider},
		{Key: "provider_message_id", Value: providerMessageID},
		{Key: "last_attempt_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "locked_by", Value: ""},
		{Key: "locked_until", Value: nil},
	}
	if delivered {
		status = string(domain.DeliveryStatusDelivered)
		set = append(set, bson.E{Key: "delivered_at", Value: deliveredAt})
	}
	set = append(bson.D{{Key: "status", Value: status}}, set...)
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("mark delivery sent: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrDeliveryNotFound
	}
	return nil
}

func (r *DeliveryRepository) MarkRetry(ctx context.Context, id string, attemptCount int, nextAttemptAt time.Time, err error) error {
	now := time.Now().UTC()
	set := bson.D{
		{Key: "status", Value: string(domain.DeliveryStatusRetryPending)},
		{Key: "attempt_count", Value: attemptCount},
		{Key: "last_attempt_at", Value: now},
		{Key: "next_attempt_at", Value: nextAttemptAt},
		{Key: "updated_at", Value: now},
		{Key: "locked_by", Value: ""},
		{Key: "locked_until", Value: nil},
	}
	if err != nil {
		set = append(set, bson.E{Key: "error", Value: err.Error()})
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("mark delivery retry: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrDeliveryNotFound
	}
	return nil
}

func (r *DeliveryRepository) MarkFailed(ctx context.Context, id string, attemptCount int, err error) error {
	now := time.Now().UTC()
	set := bson.D{
		{Key: "status", Value: string(domain.DeliveryStatusFailed)},
		{Key: "attempt_count", Value: attemptCount},
		{Key: "last_attempt_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "locked_by", Value: ""},
		{Key: "locked_until", Value: nil},
	}
	if err != nil {
		set = append(set, bson.E{Key: "error", Value: err.Error()})
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("mark delivery failed: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrDeliveryNotFound
	}
	return nil
}

func (r *DeliveryRepository) ReleaseLease(ctx context.Context, id, workerID string) error {
	_, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "locked_by", Value: workerID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "locked_by", Value: ""}, {Key: "locked_until", Value: nil}}}},
	)
	if err != nil {
		return fmt.Errorf("release delivery lease: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) UpdateDeliveryStatus(ctx context.Context, id string, status domain.DeliveryStatus) error {
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(status)}, {Key: "updated_at", Value: time.Now().UTC()}}}},
	)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrDeliveryNotFound
	}
	return nil
}

type deliveryDoc struct {
	ID                string     `bson:"_id"`
	NotificationID    string     `bson:"notification_id"`
	RecipientID       string     `bson:"recipient_id"`
	Channel           string     `bson:"channel"`
	Provider          string     `bson:"provider"`
	ProviderMessageID string     `bson:"provider_message_id"`
	Status            string     `bson:"status"`
	AttemptCount      int        `bson:"attempt_count"`
	LastAttemptAt     *time.Time `bson:"last_attempt_at,omitempty"`
	NextAttemptAt     *time.Time `bson:"next_attempt_at,omitempty"`
	DeliveredAt       *time.Time `bson:"delivered_at,omitempty"`
	LockedBy          string     `bson:"locked_by,omitempty"`
	LockedUntil       *time.Time `bson:"locked_until,omitempty"`
	Error             string     `bson:"error,omitempty"`
	CreatedAt         time.Time  `bson:"created_at"`
	UpdatedAt         time.Time  `bson:"updated_at"`
}

func toDeliveryDoc(d domain.Delivery) deliveryDoc {
	status := d.Status
	if status == "" {
		status = domain.DeliveryStatusPending
	}
	return deliveryDoc{
		ID:                d.ID,
		NotificationID:    d.NotificationID,
		RecipientID:       d.RecipientID,
		Channel:           string(d.Channel),
		Provider:          d.Provider,
		ProviderMessageID: d.ProviderMessageID,
		Status:            string(status),
		AttemptCount:      d.AttemptCount,
		LastAttemptAt:     d.LastAttemptAt,
		NextAttemptAt:     d.NextAttemptAt,
		DeliveredAt:       d.DeliveredAt,
		LockedBy:          d.LockedBy,
		LockedUntil:       d.LockedUntil,
		Error:             d.Error,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

func (d deliveryDoc) toDomain() domain.Delivery {
	status := domain.DeliveryStatus(d.Status)
	if status == "" {
		status = domain.DeliveryStatusPending
	}
	return domain.Delivery{
		ID:                d.ID,
		NotificationID:    d.NotificationID,
		RecipientID:       d.RecipientID,
		Channel:           domain.Channel(d.Channel),
		Provider:          d.Provider,
		ProviderMessageID: d.ProviderMessageID,
		Status:            status,
		AttemptCount:      d.AttemptCount,
		LastAttemptAt:     d.LastAttemptAt,
		NextAttemptAt:     d.NextAttemptAt,
		DeliveredAt:       d.DeliveredAt,
		LockedBy:          d.LockedBy,
		LockedUntil:       d.LockedUntil,
		Error:             d.Error,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}
