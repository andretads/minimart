package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeBuilders(t *testing.T) {
	t.Run("OK carries data and success", func(t *testing.T) {
		got := OK(map[string]int{"id": 1})
		assert.True(t, got.Success)
		assert.Equal(t, map[string]int{"id": 1}, got.Data)
		assert.Nil(t, got.Error)
		assert.Nil(t, got.Meta)
		assert.False(t, got.Time.IsZero())
	})

	t.Run("Created carries data and success", func(t *testing.T) {
		got := Created("widget")
		assert.True(t, got.Success)
		assert.Equal(t, "widget", got.Data)
		assert.Nil(t, got.Error)
	})

	t.Run("NoContent has neither data nor error", func(t *testing.T) {
		got := NoContent()
		assert.True(t, got.Success)
		assert.Nil(t, got.Data)
		assert.Nil(t, got.Error)
	})

	t.Run("Fail carries an error body", func(t *testing.T) {
		got := Fail(http.StatusNotFound, "missing")
		assert.False(t, got.Success)
		assert.Nil(t, got.Data)
		require.NotNil(t, got.Error)
		assert.Equal(t, http.StatusNotFound, got.Error.Code)
		assert.Equal(t, "missing", got.Error.Message)
	})

	t.Run("Paginated carries data and meta", func(t *testing.T) {
		meta := Meta{Page: 2, PageSize: 10, Total: 25, TotalPages: 3}
		got := Paginated([]string{"a"}, meta)
		assert.True(t, got.Success)
		assert.Equal(t, []string{"a"}, got.Data)
		require.NotNil(t, got.Meta)
		assert.Equal(t, meta, *got.Meta)
	})
}

func TestEnvelopeJSONShape(t *testing.T) {
	t.Run("NoContent omits data, error and meta", func(t *testing.T) {
		raw, err := json.Marshal(NoContent())
		require.NoError(t, err)
		assert.NotContains(t, string(raw), `"data"`)
		assert.NotContains(t, string(raw), `"error"`)
		assert.NotContains(t, string(raw), `"meta"`)
	})

	t.Run("Fail includes error and omits data and meta", func(t *testing.T) {
		raw, err := json.Marshal(Fail(http.StatusBadRequest, "bad"))
		require.NoError(t, err)
		assert.NotContains(t, string(raw), `"data"`)
		assert.NotContains(t, string(raw), `"meta"`)

		var got Response
		require.NoError(t, json.Unmarshal(raw, &got))
		require.NotNil(t, got.Error)
		assert.Equal(t, http.StatusBadRequest, got.Error.Code)
		assert.Equal(t, "bad", got.Error.Message)
	})
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, OK(map[string]string{"id": "p1"}))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.True(t, got.Success)
	assert.Equal(t, map[string]any{"id": "p1"}, got.Data)
}
