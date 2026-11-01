package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProduct(t *testing.T) {
	tests := []struct {
		name        string
		productName string
		priceCents  int64
		stock       int
		categoryID  string
		sku         string
		description string
		details     []ProductDetail
		wantName    string
		wantErr     error
	}{
		{"valid", "Widget", 500, 3, "cat-1", "", "", nil, "Widget", nil},
		{"trims surrounding space", "  Widget  ", 500, 0, "cat-1", "", "", nil, "Widget", nil},
		{"empty name", "   ", 500, 0, "cat-1", "", "", nil, "", ErrEmptyName},
		{"zero price", "Widget", 0, 0, "cat-1", "", "", nil, "", ErrNonPositivePrice},
		{"negative price", "Widget", -1, 0, "cat-1", "", "", nil, "", ErrNonPositivePrice},
		{"negative stock", "Widget", 500, -1, "cat-1", "", "", nil, "", ErrNegativeStock},
		{"missing category", "Widget", 500, 3, "", "", "", nil, "", ErrProductCategoryRequired},
		{"valid with SKU", "Widget", 500, 3, "cat-1", "SKU-123", "Description", nil, "Widget", nil},
		{"valid with description", "Widget", 500, 3, "cat-1", "", "Description", nil, "Widget", nil},
		{"valid with details", "Widget", 500, 3, "cat-1", "", "", []ProductDetail{{Name: "Color", Value: "Red"}}, "Widget", nil},
		{"empty SKU", "Widget", 500, 3, "cat-1", "   ", "", nil, "", ErrEmptySKU},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProduct("id-1", tt.productName, tt.priceCents, tt.stock, tt.categoryID, tt.sku, tt.description, tt.details)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "id-1", p.ID)
			assert.Equal(t, tt.wantName, p.Name)
			assert.Equal(t, tt.priceCents, p.PriceCents)
			assert.Equal(t, tt.stock, p.Stock)
			assert.Equal(t, 1, p.Version, "a new product starts at version 1")
			assert.False(t, p.CreatedAt.IsZero(), "CreatedAt is stamped on creation")
			assert.False(t, p.UpdatedAt.IsZero(), "UpdatedAt is stamped on creation")
			assert.True(t, p.CreatedAt.Equal(p.UpdatedAt), "CreatedAt and UpdatedAt start equal")
			assert.Equal(t, tt.categoryID, p.CategoryID)
			if tt.sku != "" && tt.sku != "   " {
				assert.Equal(t, tt.sku, p.SKU)
			}
			if tt.description != "" {
				assert.Equal(t, tt.description, p.Description)
			}
			if tt.details != nil {
				assert.Equal(t, tt.details, p.Details)
			}
		})
	}
}
