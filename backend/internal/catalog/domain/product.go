package domain

import (
	"strings"
	"time"
)

// ProductDetail represents a key-value attribute of a product (e.g., Color: Red, Size: Large).
type ProductDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Review status constants.
const (
	ReviewStatusPending     = "pending"
	ReviewStatusApproved    = "approved"
	ReviewStatusRejected    = "rejected"
	ReviewStatusNeedsReview = "needs_review"
)

// Moderation decision constants.
const (
	ModerationDecisionApproved    = "approved"
	ModerationDecisionRejected    = "rejected"
	ModerationDecisionNeedsReview = "needs_review"
)

// Review represents a customer review for a product.
type Review struct {
	ID                     string
	ProductID              string
	CustomerID             string
	Description            string
	Rating                 int
	Status                 string
	CreatedAt              time.Time
	ModeratedAt            *time.Time
	ModerationDecision     string
	ModerationConfidence   float64
	ModerationCategories   []string
	ModerationExplanation  string
	ModerationModel        string
	ModerationModelVersion string
}

// RatingSummary is the denormalized aggregate rating for a product.
type RatingSummary struct {
	ProductID     string
	TotalReviews  int64
	RatingSum     int64
	AverageRating float64
	UpdatedAt     time.Time
}

// Product is a catalog entry. Prices are integer cents - never floats - so money
// arithmetic is exact. Code outside tests should create Product values through
// NewProduct, so normal application flow is valid by construction.
//
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every update, which rejects writes carrying a stale version.
// CreatedAt is stamped once when the product is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
//
// CategoryID links the product to a category. It is required and validated
// against the category contract.
//
// SKU is an optional unique identifier for the product.
// Description is an optional detailed product description.
// Details is a list of flexible key-value attributes (e.g., Color: Red).
// TotalRating and TotalReviews are denormalized counters for read performance.
//
// Product images are a separate entity linked by ProductID; see ProductImage.
type Product struct {
	ID           string
	Name         string
	PriceCents   int64
	Stock        int
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CategoryID   string
	SKU          string
	Description  string
	Details      []ProductDetail
	TotalRating  int64
	TotalReviews int64
}

// NewProduct validates its inputs and returns a Product, or a domain error.
func NewProduct(id, name string, priceCents int64, stock int, categoryID string, sku, description string, details []ProductDetail) (Product, error) {
	name, err := validateName(name)
	if err != nil {
		return Product{}, err
	}
	if priceCents <= 0 {
		return Product{}, ErrNonPositivePrice
	}
	if stock < 0 {
		return Product{}, ErrNegativeStock
	}
	if categoryID == "" {
		return Product{}, ErrProductCategoryRequired
	}
	if sku != "" {
		sku = strings.TrimSpace(sku)
		if sku == "" {
			return Product{}, ErrEmptySKU
		}
	}
	now := time.Now().UTC()
	return Product{
		ID:           id,
		Name:         name,
		PriceCents:   priceCents,
		Stock:        stock,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
		CategoryID:   categoryID,
		SKU:          sku,
		Description:  description,
		Details:      details,
		TotalRating:  0,
		TotalReviews: 0,
	}, nil
}

// UpdateNameAndPrice returns a copy of the product with a validated name and
// price, leaving the other fields (ID, Stock, Version, CreatedAt, UpdatedAt)
// untouched - version and timestamps only move when the storage layer persists
// the write.
func (p Product) UpdateNameAndPrice(name string, priceCents int64) (Product, error) {
	name, err := validateName(name)
	if err != nil {
		return Product{}, err
	}
	if priceCents <= 0 {
		return Product{}, ErrNonPositivePrice
	}
	p.Name = name
	p.PriceCents = priceCents
	return p, nil
}

// UpdateCategory returns a copy of the product with a new category ID.
// The category must not be empty. Version and timestamps are left untouched.
func (p Product) UpdateCategory(categoryID string) (Product, error) {
	if categoryID == "" {
		return Product{}, ErrProductCategoryRequired
	}
	p.CategoryID = categoryID
	return p, nil
}

// UpdateDescription returns a copy of the product with a new description.
// Version and timestamps are left untouched.
func (p Product) UpdateDescription(description string) Product {
	p.Description = description
	return p
}

// UpdateSKU returns a copy of the product with a new SKU.
// The SKU must not be empty if provided. Version and timestamps are left untouched.
func (p Product) UpdateSKU(sku string) (Product, error) {
	if sku != "" {
		sku = strings.TrimSpace(sku)
		if sku == "" {
			return Product{}, ErrEmptySKU
		}
	}
	p.SKU = sku
	return p, nil
}

// UpdateDetails returns a copy of the product with new details.
// Version and timestamps are left untouched.
func (p Product) UpdateDetails(details []ProductDetail) Product {
	p.Details = details
	return p
}

// Reserve returns a copy of the product with qty units removed from stock. It
// reports ErrInvalidReservation for a non-positive quantity and
// ErrInsufficientStock when fewer than qty units are in stock. Version and
// timestamps are left untouched - only the storage layer moves those when it
// persists the change.
func (p Product) Reserve(qty int) (Product, error) {
	if qty <= 0 {
		return Product{}, ErrInvalidReservation
	}
	if qty > p.Stock {
		return Product{}, ErrInsufficientStock
	}
	p.Stock -= qty
	return p, nil
}

// Restore returns a copy of the product with qty units added back to stock. It
// reports ErrInvalidReservation for a non-positive quantity. It is used to undo
// a reservation when an order will not be fulfilled.
func (p Product) Restore(qty int) (Product, error) {
	if qty <= 0 {
		return Product{}, ErrInvalidReservation
	}
	p.Stock += qty
	return p, nil
}

// validateName trims surrounding space and rejects an empty result.
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	return name, nil
}
