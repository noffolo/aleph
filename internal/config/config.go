package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/ff3300/aleph-v2/internal/crypto"
)

type Config struct {
	Port              int
	DataRoot          string
	PostgresDSN       string
	DuckDBPath        string
	DuckDBSchema      string
	EmbeddingModel    string
	NLPAddr           string
	OllamaBaseURL     string
	MCPServerURIs     []string
	KeyEncryptionKey  string // raw hex string from env (kept for compat/display)
	EncryptionKey     []byte // decoded 32-byte AES-256 key (nil if not set)
	BackupInterval    string
	BackupDir         string
	BackupKeep        int
	RateLimitChat     int
	RateLimitHealth   int
	RateLimitDefault  int
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("PORT", 8080)
	
	wd, _ := os.Getwd()
	viper.SetDefault("DATA_ROOT", filepath.Join(wd, "data", "raw"))
	viper.SetDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable")
	viper.SetDefault("DUCKDB_PATH", filepath.Join(wd, "data", "aleph.duckdb"))
	viper.SetDefault("NLP_ADDR", "localhost:8001")
	viper.SetDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	viper.SetDefault("DUCKDB_SCHEMA", "main")
	viper.SetDefault("EMBEDDING_MODEL", "nomic-embed-text")
	viper.SetDefault("BACKUP_KEEP", 7)
	viper.SetDefault("RATE_LIMIT_CHAT", 10)
	viper.SetDefault("RATE_LIMIT_HEALTH", 100)
	viper.SetDefault("RATE_LIMIT_DEFAULT", 500)

	viper.AutomaticEnv()

	rawKey := viper.GetString("KEY_ENCRYPTION_KEY")
	var encKey []byte
	if rawKey != "" {
		decoded, err := crypto.LoadEncryptionKey(rawKey)
		if err != nil {
			log.Printf("[config] WARNING: KEY_ENCRYPTION_KEY is invalid (%v) — API keys stored in PLAINTEXT", err)
		} else {
			encKey = decoded
		}
	} else {
		log.Printf("[config] WARNING: KEY_ENCRYPTION_KEY not set — API keys stored in PLAINTEXT")
	}

	return &Config{
		Port:              viper.GetInt("PORT"),
		DataRoot:          viper.GetString("DATA_ROOT"),
		PostgresDSN:       viper.GetString("POSTGRES_DSN"),
		DuckDBPath:        viper.GetString("DUCKDB_PATH"),
		NLPAddr:           viper.GetString("NLP_ADDR"),
		OllamaBaseURL:     viper.GetString("OLLAMA_BASE_URL"),
		DuckDBSchema:      viper.GetString("DUCKDB_SCHEMA"),
		EmbeddingModel:    viper.GetString("EMBEDDING_MODEL"),
		MCPServerURIs:     parseMCPServerURIs(viper.GetString("MCP_SERVER_URIS")),
		KeyEncryptionKey:  rawKey,
		EncryptionKey:     encKey,
		BackupKeep:        viper.GetInt("BACKUP_KEEP"),
		RateLimitChat:     viper.GetInt("RATE_LIMIT_CHAT"),
		RateLimitHealth:   viper.GetInt("RATE_LIMIT_HEALTH"),
		RateLimitDefault:  viper.GetInt("RATE_LIMIT_DEFAULT"),
	}, nil
}

func parseMCPServerURIs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
