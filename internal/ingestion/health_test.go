package ingestion

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	db := setupTestDB(t)
	wm := NewWatermarkManager(db)
	require.NoError(t, wm.Set("election", time.Now(), "cursor_1", `{"records":1500}`))
	require.NoError(t, wm.Set("pep", time.Now().Add(-24*time.Hour), "", `{"records":500}`))

	handler := NewHealthHandler(db)

	req := httptest.NewRequest("GET", "/api/health/ingestion", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	sources := resp["sources"].([]interface{})
	assert.GreaterOrEqual(t, len(sources), 2)
	t.Logf("response: %+v", resp)
}

func TestHealthEndpointEmpty(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHealthHandler(db)

	req := httptest.NewRequest("GET", "/api/health/ingestion", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	sources := resp["sources"].([]interface{})
	assert.Equal(t, 0, len(sources))
	assert.Equal(t, 0, int(resp["total_sources"].(float64)))
}
