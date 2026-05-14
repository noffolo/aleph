package handler

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeCursor(t *testing.T) {
	result := encodeCursor("item-42")
	// Must be valid base64 and decode back
	decoded, err := base64.RawURLEncoding.DecodeString(result)
	assert.NoError(t, err)
	var d cursorData
	err = json.Unmarshal(decoded, &d)
	assert.NoError(t, err)
	assert.Equal(t, "item-42", d.ID)
}

func TestEncodeCursor_Empty(t *testing.T) {
	result := encodeCursor("")
	decoded, _ := base64.RawURLEncoding.DecodeString(result)
	var d cursorData
	json.Unmarshal(decoded, &d)
	assert.Equal(t, "", d.ID)
}

func TestEncodeCursor_SpecialChars(t *testing.T) {
	result := encodeCursor("user:123/project=abc")
	assert.NotEmpty(t, result)
	decoded, _ := base64.RawURLEncoding.DecodeString(result)
	var d cursorData
	json.Unmarshal(decoded, &d)
	assert.Equal(t, "user:123/project=abc", d.ID)
}

func TestDecodeCursor_Empty(t *testing.T) {
	assert.Equal(t, "", decodeCursor(""))
}

func TestDecodeCursor_Valid(t *testing.T) {
	cursor := encodeCursor("item-99")
	result := decodeCursor(cursor)
	assert.Equal(t, "item-99", result)
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	assert.Equal(t, "", decodeCursor("not-valid-base64!!!"))
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	invalid := base64.RawURLEncoding.EncodeToString([]byte("{bad json"))
	assert.Equal(t, "", decodeCursor(invalid))
}

func TestDecodeCursor_Roundtrip(t *testing.T) {
	ids := []string{"a", "abc-123", "", "zzz"}
	for _, id := range ids {
		encoded := encodeCursor(id)
		decoded := decodeCursor(encoded)
		assert.Equal(t, id, decoded, "roundtrip failed for id=%q", id)
	}
}

func TestClampLimit(t *testing.T) {
	assert.Equal(t, int32(DefaultPageSize), clampLimit(0))
	assert.Equal(t, int32(DefaultPageSize), clampLimit(-1))
	assert.Equal(t, int32(DefaultPageSize), clampLimit(-100))
	assert.Equal(t, int32(1), clampLimit(1))
	assert.Equal(t, int32(50), clampLimit(50))
	assert.Equal(t, int32(MaxPageSize), clampLimit(100))
	assert.Equal(t, int32(MaxPageSize), clampLimit(101))
	assert.Equal(t, int32(MaxPageSize), clampLimit(999999))
}

func TestParsePagination(t *testing.T) {
	t.Run("empty cursor, default limit", func(t *testing.T) {
		result := ParsePagination("", 0)
		assert.Equal(t, "", result.Cursor)
		assert.Equal(t, int32(DefaultPageSize), result.Limit)
	})

	t.Run("valid cursor, custom limit", func(t *testing.T) {
		encoded := encodeCursor("cursor-123")
		result := ParsePagination(encoded, 50)
		assert.Equal(t, "cursor-123", result.Cursor)
		assert.Equal(t, int32(50), result.Limit)
	})

	t.Run("clamps excessive limit", func(t *testing.T) {
		result := ParsePagination("", 999)
		assert.Equal(t, int32(MaxPageSize), result.Limit)
	})

	t.Run("invalid cursor returns empty", func(t *testing.T) {
		result := ParsePagination("garbage", 25)
		assert.Equal(t, "", result.Cursor)
	})

	t.Run("negative limit uses default", func(t *testing.T) {
		result := ParsePagination("", -5)
		assert.Equal(t, int32(DefaultPageSize), result.Limit)
	})
}
