package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvSecrets_MustString_Present(t *testing.T) {
	os.Setenv("JWT_SECRET", "must-string-value")
	defer os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	v := e.MustString("jwt.secret")
	assert.Equal(t, "must-string-value", v)
}

func TestEnvSecrets_MustString_Panics(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	defer func() {
		r := recover()
		assert.NotNil(t, r, "MustString should panic when env var is missing")
	}()
	e.MustString("jwt.secret")
}

func TestEnvSecrets_MustString_CustomKey(t *testing.T) {
	os.Setenv("CUSTOM_KEY", "custom-val")
	defer os.Unsetenv("CUSTOM_KEY")

	e := &envSecrets{}
	v := e.MustString("custom.key")
	assert.Equal(t, "custom-val", v)
}

func TestEnvSecrets_Has_Present(t *testing.T) {
	os.Setenv("JWT_SECRET", "present")
	defer os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	assert.True(t, e.Has("jwt.secret"))
}

func TestEnvSecrets_Has_Absent(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	assert.False(t, e.Has("jwt.secret"))
}

func TestEnvSecrets_Has_Custom(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		envKey  string
		envVal  string
		wantHas bool
	}{
		{"database_url_present", "database.url", "POSTGRES_DSN", "pg://localhost", true},
		{"database_url_absent", "database.url", "", "", false},
		{"ollama_base_url_present", "ollama.base_url", "OLLAMA_BASE_URL", "http://local:11434", true},
		{"ollama_base_url_absent", "ollama.base_url", "", "", false},
		{"generic_env_present", "generic.key", "GENERIC_KEY", "value", true},
		{"generic_env_absent", "generic.key", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				os.Setenv(tt.envKey, tt.envVal)
				defer os.Unsetenv(tt.envKey)
			}
			e := &envSecrets{}
			assert.Equal(t, tt.wantHas, e.Has(tt.key))
		})
	}
}

func TestEnvSecrets_String_Fallback(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}

	t.Run("with_fallback", func(t *testing.T) {
		v := e.String("jwt.secret", "fallback-value")
		assert.Equal(t, "fallback-value", v)
	})

	t.Run("without_fallback", func(t *testing.T) {
		v := e.String("jwt.secret")
		assert.Equal(t, "", v)
	})
}

func TestEnvSecrets_String_Present(t *testing.T) {
	os.Setenv("JWT_SECRET", "real-value")
	defer os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	v := e.String("jwt.secret")
	assert.Equal(t, "real-value", v)
}

func TestEnvSecrets_String_FallbackIgnoredWhenPresent(t *testing.T) {
	os.Setenv("JWT_SECRET", "real-value")
	defer os.Unsetenv("JWT_SECRET")

	e := &envSecrets{}
	v := e.String("jwt.secret", "ignored-fallback")
	assert.Equal(t, "real-value", v)
}

func TestEnvSecrets_String_DatabaseURL(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "postgres://localhost:5432/db")
	defer os.Unsetenv("POSTGRES_DSN")

	e := &envSecrets{}
	v := e.String("database.url")
	assert.Equal(t, "postgres://localhost:5432/db", v)
}

func TestEnvSecrets_String_NLPAddr(t *testing.T) {
	os.Setenv("NLP_ADDR", "localhost:8001")
	defer os.Unsetenv("NLP_ADDR")

	e := &envSecrets{}
	v := e.String("nlp.sidecar_url")
	assert.Equal(t, "localhost:8001", v)
}

func TestEnvSecrets_String_SMTPPassword(t *testing.T) {
	os.Setenv("SMTP_PASSWORD", "smtp-secret-123")
	defer os.Unsetenv("SMTP_PASSWORD")

	e := &envSecrets{}
	v := e.String("smtp.password")
	assert.Equal(t, "smtp-secret-123", v)
}

func TestEnvSecrets_String_KeyEncryptionKey(t *testing.T) {
	os.Setenv("KEY_ENCRYPTION_KEY", "aes256key-material")
	defer os.Unsetenv("KEY_ENCRYPTION_KEY")

	e := &envSecrets{}
	v := e.String("key_encryption_key")
	assert.Equal(t, "aes256key-material", v)
}

func TestEnvSecrets_MustString_DatabaseURL(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "pg://host/db")
	defer os.Unsetenv("POSTGRES_DSN")

	e := &envSecrets{}
	v := e.MustString("database.url")
	assert.Equal(t, "pg://host/db", v)
}

func TestSecretToEnvKey_UnknownFollowsDefault(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"unknown_key", "UNKNOWN_KEY"},
		{"a.b.c", "A_B_C"},
		{"simple", "SIMPLE"},
		{"multi.dot.notation.key", "MULTI_DOT_NOTATION_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, secretToEnvKey(tt.key))
		})
	}
}

func TestLoadSecrets_ExplicitProductionEnv(t *testing.T) {
	os.Setenv("GOSECRETS_ENV", "production")
	defer os.Unsetenv("GOSECRETS_ENV")

	s := LoadSecrets()
	assert.NotNil(t, s)
	_, ok := s.(*envSecrets)
	assert.True(t, ok, "without gosecrets key file, production should fall back to envSecrets")
}

func TestLoadSecrets_CustomEnv(t *testing.T) {
	os.Setenv("GOSECRETS_ENV", "staging")
	defer os.Unsetenv("GOSECRETS_ENV")

	s := LoadSecrets()
	assert.NotNil(t, s)
	_, ok := s.(*envSecrets)
	assert.True(t, ok, "staging without key file should fall back to envSecrets")
}
