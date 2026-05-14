package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"

	"connectrpc.com/connect"
)

func TestSanitizePath_ValidPath(t *testing.T) {
	tmp := t.TempDir()
	result, err := sanitizePath(tmp, "subdir", "file.txt")
	assert.NoError(t, err)
	assert.Contains(t, result, "subdir")
	assert.Contains(t, result, "file.txt")
}

func TestSanitizePath_EmptyBase(t *testing.T) {
	_, err := sanitizePath("", "parts")
	assert.NoError(t, err)
}

func TestSanitizePath_PathTraversalParent(t *testing.T) {
	tmp := t.TempDir()
	_, err := sanitizePath(tmp, "..", "etc", "passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestSanitizePath_PathTraversalAbsolute(t *testing.T) {
	tmp := t.TempDir()
	_, err := sanitizePath(tmp, "/etc", "passwd")
	assert.NoError(t, err)
}

func TestSanitizePath_SameDirectory(t *testing.T) {
	tmp := t.TempDir()
	result, err := sanitizePath(tmp)
	assert.NoError(t, err)
	assert.Equal(t, tmp, result)
}

func TestSanitizePath_BaseInvalid(t *testing.T) {
	result, err := sanitizePath("/nonexistent/really/not/there/12345/base", "file.txt")
	assert.NoError(t, err)
	assert.Contains(t, result, "file.txt")
}

func TestSanitizePath_MultipleTraversalAttempts(t *testing.T) {
	tmp := t.TempDir()
	_, err := sanitizePath(tmp, "a", "..", "..", "..", "etc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestSanitizePath_EncodedTraversal(t *testing.T) {
	tmp := t.TempDir()
	result, err := sanitizePath(tmp, "..%2f..%2fetc")
	assert.NoError(t, err)
	assert.Contains(t, result, "..%2f..%2fetc")
}

func TestSanitizePath_NestedValid(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "a", "b", "c")
	os.MkdirAll(nested, 0755)
	result, err := sanitizePath(tmp, "a", "b", "c", "file.txt")
	assert.NoError(t, err)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "file.txt")
}

func TestSanitizePdfString_NoSpecial(t *testing.T) {
	assert.Equal(t, "hello world", sanitizePdfString("hello world"))
}

func TestSanitizePdfString_Backslash(t *testing.T) {
	assert.Equal(t, "\\\\", sanitizePdfString("\\"))
}

func TestSanitizePdfString_Parentheses(t *testing.T) {
	assert.Equal(t, "\\(test\\)", sanitizePdfString("(test)"))
}

func TestSanitizePdfString_Mixed(t *testing.T) {
	result := sanitizePdfString(`path\to\file (version)`)
	assert.Contains(t, result, "\\\\")
	assert.Contains(t, result, "\\(")
	assert.Contains(t, result, "\\)")
}

func TestSanitizePdfString_Empty(t *testing.T) {
	assert.Equal(t, "", sanitizePdfString(""))
}

func TestNewLibraryHandler(t *testing.T) {
	h := NewLibraryHandler("/tmp/lib")
	assert.NotNil(t, h)
	assert.Equal(t, "/tmp/lib", h.projectsRoot)
}

func TestLibraryHandler_UploadAsset_EmptyFilename(t *testing.T) {
	h := NewLibraryHandler("/tmp/lib")
	req := connect.NewRequest(&v1.UploadAssetRequest{
		ProjectId: "proj",
		Filename:  "",
		Content:   []byte("test"),
	})
	_, err := h.UploadAsset(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename is required")
}

func TestLibraryHandler_UploadAsset_EmptyContent(t *testing.T) {
	h := NewLibraryHandler("/tmp/lib")
	req := connect.NewRequest(&v1.UploadAssetRequest{
		ProjectId: "proj",
		Filename:  "test.txt",
		Content:   []byte{},
	})
	_, err := h.UploadAsset(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}
