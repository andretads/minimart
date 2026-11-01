package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres/reader"
	"minimart/internal/catalog/infrastructure/postgres/writer"
	"minimart/pkg/message"
	"minimart/pkg/moderation"
	"minimart/pkg/outbox"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// EventPublisher writes a domain event into the transactional outbox within the
// caller's transaction, so the event commits atomically with the aggregate.
type EventPublisher interface {
	Send(ctx context.Context, msg outbox.Message, tx pgx.Tx) error
}

// CatalogRepository implements the catalog domain.Repository port over Postgres.
// It holds sqlc query objects bound to a read pool and a write pool, plus the
// write pool itself so multi-statement stock operations can run in a transaction.
type CatalogRepository struct {
	reader    *reader.Queries
	writer    *writer.Queries
	writePool *pgxpool.Pool
	events    EventPublisher
	cache     *goredis.Client
}

// Compile-time proof the Postgres adapter satisfies the same port as the
// in-memory one. This is why swapping storage touches no business code.
var _ domain.Repository = (*CatalogRepository)(nil)

func NewCatalogRepository(readPool, writePool *pgxpool.Pool, events EventPublisher, cache ...*goredis.Client) *CatalogRepository {
	repo := &CatalogRepository{
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
		writePool: writePool,
		events:    events,
	}
	if len(cache) > 0 {
		repo.cache = cache[0]
	}
	return repo
}

// Save upserts the product by id, using the timestamps carried on the value:
// created_at and updated_at are written on a fresh insert. The ON CONFLICT
// branch deliberately leaves created_at alone and only refreshes updated_at, so
// re-saving an existing product never rewrites its creation time - matching the
// in-memory adapter.
func (r *CatalogRepository) Save(ctx context.Context, p domain.Product) error {
	// The stock column is a 4-byte integer; reject values that would overflow it
	// rather than silently truncating.
	if p.Stock > math.MaxInt32 {
		return fmt.Errorf("stock %d exceeds int32 range", p.Stock)
	}
	// The version column is a 4-byte integer; reject values that would overflow it.
	if p.Version > math.MaxInt32 {
		return fmt.Errorf("version %d exceeds int32 range", p.Version)
	}

	detailsJSON, err := json.Marshal(p.Details)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	return r.writer.SaveProduct(ctx, writer.SaveProductParams{
		ID:           p.ID,
		Name:         p.Name,
		PriceCents:   p.PriceCents,
		Stock:        int32(p.Stock),   //nolint:gosec // bounded by the range check above
		Version:      int32(p.Version), //nolint:gosec // bounded by the range check above
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		CategoryID:   p.CategoryID,
		Sku:          stringPtr(p.SKU),
		Description:  stringPtr(p.Description),
		Details:      detailsJSON,
		TotalRating:  int64Ptr(p.TotalRating),
		TotalReviews: int64Ptr(p.TotalReviews),
	})
}

// Update writes name, price_cents, category_id, sku, description, and details under the product's optimistic lock.
// The statement only matches while the stored version equals p.Version, so a
// concurrent writer that bumped the version first is reported as
// ErrProductOptimisticLock instead of silently overwriting its change. Stock is
// intentionally left alone.
func (r *CatalogRepository) Update(ctx context.Context, p domain.Product) (domain.Product, error) {
	// The version column is a 4-byte integer; reject values that would overflow it.
	if p.Version > math.MaxInt32 {
		return domain.Product{}, fmt.Errorf("version %d exceeds int32 range", p.Version)
	}

	detailsJSON, err := json.Marshal(p.Details)
	if err != nil {
		return domain.Product{}, fmt.Errorf("marshal details: %w", err)
	}

	row, err := r.writer.UpdateProduct(ctx, writer.UpdateProductParams{
		ID:          p.ID,
		Name:        p.Name,
		PriceCents:  p.PriceCents,
		CategoryID:  p.CategoryID,
		Sku:         stringPtr(p.SKU),
		Description: stringPtr(p.Description),
		Details:     detailsJSON,
		Version:     int32(p.Version), //nolint:gosec // bounded by the range check above
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.classifyNoMatch(ctx, p.ID)
		}
		return domain.Product{}, err
	}
	p.Version = int(row.Version)
	p.UpdatedAt = row.UpdatedAt
	return p, nil
}

// classifyNoMatch tells a stale version apart from a missing row: the UPDATE
// matched nothing, and the product either exists (someone else bumped the
// version) or does not exist at all. The existence probe goes to the writer so
// a lagging read replica cannot misreport a live row as missing.
func (r *CatalogRepository) classifyNoMatch(ctx context.Context, id string) (domain.Product, error) {
	exists, err := r.writer.ProductExists(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	if exists {
		return domain.Product{}, domain.ErrProductOptimisticLock
	}
	return domain.Product{}, domain.ErrProductNotFound
}

func (r *CatalogRepository) GetByID(ctx context.Context, id string) (domain.Product, error) {
	row, err := r.reader.GetProduct(ctx, id)
	if err != nil {
		// Translate the driver's "no rows" into the domain's not-found error, so
		// the application layer never learns which database we use.
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Product{}, domain.ErrProductNotFound
		}
		return domain.Product{}, err
	}

	var details []domain.ProductDetail
	if len(row.Details) > 0 {
		if err := json.Unmarshal(row.Details, &details); err != nil {
			return domain.Product{}, fmt.Errorf("unmarshal product details: %w", err)
		}
	}

	return domain.Product{
		ID:           row.ID,
		Name:         row.Name,
		PriceCents:   row.PriceCents,
		Stock:        int(row.Stock),
		Version:      int(row.Version),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		CategoryID:   row.CategoryID,
		SKU:          derefString(row.Sku),
		Description:  derefString(row.Description),
		Details:      details,
		TotalRating:  derefInt64(row.TotalRating),
		TotalReviews: derefInt64(row.TotalReviews),
	}, nil
}

func (r *CatalogRepository) List(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.reader.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		var details []domain.ProductDetail
		if len(row.Details) > 0 {
			if err := json.Unmarshal(row.Details, &details); err != nil {
				return nil, fmt.Errorf("unmarshal product details: %w", err)
			}
		}
		out = append(out, domain.Product{
			ID:           row.ID,
			Name:         row.Name,
			PriceCents:   row.PriceCents,
			Stock:        int(row.Stock),
			Version:      int(row.Version),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			CategoryID:   row.CategoryID,
			SKU:          derefString(row.Sku),
			Description:  derefString(row.Description),
			Details:      details,
			TotalRating:  derefInt64(row.TotalRating),
			TotalReviews: derefInt64(row.TotalReviews),
		})
	}
	return out, nil
}

// ListByCategory returns all products in the given category.
func (r *CatalogRepository) ListByCategory(ctx context.Context, categoryID string) ([]domain.Product, error) {
	rows, err := r.reader.ListProductsByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		var details []domain.ProductDetail
		if len(row.Details) > 0 {
			if err := json.Unmarshal(row.Details, &details); err != nil {
				return nil, fmt.Errorf("unmarshal product details: %w", err)
			}
		}
		out = append(out, domain.Product{
			ID:           row.ID,
			Name:         row.Name,
			PriceCents:   row.PriceCents,
			Stock:        int(row.Stock),
			Version:      int(row.Version),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			CategoryID:   row.CategoryID,
			SKU:          derefString(row.Sku),
			Description:  derefString(row.Description),
			Details:      details,
			TotalRating:  derefInt64(row.TotalRating),
			TotalReviews: derefInt64(row.TotalReviews),
		})
	}
	return out, nil
}

// ReserveStock reserves every item's quantity for orderID in one transaction. It
// is all-or-nothing: any shortage rolls back the whole reservation, so there is
// never a partially reserved order. Each product is decremented with a
// conditional UPDATE, which is what prevents overselling under concurrency.
//
// A reservation row already present for (orderID, product) means a previous call
// already reserved that product, so it is skipped: a retried placement does not
// double-reserve.
func (r *CatalogRepository) ReserveStock(ctx context.Context, orderID string, items []domain.ReservationItem) error {
	if len(items) == 0 {
		return domain.ErrInvalidReservation
	}
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	q := r.writer.WithTx(tx)
	for _, it := range items {
		// The quantity column is a 4-byte integer; reject values that would
		// overflow it rather than silently truncating.
		if it.Quantity <= 0 {
			return domain.ErrInvalidReservation
		}
		if it.Quantity > math.MaxInt32 {
			return fmt.Errorf("reservation quantity %d exceeds int32 range", it.Quantity)
		}
		inserted, err := q.CreateStockReservation(ctx, writer.CreateStockReservationParams{
			OrderID:   orderID,
			ProductID: it.ProductID,
			Quantity:  int32(it.Quantity),
		})
		if err != nil {
			return err
		}
		if inserted == 0 {
			continue // already reserved for this order
		}
		rows, err := q.ReserveProductStock(ctx, writer.ReserveProductStockParams{
			ID:       it.ProductID,
			Quantity: int32(it.Quantity),
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			// Disambiguate a sold-out product from a missing one, on the write
			// pool so replica lag cannot misreport a live product as missing.
			exists, err := q.ProductExists(ctx, it.ProductID)
			if err != nil {
				return err
			}
			if !exists {
				return domain.ErrProductNotFound
			}
			return domain.ErrInsufficientStock
		}
	}
	return tx.Commit(ctx)
}

// ReleaseStock restores the stock for every outstanding reservation of orderID in
// one transaction, then marks those reservations released. Marking and returning
// the rows atomically is what makes release idempotent: a redelivered release
// event finds nothing outstanding and writes nothing.
func (r *CatalogRepository) ReleaseStock(ctx context.Context, orderID string) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	released, err := q.ReleaseStockReservations(ctx, orderID)
	if err != nil {
		return err
	}
	for _, row := range released {
		if err := q.RestoreProductStock(ctx, writer.RestoreProductStockParams{
			ID:       row.ProductID,
			Quantity: row.Quantity,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SaveReview persists a new review for a product and publishes the review-created
// event atomically in the same transaction.
func (r *CatalogRepository) SaveReview(ctx context.Context, rev domain.Review) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	var reviewID pgtype.UUID
	if err := reviewID.Scan(rev.ID); err != nil {
		return err
	}
	if err := q.SaveReview(ctx, writer.SaveReviewParams{
		ID:          reviewID,
		ProductID:   rev.ProductID,
		CustomerID:  rev.CustomerID,
		Description: rev.Description,
		Rating:      clampInt32(rev.Rating),
		Status:      rev.Status,
		CreatedAt:   rev.CreatedAt,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateReview
		}
		return err
	}

	if r.events != nil {
		msg, err := outboxReviewCreatedMessage(rev)
		if err != nil {
			return err
		}
		if err := r.events.Send(ctx, msg, tx); err != nil {
			return fmt.Errorf("enqueue review event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.Del(ctx, "catalog:rating:"+rev.ProductID).Err()
	}
	return nil
}

// GetReviews returns all approved reviews for a product.
func (r *CatalogRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	rows, err := r.reader.GetReviews(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Review{
			ID:          row.ID.String(),
			ProductID:   row.ProductID,
			CustomerID:  row.CustomerID,
			Description: row.Description,
			Rating:      int(row.Rating),
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil
}

// GetReviewsPaginated returns a page of approved reviews for a product, newest first.
func (r *CatalogRepository) GetReviewsPaginated(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	if r.cache != nil {
		key := fmt.Sprintf("catalog:reviews:%s:%d:%d", productID, limit, offset)
		if v, err := r.cache.Get(ctx, key).Result(); err == nil {
			var cached []domain.Review
			if json.Unmarshal([]byte(v), &cached) == nil {
				return cached, nil
			}
		}
	}
	rows, err := r.reader.GetReviewsPaginated(ctx, reader.GetReviewsPaginatedParams{
		ProductID: productID,
		Limit:     int32(limit),  //nolint:gosec // limit validated <= 100
		Offset:    int32(offset), //nolint:gosec // offset from validated page/page_size
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Review{
			ID:          row.ID.String(),
			ProductID:   row.ProductID,
			CustomerID:  row.CustomerID,
			Description: row.Description,
			Rating:      int(row.Rating),
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
		})
	}
	if r.cache != nil {
		key := fmt.Sprintf("catalog:reviews:%s:%d:%d", productID, limit, offset)
		if b, err := json.Marshal(out); err == nil {
			_ = r.cache.Set(ctx, key, b, time.Minute).Err()
		}
	}
	return out, nil
}

// CountReviews returns the number of approved reviews for a product.
func (r *CatalogRepository) CountReviews(ctx context.Context, productID string) (int, error) {
	count, err := r.reader.CountReviews(ctx, productID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetRatingSummary returns the rating summary for a product.
func (r *CatalogRepository) GetRatingSummary(ctx context.Context, productID string) (domain.RatingSummary, error) {
	if r.cache != nil {
		key := "catalog:rating:" + productID
		if v, err := r.cache.Get(ctx, key).Result(); err == nil {
			var cached domain.RatingSummary
			if json.Unmarshal([]byte(v), &cached) == nil {
				return cached, nil
			}
		}
	}
	row, err := r.reader.GetRatingSummary(ctx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RatingSummary{ProductID: productID}, nil
		}
		return domain.RatingSummary{}, err
	}
	avg, _ := row.AverageRating.Float64Value()
	summary := domain.RatingSummary{
		ProductID:     row.ProductID,
		TotalReviews:  row.TotalReviews,
		RatingSum:     row.RatingSum,
		AverageRating: avg.Float64,
		UpdatedAt:     row.UpdatedAt,
	}
	if r.cache != nil {
		key := "catalog:rating:" + productID
		if b, err := json.Marshal(summary); err == nil {
			_ = r.cache.Set(ctx, key, b, time.Minute).Err()
		}
	}
	return summary, nil
}

// UpdateRatingSummary updates the rating summary aggregate for a product.
func (r *CatalogRepository) UpdateRatingSummary(ctx context.Context, productID string, ratingSum, totalReviews int64) error {
	avg := 0.0
	if totalReviews > 0 {
		avg = float64(ratingSum) / float64(totalReviews)
	}
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.4f", avg))
	return r.writer.UpsertRatingSummary(ctx, writer.UpsertRatingSummaryParams{
		ProductID:     productID,
		TotalReviews:  totalReviews,
		RatingSum:     ratingSum,
		AverageRating: n,
	})
}

// GetReviewByID returns a review by its ID.
func (r *CatalogRepository) GetReviewByID(ctx context.Context, id string) (domain.Review, error) {
	// Parse UUID string
	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return domain.Review{}, domain.ErrProductReviewNotFound
	}

	row, err := r.reader.GetReviewByID(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Review{}, domain.ErrProductReviewNotFound
		}
		return domain.Review{}, err
	}

	var categories []string
	if len(row.ModerationCategories) > 0 {
		if err := json.Unmarshal(row.ModerationCategories, &categories); err != nil {
			return domain.Review{}, fmt.Errorf("unmarshal moderation categories: %w", err)
		}
	}

	return domain.Review{
		ID:                     row.ID.String(),
		ProductID:              row.ProductID,
		CustomerID:             row.CustomerID,
		Description:            row.Description,
		Rating:                 int(row.Rating),
		Status:                 row.Status,
		CreatedAt:              row.CreatedAt,
		ModeratedAt:            row.ModeratedAt,
		ModerationDecision:     derefString(row.ModerationDecision),
		ModerationConfidence:   derefFloat64(row.ModerationConfidence),
		ModerationCategories:   categories,
		ModerationExplanation:  derefString(row.ModerationExplanation),
		ModerationModel:        derefString(row.ModerationModel),
		ModerationModelVersion: derefString(row.ModerationModelVersion),
	}, nil
}

// UpdateReviewStats updates the denormalized review counters on the product.
func (r *CatalogRepository) UpdateReviewStats(ctx context.Context, productID string, totalRating, totalReviews int64) error {
	return r.writer.UpdateReviewStats(ctx, writer.UpdateReviewStatsParams{
		ID:           productID,
		TotalRating:  int64Ptr(totalRating),
		TotalReviews: int64Ptr(totalReviews),
	})
}

// UpdateReviewModeration updates the moderation fields of a review.
func (r *CatalogRepository) UpdateReviewModeration(ctx context.Context, reviewID string, result moderation.Result) error {
	var reviewUUID pgtype.UUID
	if err := reviewUUID.Scan(reviewID); err != nil {
		return err
	}

	// Load the review first so the outbox event can carry its product and rating.
	revRow, err := r.reader.GetReviewByID(ctx, reviewUUID)
	if err != nil {
		return err
	}

	var moderatedAt *time.Time
	if !result.ModeratedAt.IsZero() {
		moderatedAt = &result.ModeratedAt
	}

	categoriesJSON, err := json.Marshal(result.Categories)
	if err != nil {
		return err
	}

	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	if err := q.UpdateReviewModeration(ctx, writer.UpdateReviewModerationParams{
		ID:                     reviewUUID,
		Status:                 result.Decision,
		ModeratedAt:            moderatedAt,
		ModerationDecision:     stringPtr(result.Decision),
		ModerationConfidence:   float64Ptr(result.Confidence),
		ModerationCategories:   categoriesJSON,
		ModerationExplanation:  stringPtr(result.Explanation),
		ModerationModel:        stringPtr(result.Model),
		ModerationModelVersion: stringPtr(result.ModelVersion),
	}); err != nil {
		return err
	}

	if r.events != nil {
		body, err := json.Marshal(message.ReviewModerated{
			ReviewID:  reviewID,
			ProductID: revRow.ProductID,
			Decision:  result.Decision,
			Rating:    int(revRow.Rating),
		})
		if err != nil {
			return err
		}
		msg := outbox.Message{
			Topic: "review-moderated",
			Key:   reviewID,
			Body:  body,
		}
		if err := r.events.Send(ctx, msg, tx); err != nil {
			return fmt.Errorf("enqueue review moderated event: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// clampInt32 clamps an int to the int32 range.
func clampInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// stringPtr returns a pointer to the string if non-empty, otherwise nil.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString returns the string value if the pointer is non-nil, otherwise empty string.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// int64Ptr returns a pointer to the int64 if non-zero, otherwise nil.
func int64Ptr(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}

// derefInt64 returns the int64 value if the pointer is non-nil, otherwise 0.
func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// derefFloat64 returns the float64 value if the pointer is non-nil, otherwise 0.
func derefFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// float64Ptr returns a pointer to the float64 if non-zero, otherwise nil.
func float64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// outboxReviewCreatedMessage builds the outbox message for a review-created event.
func outboxReviewCreatedMessage(rev domain.Review) (outbox.Message, error) {
	type ReviewCreatedEvent struct {
		EventID   string    `json:"event_id"`
		ReviewID  string    `json:"review_id"`
		CreatedAt time.Time `json:"created_at"`
	}
	evt := ReviewCreatedEvent{
		EventID:   rev.ID,
		ReviewID:  rev.ID,
		CreatedAt: rev.CreatedAt,
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return outbox.Message{}, err
	}
	return outbox.Message{
		Topic: "review-created",
		Key:   rev.ID,
		Body:  body,
	}, nil
}
