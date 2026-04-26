package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKey      = errors.New("KEY_ENCRYPTION_KEY must be 32 bytes (64 hex chars) for AES-256")
	ErrInvalidCipher   = errors.New("ciphertext too short to contain nonce")
	ErrDecryptFailed   = errors.New("decryption failed: authentication error or corrupted data")
	ErrKeyNotSet       = errors.New("encryption key not configured")
)

const (
	KeyLength = 32 // AES-256 requires 32-byte key
)

// Encrypt encrypts plaintext with AES-256-GCM using the provided key.
// Returns hex-encoded: nonce || ciphertext || auth_tag.
// The key must be exactly 32 bytes (64 hex chars when hex-encoded).
// If key is nil or zero-length, returns plaintext hex-encoded (no-op for dev).
func Encrypt(plaintext []byte, key []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrKeyNotSet
	}
	if len(key) != KeyLength {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher init: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm init: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded ciphertext with AES-256-GCM.
// The hex payload must be: nonce || ciphertext || auth_tag.
// Returns the original plaintext bytes.
// If key is nil or zero-length, attempts hex decode only (no-op for dev).
func Decrypt(cipherHex string, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrKeyNotSet
	}
	if len(key) != KeyLength {
		return nil, ErrInvalidKey
	}

	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return nil, fmt.Errorf("hex decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher init: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCipher
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}

	return plaintext, nil
}

// LoadEncryptionKey reads the KEY_ENCRYPTION_KEY environment variable
// and validates it is exactly 32 bytes for AES-256.
func LoadEncryptionKey(encodedKey string) ([]byte, error) {
	if encodedKey == "" {
		return nil, ErrKeyNotSet
	}
	key, err := hex.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("KEY_ENCRYPTION_KEY must be hex-encoded: %w", err)
	}
	if len(key) != KeyLength {
		return nil, ErrInvalidKey
	}
	return key, nil
}
