package sources

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
)

//go:embed social_schema.sql
var socialSchemaSQL string

// RegisterSocialMediaTables creates DuckDB tables and views for social media storage
// (posts_x, posts_instagram, posts_facebook, politici, v_posts_unified).
// Idempotent via CREATE TABLE IF NOT EXISTS and CREATE OR REPLACE VIEW.
func RegisterSocialMediaTables(db *sql.DB) error {
	statements := splitDDL(socialSchemaSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("register social media schema: %w", err)
		}
	}
	slog.Info("social media tables registered")
	return nil
}

func splitDDL(sqlText string) []string {
	return strings.Split(sqlText, ";")
}
