package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type SessionHandler struct {
	metaRepo *repository.MetadataRepository
}

func NewSessionHandler(metaRepo *repository.MetadataRepository) *SessionHandler {
	return &SessionHandler{metaRepo: metaRepo}
}

type createSessionRequest struct {
	APIKey string `json:"api_key"`
}

type createSessionResponse struct {
	ProjectID string `json:"project_id"`
}

func (h *SessionHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.APIKey == "" {
		http.Error(w, `{"error":"api_key is required"}`, http.StatusBadRequest)
		return
	}

	projectID, _, err := middleware.ValidateAPIKey(h.metaRepo, req.APIKey)
	if err != nil {
		http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "aleph_session",
		Value:    req.APIKey,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   86400,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createSessionResponse{ProjectID: projectID})
}

func (h *SessionHandler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "aleph_session",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *SessionHandler) HandleValidateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	apiKey := middleware.ExtractAPIKeyFromCookie(r)
	if apiKey == "" {
		http.Error(w, `{"error":"no session"}`, http.StatusUnauthorized)
		return
	}
	projectID, _, err := middleware.ValidateAPIKey(h.metaRepo, apiKey)
	if err != nil {
		http.Error(w, `{"error":"invalid session"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createSessionResponse{ProjectID: projectID})
}
