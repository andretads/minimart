package http

import (
	"minimart/internal/showcase/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetProduct(t *testing.T) {
	mux, repo := newTestMux(t)

	product := domain.Product{ID: "p1", Name: "Product 1", PriceCents: 100}
	seedProduct(t, repo, product)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantSuccess bool
		wantID      string
	}{
		{
			name:        "found",
			path:        "/v1/showcase/products/p1",
			wantStatus:  200,
			wantSuccess: true,
			wantID:      "p1",
		},
		{
			name:        "not found",
			path:        "/v1/showcase/products/missing",
			wantStatus:  404,
			wantSuccess: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, "GET", tc.path)
			env := assertJSONResponse(t, rec, tc.wantStatus, tc.wantSuccess)

			if tc.wantSuccess {
				var p productResponse
				unmarshalData(t, env, &p)
				require.Equal(t, tc.wantID, p.ID)
				require.Equal(t, "Product 1", p.Name)
				require.Equal(t, int64(100), p.PriceCents)
			} else {
				require.NotNil(t, env.Error)
			}
		})
	}
}
