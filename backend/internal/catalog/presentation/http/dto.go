// Package http is the catalog's input adapter for HTTP. It translates between the
// wire (JSON) and the application layer, and never contains business rules.
package http

import (
	"minimart/internal/catalog/domain"
	"time"
)

// createProductRequest is the JSON body accepted by POST /v1/products.
type createProductRequest struct {
	Name        string                 `json:"name" form:"name"`
	PriceCents  int64                  `json:"price_cents" form:"price_cents"`
	Stock       int                    `json:"stock" form:"stock"`
	CategoryID  string                 `json:"category_id" form:"category_id"`
	SKU         string                 `json:"sku" form:"sku"`
	Description string                 `json:"description" form:"description"`
	Details     []domain.ProductDetail `json:"details" form:"details"`
} //	@name	CatalogCreateProductRequest

// updateProductRequest is the JSON body accepted by PATCH /v1/products/{id}.
// Stock is deliberately absent: this use case changes only name, price, and category.
// Version is required and must be the value last read from GET - it is the
// optimistic lock the write is conditioned on.
type updateProductRequest struct {
	Name        string                 `json:"name" form:"name"`
	PriceCents  int64                  `json:"price_cents" form:"price_cents"`
	Version     int                    `json:"version" form:"version"`
	CategoryID  string                 `json:"category_id" form:"category_id"`
	SKU         string                 `json:"sku" form:"sku"`
	Description string                 `json:"description" form:"description"`
	Details     []domain.ProductDetail `json:"details" form:"details"`
} //	@name	CatalogUpdateProductRequest

// productResponse is the JSON returned to clients. It is kept separate from the
// domain entity so the wire contract stays stable even if the entity changes.
// Version is the optimistic lock a client reads back before its next update.
// Images are ordered by position; the first image is the primary image.
type productResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	PriceCents   int64                  `json:"price_cents"`
	Stock        int                    `json:"stock"`
	Version      int                    `json:"version"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CategoryID   string                 `json:"category_id"`
	SKU          string                 `json:"sku,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Details      []domain.ProductDetail `json:"details,omitempty"`
	TotalRating  int64                  `json:"total_rating,omitempty"`
	TotalReviews int64                  `json:"total_reviews,omitempty"`
	Images       []productImageResponse `json:"images"`
} //	@name	CatalogProduct

// productImageResponse is the JSON returned for a product image. URL is a
// short-lived presigned URL; Position determines display order and the lowest
// position is the primary image.
type productImageResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
} //	@name	CatalogProductImage

// reviewRequest is the JSON body accepted by POST /v1/products/{id}/reviews.
// The customer is taken from the authenticated session, not from the body.
type reviewRequest struct {
	Description string `json:"description"`
	Rating      int    `json:"rating"`
} //	@name	CatalogAddReviewRequest

// reviewResponse is the JSON returned for a review.
type reviewResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	CustomerID  string    `json:"customer_id"`
	Description string    `json:"description"`
	Rating      int       `json:"rating"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
} //	@name	CatalogReviewResponse

func toProductResponse(p domain.Product, images []productImageResponse) productResponse {
	return productResponse{
		ID:           p.ID,
		Name:         p.Name,
		PriceCents:   p.PriceCents,
		Stock:        p.Stock,
		Version:      p.Version,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		CategoryID:   p.CategoryID,
		SKU:          p.SKU,
		Description:  p.Description,
		Details:      p.Details,
		TotalRating:  p.TotalRating,
		TotalReviews: p.TotalReviews,
		Images:       images,
	}
}

func toProductImageResponse(img domain.ProductImage, url string) productImageResponse {
	return productImageResponse{
		ID:          img.ID,
		ProductID:   img.ProductID,
		URL:         url,
		ContentType: img.ContentType,
		SizeBytes:   img.SizeBytes,
		Position:    img.Position,
		CreatedAt:   img.CreatedAt,
		UpdatedAt:   img.UpdatedAt,
	}
}

func toReviewResponse(r domain.Review) reviewResponse {
	return reviewResponse{
		ID:          r.ID,
		ProductID:   r.ProductID,
		CustomerID:  r.CustomerID,
		Description: r.Description,
		Rating:      r.Rating,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
}
