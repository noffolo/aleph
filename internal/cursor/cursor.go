// Package cursor provides opaque cursor encoding for cursor-based pagination.
//
// Cursors encode a position marker (row offset or entity ID) as a base64 string,
// making the internal pagination mechanism opaque to clients. This allows the
// backend to change pagination strategy without breaking clients.
//
// Format: base64("<prefix>:<value>")
//
//	prefix = "pos" for position-based cursors (ExecuteQuery)
//	prefix = "id"  for ID-based cursors (ListAgents, ListTools, ListSkills)
package cursor

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncodePosition creates an opaque cursor encoding a row offset position.
// Used for query results where items don't have stable IDs.
func EncodePosition(offset int) string {
	return encode("pos", strconv.Itoa(offset))
}

// EncodeID creates an opaque cursor encoding an entity ID.
// Used for list endpoints where each item has a unique ID.
func EncodeID(id string) string {
	return encode("id", id)
}

// DecodePosition decodes a position-based cursor returning the row offset.
// Returns 0 if the cursor is empty or cannot be decoded.
func DecodePosition(cursor string) int {
	prefix, val := decode(cursor)
	if prefix != "pos" || val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return n
}

// DecodeID decodes an ID-based cursor returning the entity ID.
// Returns empty string if the cursor is empty or cannot be decoded.
func DecodeID(cursor string) string {
	prefix, val := decode(cursor)
	if prefix != "id" || val == "" {
		return ""
	}
	return val
}

// encode returns base64("<prefix>:<value>")
func encode(prefix, value string) string {
	raw := fmt.Sprintf("%s:%s", prefix, value)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// decode parses a base64-encoded cursor into its prefix and value components.
func decode(cursor string) (prefix, value string) {
	if cursor == "" {
		return "", ""
	}
	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", ""
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
