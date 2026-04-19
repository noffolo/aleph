package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type AuthHandler struct {
	metaRepo *repository.MetadataRepository
}

func NewAuthHandler(metaRepo *repository.MetadataRepository) *AuthHandler {
	return &AuthHandler{metaRepo: metaRepo}
}

func (h *AuthHandler) ListApiKeys(
	ctx context.Context,
	req *connect.Request[v1.ListApiKeysRequest],
) (*connect.Response[v1.ListApiKeysResponse], error) {
	projectID := req.Msg.ProjectId
	rows, err := h.metaRepo.DB().Query("SELECT id, label, key, created_at FROM system_api_keys WHERE project_id = ?", projectID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	defer rows.Close()

	var keys []*v1.ApiKey
	for rows.Next() {
		var k v1.ApiKey
		var createdAt string
		var hashedKey string
		rows.Scan(&k.Id, &k.Label, &hashedKey, &createdAt)
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		k.CreatedAt = t.Unix()
		// Securely mask the hashed key in listings
		k.Key = "********" 
		keys = append(keys, &k)
	}
	return connect.NewResponse(&v1.ListApiKeysResponse{Keys: keys}), nil
}

func (h *AuthHandler) CreateApiKey(
	ctx context.Context,
	req *connect.Request[v1.CreateApiKeyRequest],
) (*connect.Response[v1.CreateApiKeyResponse], error) {
	projectID := req.Msg.ProjectId
	label := req.Msg.Label

	// Generate random key
	b := make([]byte, 16)
	rand.Read(b)
	key := hex.EncodeToString(b)
	id := hex.EncodeToString(b[:4])

	// Hash the key for storage
	hsh := sha256.New()
	hsh.Write([]byte(key))
	hashedKey := hex.EncodeToString(hsh.Sum(nil))

	_, err := h.metaRepo.DB().Exec(
		"INSERT INTO system_api_keys (id, project_id, label, key) VALUES (?, ?, ?, ?)",
		id, projectID, label, hashedKey,
	)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	return connect.NewResponse(&v1.CreateApiKeyResponse{
		Key: &v1.ApiKey{Id: id, Label: label, Key: key, CreatedAt: time.Now().Unix()},
	}), nil
}

func (h *AuthHandler) DeleteApiKey(
	ctx context.Context,
	req *connect.Request[v1.DeleteApiKeyRequest],
) (*connect.Response[v1.DeleteApiKeyResponse], error) {
	projectID := req.Msg.ProjectId
	id := req.Msg.Id
	_, err := h.metaRepo.DB().Exec("DELETE FROM system_api_keys WHERE project_id = ? AND id = ?", projectID, id)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteApiKeyResponse{Success: true}), nil
}
