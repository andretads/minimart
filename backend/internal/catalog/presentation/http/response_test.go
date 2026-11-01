package http

import (
	"encoding/json"
	"minimart/internal/catalog/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRespondDomainErrorOptimisticLock covers the conflict mapping directly at
// the edge; the end-to-end stale-version path is exercised by the
// "stale version" case in TestUpdateProductStatusMapping.
func TestRespondDomainErrorOptimisticLock(t *testing.T) {
	rec := httptest.NewRecorder()
	respondDomainError(rec, domain.ErrProductOptimisticLock)

	require.Equal(t, http.StatusConflict, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.False(t, env.Success)
	require.NotNil(t, env.Error)
	assert.Equal(t, http.StatusConflict, env.Error.Code)
	assert.Equal(t, domain.ErrProductOptimisticLock.Error(), env.Error.Message)
}
