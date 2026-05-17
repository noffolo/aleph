package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretToEnvKey_ExplicitMapping(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"jwt.secret", "JWT_SECRET"},
		{"key_encryption_key", "KEY_ENCRYPTION_KEY"},
		{"database.url", "POSTGRES_DSN"},
		{"postgres.dsn", "POSTGRES_DSN"},
		{"ollama.base_url", "OLLAMA_BASE_URL"},
		{"nlp.sidecar_url", "NLP_ADDR"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := secretToEnvKey(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretToEnvKey_DefaultUppercase(t *testing.T) {
	got := secretToEnvKey("custom.setting")
	assert.Equal(t, "CUSTOM_SETTING", got)
}

func TestEnvSecrets_String_WithFallback(t *testing.T) {
	e := &envSecrets{}
	got := e.String("nonexistent.key_12345", "fallback_value")
	assert.Equal(t, "fallback_value", got)
}

func TestEnvSecrets_String_WithoutFallback(t *testing.T) {
	e := &envSecrets{}
	got := e.String("nonexistent.key_12345")
	assert.Equal(t, "", got)
}

func TestEnvSecrets_String_FromEnv(t *testing.T) {
	os.Setenv("TEST_SECRETS_KEY", "env_value")
	defer os.Unsetenv("TEST_SECRETS_KEY")

	e := &envSecrets{}
	got := e.String("TEST_SECRETS_KEY")
	assert.Equal(t, "env_value", got)
}

func TestEnvSecrets_MustString_Exists(t *testing.T) {
	os.Setenv("MUST_STRING_TEST", "found")
	defer os.Unsetenv("MUST_STRING_TEST")

	e := &envSecrets{}
	got := e.MustString("MUST_STRING_TEST")
	assert.Equal(t, "found", got)
}

func TestEnvSecrets_MustString_NotFound(t *testing.T) {
	e := &envSecrets{}
	assert.Panics(t, func() {
		e.MustString("nonexistent_must_key_xyz")
	})
}

func TestEnvSecrets_Has(t *testing.T) {
	os.Setenv("HAS_TEST_KEY", "value")
	defer os.Unsetenv("HAS_TEST_KEY")

	e := &envSecrets{}

	assert.True(t, e.Has("HAS_TEST_KEY"))
	assert.False(t, e.Has("NONEXISTENT_HAS_KEY"))
}

func TestEnvName_WithEnv(t *testing.T) {
	os.Setenv("GOSECRETS_ENV", "test-env")
	defer os.Unsetenv("GOSECRETS_ENV")
	assert.Equal(t, "test-env", envName())
}

func TestEnvName_Default(t *testing.T) {
	os.Unsetenv("GOSECRETS_ENV")
	assert.Equal(t, "development", envName())
}

func TestLoadSecrets_CI_Mode(t *testing.T) {
	os.Setenv("GOSECRETS_ENV", "ci")
	defer os.Unsetenv("GOSECRETS_ENV")

	s := LoadSecrets()
	require.NotNil(t, s)
	_, ok := s.(*envSecrets)
	assert.True(t, ok)
}
