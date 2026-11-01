// Package mongo is a MongoDB adapter for the cart Repository port.
// Documents live in one collection with a unique partial index on
// (customer_id) with partialFilterExpression for status in (active, checkout_pending)
// - the idempotency key the domain's Create contract promises - and a supporting
// (customer_id, status) index for queries. sqlc only generates code for SQL, so
// this adapter is hand-written the way any non-SQL adapter is: by translating
// between bson documents and the domain entity.
package mongo

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/cart/domain"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionName is the MongoDB collection the carts live in.
const collectionName = "carts"

// CartRepository stores carts in MongoDB. It is safe for concurrent use:
// MongoDB serializes the single-document operations it issues.
type CartRepository struct {
	coll *mongo.Collection
}

// Compile-time proof that Repository satisfies the port. If a method drifts, the
// build fails here, not at runtime.
var _ domain.Repository = (*CartRepository)(nil)

// NewCartRepository adapts a mongo.Database handle to the domain port. The
// composition root owns the client's lifecycle; this only names the collection.
func NewCartRepository(db *mongo.Database) *CartRepository {
	return &CartRepository{coll: db.Collection(collectionName)}
}

// EnsureIndexes creates the indexes the adapter's queries depend on: the unique
// partial index on (customer_id) for active/checkout_pending carts backing
// Create's first-write-wins guarantee, and the (customer_id, status) index
// backing GetByCustomerID's filter. The composition root calls it once during
// boot, the way a SQL schema is migrated before the first query.
func (r *CartRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{
					{Key: "status", Value: bson.D{{Key: "$in", Value: []string{"active", "checkout_pending"}}}},
				}),
		},
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ensure cart indexes: %w", err)
	}
	return nil
}

// GetByCustomerID returns the customer's active or checkout_pending cart,
// or domain.ErrCartNotFound. It uses the (customer_id, status) index for the
// filter and the status range.
func (r *CartRepository) GetByCustomerID(ctx context.Context, customerID string) (domain.Cart, error) {
	var doc cartDoc
	filter := bson.D{
		{Key: "customer_id", Value: customerID},
		{Key: "status", Value: bson.D{{Key: "$in", Value: []string{"active", "checkout_pending"}}}},
	}
	err := r.coll.FindOne(ctx, filter).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Cart{}, domain.ErrCartNotFound
	case err != nil:
		return domain.Cart{}, fmt.Errorf("get cart by customer: %w", err)
	}
	return doc.toDomain(), nil
}

// GetByCustomerIDIncludingConsumed returns the customer's cart regardless of status
// (active, checkout_pending, or consumed), or domain.ErrCartNotFound.
func (r *CartRepository) GetByCustomerIDIncludingConsumed(ctx context.Context, customerID string) (domain.Cart, error) {
	var doc cartDoc
	filter := bson.D{{Key: "customer_id", Value: customerID}}
	err := r.coll.FindOne(ctx, filter).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.Cart{}, domain.ErrCartNotFound
	case err != nil:
		return domain.Cart{}, fmt.Errorf("get cart by customer: %w", err)
	}
	return doc.toDomain(), nil
}

// Create records the cart idempotently: a unique partial index on (customer_id)
// where status in (active, checkout_pending) ensures at most one active cart
// per customer. A duplicate-key error is reported as domain.ErrCartAlreadyExists.
func (r *CartRepository) Create(ctx context.Context, cart domain.Cart) error {
	doc := toCartDoc(cart)
	_, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrCartAlreadyExists
		}
		return fmt.Errorf("create cart: %w", err)
	}
	return nil
}

// Update performs an optimistic-lock CAS on (id, version). It sets the mutable
// fields and increments version. A stale version returns
// domain.ErrCartOptimisticLock; a missing cart returns domain.ErrCartNotFound.
func (r *CartRepository) Update(ctx context.Context, cart domain.Cart, expectedVersion int) error {
	filter := bson.D{
		{Key: "_id", Value: cart.ID},
		{Key: "version", Value: expectedVersion},
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "status", Value: string(cart.Status)},
			{Key: "checkout_key", Value: cart.CheckoutKey},
			{Key: "order_id", Value: cart.OrderID},
			{Key: "items", Value: toItemDocs(cart.Items)},
			{Key: "updated_at", Value: cart.UpdatedAt},
			{Key: "version", Value: expectedVersion + 1},
		}},
	}
	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update cart: %w", err)
	}
	if res.MatchedCount == 0 {
		var existing cartDoc
		err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: cart.ID}}).Decode(&existing)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrCartNotFound
		}
		if err != nil {
			return fmt.Errorf("probe cart for version conflict: %w", err)
		}
		return domain.ErrCartOptimisticLock
	}
	return nil
}

// cartDoc is the bson shape of one stored cart. It mirrors the domain entity
// field for field; the mapping lives here so a schema change never leaks into
// the domain.
type cartDoc struct {
	ID          string    `bson:"_id"`
	CustomerID  string    `bson:"customer_id"`
	Status      string    `bson:"status"`
	CheckoutKey string    `bson:"checkout_key"`
	OrderID     string    `bson:"order_id"`
	Version     int       `bson:"version"`
	Items       []itemDoc `bson:"items"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

// itemDoc is the bson shape of a single cart item.
type itemDoc struct {
	ProductID              string    `bson:"product_id"`
	ProductNameSnapshot    string    `bson:"product_name_snapshot"`
	UnitPriceSnapshotCents int64     `bson:"unit_price_snapshot_cents"`
	Quantity               int       `bson:"quantity"`
	CreatedAt              time.Time `bson:"created_at"`
	UpdatedAt              time.Time `bson:"updated_at"`
}

// toCartDoc converts a domain.Cart to its bson representation.
func toCartDoc(c domain.Cart) cartDoc {
	items := make([]itemDoc, len(c.Items))
	for i, item := range c.Items {
		items[i] = itemDoc{
			ProductID:              item.ProductID,
			ProductNameSnapshot:    item.ProductNameSnapshot,
			UnitPriceSnapshotCents: item.UnitPriceSnapshotCents,
			Quantity:               item.Quantity,
			CreatedAt:              item.CreatedAt,
			UpdatedAt:              item.UpdatedAt,
		}
	}
	return cartDoc{
		ID:          c.ID,
		CustomerID:  c.CustomerID,
		Status:      string(c.Status),
		CheckoutKey: c.CheckoutKey,
		OrderID:     c.OrderID,
		Version:     c.Version,
		Items:       items,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// toItemDocs converts a slice of domain.Item to itemDoc slice.
func toItemDocs(items []domain.Item) []itemDoc {
	docs := make([]itemDoc, len(items))
	for i, item := range items {
		docs[i] = itemDoc{
			ProductID:              item.ProductID,
			ProductNameSnapshot:    item.ProductNameSnapshot,
			UnitPriceSnapshotCents: item.UnitPriceSnapshotCents,
			Quantity:               item.Quantity,
			CreatedAt:              item.CreatedAt,
			UpdatedAt:              item.UpdatedAt,
		}
	}
	return docs
}

// toDomain converts a bson cartDoc to its domain representation.
func (d cartDoc) toDomain() domain.Cart {
	items := make([]domain.Item, len(d.Items))
	for i, item := range d.Items {
		items[i] = domain.Item{
			ProductID:              item.ProductID,
			ProductNameSnapshot:    item.ProductNameSnapshot,
			UnitPriceSnapshotCents: item.UnitPriceSnapshotCents,
			Quantity:               item.Quantity,
			CreatedAt:              item.CreatedAt,
			UpdatedAt:              item.UpdatedAt,
		}
	}
	return domain.Cart{
		ID:          d.ID,
		CustomerID:  d.CustomerID,
		Status:      domain.Status(d.Status),
		CheckoutKey: d.CheckoutKey,
		OrderID:     d.OrderID,
		Version:     d.Version,
		Items:       items,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
