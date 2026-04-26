package crypto

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key, err := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("key length: got %d, want 32", len(key))
	}

	plaintexts := []string{
		"sk-ant-my-test-api-key-12345",
		"",
		"a",
		"hello-world",
		"very long key that exceeds typical bounds for testing purposes x123",
		"sk-" + strings.Repeat("a", 48),
	}

	for _, pt := range plaintexts {
		t.Run("key_"+pt[:min(len(pt), 16)], func(t *testing.T) {
			cipherHex, err := Encrypt([]byte(pt), key)
			if err != nil {
				t.Fatalf("Encrypt(%q) unexpected error: %v", pt, err)
			}
			if cipherHex == "" {
				t.Fatal("Encrypt returned empty string")
			}
			decrypted, err := Decrypt(cipherHex, key)
			if err != nil {
				t.Fatalf("Decrypt(%q) unexpected error: %v", cipherHex, err)
			}
			if string(decrypted) != pt {
				t.Fatalf("round-trip mismatch: got %q, want %q", string(decrypted), pt)
			}
		})
	}
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	plaintext := []byte("same-key-each-time")

	c1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	if c1 == c2 {
		t.Error("two encryptions of same plaintext produced identical ciphertext (nonce reuse?)")
	}
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	shortKey := []byte("too-short")
	_, err := Encrypt([]byte("test"), shortKey)
	if err == nil {
		t.Error("expected error for short key")
	}
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	cipherHex, err := Encrypt([]byte("secret-api-key"), key)
	if err != nil {
		t.Fatal(err)
	}

	tampered := cipherHex[:len(cipherHex)-2] + "00"
	_, err = Decrypt(tampered, key)
	if err == nil {
		t.Error("expected error for tampered ciphertext")
	}
	if err != ErrDecryptFailed {
		t.Errorf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestDecrypt_InvalidHex(t *testing.T) {
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	_, err := Decrypt("this-is-not-hex", key)
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestDecrypt_EmptyKey(t *testing.T) {
	_, err := Decrypt("someciphertext", nil)
	if err != ErrKeyNotSet {
		t.Errorf("expected ErrKeyNotSet, got %v", err)
	}
}

func TestEncrypt_EmptyKey(t *testing.T) {
	_, err := Encrypt([]byte("test"), nil)
	if err != ErrKeyNotSet {
		t.Errorf("expected ErrKeyNotSet, got %v", err)
	}
}

func TestLoadEncryptionKey(t *testing.T) {
	validHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	key, err := LoadEncryptionKey(validHex)
	if err != nil {
		t.Fatalf("LoadEncryptionKey(%q) unexpected error: %v", validHex, err)
	}
	if len(key) != 32 {
		t.Fatalf("key length: got %d, want 32", len(key))
	}

	_, err = LoadEncryptionKey("")
	if err != ErrKeyNotSet {
		t.Errorf("empty key: expected ErrKeyNotSet, got %v", err)
	}

	_, err = LoadEncryptionKey("short")
	if err == nil {
		t.Error("short hex key should error")
	}

	_, err = LoadEncryptionKey("not-hex!!")
	if err == nil {
		t.Error("non-hex key should error")
	}
}

// TestSerializedResponseDoesNotContainKey verifies that a simulated
// serialized response does not expose the raw encryption key or API key.
func TestSerializedResponseDoesNotContainKey(t *testing.T) {
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	apiKey := "sk-secret-key-12345"

	cipherHex, err := Encrypt([]byte(apiKey), key)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a response JSON that would be returned to the client
	response := `{"id":"agent-1","name":"test","api_key":"` + maskAPIKey(apiKey) + `","provider":"openai"}`
	if strings.Contains(response, apiKey) {
		t.Error("response contains readable API key")
	}
	if strings.Contains(response, hex.EncodeToString(key)) {
		t.Error("response contains encryption key")
	}

	// The ciphertext is hex-encoded and safe for transport
	if cipherHex == apiKey {
		t.Error("ciphertext should not equal plaintext")
	}
}

func maskAPIKey(key string) string {
	if len(key) > 8 {
		return key[:8] + "****"
	}
	return "****"
}

func TestEncrypt_SameKey(t *testing.T) {
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	key2 := make([]byte, 32)
	copy(key2, key)
	key2[0] ^= 0x01

	pt := []byte("my-api-key")
	c1, _ := Encrypt(pt, key)
	c2, _ := Encrypt(pt, key2)

	if c1 == c2 {
		t.Error("different keys should produce different ciphertexts")
	}

	// Decrypt c1 with key2 should fail
	_, err := Decrypt(c1, key2)
	if err == nil {
		t.Error("decrypting with wrong key should fail")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
