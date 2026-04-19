package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

type LibraryHandler struct {
	projectsRoot string
}

func NewLibraryHandler(projectsRoot string) *LibraryHandler {
	return &LibraryHandler{projectsRoot: projectsRoot}
}

func (h *LibraryHandler) ListAssets(
	ctx context.Context,
	req *connect.Request[v1.ListAssetsRequest],
) (*connect.Response[v1.ListAssetsResponse], error) {
	projectID := req.Msg.ProjectId
	libPath := filepath.Join(h.projectsRoot, projectID, "library")
	os.MkdirAll(libPath, 0755)

	entries, err := os.ReadDir(libPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var assets []*v1.Asset
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			assets = append(assets, &v1.Asset{
				Id:        entry.Name(),
				Name:      entry.Name(),
				Type:      filepath.Ext(entry.Name()),
				CreatedAt: info.ModTime().Unix(),
			})
		}
	}

	return connect.NewResponse(&v1.ListAssetsResponse{Assets: assets}), nil
}

func (h *LibraryHandler) GetAssetContent(
	ctx context.Context,
	req *connect.Request[v1.GetAssetContentRequest],
) (*connect.Response[v1.GetAssetContentResponse], error) {
	projectID := req.Msg.ProjectId
	assetID := req.Msg.AssetId
	path := filepath.Join(h.projectsRoot, projectID, "library", assetID)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset not found: %v", err))
	}

	return connect.NewResponse(&v1.GetAssetContentResponse{Content: string(content)}), nil
}

func (h *LibraryHandler) DeleteAsset(
	ctx context.Context,
	req *connect.Request[v1.DeleteAssetRequest],
) (*connect.Response[v1.DeleteAssetResponse], error) {
	projectID := req.Msg.ProjectId
	assetID := req.Msg.Id
	path := filepath.Join(h.projectsRoot, projectID, "library", assetID)

	if err := os.Remove(path); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteAssetResponse{Success: true}), nil
}
