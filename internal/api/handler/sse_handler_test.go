package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSEEventConstants(t *testing.T) {
	assert.Equal(t, "tool_status", EventToolStatus)
	assert.Equal(t, "notification", EventNotification)
	assert.Equal(t, "ingestion_progress", EventIngestionProgress)
	assert.Equal(t, "system_alert", EventSystemAlert)
	assert.Equal(t, "health_change", EventHealthChange)
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()
	assert.NotEqual(t, id1, id2, "each client ID must be unique")
	assert.Len(t, id1, 4+32, "format: 'sse-' + 32 hex chars")
	assert.Contains(t, id1, "sse-")
	assert.Contains(t, id2, "sse-")
}

func TestGenerateClientID_DifferentOnEachCall(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateClientID()
		assert.False(t, seen[id], "duplicate client ID generated: %s", id)
		seen[id] = true
	}
}

func TestExtractAPIKeyFromSSE(t *testing.T) {
	r := httptest.NewRequest("GET", "/events", nil)
	r.Header.Set("X-Aleph-Api-Key", "test-api-key-123")
	assert.Equal(t, "test-api-key-123", extractAPIKeyFromSSE(r))
}

func TestExtractAPIKeyFromSSE_Missing(t *testing.T) {
	r := httptest.NewRequest("GET", "/events", nil)
	assert.Empty(t, extractAPIKeyFromSSE(r))
}

func TestNewSSEHandler(t *testing.T) {
	h := NewSSEHandler(nil, nil)
	assert.NotNil(t, h)
	assert.NotNil(t, h.logger)
	assert.Nil(t, h.broker)
}

func TestSSEHandler_WithMetaRepo(t *testing.T) {
	h := NewSSEHandler(nil, nil)
	assert.Nil(t, h.metaRepo)
	result := h.WithMetaRepo(nil)
	assert.Same(t, h, result)
}

func TestSSEHandler_WithJWTSecret(t *testing.T) {
	h := NewSSEHandler(nil, nil)
	assert.Nil(t, h.jwtSecret)
	result := h.WithJWTSecret([]byte("secret-key"))
	assert.Same(t, h, result)
	assert.Equal(t, []byte("secret-key"), h.jwtSecret)
}

func TestSSEHandler_Broker(t *testing.T) {
	h := NewSSEHandler(nil, nil)
	assert.Nil(t, h.Broker())
}

func TestToolStatusPayload(t *testing.T) {
	p := ToolStatusPayload{
		ToolID:     "t1",
		ToolName:   "test-tool",
		Status:     "completed",
		Progress:   1.0,
		DurationMs: 42,
	}
	data, err := json.Marshal(p)
	assert.NoError(t, err)
	var decoded ToolStatusPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "t1", decoded.ToolID)
	assert.Equal(t, "test-tool", decoded.ToolName)
	assert.Equal(t, "completed", decoded.Status)
	assert.Equal(t, 1.0, decoded.Progress)
	assert.Equal(t, int64(42), decoded.DurationMs)
}

func TestNotificationPayload(t *testing.T) {
	p := NotificationPayload{
		Title:   "Test",
		Message: "Hello",
		Type:    "info",
		Link:    "/details",
	}
	data, _ := json.Marshal(p)
	var decoded NotificationPayload
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Test", decoded.Title)
	assert.Equal(t, "Hello", decoded.Message)
	assert.Equal(t, "info", decoded.Type)
	assert.Equal(t, "/details", decoded.Link)
}

func TestIngestionProgressPayload(t *testing.T) {
	p := IngestionProgressPayload{
		TaskID:      "task-1",
		TaskName:    "import",
		Progress:    0.75,
		Phase:       "parsing",
		RowsProcess: 1500,
		TotalRows:   2000,
	}
	data, _ := json.Marshal(p)
	var decoded IngestionProgressPayload
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "task-1", decoded.TaskID)
	assert.Equal(t, 0.75, decoded.Progress)
	assert.Equal(t, "parsing", decoded.Phase)
	assert.Equal(t, int64(1500), decoded.RowsProcess)
	assert.Equal(t, int64(2000), decoded.TotalRows)
}

func TestSystemAlertPayload(t *testing.T) {
	p := SystemAlertPayload{
		Severity:    "critical",
		Title:       "Disk Full",
		Description: "Storage at 99%",
		Component:   "duckdb",
	}
	data, _ := json.Marshal(p)
	var decoded SystemAlertPayload
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "critical", decoded.Severity)
	assert.Equal(t, "duckdb", decoded.Component)
}
