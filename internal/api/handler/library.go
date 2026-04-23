package handler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jung-kurt/gofpdf"
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
	path, err := sanitizePath(h.projectsRoot, projectID, "library", assetID)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }

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
	path, err := sanitizePath(h.projectsRoot, projectID, "library", assetID)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }

	if err := os.Remove(path); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteAssetResponse{Success: true}), nil
}

func (h *LibraryHandler) GeneratePdf(
	ctx context.Context,
	req *connect.Request[v1.GeneratePdfRequest],
) (*connect.Response[v1.GeneratePdfResponse], error) {
	projectID := req.Msg.ProjectId
	assetID := req.Msg.AssetId
	path, err := sanitizePath(h.projectsRoot, projectID, "library", assetID)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset not found: %v", err))
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, assetID)
	pdf.Ln(14)
	pdf.SetFont("Helvetica", "", 10)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		escaped := strings.ToValidUTF8(line, "?")
		if escaped == "" {
			pdf.Ln(5)
			continue
		}
		safe := sanitizePdfString(escaped)
		pdf.MultiCell(0, 5, safe, "", "", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("pdf generation failed: %v", err))
	}

	filename := strings.TrimSuffix(assetID, filepath.Ext(assetID)) + ".pdf"

	return connect.NewResponse(&v1.GeneratePdfResponse{
		PdfData:  buf.Bytes(),
		Filename: filename,
	}), nil
}

func sanitizePdfString(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"(", "\\(",
		")", "\\)",
	)
	return replacer.Replace(s)
}

func (h *LibraryHandler) UploadAsset(
	ctx context.Context,
	req *connect.Request[v1.UploadAssetRequest],
) (*connect.Response[v1.UploadAssetResponse], error) {
	projectID := req.Msg.ProjectId
	filename := req.Msg.Filename
	content := req.Msg.Content

	if filename == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("filename is required"))
	}
	if len(content) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is required"))
	}

	libPath := filepath.Join(h.projectsRoot, projectID, "library")
	os.MkdirAll(libPath, 0755)

	safeName := filepath.Base(filename)

	safeDestPath, perr := sanitizePath(h.projectsRoot, projectID, "library", safeName)
	if perr != nil { return nil, connect.NewError(connect.CodeInvalidArgument, perr) }

	if err := os.WriteFile(safeDestPath, content, 0644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write asset: %v", err))
	}

	asset := &v1.Asset{
		Id:        safeName,
		Name:      safeName,
		Type:      filepath.Ext(safeName),
		CreatedAt: time.Now().Unix(),
	}

	return connect.NewResponse(&v1.UploadAssetResponse{Asset: asset}), nil
}
