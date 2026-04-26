package handler

import (
	"encoding/base64"
	"encoding/json"
)

type PaginationRequest struct {
	Cursor string // decoded opaque cursor; empty means "start from beginning"
	Limit  int32  // requested page size, clamped to [1, MaxPageSize]
}

type PaginationResponse struct {
	NextCursor string // base64-encoded cursor for the next page; empty if no more
}

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

type cursorData struct {
	ID string `json:"i"`
}

func encodeCursor(id string) string {
	data, _ := json.Marshal(cursorData{ID: id})
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeCursor(cursor string) string {
	if cursor == "" {
		return ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return ""
	}
	var d cursorData
	if err := json.Unmarshal(decoded, &d); err != nil {
		return ""
	}
	return d.ID
}

func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return DefaultPageSize
	}
	if limit > MaxPageSize {
		return MaxPageSize
	}
	return limit
}

// ParsePagination decodes the cursor string and clamps the limit from a
// protobuf list request.
func ParsePagination(after string, limit int32) PaginationRequest {
	return PaginationRequest{
		Cursor: decodeCursor(after),
		Limit:  clampLimit(limit),
	}
}
