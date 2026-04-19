package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Port        int
	DataRoot    string
	PostgresDSN string
	DuckDBPath  string
	NLPAddr     string
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("PORT", 8080)
	
	wd, _ := os.Getwd()
	viper.SetDefault("DATA_ROOT", filepath.Join(wd, "data", "raw"))
	viper.SetDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/aleph?sslmode=disable")
	viper.SetDefault("DUCKDB_PATH", filepath.Join(wd, "data", "aleph.duckdb"))
	viper.SetDefault("NLP_ADDR", "localhost:8001")

	viper.AutomaticEnv()

	return &Config{
		Port:        viper.GetInt("PORT"),
		DataRoot:    viper.GetString("DATA_ROOT"),
		PostgresDSN: viper.GetString("POSTGRES_DSN"),
		DuckDBPath:  viper.GetString("DUCKDB_PATH"),
		NLPAddr:     viper.GetString("NLP_ADDR"),
	}, nil
}
