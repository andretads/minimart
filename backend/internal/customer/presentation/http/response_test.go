package http

import (
	"encoding/json"
	"minimart/internal/customer/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRespondDomainErrorOptimisticLock covers the conflict mapping directly at
// the edge; the end-to-end stale-version path is exercised by the
// "stale version" case in TestUpdateCustomer.
func TestRespondDomainErrorOptimisticLock(t *testing.T) {
	rec := httptest.NewRecorder()
	respondDomainError(rec, domain.ErrCustomerOptimisticLock)

	require.Equal(t, http.StatusConflict, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.False(t, env.Success)
	require.NotNil(t, env.Error)
	assert.Equal(t, http.StatusConflict, env.Error.Code)
	assert.Equal(t, domain.ErrCustomerOptimisticLock.Error(), env.Error.Message)
}

func TestRespondDomainErrorInvalidVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	respondDomainError(rec, domain.ErrInvalidVersion)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.False(t, env.Success)
	require.NotNil(t, env.Error)
	assert.Equal(t, http.StatusBadRequest, env.Error.Code)
	assert.Equal(t, domain.ErrInvalidVersion.Error(), env.Error.Message)
}
