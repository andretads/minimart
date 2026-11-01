package domain

import "time"

// ProductDetail represents a key-value attribute of a product (e.g., Color: Red).
type ProductDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ProductImage is the public view of a stored product image. ObjectKey is the
// storage key; the HTTP layer turns it into a short-lived presigned URL.
type ProductImage struct {
	ID          string
	ProductID   string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Position    int
}

// Review is the public view of an approved product review.
type Review struct {
	ID          string
	ProductID   string
	CustomerID  string
	Description string
	Rating      int
	Status      string
	CreatedAt   time.Time
}

// Product is the public read-model view of a catalog product exposed by the
// showcase domain. It contains only the fields intended for public display.
// The showcase domain owns its own read queries over the shared products,
// product_images, and product_reviews tables; it does not depend on the catalog
// contract.
type Product struct {
	ID           string
	Name         string
	PriceCents   int64
	CategoryID   string
	Stock        int
	SKU          string
	Description  string
	Details      []ProductDetail
	TotalRating  int64
	TotalReviews int64
	Images       []ProductImage
	CreatedAt    time.Time
}
