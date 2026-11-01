package domain

import (
	"strings"
	"time"
)

// ProductImage is an image attached to a product. The binary lives in S3 and
// only its metadata is stored here. A product may have many images; the lowest
// position is the primary image.
//
// Code outside tests should create ProductImage values through NewProductImage,
// so normal application flow is valid by construction.
//
// Position is assigned by the storage layer when the image is saved, keeping
// concurrent uploads in a deterministic order.
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every persisted write. It is not set by domain logic.
// CreatedAt is stamped once when the image is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
type ProductImage struct {
	ID          string
	ProductID   string
	ObjectKey   string
	Bucket      string
	ContentType string
	SizeBytes   int64
	Position    int
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProductImage validates the inputs and returns a ProductImage, or a domain
// error. productID, objectKey, and bucket must be non-empty. contentType must
// be a supported image type. sizeBytes must be positive.
func NewProductImage(productID, objectKey, bucket, contentType string, sizeBytes int64) (ProductImage, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ProductImage{}, ErrEmptyProductID
	}

	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return ProductImage{}, ErrEmptyObjectKey
	}

	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ProductImage{}, ErrEmptyBucket
	}

	if !IsSupportedImageType(contentType) {
		return ProductImage{}, ErrInvalidImageFormat
	}

	if sizeBytes <= 0 {
		return ProductImage{}, ErrInvalidImageSize
	}

	now := time.Now().UTC()
	return ProductImage{
		ProductID:   productID,
		ObjectKey:   objectKey,
		Bucket:      bucket,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// IsSupportedImageType reports whether the content type is a supported image format.
func IsSupportedImageType(contentType string) bool {
	return contentType == "image/jpeg" ||
		contentType == "image/png" ||
		contentType == "image/webp" ||
		contentType == "image/gif"
}
