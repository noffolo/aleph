package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSecrets_CI(t *testing.T) {
	os.Setenv("GOSECRETS_ENV", "ci")
	defer os.Unsetenv("GOSECRETS_ENV")

	s := LoadSecrets()
	assert.NotNil(t, s)
	_, ok := s.(*envSecrets)
	assert.True(t, ok, "CI mode should return envSecrets")
}

func TestLoadSecrets_Fallback(t *testing.T) {
	os.Unsetenv("GOSECRETS_ENV")
	s := LoadSecrets()
	assert.NotNil(t, s)
	_, ok := s.(*envSecrets)
	assert.True(t, ok, "fallback without key file should return envSecrets")
}

func TestEnvSecrets_String(t *testing.T) {
	os.Setenv("JWT_SECRET", "jwt-value-123")
	defer os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	v := e.String("jwt.secret")
	assert.Equal(t, "jwt-value-123", v)
}

func TestSecretToEnvKey(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"jwt.secret", "JWT_SECRET"},
		{"key_encryption_key", "KEY_ENCRYPTION_KEY"},
		{"aleph.api_key_secret_backend", "ALEPH_API_KEY_SECRET_BACKEND"},
		{"database.url", "POSTGRES_DSN"},
		{"postgres.dsn", "POSTGRES_DSN"},
		{"smtp.password", "SMTP_PASSWORD"},
		{"ollama.base_url", "OLLAMA_BASE_URL"},
		{"nlp.sidecar_url", "NLP_ADDR"},
		{"custom.key", "CUSTOM_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, secretToEnvKey(tt.key))
		})
	}
}

func TestEnvName(t *testing.T) {
	t.Run("default_development", func(t *testing.T) {
		os.Unsetenv("GOSECRETS_ENV")
		assert.Equal(t, "development", envName())
	})

	t.Run("custom_env", func(t *testing.T) {
		os.Setenv("GOSECRETS_ENV", "production")
		defer os.Unsetenv("GOSECRETS_ENV")
		assert.Equal(t, "production", envName())
	})

	t.Run("ci_env", func(t *testing.T) {
		os.Setenv("GOSECRETS_ENV", "ci")
		defer os.Unsetenv("GOSECRETS_ENV")
		assert.Equal(t, "ci", envName())
	})
}
