package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Unset env vars to test defaults
	for _, k := range []string{"PORT", "DATA_ROOT", "POSTGRES_DSN", "DUCKDB_PATH", "NLP_ADDR", "OLLAMA_BASE_URL", "MCP_SERVER_URIS", "KEY_ENCRYPTION_KEY", "BACKUP_INTERVAL", "BACKUP_DIR", "BACKUP_KEEP"} {
		os.Unsetenv(k)
	}

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable", cfg.PostgresDSN)
	assert.Equal(t, "localhost:8001", cfg.NLPAddr)
	assert.Equal(t, "http://localhost:11434", cfg.OllamaBaseURL)
	assert.Empty(t, cfg.MCPServerURIs)
	assert.Empty(t, cfg.KeyEncryptionKey)
	assert.Nil(t, cfg.EncryptionKey)
	assert.Equal(t, 7, cfg.BackupKeep)
}

func TestLoadConfig_WithEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/test")
	os.Setenv("NLP_ADDR", "nlp:8002")
	os.Setenv("MCP_SERVER_URIS", "http://server1, http://server2")
	os.Setenv("KEY_ENCRYPTION_KEY", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	os.Setenv("BACKUP_INTERVAL", "30m")
	os.Setenv("BACKUP_KEEP", "14")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("NLP_ADDR")
		os.Unsetenv("MCP_SERVER_URIS")
		os.Unsetenv("KEY_ENCRYPTION_KEY")
		os.Unsetenv("BACKUP_INTERVAL")
		os.Unsetenv("BACKUP_KEEP")
	}()

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "postgres://user:pass@localhost:5432/test", cfg.PostgresDSN)
	assert.Equal(t, "nlp:8002", cfg.NLPAddr)
	assert.Equal(t, []string{"http://server1", "http://server2"}, cfg.MCPServerURIs)
	assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", cfg.KeyEncryptionKey)
	assert.NotNil(t, cfg.EncryptionKey)
	assert.Equal(t, 32, len(cfg.EncryptionKey))
	assert.Equal(t, 14, cfg.BackupKeep)
}

func TestLoadConfig_EmptyMCPURIs(t *testing.T) {
	os.Setenv("MCP_SERVER_URIS", "")
	defer os.Unsetenv("MCP_SERVER_URIS")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Empty(t, cfg.MCPServerURIs)
}

func TestLoadConfig_OllamaDefault(t *testing.T) {
	os.Unsetenv("OLLAMA_BASE_URL")
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:11434", cfg.OllamaBaseURL)
}
