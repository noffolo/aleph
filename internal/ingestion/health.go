package ingestion

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// HealthHandler serves /api/health/ingestion with per-source status.
type HealthHandler struct {
	db *sql.DB
	wm *WatermarkManager
}

// NewHealthHandler creates a HealthHandler backed by the given DB.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db, wm: NewWatermarkManager(db)}
}

// SourceHealth represents a single source's health status.
type SourceHealth struct {
	SourceName  string    `json:"source_name"`
	LastRun     time.Time `json:"last_run"`
	Status      string    `json:"status"`
	Cursor      string    `json:"cursor,omitempty"`
}

// ServeHTTP returns a JSON response with the health of all ingestion sources.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	watermarks, err := h.wm.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sources := make([]SourceHealth, 0, len(watermarks))
	for _, wm := range watermarks {
		sh := SourceHealth{
			SourceName: wm.SourceName,
			LastRun:    wm.LastRun,
			Status:     "healthy",
			Cursor:     wm.Cursor,
		}
		if time.Since(wm.LastRun) > 7*24*time.Hour {
			sh.Status = "stale"
		}
		sources = append(sources, sh)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources":       sources,
		"total_sources": len(sources),
		"timestamp":     time.Now().UTC(),
	})
}
