package sources

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"log/slog"
)

// ensureSocialRawTable creates the social_raw table if it doesn't exist.
func ensureSocialRawTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS social_raw (
			id            VARCHAR,
			platform      VARCHAR,
			post_text     VARCHAR,
			post_url      VARCHAR,
			author        VARCHAR,
			post_timestamp VARCHAR,
			fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(id, platform)
		)
	`)
	if err != nil {
		return fmt.Errorf("create social_raw table: %w", err)
	}
	return nil
}

// SocialCrawlConfig holds configuration for the social crawl scheduler.
type SocialCrawlConfig struct {
	WorkDir     string
	ScriptsDir  string
	DBPath      string
	RunSentiment bool
}

// DefaultSocialCrawlConfig returns sensible defaults for the social crawl.
func DefaultSocialCrawlConfig() SocialCrawlConfig {
	return SocialCrawlConfig{
		WorkDir:      ".",
		ScriptsDir:   "scripts",
		DBPath:       "data/aleph.duckdb",
		RunSentiment: true,
	}
}

// RunSocialCrawlSchedule runs the Python crawl_all.py script, then consolidates
// newly fetched posts from each platform table into social_raw. If cfg.RunSentiment
// is true, it also triggers the sentiment pipeline on the consolidated data.
func RunSocialCrawlSchedule(ctx context.Context, db *sql.DB, cfg SocialCrawlConfig) error {
	slog.Info("starting social crawl schedule")

	scriptPath := filepath.Join(cfg.WorkDir, cfg.ScriptsDir, "crawl_all.py")
	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	cmd.Dir = cfg.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("crawl_all.py exited with error", "err", err, "output", string(output))
	} else {
		slog.Info("crawl_all.py completed successfully")
	}

	if err := ensureSocialRawTable(db); err != nil {
		return fmt.Errorf("ensure social_raw table: %w", err)
	}

	if err := consolidateSocialPosts(ctx, db); err != nil {
		return fmt.Errorf("consolidate social posts: %w", err)
	}

	slog.Info("social raw consolidation complete")

	if cfg.RunSentiment {
		if err := RunSentimentOnSocial(ctx, db); err != nil {
			slog.Warn("sentiment on social failed", "err", err)
		}
	}

	return nil
}

// consolidateSocialPosts copies posts from per-platform tables into social_raw.
// Uses INSERT OR IGNORE to handle deduplication on (id, platform).
func consolidateSocialPosts(ctx context.Context, db *sql.DB) error {
	queries := []struct {
		platform string
		query    string
	}{
		{
			platform: "x",
			query: fmt.Sprintf(`
				INSERT OR IGNORE INTO social_raw (id, platform, post_text, post_url, author, post_timestamp)
				SELECT
					tweet_id,
					'%s',
					content,
					url,
					COALESCE(p.full_name, ''),
					CAST(posted_at AS VARCHAR)
				FROM posts_x px
				LEFT JOIN politici p ON px.politico_id = p.id
				WHERE content IS NOT NULL
			`, "x"),
		},
		{
			platform: "instagram",
			query: fmt.Sprintf(`
				INSERT OR IGNORE INTO social_raw (id, platform, post_text, post_url, author, post_timestamp)
				SELECT
					post_id,
					'%s',
					caption,
					post_url,
					COALESCE(p.full_name, ''),
					CAST(posted_at AS VARCHAR)
				FROM posts_ig pg
				LEFT JOIN politici p ON pg.politico_id = p.id
				WHERE caption IS NOT NULL
			`, "instagram"),
		},
		{
			platform: "facebook",
			query: fmt.Sprintf(`
				INSERT OR IGNORE INTO social_raw (id, platform, post_text, post_url, author, post_timestamp)
				SELECT
					post_id,
					'%s',
					content,
					post_url,
					COALESCE(p.full_name, ''),
					CAST(posted_at AS VARCHAR)
				FROM posts_fb pf
				LEFT JOIN politici p ON pf.politico_id = p.id
				WHERE content IS NOT NULL
			`, "facebook"),
		},
		{
			platform: "telegram",
			query: fmt.Sprintf(`
				INSERT OR IGNORE INTO social_raw (id, platform, post_text, post_url, author, post_timestamp)
				SELECT
					message_id,
					'%s',
					content,
					'',
					channel_name,
					CAST(posted_at AS VARCHAR)
				FROM posts_telegram
				WHERE content IS NOT NULL
			`, "telegram"),
		},
	}

	for _, q := range queries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := db.ExecContext(ctx, q.query)
		if err != nil {
			if isMissingTableError(err) {
				slog.Info("platform table not found, skipping", "platform", q.platform)
				continue
			}
			return fmt.Errorf("consolidate %s posts: %w", q.platform, err)
		}

		n, _ := result.RowsAffected()
		slog.Info("consolidated social posts", "platform", q.platform, "rows", n)
	}

	return nil
}

// isMissingTableError checks if the error indicates a missing table in DuckDB.
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "table") && (strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found"))
}

// RunSocialScheduledLoop runs the social crawl on a ticker for polling scenarios.
// blocks until ctx is cancelled.
func RunSocialScheduledLoop(ctx context.Context, db *sql.DB, cfg SocialCrawlConfig, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("social crawl scheduler started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("social crawl scheduler stopped")
			return
		case <-ticker.C:
			if err := RunSocialCrawlSchedule(ctx, db, cfg); err != nil {
				slog.Error("social crawl schedule failed", "err", err)
			}
		}
	}
}
