// Package auth provides argon2id-based API key hashing and verification.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters as recommended by OWASP.
	argonMemory  = 64 * 1024 // 64 MB
	argonTime    = 1
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashAPIKey generates a random salt, computes an argon2id hash of the key,
// and returns the encoded hash string in the format:
//
//	$argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
func HashAPIKey(key string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hashAPIKey: failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(key), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash)

	return encoded, nil
}

// VerifyAPIKey parses the encoded argon2id hash string and verifies that the
// given key matches the stored hash using constant-time comparison.
func VerifyAPIKey(key string, encodedHash string) (bool, error) {
	// Parse the encoded hash string.
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("verifyAPIKey: invalid encoded hash format: expected 6 parts, got %d", len(parts))
	}

	// parts[0] is empty (string starts with $)
	// parts[1] = "argon2id"
	// parts[2] = "v=19"
	// parts[3] = "m=65536,t=1,p=4"
	// parts[4] = base64-salt
	// parts[5] = base64-hash

	if parts[1] != "argon2id" {
		return false, fmt.Errorf("verifyAPIKey: unexpected algorithm: %s", parts[1])
	}

	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false, fmt.Errorf("verifyAPIKey: unexpected version: %s", parts[2])
	}

	// Decode salt.
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("verifyAPIKey: failed to decode salt: %w", err)
	}

	// Decode stored hash.
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("verifyAPIKey: failed to decode hash: %w", err)
	}

	// Recompute hash with the same salt and parameters.
	computedHash := argon2.IDKey([]byte(key), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Constant-time comparison.
	if len(storedHash) != len(computedHash) {
		return false, nil
	}

	return subtle.ConstantTimeCompare(storedHash, computedHash) == 1, nil
}
