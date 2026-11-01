package application

import (
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, newFakeIdem(), approvedProvider(), metrics.Noop{})

	created, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	got, err := svc.GetPayment(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	_, err = svc.GetPayment(t.Context(), "missing")
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}
