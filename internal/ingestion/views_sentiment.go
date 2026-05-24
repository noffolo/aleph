package ingestion

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// RegisterSentimentViews creates DuckDB views for sentiment analysis and
// sentiment-electoral correlation. Idempotent via CREATE OR REPLACE VIEW.
func RegisterSentimentViews(db *sql.DB) error {
	views := []string{
		`CREATE OR REPLACE VIEW v_party_sentiment_timeline AS
		 SELECT
			party,
			DATE_TRUNC('week', TRY_CAST(date AS DATE)) AS week,
			AVG(score) AS avg_score,
			COUNT(*) AS count
		 FROM sentiment_scores
		 WHERE TRY_CAST(date AS DATE) IS NOT NULL
		 GROUP BY party, week
		 ORDER BY week, party`,

		`CREATE OR REPLACE VIEW v_coalition_sentiment_timeline AS
		 SELECT
			coalition,
			DATE_TRUNC('week', TRY_CAST(date AS DATE)) AS week,
			AVG(score) AS avg_score,
			COUNT(*) AS count
		 FROM sentiment_scores
		 WHERE coalition != '' AND TRY_CAST(date AS DATE) IS NOT NULL
		 GROUP BY coalition, week
		 ORDER BY week, coalition`,

		`CREATE OR REPLACE VIEW v_social_electoral_correlation AS
		 SELECT
			s.party,
			s.coalition,
			TRY_CAST(strftime(s.date, '%Y') AS INTEGER) AS sentiment_year,
			AVG(s.score) AS avg_sentiment_score,
			COUNT(DISTINCT s.article_id) AS articles_count,
			e.year AS election_year,
			AVG(e.percentuale) AS avg_vote_share,
			COUNT(DISTINCT e.comune_istat) AS comunes_count
		 FROM sentiment_scores s
		 LEFT JOIN election_results e
			ON LOWER(s.party) = LOWER(e.party_canonical)
			AND TRY_CAST(strftime(s.date, '%Y') AS INTEGER) = e.year
		 WHERE TRY_CAST(strftime(s.date, '%Y') AS INTEGER) IS NOT NULL
		 GROUP BY s.party, s.coalition, sentiment_year, e.year
		 ORDER BY sentiment_year, s.party`,

		`CREATE OR REPLACE VIEW v_social_source_sentiment AS
		 SELECT
			source,
			party,
			AVG(score) AS avg_score,
			COUNT(*) AS count,
			MIN(TRY_CAST(date AS DATE)) AS first_date,
			MAX(TRY_CAST(date AS DATE)) AS last_date
		 FROM sentiment_scores
		 WHERE category = 'social'
		 GROUP BY source, party
		 ORDER BY source, avg_score DESC`,
	}

	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create sentiment view: %w", err)
		}
	}

	slog.Info("sentiment correlation views registered")
	return nil
}
