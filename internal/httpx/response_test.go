package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"komecore/internal/httpx"
	"komecore/internal/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaginatedResponse_EmptyItemsNotNull(t *testing.T) {
	meta := pagination.NewPaginationMeta(1, 10, 0)

	// Case 1: nil slice passed
	respNil := httpx.NewPaginatedResponse[string](nil, meta)
	bytesNil, err := json.Marshal(respNil)
	require.NoError(t, err)

	expectedJSON := `{"items":[],"pagination":{"page":1,"limit":10,"total_items":0,"total_pages":0,"next_cursor":null}}`
	assert.JSONEq(t, expectedJSON, string(bytesNil), "nil items must serialize to [] and never null")

	// Case 2: empty slice passed
	emptyItems := make([]string, 0)
	respEmpty := httpx.NewPaginatedResponse(emptyItems, meta)
	bytesEmpty, err := json.Marshal(respEmpty)
	require.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(bytesEmpty))

	// Case 3: populated slice
	populated := []string{"item1", "item2"}
	metaPopulated := pagination.NewPaginationMeta(1, 10, 2)
	respPopulated := httpx.NewPaginatedResponse(populated, metaPopulated)
	bytesPopulated, err := json.Marshal(respPopulated)
	require.NoError(t, err)

	expectedPopulatedJSON := `{"items":["item1","item2"],"pagination":{"page":1,"limit":10,"total_items":2,"total_pages":1,"next_cursor":null}}`
	assert.JSONEq(t, expectedPopulatedJSON, string(bytesPopulated))
}

func TestToUTCPtr(t *testing.T) {
	// Nil pointer remains nil
	assert.Nil(t, httpx.ToUTCPtr(nil))

	// Local / non-UTC time converts to UTC
	loc := time.FixedZone("WIB", 7*3600)
	localTime := time.Date(2026, 10, 15, 15, 30, 0, 0, loc)
	utcPtr := httpx.ToUTCPtr(&localTime)

	require.NotNil(t, utcPtr)
	assert.Equal(t, time.UTC, utcPtr.Location())
	assert.Equal(t, "2026-10-15T08:30:00Z", utcPtr.Format(time.RFC3339))
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]string{"status": "ok"}
	httpx.WriteJSON(w, http.StatusCreated, payload)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}
