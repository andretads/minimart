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

const preferencesCollectionName = "notification_preferences"

type PreferencesRepository struct {
	coll *mongo.Collection
}

var _ domain.PreferencesRepository = (*PreferencesRepository)(nil)

func NewPreferencesRepository(db *mongo.Database) *PreferencesRepository {
	return &PreferencesRepository{coll: db.Collection(preferencesCollectionName)}
}

func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID string) (domain.Preferences, error) {
	var doc preferencesDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Preferences{}, domain.ErrPreferencesNotFound
	case err != nil:
		return domain.Preferences{}, fmt.Errorf("get notification preferences: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *PreferencesRepository) Upsert(ctx context.Context, p domain.Preferences) error {
	doc := toPreferencesDoc(p)
	_, err := r.coll.ReplaceOne(ctx,
		bson.D{{Key: "_id", Value: p.UserID}},
		doc,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("save notification preferences: %w", err)
	}
	return nil
}

type preferencesDoc struct {
	ID               string `bson:"_id"`
	EmailEnabled     bool   `bson:"email_enabled"`
	SMSEnabled       bool   `bson:"sms_enabled"`
	PushEnabled      bool   `bson:"push_enabled"`
	MarketingEnabled bool   `bson:"marketing_enabled"`

	OrderUpdates   *bool     `bson:"order_updates,omitempty"`
	Promotions     *bool     `bson:"promotions,omitempty"`
	SecurityAlerts *bool     `bson:"security_alerts,omitempty"`
	Newsletter     *bool     `bson:"newsletter,omitempty"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

func toPreferencesDoc(p domain.Preferences) preferencesDoc {
	email := p.EmailEnabled
	sms := p.SMSEnabled
	push := p.PushEnabled
	marketing := p.MarketingEnabled
	return preferencesDoc{
		ID:               p.UserID,
		EmailEnabled:     email,
		SMSEnabled:       sms,
		PushEnabled:      push,
		MarketingEnabled: marketing,
		UpdatedAt:        p.UpdatedAt,
	}
}

func (d preferencesDoc) toDomain() domain.Preferences {
	email := d.EmailEnabled
	sms := d.SMSEnabled
	push := d.PushEnabled
	marketing := d.MarketingEnabled

	if !email && !sms && !push && !marketing {
		if d.OrderUpdates != nil && *d.OrderUpdates {
			email = true
		}
		if d.SecurityAlerts != nil && *d.SecurityAlerts {
			email = true
		}
		if d.Promotions != nil && *d.Promotions {
			marketing = true
		}
		if d.Newsletter != nil && *d.Newsletter {
			marketing = true
		}
	}
	return domain.Preferences{
		UserID:           d.ID,
		EmailEnabled:     email,
		SMSEnabled:       sms,
		PushEnabled:      push,
		MarketingEnabled: marketing,
		UpdatedAt:        d.UpdatedAt,
	}
}
