package application

import (
	"context"
	"fmt"
	"minimart/internal/catalog/domain"
	"minimart/pkg/logger"
)

// AddProductImage uploads an image for an existing product and persists its
// metadata. The product's version is bumped by the repository.
func (s *Service) AddProductImage(ctx context.Context, productID string, img *ImageUpload) (domain.ProductImage, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		return domain.ProductImage{}, err
	}
	if img == nil {
		return domain.ProductImage{}, domain.ErrImageRequired
	}
	if err := s.validateImage(img); err != nil {
		return domain.ProductImage{}, err
	}

	imageID := newID()
	key, err := s.storeImage(ctx, productID, imageID, img)
	if err != nil {
		return domain.ProductImage{}, err
	}

	productImage, err := domain.NewProductImage(productID, key, s.s3Client.Bucket(), img.ContentType, img.Size)
	if err != nil {
		return domain.ProductImage{}, err
	}
	productImage.ID = imageID

	saved, err := s.repo.SaveProductImage(ctx, productImage)
	if err != nil {
		if deleteErr := s.s3Client.Delete(ctx, key); deleteErr != nil {
			logger.FromContext(ctx).Warn("product image cleanup failed",
				"operation", "s3_delete",
				"product_id", productID,
				"key", key,
				"error", deleteErr,
			)
		}
		return domain.ProductImage{}, err
	}
	return saved, nil
}

// ListProductImages returns a product's images ordered by position.
func (s *Service) ListProductImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.repo.ListProductImages(ctx, productID)
}

// ListProductImagesByProductIDs returns images grouped by product ID.
func (s *Service) ListProductImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	return s.repo.ListProductImagesByProductIDs(ctx, productIDs)
}

// DeleteProductImage removes an image from a product and deletes its S3 object.
// The database delete is authoritative; a failed S3 cleanup leaves an orphaned
// object but does not fail the request.
func (s *Service) DeleteProductImage(ctx context.Context, productID, imageID string) (domain.ProductImage, error) {
	deleted, err := s.repo.DeleteProductImage(ctx, productID, imageID)
	if err != nil {
		return domain.ProductImage{}, err
	}

	if err := s.s3Client.Delete(ctx, deleted.ObjectKey); err != nil {
		logger.FromContext(ctx).Warn("product image cleanup failed",
			"operation", "s3_delete",
			"product_id", productID,
			"image_id", imageID,
			"key", deleted.ObjectKey,
			"error", fmt.Errorf("delete product image object: %w", err),
		)
	}
	return deleted, nil
}
