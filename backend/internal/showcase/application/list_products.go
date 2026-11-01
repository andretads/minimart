package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"minimart/internal/showcase/domain"
	"time"
)

// ListProductsRequest carries pagination parameters.
// PageSize: number of items per page (default 20, max 100).
// Cursor: opaque keyset token produced by a previous response; empty for the
// first page.
// CategoryID: optional category filter.
type ListProductsRequest struct {
	PageSize   int
	Cursor     string
	CategoryID string
}

// ListProductsResult wraps a page of products with keyset metadata.
type ListProductsResult struct {
	Items      []domain.Product
	PageSize   int
	NextCursor string
	HasMore    bool
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ListProducts returns a page of products ordered newest-first. It fetches
// PageSize+1 rows so the presence of an extra row tells whether a next page
// exists without an additional COUNT query.
func (s *Service) ListProducts(ctx context.Context, req ListProductsRequest) (ListProductsResult, error) {
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 1 || pageSize > maxPageSize {
		return ListProductsResult{}, domain.ErrInvalidPageSize
	}

	var after *domain.Cursor
	if req.Cursor != "" {
		c, err := DecodeCursor(req.Cursor)
		if err != nil {
			return ListProductsResult{}, domain.ErrInvalidCursor
		}
		after = &c
	}

	var items []domain.Product
	var err error
	if req.CategoryID != "" {
		items, err = s.repo.ListByCategory(ctx, req.CategoryID, pageSize+1, after)
	} else {
		items, err = s.repo.List(ctx, pageSize+1, after)
	}
	if err != nil {
		return ListProductsResult{}, err
	}

	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}

	if len(items) > 0 {
		productIDs := make([]string, 0, len(items))
		for _, p := range items {
			productIDs = append(productIDs, p.ID)
		}
		imagesByProduct, err := s.repo.ListImagesByProductIDs(ctx, productIDs)
		if err != nil {
			return ListProductsResult{}, err
		}
		for i := range items {
			items[i].Images = imagesByProduct[items[i].ID]
		}
	}

	result := ListProductsResult{
		Items:    items,
		PageSize: pageSize,
		HasMore:  hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		result.NextCursor = EncodeCursor(domain.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return result, nil
}

// cursorPayload is the JSON-encoded cursor body.
type cursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// EncodeCursor serializes and base64-encodes a cursor into an opaque token.
func EncodeCursor(c domain.Cursor) string {
	b, err := json.Marshal(cursorPayload{CreatedAt: c.CreatedAt, ID: c.ID})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor decodes and validates an opaque cursor token.
func DecodeCursor(token string) (domain.Cursor, error) {
	if token == "" {
		return domain.Cursor{}, errors.New("empty cursor")
	}
	if len(token) > 1024 {
		return domain.Cursor{}, errors.New("cursor too long")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return domain.Cursor{}, err
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return domain.Cursor{}, err
	}
	if p.ID == "" || p.CreatedAt.IsZero() {
		return domain.Cursor{}, errors.New("invalid cursor payload")
	}
	return domain.Cursor{CreatedAt: p.CreatedAt, ID: p.ID}, nil
}
