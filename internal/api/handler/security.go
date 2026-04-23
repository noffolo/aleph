package handler

import (
	"fmt"
	"path/filepath"
	"strings"
)

func sanitizePath(base string, parts ...string) (string, error) {
	joined := filepath.Join(append([]string{base}, parts...)...)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("invalid path: %v", err)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %v", err)
	}
	if !strings.HasPrefix(abs, absBase+string(filepath.Separator)) && abs != absBase {
		return "", fmt.Errorf("path traversal detected")
	}
	return abs, nil
}
