package application

import (
	"context"
	"fmt"
	"io"
	"minimart/internal/catalog/domain"
)

// ImageUpload is an inbound image stream supplied to a use case.
type ImageUpload struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Size        int64
}

// validateImage checks that the upload has a supported content type and a size
// within the configured limit.
func (s *Service) validateImage(img *ImageUpload) error {
	if img == nil {
		return nil
	}
	if !domain.IsSupportedImageType(img.ContentType) {
		return domain.ErrInvalidImageFormat
	}
	if img.Size <= 0 {
		return domain.ErrInvalidImageSize
	}
	if img.Size > s.maxImageSize {
		return domain.ErrImageTooLarge
	}
	return nil
}

// storeImage uploads the image to S3 and returns the object key.
func (s *Service) storeImage(ctx context.Context, productID, imageID string, img *ImageUpload) (string, error) {
	if err := s.validateImage(img); err != nil {
		return "", err
	}
	key := "products/" + productID + "/images/" + imageID + "/" + sanitizeFilename(img.Filename)
	if err := s.s3Client.Upload(ctx, key, img.Reader, img.ContentType, img.Size); err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}
	return key, nil
}

// sanitizeFilename removes potentially problematic characters from a filename.
func sanitizeFilename(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
