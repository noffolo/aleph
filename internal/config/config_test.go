package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeSecrets struct {
	data map[string]string
}

func (f *fakeSecrets) String(key string, fallback ...string) string {
	if v, ok := f.data[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

func (f *fakeSecrets) MustString(key string) string {
	v, ok := f.data[key]
	if !ok || v == "" {
		panic("secrets: required key " + key + " not found")
	}
	return v
}

func (f *fakeSecrets) Has(key string) bool {
	_, ok := f.data[key]
	return ok
}

func testSecrets() SecretsProvider {
	return &fakeSecrets{data: map[string]string{
		"jwt.secret":          "test-jwt-secret-key-for-unit-tests",
		"key_encryption_key":  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"database.url":        "postgres://user:pass@localhost:5432/test",
		"ollama.base_url":     "http://localhost:11434",
		"nlp.sidecar_url":     "localhost:8001",
	}}
}

func TestLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("DATA_ROOT")
	os.Unsetenv("DUCKDB_PATH")
	os.Unsetenv("BACKUP_INTERVAL")
	os.Unsetenv("BACKUP_DIR")
	os.Unsetenv("BACKUP_KEEP")

	cfg, err := LoadConfigWithSecrets(testSecrets())
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "postgres://user:pass@localhost:5432/test", cfg.PostgresDSN)
	assert.Contains(t, cfg.DuckDBPath, "data/aleph.duckdb")
	assert.NotNil(t, cfg.JWTSecret)
	assert.NotNil(t, cfg.EncryptionKey)
	assert.Equal(t, 32, len(cfg.EncryptionKey))
}

func TestLoadConfig_WithEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("MCP_SERVER_URIS", "http://server1, http://server2")
	os.Setenv("BACKUP_INTERVAL", "30m")
	os.Setenv("BACKUP_KEEP", "14")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("MCP_SERVER_URIS")
		os.Unsetenv("BACKUP_INTERVAL")
		os.Unsetenv("BACKUP_KEEP")
	}()

	sec := &fakeSecrets{data: map[string]string{
		"jwt.secret":         "test-jwt-secret-key-for-unit-tests",
		"key_encryption_key": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"database.url":        "postgres://user:pass@localhost:5432/test",
		"nlp.sidecar_url":    "nlp:8002",
		"ollama.base_url":     "http://ollama:11434",
	}}

	cfg, err := LoadConfigWithSecrets(sec)
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "postgres://user:pass@localhost:5432/test", cfg.PostgresDSN)
	assert.Equal(t, "nlp:8002", cfg.NLPAddr)
	assert.Equal(t, []string{"http://server1", "http://server2"}, cfg.MCPServerURIs)
	assert.Equal(t, "http://ollama:11434", cfg.OllamaBaseURL)
	assert.NotNil(t, cfg.EncryptionKey)
	assert.Equal(t, 32, len(cfg.EncryptionKey))
	assert.Equal(t, 14, cfg.BackupKeep)
}

func TestLoadConfig_EmptyMCPURIs(t *testing.T) {
	os.Unsetenv("MCP_SERVER_URIS")

	cfg, err := LoadConfigWithSecrets(testSecrets())
	assert.NoError(t, err)
	assert.Empty(t, cfg.MCPServerURIs)
}

func TestLoadConfig_OllamaDefault(t *testing.T) {
	os.Unsetenv("OLLAMA_BASE_URL")

	cfg, err := LoadConfigWithSecrets(testSecrets())
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:11434", cfg.OllamaBaseURL)
}

func TestLoadConfig_MissingJWTSecret(t *testing.T) {
	sec := &fakeSecrets{data: map[string]string{
		"key_encryption_key": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"database.url":        "postgres://user:pass@localhost:5432/test",
	}}
	_, err := LoadConfigWithSecrets(sec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jwt.secret")
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	sec := &fakeSecrets{data: map[string]string{
		"jwt.secret":         "test-jwt-secret",
		"key_encryption_key": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}}
	_, err := LoadConfigWithSecrets(sec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres.dsn")
}

func TestLoadConfig_PostgresDSNFallback(t *testing.T) {
	sec := &fakeSecrets{data: map[string]string{
		"jwt.secret":         "test-jwt-secret",
		"key_encryption_key": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"postgres.dsn":        "postgres://fallback:5432/db",
	}}
	cfg, err := LoadConfigWithSecrets(sec)
	assert.NoError(t, err)
	assert.Equal(t, "postgres://fallback:5432/db", cfg.PostgresDSN)
}