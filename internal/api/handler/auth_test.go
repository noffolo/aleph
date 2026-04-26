package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
)

func AuthHash(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func TestAuthHandler_NewAuthHandler(t *testing.T) {
	h := NewAuthHandler((*repository.MetadataRepository)(nil))
	assert.NotNil(t, h)
}

func TestAuthHandler_HashValidation(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"test-key-123", 64},
		{"", 64},
		{"very-long-key-with-lots-of-characters-1234567890", 64},
	}

	for _, tc := range cases {
		hash := AuthHash(tc.input)
		assert.Len(t, hash, tc.expected, "SHA-256 hex must always be 64 chars")
		assert.Equal(t, hash, AuthHash(tc.input), "hashing must be deterministic")
	}
}

func TestAuthHandler_HashCollision(t *testing.T) {
	assert.NotEqual(t, AuthHash("a"), AuthHash("b"), "different inputs must produce different hashes")
	assert.NotEqual(t, AuthHash("key1"), AuthHash("key2"), "nearby strings must not collide")
}

func TestAuthHandler_CreateApiKey_KeyFormat(t *testing.T) {
	keyLen := 16
	raw := make([]byte, keyLen)
	raw[0] = 0xFF
	raw[15] = 0x00

	id := hex.EncodeToString(raw[:4])
	key := hex.EncodeToString(raw)

	assert.Len(t, id, 8, "id is first 4 bytes hex encoded")
	assert.Len(t, key, 32, "key is 16 bytes hex encoded")

	hashed := AuthHash(key)
	assert.Len(t, hashed, 64, "hashed key must be SHA-256 hex")
	assert.NotEqual(t, key, hashed, "hash must differ from raw key")
}
