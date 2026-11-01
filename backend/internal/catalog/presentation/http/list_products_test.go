package http

import (
	"encoding/json"
	"minimart/internal/catalog/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	mux, repo := newTestMux(t)
	for _, p := range []struct {
		id, name string
	}{{"p1", "Widget"}, {"p2", "Gadget"}} {
		product, err := domain.NewProduct(p.id, p.name, 100, 1, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), product))
	}

	rec := do(t, mux, http.MethodGet, "/v1/products", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Data []productResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Len(t, env.Data, 2)
}
