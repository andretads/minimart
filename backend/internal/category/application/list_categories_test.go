package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCategories(t *testing.T) {
	t.Run("Empty storage returns an empty list", func(t *testing.T) {
		svc, _ := newTestService(t)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("All stored categories are returned ordered by name", func(t *testing.T) {
		svc, _ := newTestService(t)

		createTestCategory(t, svc, validCreateCategoryRequest())

		booksReq := validCreateCategoryRequest()
		booksReq.Name = "Books"
		booksReq.Icon = "📚"
		createTestCategory(t, svc, booksReq)

		clothingReq := validCreateCategoryRequest()
		clothingReq.Name = "Clothing"
		clothingReq.Icon = "👕"
		createTestCategory(t, svc, clothingReq)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "Books", got[0].Name)
		assert.Equal(t, "Clothing", got[1].Name)
		assert.Equal(t, "Electronics", got[2].Name)
	})
}
