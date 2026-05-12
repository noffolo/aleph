package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type SessionHandler struct {
	metaRepo       *repository.MetadataRepository
	jwtSecret      []byte
	revocationStore *middleware.TokenRevocationStore
}

func NewSessionHandler(metaRepo *repository.MetadataRepository, jwtSecret []byte) *SessionHandler {
	return &SessionHandler{metaRepo: metaRepo, jwtSecret: jwtSecret}
}

func (h *SessionHandler) WithRevocationStore(store *middleware.TokenRevocationStore) *SessionHandler {
	h.revocationStore = store
	return h
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

	projectID, role, err := middleware.ValidateAPIKey(h.metaRepo, req.APIKey)
	if err != nil {
		http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
		return
	}

	maskedKey := maskAPIKey(req.APIKey)

	token, err := auth.GenerateToken(auth.SessionToken{
		UserID:    maskedKey,
		ProjectID: projectID,
		Role:      string(role),
	}, h.jwtSecret, auth.JWTTTL)
	if err != nil {
		http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "aleph_jwt",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   3600,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createSessionResponse{ProjectID: projectID})
}

func (h *SessionHandler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("aleph_jwt"); err == nil && h.revocationStore != nil {
		if claims, verr := auth.ValidateToken(c.Value, h.jwtSecret); verr == nil && claims.ID != "" {
			h.revocationStore.Revoke(claims.ID)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "aleph_jwt",
		Value:    "",
		HttpOnly: true,
		Secure:   false,
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

	c, err := r.Cookie("aleph_jwt")
	if err != nil {
		http.Error(w, `{"error":"no session"}`, http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateToken(c.Value, h.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid session"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createSessionResponse{ProjectID: claims.ProjectID})
}

func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[len(key)-4:]
}
