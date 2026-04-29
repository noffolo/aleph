package auth

import (
	"strings"
	"testing"
)

func TestHashAPIKey_ReturnsValidEncodedString(t *testing.T) {
	key := "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
	encoded, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey returned error: %v", err)
	}

	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Errorf("expected prefix $argon2id$v=19$, got: %s", encoded)
	}

	// Verify format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		t.Fatalf("expected 6 parts separated by $, got %d parts in: %s", len(parts), encoded)
	}

	if parts[1] != "argon2id" {
		t.Errorf("expected algorithm argon2id, got: %s", parts[1])
	}

	if parts[2] != "v=19" {
		t.Errorf("expected version v=19, got: %s", parts[2])
	}

	if parts[3] != "m=65536,t=1,p=4" {
		t.Errorf("expected params m=65536,t=1,p=4, got: %s", parts[3])
	}

	if parts[4] == "" {
		t.Error("salt part is empty")
	}

	if parts[5] == "" {
		t.Error("hash part is empty")
	}
}

func TestHashAPIKey_ProducesDifferentHashesForSameKey(t *testing.T) {
	key := "same-key-different-salt"
	h1, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("first HashAPIKey: %v", err)
	}
	h2, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("second HashAPIKey: %v", err)
	}

	// Same key with different salts should produce different encoded strings.
	if h1 == h2 {
		t.Error("expected different hashes for same key due to random salt, got identical strings")
	}
}

func TestVerifyAPIKey_SucceedsWithCorrectKey(t *testing.T) {
	key := "correct-api-key-value"
	encoded, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey: %v", err)
	}

	ok, err := VerifyAPIKey(key, encoded)
	if err != nil {
		t.Fatalf("VerifyAPIKey returned error: %v", err)
	}
	if !ok {
		t.Error("VerifyAPIKey returned false for correct key")
	}
}

func TestVerifyAPIKey_FailsWithWrongKey(t *testing.T) {
	key := "original-key"
	wrongKey := "wrong-key"
	encoded, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey: %v", err)
	}

	ok, err := VerifyAPIKey(wrongKey, encoded)
	if err != nil {
		t.Fatalf("VerifyAPIKey returned error: %v", err)
	}
	if ok {
		t.Error("VerifyAPIKey returned true for wrong key")
	}
}

func TestVerifyAPIKey_FailsWithEmptyKey(t *testing.T) {
	key := "some-key"
	encoded, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey: %v", err)
	}

	ok, err := VerifyAPIKey("", encoded)
	if err != nil {
		t.Fatalf("VerifyAPIKey returned error: %v", err)
	}
	if ok {
		t.Error("VerifyAPIKey returned true for empty key")
	}
}

func TestVerifyAPIKey_InvalidEncodedHashFormat(t *testing.T) {
	_, err := VerifyAPIKey("key", "invalid-format")
	if err == nil {
		t.Error("expected error for invalid format, got nil")
	}
}

func TestVerifyAPIKey_WrongAlgorithm(t *testing.T) {
	// Valid format but wrong algorithm name.
	encoded := "$argon2i$v=19$m=65536,t=1,p=4$c2FsdHNhbHRzYWx0$dGVzdGhhc2g="
	_, err := VerifyAPIKey("key", encoded)
	if err == nil {
		t.Error("expected error for wrong algorithm, got nil")
	}
}

func TestVerifyAPIKey_EncodedFormatPrefix(t *testing.T) {
	key := "test-key-format-check"
	encoded, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey: %v", err)
	}

	// Must start with $argon2id$v=19$m=65536,t=1,p=4$
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=1,p=4$") {
		t.Errorf("encoded hash does not start with expected OWASP prefix, got: %s", encoded)
	}
}
