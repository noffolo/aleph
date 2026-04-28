package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// SSEHandler provides an HTTP handler for Server-Sent Events streaming.
// It uses the SSE broker to manage client connections and broadcast events.
//
// SSE is used here (rather than gRPC streaming via connect-go) because:
//   - Notification delivery is unidirectional (server→client only)
//   - EventSource is native in browsers — no gRPC-Web transport needed
//   - Auto-reconnect with Last-Event-ID is built into the browser
//
// gRPC streaming (connect-go) sources like Chat and StreamPredictions
// remain unchanged — they require bidirectional or protocol-level semantics.
type SSEHandler struct {
	broker   *sse.Broker
	logger   *slog.Logger
	metaRepo *repository.MetadataRepository
}

// NewSSEHandler creates a new SSEHandler with the given broker and logger.
func NewSSEHandler(broker *sse.Broker, logger *slog.Logger) *SSEHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SSEHandler{broker: broker, logger: logger}
}

// WithMetaRepo sets the MetadataRepository for API key validation.
func (h *SSEHandler) WithMetaRepo(repo *repository.MetadataRepository) *SSEHandler {
	h.metaRepo = repo
	return h
}

// Stream is the HTTP handler for SSE connections.
// GET /api/v1/events — opens an SSE stream for real-time notifications.
//
// Authentication uses the X-Aleph-Api-Key header.
// Events include: tool_status, notification, ingestion_progress, system_alert.
func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate authentication
	if !isAuthenticatedForSSE(r, h.metaRepo) {
		http.Error(w, "unauthorized — valid X-Aleph-Api-Key header required", http.StatusUnauthorized)
		return
	}

	// Generate a unique client ID for this connection
	clientID := generateClientID()

	client := h.broker.Subscribe(clientID)

	// Ensure cleanup on disconnect
	defer h.broker.Unsubscribe(clientID)

	h.logger.Info("sse client connected",
		"client_id", clientID,
		"remote", r.RemoteAddr,
		"total_clients", h.broker.ClientCount())

	if err := sse.StreamEvents(w, r, client, h.logger); err != nil {
		h.logger.Warn("sse stream error",
			"client_id", clientID,
			"error", err)
	}

	h.logger.Info("sse client disconnected",
		"client_id", clientID,
		"total_clients", h.broker.ClientCount())
}

// Broker returns the underlying SSE broker so callers can publish events.
func (h *SSEHandler) Broker() *sse.Broker {
	return h.broker
}

// Event types published via SSE. These are the "event:" field values
// that the frontend EventSource addEventListener listens for.
const (
	EventToolStatus         = "tool_status"
	EventNotification       = "notification"
	EventIngestionProgress  = "ingestion_progress"
	EventSystemAlert        = "system_alert"
	EventHealthChange       = "health_change"
)

// ToolStatusPayload is published via SSE when a tool execution status changes.
type ToolStatusPayload struct {
	ToolID     string      `json:"tool_id"`
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"` // started, running, completed, failed
	Progress   float64     `json:"progress,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"duration_ms,omitempty"`
}

// NotificationPayload is published via SSE for general notifications.
type NotificationPayload struct {
	Title   string      `json:"title"`
	Message string      `json:"message"`
	Type    string      `json:"type"` // info, success, warning, error
	Link    string      `json:"link,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// IngestionProgressPayload is published via SSE when an ingestion task progresses.
type IngestionProgressPayload struct {
	TaskID      string  `json:"task_id"`
	TaskName    string  `json:"task_name"`
	Progress    float64 `json:"progress"`       // 0.0-1.0
	Phase       string  `json:"phase"`          // e.g. "downloading", "parsing", "importing"
	RowsProcess int64   `json:"rows_processed"`
	TotalRows   int64   `json:"total_rows,omitempty"`
}

// SystemAlertPayload is published via SSE for system-level alerts.
type SystemAlertPayload struct {
	Severity    string `json:"severity"` // critical, warning, info
	Title       string `json:"title"`
	Description string `json:"description"`
	Component   string `json:"component"`
}

// generateClientID creates a unique SSE client identifier.
func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("sse-%s", hex.EncodeToString(b))
}

// extractAPIKeyFromSSE extracts the API key from the X-Aleph-Api-Key header.
// Query parameter auth (?api_key=...) was removed for security (W11-02).
func extractAPIKeyFromSSE(r *http.Request) string {
	return r.Header.Get("X-Aleph-Api-Key")
}

// isAuthenticatedForSSE checks if the request has valid authentication.
// Validates the X-Aleph-Api-Key header against the repository.
func isAuthenticatedForSSE(r *http.Request, metaRepo *repository.MetadataRepository) bool {
	key := extractAPIKeyFromSSE(r)
	if key == "" {
		return false
	}
	_, err := middleware.ValidateAPIKey(metaRepo, key)
	return err == nil
}
