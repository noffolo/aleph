package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ff3300/aleph-v2/internal/political"
)

// ensureSentimentScoresTable creates the sentiment_scores table if it doesn't exist.
func ensureSentimentScoresTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sentiment_scores (
			article_id  VARCHAR,
			title       VARCHAR,
			date        VARCHAR,
			party       VARCHAR,
			score       DOUBLE,
			coalition   VARCHAR,
			category    VARCHAR,
			source      VARCHAR,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(article_id, party)
		)
	`)
	if err != nil {
		return fmt.Errorf("create sentiment_scores table: %w", err)
	}
	return nil
}

// StoreSentimentResults inserts or replaces sentiment results into DuckDB.
// Deduplicates on (article_id, party) via ON CONFLICT DO UPDATE.
func StoreSentimentResults(db *sql.DB, results []political.SentimentResult) error {
	if err := ensureSentimentScoresTable(db); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO sentiment_scores
			(article_id, title, date, party, score, coalition, category, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (article_id, party) DO UPDATE SET
			title     = excluded.title,
			date      = excluded.date,
			score     = excluded.score,
			coalition = excluded.coalition,
			category  = excluded.category,
			source    = excluded.source
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, r := range results {
		if _, err := stmt.Exec(
			r.ArticleID, r.Title, r.Date, r.Party, r.Score,
			r.Coalition, r.Category, r.Source,
		); err != nil {
			return fmt.Errorf("insert sentiment row: %w", err)
		}
	}

	return tx.Commit()
}

// articleInput is the JSON shape expected from sentiment pipeline input files.
type articleInput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Date    string `json:"date"`
	Content string `json:"content"`
	Source  string `json:"source"`
}

// RunSentimentPipeline reads JSON article files from inputDir, extracts party mentions
// via EntityExtractor, scores sentiment, and stores results in sentiment_scores.
func RunSentimentPipeline(ctx context.Context, db *sql.DB, inputDir string) error {
	extractor := political.NewEntityExtractor(political.DefaultPartyKeywords)

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("read input dir %s: %w", inputDir, err)
	}

	var allResults []political.SentimentResult

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path := filepath.Join(inputDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("skip unreadable file", "path", path, "err", err)
			continue
		}

		// Try JSON array first, fall back to single object
		var articles []articleInput
		if err := json.Unmarshal(data, &articles); err != nil {
			var single articleInput
			if err2 := json.Unmarshal(data, &single); err2 != nil {
				slog.Warn("skip invalid JSON", "path", path, "err", err2)
				continue
			}
			articles = []articleInput{single}
		}

		for _, a := range articles {
			mentions := extractor.ExtractEntities(a.Title, a.Content)
			if len(mentions) == 0 {
				continue
			}

			text := a.Title + " " + a.Content
			score := political.ScoreSentiment(text)

			source := a.Source
			if source == "" {
				source = "unknown"
			}

			for _, m := range mentions {
				allResults = append(allResults, political.SentimentResult{
					ArticleID: a.ID,
					Title:     a.Title,
					Date:      a.Date,
					Party:     m.Party,
					Score:     score,
					Coalition: political.CoalitionFor(m.Party),
					Category:  "",
					Source:    source,
				})
			}
		}
	}

	if len(allResults) == 0 {
		slog.Info("no sentiment results to store")
		return nil
	}

	if err := StoreSentimentResults(db, allResults); err != nil {
		return fmt.Errorf("store results: %w", err)
	}

	slog.Info("sentiment pipeline complete", "results", len(allResults), "files", len(entries))
	return nil
}

// RunSentimentOnSocial queries social_raw for recent posts (last 48h), runs
// entity extraction and sentiment scoring, and stores results in sentiment_scores.
// This is called after social crawl to process newly fetched content.
func RunSentimentOnSocial(ctx context.Context, db *sql.DB) error {
	extractor := political.NewEntityExtractor(political.DefaultPartyKeywords)

	rows, err := db.QueryContext(ctx, `
		SELECT id, platform, post_text, post_timestamp, author
		FROM social_raw
		ORDER BY fetched_at DESC
	`)
	if err != nil {
		// social_raw may not exist yet (first run before crawl)
		return fmt.Errorf("query social_raw: %w", err)
	}
	defer rows.Close()

	var allResults []political.SentimentResult

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var id, platform, postText, postTS, author string
		if err := rows.Scan(&id, &platform, &postText, &postTS, &author); err != nil {
			return fmt.Errorf("scan social row: %w", err)
		}

		mentions := extractor.ExtractEntities(postText, "")
		if len(mentions) == 0 {
			continue
		}

		score := political.ScoreSentiment(postText)

		for _, m := range mentions {
			articleID := fmt.Sprintf("%s-%s-%s", platform, id, m.Party)
			allResults = append(allResults, political.SentimentResult{
				ArticleID: articleID,
				Title:     truncateString(postText, 100),
				Date:      postTS,
				Party:     m.Party,
				Score:     score,
				Coalition: political.CoalitionFor(m.Party),
				Category:  "social",
				Source:    platform,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(allResults) == 0 {
		slog.Info("no sentiment results from social posts")
		return nil
	}

	if err := StoreSentimentResults(db, allResults); err != nil {
		return fmt.Errorf("store social sentiment: %w", err)
	}

	slog.Info("social sentiment analysis complete", "results", len(allResults))
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
