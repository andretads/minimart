// Package mongo is a MongoDB adapter for the wishlist Repository port.
package mongo

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/wishlist/domain"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "wishlists"

type Repository struct {
	coll *mongo.Collection
}

var _ domain.Repository = (*Repository)(nil)

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{coll: db.Collection(collectionName)}
}

func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("ensure wishlist indexes: %w", err)
	}
	return nil
}

func (r *Repository) GetByCustomerID(ctx context.Context, customerID string) (domain.Wishlist, error) {
	var doc wishlistDoc
	filter := bson.D{{Key: "customer_id", Value: customerID}}
	err := r.coll.FindOne(ctx, filter).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Wishlist{}, domain.ErrWishlistNotFound
	case err != nil:
		return domain.Wishlist{}, fmt.Errorf("get wishlist by customer: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *Repository) Create(ctx context.Context, w domain.Wishlist) error {
	doc := toDoc(w)
	_, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrWishlistAlreadyExists
		}
		return fmt.Errorf("create wishlist: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, w domain.Wishlist) error {
	filter := bson.D{{Key: "_id", Value: w.ID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "items", Value: toItemDocs(w.Items)},
			{Key: "updated_at", Value: w.UpdatedAt},
		}},
	}
	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update wishlist: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrWishlistNotFound
	}
	return nil
}

func toItemDocs(items []domain.WishlistItem) []itemDoc {
	docs := make([]itemDoc, len(items))
	for i, item := range items {
		docs[i] = itemDoc{
			ProductID: item.ProductID,
			AddedAt:   item.AddedAt,
		}
	}
	return docs
}

type wishlistDoc struct {
	ID         string    `bson:"_id"`
	CustomerID string    `bson:"customer_id"`
	Items      []itemDoc `bson:"items"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
}

type itemDoc struct {
	ProductID string    `bson:"product_id"`
	AddedAt   time.Time `bson:"added_at"`
}

func toDoc(w domain.Wishlist) wishlistDoc {
	items := make([]itemDoc, len(w.Items))
	for i, item := range w.Items {
		items[i] = itemDoc{
			ProductID: item.ProductID,
			AddedAt:   item.AddedAt,
		}
	}
	return wishlistDoc{
		ID:         w.ID,
		CustomerID: w.CustomerID,
		Items:      items,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}
}

func (d wishlistDoc) toDomain() domain.Wishlist {
	items := make([]domain.WishlistItem, len(d.Items))
	for i, item := range d.Items {
		items[i] = domain.WishlistItem{
			ProductID: item.ProductID,
			AddedAt:   item.AddedAt,
		}
	}
	return domain.Wishlist{
		ID:         d.ID,
		CustomerID: d.CustomerID,
		Items:      items,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}
