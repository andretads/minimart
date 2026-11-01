package postgres

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres/reader"
	"minimart/internal/catalog/infrastructure/postgres/writer"

	"github.com/jackc/pgx/v5"
)

// SaveProductImage inserts an image and bumps the owning product's version in
// one transaction. Bumping first locks the product row, which serializes image
// writes for that product and keeps position assignment deterministic.
func (r *CatalogRepository) SaveProductImage(ctx context.Context, img domain.ProductImage) (domain.ProductImage, error) {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return domain.ProductImage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	bumped, err := q.BumpProductVersion(ctx, img.ProductID)
	if err != nil {
		return domain.ProductImage{}, err
	}
	if bumped == 0 {
		return domain.ProductImage{}, domain.ErrProductNotFound
	}

	position, err := q.NextProductImagePosition(ctx, img.ProductID)
	if err != nil {
		return domain.ProductImage{}, err
	}

	row, err := q.SaveProductImage(ctx, writer.SaveProductImageParams{
		ID:          img.ID,
		ProductID:   img.ProductID,
		ObjectKey:   img.ObjectKey,
		Bucket:      img.Bucket,
		ContentType: img.ContentType,
		SizeBytes:   img.SizeBytes,
		Position:    position,
		CreatedAt:   img.CreatedAt,
		UpdatedAt:   img.UpdatedAt,
	})
	if err != nil {
		return domain.ProductImage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ProductImage{}, err
	}
	return toDomainImageFromWriter(row), nil
}

// DeleteProductImage removes an image and bumps the owning product's version in
// one transaction. The product is bumped first so a missing product is reported
// distinctly from a missing image.
func (r *CatalogRepository) DeleteProductImage(ctx context.Context, productID, imageID string) (domain.ProductImage, error) {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return domain.ProductImage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	bumped, err := q.BumpProductVersion(ctx, productID)
	if err != nil {
		return domain.ProductImage{}, err
	}
	if bumped == 0 {
		return domain.ProductImage{}, domain.ErrProductNotFound
	}

	row, err := q.DeleteProductImage(ctx, writer.DeleteProductImageParams{
		ID:        imageID,
		ProductID: productID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ProductImage{}, domain.ErrProductImageNotFound
		}
		return domain.ProductImage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ProductImage{}, err
	}
	return toDomainImageFromWriter(row), nil
}

// ListProductImages returns a product's images ordered by position.
func (r *CatalogRepository) ListProductImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	rows, err := r.reader.ListProductImages(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProductImage, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainImage(row))
	}
	return out, nil
}

// ListProductImagesByProductIDs returns images grouped by product ID.
func (r *CatalogRepository) ListProductImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	out := make(map[string][]domain.ProductImage, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}

	rows, err := r.reader.ListProductImagesByProductIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ProductID] = append(out[row.ProductID], toDomainImage(row))
	}
	return out, nil
}

func toDomainImage(row reader.ProductImage) domain.ProductImage {
	return domain.ProductImage{
		ID:          row.ID,
		ProductID:   row.ProductID,
		ObjectKey:   row.ObjectKey,
		Bucket:      row.Bucket,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		Position:    int(row.Position),
		Version:     int(row.Version),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toDomainImageFromWriter(row writer.ProductImage) domain.ProductImage {
	return domain.ProductImage{
		ID:          row.ID,
		ProductID:   row.ProductID,
		ObjectKey:   row.ObjectKey,
		Bucket:      row.Bucket,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		Position:    int(row.Position),
		Version:     int(row.Version),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
