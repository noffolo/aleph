package config

import (
	"fmt"
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
	KeyEncryptionKey  string
	EncryptionKey     []byte
	JWTSecret         []byte
	BackupInterval    string
	BackupDir         string
	BackupKeep        int
	RateLimitChat          int
	RateLimitHealth        int
	RateLimitDefault       int
	MaxProjects            int
	MaxAgentsPerProject    int
	CORSAllowedOrigins     []string
	LLMTimeoutSeconds      int
	SlowQueryThresholdMs   int
}

func LoadConfig() (*Config, error) {
	return LoadConfigWithSecrets(LoadSecrets())
}

func LoadConfigWithSecrets(secrets SecretsProvider) (*Config, error) {
	viper.SetDefault("PORT", 8080)

	wd, _ := os.Getwd()
	viper.SetDefault("DATA_ROOT", filepath.Join(wd, "data", "raw"))
	viper.SetDefault("DUCKDB_PATH", filepath.Join(wd, "data", "aleph.duckdb"))
	viper.SetDefault("DUCKDB_SCHEMA", "main")
	viper.SetDefault("EMBEDDING_MODEL", "nomic-embed-text")
	viper.SetDefault("BACKUP_INTERVAL", "24h")
	viper.SetDefault("BACKUP_DIR", filepath.Join(wd, "data", "backups", "duckdb"))
	viper.SetDefault("BACKUP_KEEP", 7)
	viper.SetDefault("RATE_LIMIT_CHAT", 10)
	viper.SetDefault("RATE_LIMIT_HEALTH", 100)
	viper.SetDefault("RATE_LIMIT_DEFAULT", 500)
	viper.SetDefault("LLM_TIMEOUT_SECONDS", 30)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")
	viper.SetDefault("MAX_PROJECTS", 50)
	viper.SetDefault("MAX_AGENTS_PER_PROJECT", 20)
	viper.SetDefault("SLOW_QUERY_THRESHOLD_MS", 500)

	viper.AutomaticEnv()

	jwtSecret := secrets.String("jwt.secret", viper.GetString("JWT_SECRET"))
	if jwtSecret == "" {
		return nil, fmt.Errorf("FATAL: jwt.secret is required — set in gosecrets or JWT_SECRET env var")
	}

	postgresDSN := secrets.String("postgres.dsn", secrets.String("database.url", viper.GetString("POSTGRES_DSN")))
	if postgresDSN == "" {
		return nil, fmt.Errorf("FATAL: postgres.dsn is required — set in gosecrets or POSTGRES_DSN env var")
	}

	rawKey := secrets.String("key_encryption_key", "")
	if rawKey == "" {
		if data, err := os.ReadFile("/run/secrets/key_encryption_key"); err == nil {
			rawKey = strings.TrimSpace(string(data))
		}
	}
	if rawKey == "" {
		rawKey = viper.GetString("KEY_ENCRYPTION_KEY")
	}
	if rawKey == "" {
		return nil, fmt.Errorf("FATAL: key_encryption_key is required — API keys must be encrypted at rest")
	}

	decoded, err := crypto.LoadEncryptionKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("FATAL: key_encryption_key is invalid (%v)", err)
	}

	ollamaBaseURL := secrets.String("ollama.base_url", viper.GetString("OLLAMA_BASE_URL"))
	nlpAddr := secrets.String("nlp.sidecar_url", viper.GetString("NLP_ADDR"))

	return &Config{
		Port:              viper.GetInt("PORT"),
		DataRoot:          viper.GetString("DATA_ROOT"),
		PostgresDSN:       postgresDSN,
		DuckDBPath:        viper.GetString("DUCKDB_PATH"),
		NLPAddr:           nlpAddr,
		OllamaBaseURL:     ollamaBaseURL,
		DuckDBSchema:      viper.GetString("DUCKDB_SCHEMA"),
		EmbeddingModel:    viper.GetString("EMBEDDING_MODEL"),
		MCPServerURIs:     parseMCPServerURIs(viper.GetString("MCP_SERVER_URIS")),
		KeyEncryptionKey:  rawKey,
		EncryptionKey:     decoded,
		JWTSecret:         []byte(jwtSecret),
		BackupInterval:    viper.GetString("BACKUP_INTERVAL"),
		BackupDir:         viper.GetString("BACKUP_DIR"),
		BackupKeep:        viper.GetInt("BACKUP_KEEP"),
		RateLimitChat:        viper.GetInt("RATE_LIMIT_CHAT"),
		RateLimitHealth:      viper.GetInt("RATE_LIMIT_HEALTH"),
		RateLimitDefault:     viper.GetInt("RATE_LIMIT_DEFAULT"),
		MaxProjects:          viper.GetInt("MAX_PROJECTS"),
		MaxAgentsPerProject:  viper.GetInt("MAX_AGENTS_PER_PROJECT"),
		CORSAllowedOrigins:   parseCORSOrigins(viper.GetString("CORS_ALLOWED_ORIGINS")),
		LLMTimeoutSeconds:    viper.GetInt("LLM_TIMEOUT_SECONDS"),
		SlowQueryThresholdMs: viper.GetInt("SLOW_QUERY_THRESHOLD_MS"),
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

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{"http://localhost:5173", "http://localhost:3000"}
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