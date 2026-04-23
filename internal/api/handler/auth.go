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
	keys, err := h.metaRepo.ListAPIKeys(projectID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	var result []*v1.ApiKey
	for _, k := range keys {
		result = append(result, &v1.ApiKey{
			Id:        k.ID,
			Label:     k.Label,
			Key:       "********",
			CreatedAt: k.CreatedAt.Unix(),
		})
	}
	return connect.NewResponse(&v1.ListApiKeysResponse{Keys: result}), nil
}

func (h *AuthHandler) CreateApiKey(
	ctx context.Context,
	req *connect.Request[v1.CreateApiKeyRequest],
) (*connect.Response[v1.CreateApiKeyResponse], error) {
	projectID := req.Msg.ProjectId
	label := req.Msg.Label

	b := make([]byte, 16)
	rand.Read(b)
	key := hex.EncodeToString(b)
	id := hex.EncodeToString(b[:4])

	hsh := sha256.New()
	hsh.Write([]byte(key))
	hashedKey := hex.EncodeToString(hsh.Sum(nil))

	err := h.metaRepo.CreateAPIKey(id, projectID, label, hashedKey)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

	return connect.NewResponse(&v1.CreateApiKeyResponse{
		Key: &v1.ApiKey{Id: id, Label: label, Key: key, CreatedAt: time.Now().Unix()},
	}), nil
}

func (h *AuthHandler) DeleteApiKey(
	ctx context.Context,
	req *connect.Request[v1.DeleteApiKeyRequest],
) (*connect.Response[v1.DeleteApiKeyResponse], error) {
	err := h.metaRepo.DeleteAPIKey(req.Msg.Id, req.Msg.ProjectId)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.DeleteApiKeyResponse{Success: true}), nil
}