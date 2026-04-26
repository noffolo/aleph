package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port              int
	DataRoot          string
	PostgresDSN       string
	DuckDBPath        string
	NLPAddr           string
	OllamaBaseURL     string
	MCPServerURIs     []string
	KeyEncryptionKey  string
	BackupInterval    string
	BackupDir         string
	BackupKeep        int
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("PORT", 8080)
	
	wd, _ := os.Getwd()
	viper.SetDefault("DATA_ROOT", filepath.Join(wd, "data", "raw"))
	viper.SetDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable")
	viper.SetDefault("DUCKDB_PATH", filepath.Join(wd, "data", "aleph.duckdb"))
	viper.SetDefault("NLP_ADDR", "localhost:8001")
	viper.SetDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	viper.SetDefault("BACKUP_KEEP", 7)

	viper.AutomaticEnv()

	return &Config{
		Port:              viper.GetInt("PORT"),
		DataRoot:          viper.GetString("DATA_ROOT"),
		PostgresDSN:       viper.GetString("POSTGRES_DSN"),
		DuckDBPath:        viper.GetString("DUCKDB_PATH"),
		NLPAddr:           viper.GetString("NLP_ADDR"),
		OllamaBaseURL:     viper.GetString("OLLAMA_BASE_URL"),
		MCPServerURIs:     parseMCPServerURIs(viper.GetString("MCP_SERVER_URIS")),
		KeyEncryptionKey:  viper.GetString("KEY_ENCRYPTION_KEY"),
		BackupKeep:        viper.GetInt("BACKUP_KEEP"),
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
