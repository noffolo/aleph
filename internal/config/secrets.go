package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/bilustek/gosecrets"
)

// SecretsProvider abstracts credential loading so tests can inject fakes.
type SecretsProvider interface {
	// String returns the secret value for key, or fallback if not found.
	String(key string, fallback ...string) string
	// MustString returns the secret value for key, panicking if not found.
	MustString(key string) string
	// Has reports whether the key exists.
	Has(key string) bool
}

type realSecrets struct {
	inner *gosecrets.Secrets
}

func (r *realSecrets) String(key string, fallback ...string) string { return r.inner.String(key, fallback...) }
func (r *realSecrets) MustString(key string) string                  { return r.inner.MustString(key) }
func (r *realSecrets) Has(key string) bool                           { return r.inner.Has(key) }

// envSecrets implements SecretsProvider using environment variables.
// Keys use dot-notation (e.g. "database.url") and are mapped to env var names
// (e.g. "DATABASE_URL") via secretToEnvKey().
type envSecrets struct{}

func (e *envSecrets) String(key string, fallback ...string) string {
	if v := os.Getenv(secretToEnvKey(key)); v != "" {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

func (e *envSecrets) MustString(key string) string {
	envKey := secretToEnvKey(key)
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("secrets: required key %q (env %q) not found", key, envKey))
	}
	return v
}

func (e *envSecrets) Has(key string) bool {
	return os.Getenv(secretToEnvKey(key)) != ""
}

// secretToEnvKey maps a dot-notation secret key to an environment variable name.
// Examples: "jwt.secret" → "JWT_SECRET", "key_encryption_key" → "KEY_ENCRYPTION_KEY",
// "database.url" → "DATABASE_URL", "aleph.api_key_secret_backend" → "ALEPH_API_KEY_SECRET_BACKEND"
func secretToEnvKey(key string) string {
	// Explicit mappings for keys that don't follow simple dot→underscore convention
	mappings := map[string]string{
		"jwt.secret":                    "JWT_SECRET",
		"key_encryption_key":            "KEY_ENCRYPTION_KEY",
		"aleph.api_key_secret_backend":  "ALEPH_API_KEY_SECRET_BACKEND",
		"database.url":                   "POSTGRES_DSN",
		"postgres.dsn":                   "POSTGRES_DSN",
		"smtp.password":                 "SMTP_PASSWORD",
		"ollama.base_url":               "OLLAMA_BASE_URL",
		"nlp.sidecar_url":               "NLP_ADDR",
	}
	if mapped, ok := mappings[key]; ok {
		return mapped
	}
	// Default: replace dots with underscores, uppercase
	s := strings.ReplaceAll(key, ".", "_")
	return strings.ToUpper(s)
}

// LoadSecrets returns a SecretsProvider.
//
// Resolution order:
//  1. GOSECRETS_ENV=ci → read from environment variables (CI mode, no encrypted file needed)
//  2. gosecrets.Load()  → read from encrypted credentials file
//  3. Fallback          → read from environment variables with a warning
func LoadSecrets() SecretsProvider {
	env := os.Getenv("GOSECRETS_ENV")

	if env == "ci" {
		slog.Info("[secrets] CI mode: reading secrets from environment variables")
		return &envSecrets{}
	}

	s, err := gosecrets.Load(gosecrets.WithEnv(env))
	if err == nil {
		slog.Info("[secrets] loaded encrypted secrets from gosecrets", "env", envName())
		return &realSecrets{inner: s}
	}

	slog.Warn("[secrets] gosecrets unavailable; falling back to environment variables",
		"error", err.Error(),
		"hint", "set GOSECRETS_ENV=ci to suppress this warning",
	)
	return &envSecrets{}
}

func envName() string {
	if v := os.Getenv("GOSECRETS_ENV"); v != "" {
		return v
	}
	return "development"
}