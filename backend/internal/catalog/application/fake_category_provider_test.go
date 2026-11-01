package application

import (
	"context"
	"minimart/internal/category/presentation/module/client"
)

// fakeCategoryProvider is a test double for CategoryProvider.
type fakeCategoryProvider struct {
	err error
}

func (f fakeCategoryProvider) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	if f.err != nil {
		return client.CategoryDTO{}, f.err
	}
	return client.CategoryDTO{ID: id, Name: "Test Category"}, nil
}
