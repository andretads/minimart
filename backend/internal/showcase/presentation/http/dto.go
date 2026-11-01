package http

import (
	"minimart/internal/showcase/domain"
	"time"
)

// productResponse is the public JSON representation of a showcase product.
// List responses include image_url (the primary image); detail responses
// include the full images array.
type productResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	PriceCents   int64                  `json:"price_cents"`
	CategoryID   string                 `json:"category_id"`
	Stock        int                    `json:"stock"`
	SKU          string                 `json:"sku,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Details      []domain.ProductDetail `json:"details,omitempty"`
	TotalRating  int64                  `json:"total_rating"`
	TotalReviews int64                  `json:"total_reviews"`
	ImageURL     string                 `json:"image_url,omitempty"`
	Images       []productImageResponse `json:"images,omitempty"`
} //	@name	ShowcaseProduct

// productImageResponse is the public JSON representation of a product image.
// URL is a short-lived presigned GET URL.
type productImageResponse struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Position    int    `json:"position"`
} //	@name	ShowcaseProductImage

// categoryResponse is the public JSON representation of a category.
type categoryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
} //	@name	ShowcaseCategory

// reviewResponse is the public JSON representation of an approved review.
type reviewResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	CustomerID  string    `json:"customer_id"`
	Description string    `json:"description"`
	Rating      int       `json:"rating"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
} //	@name	ShowcaseProductReview

func toProductSummaryResponse(p domain.Product, primaryImageURL string) productResponse {
	resp := baseProductResponse(p)
	resp.ImageURL = primaryImageURL
	return resp
}

func toProductDetailResponse(p domain.Product, images []productImageResponse) productResponse {
	resp := baseProductResponse(p)
	if len(images) > 0 {
		resp.ImageURL = images[0].URL
		resp.Images = images
	}
	return resp
}

func baseProductResponse(p domain.Product) productResponse {
	return productResponse{
		ID:           p.ID,
		Name:         p.Name,
		PriceCents:   p.PriceCents,
		CategoryID:   p.CategoryID,
		Stock:        p.Stock,
		SKU:          p.SKU,
		Description:  p.Description,
		Details:      p.Details,
		TotalRating:  p.TotalRating,
		TotalReviews: p.TotalReviews,
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
