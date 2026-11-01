package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/postgres/reader"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implements domain.Repository using the Postgres read pool.
// It holds the sqlc-generated reader queries bound to the replica pool.
type Repository struct {
	reader *reader.Queries
}

var _ domain.Repository = (*Repository)(nil)

// NewRepository builds the repository with the provided read pool.
func NewRepository(readPool *pgxpool.Pool) *Repository {
	return &Repository{reader: reader.New(readPool)}
}

// ListCategories returns all categories ordered by name.
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.reader.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, domain.Category{
			ID:   row.ID,
			Name: row.Name,
			Icon: row.Icon,
		})
	}
	return categories, nil
}

// List returns a page of products ordered newest-first.
func (r *Repository) List(ctx context.Context, limit int, after *domain.Cursor) ([]domain.Product, error) {
	if after == nil {
		rows, err := r.reader.ListProducts(ctx, int32(limit)) //nolint:gosec // limit validated <= 101
		if err != nil {
			return nil, err
		}
		products := make([]domain.Product, 0, len(rows))
		for _, row := range rows {
			products = append(products, toProduct(row))
		}
		return products, nil
	}
	rows, err := r.reader.ListProductsAfter(ctx, reader.ListProductsAfterParams{
		CreatedAt: after.CreatedAt,
		ID:        after.ID,
		PageSize:  int32(limit), //nolint:gosec // limit validated <= 101
	})
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, toProduct(row))
	}
	return products, nil
}

// ListByCategory returns a page of products in the given category ordered newest-first.
func (r *Repository) ListByCategory(ctx context.Context, categoryID string, limit int, after *domain.Cursor) ([]domain.Product, error) {
	if after == nil {
		rows, err := r.reader.ListProductsByCategory(ctx, reader.ListProductsByCategoryParams{
			CategoryID: categoryID,
			PageSize:   int32(limit), //nolint:gosec // limit validated <= 101
		})
		if err != nil {
			return nil, err
		}
		products := make([]domain.Product, 0, len(rows))
		for _, row := range rows {
			products = append(products, toProduct(row))
		}
		return products, nil
	}
	rows, err := r.reader.ListProductsByCategoryAfter(ctx, reader.ListProductsByCategoryAfterParams{
		CategoryID: categoryID,
		CreatedAt:  after.CreatedAt,
		ID:         after.ID,
		PageSize:   int32(limit), //nolint:gosec // limit validated <= 101
	})
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, toProduct(row))
	}
	return products, nil
}

// GetByID reads a product by ID from the read pool.
// Returns ErrProductNotFound when no row exists.
func (r *Repository) GetByID(ctx context.Context, id string) (domain.Product, error) {
	row, err := r.reader.GetProduct(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Product{}, domain.ErrProductNotFound
		}
		return domain.Product{}, err
	}
	return toProduct(row), nil
}

// ListImages returns the images of a product ordered by position.
func (r *Repository) ListImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	rows, err := r.reader.ListProductImages(ctx, productID)
	if err != nil {
		return nil, err
	}
	images := make([]domain.ProductImage, 0, len(rows))
	for _, row := range rows {
		images = append(images, domain.ProductImage{
			ID:          row.ID,
			ProductID:   row.ProductID,
			ObjectKey:   row.ObjectKey,
			ContentType: row.ContentType,
			SizeBytes:   row.SizeBytes,
			Position:    int(row.Position),
		})
	}
	return images, nil
}

// ListImagesByProductIDs returns images for the given products grouped by
// product ID, ordered by position within each product.
func (r *Repository) ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	if len(productIDs) == 0 {
		return map[string][]domain.ProductImage{}, nil
	}
	rows, err := r.reader.ListProductImagesByProductIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	imagesByProduct := make(map[string][]domain.ProductImage, len(productIDs))
	for _, row := range rows {
		imagesByProduct[row.ProductID] = append(imagesByProduct[row.ProductID], domain.ProductImage{
			ID:          row.ID,
			ProductID:   row.ProductID,
			ObjectKey:   row.ObjectKey,
			ContentType: row.ContentType,
			SizeBytes:   row.SizeBytes,
			Position:    int(row.Position),
		})
	}
	return imagesByProduct, nil
}

// ListReviews returns a page of approved reviews of a product, newest first.
func (r *Repository) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	rows, err := r.reader.ListApprovedReviewsPaginated(ctx, reader.ListApprovedReviewsPaginatedParams{
		ProductID: productID,
		Limit:     int32(limit),  //nolint:gosec // limit validated <= 100
		Offset:    int32(offset), //nolint:gosec // offset from validated page/page_size
	})
	if err != nil {
		return nil, err
	}
	reviews := make([]domain.Review, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, domain.Review{
			ID:          row.ID.String(),
			ProductID:   row.ProductID,
			CustomerID:  row.CustomerID,
			Description: row.Description,
			Rating:      int(row.Rating),
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
		})
	}
	return reviews, nil
}

// CountReviews returns the total number of approved reviews of a product.
func (r *Repository) CountReviews(ctx context.Context, productID string) (int, error) {
	count, err := r.reader.CountApprovedReviews(ctx, productID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// toProduct maps the shared sqlc product row into the domain model.
func toProduct(row reader.Product) domain.Product {
	return domain.Product{
		ID:           row.ID,
		Name:         row.Name,
		PriceCents:   row.PriceCents,
		CategoryID:   row.CategoryID,
		Stock:        int(row.Stock),
		SKU:          derefString(row.Sku),
		Description:  derefString(row.Description),
		Details:      parseDetails(row.Details),
		TotalRating:  derefInt64(row.TotalRating),
		TotalReviews: derefInt64(row.TotalReviews),
		CreatedAt:    row.CreatedAt,
	}
}

func parseDetails(raw []byte) []domain.ProductDetail {
	if len(raw) == 0 {
		return nil
	}
	var details []domain.ProductDetail
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil
	}
	return details
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
