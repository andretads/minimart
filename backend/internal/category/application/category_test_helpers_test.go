package application

import (
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, metrics.Noop{}), repo
}

func validCreateCategoryRequest() CreateCategoryRequest {
	return CreateCategoryRequest{
		Name: "Electronics",
		Icon: "📱",
	}
}

func validUpdateCategoryRequest(id string, version int) UpdateCategoryRequest {
	return UpdateCategoryRequest{
		ID:      id,
		Name:    "Gadgets",
		Icon:    "🔧",
		Version: version,
	}
}

func createTestCategory(t *testing.T, svc *Service, req CreateCategoryRequest) domain.Category {
	t.Helper()
	c, err := svc.CreateCategory(t.Context(), req)
	require.NoError(t, err)
	return c
}
