package sources

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/political"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSentimentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEnsureSentimentScoresTable(t *testing.T) {
	db := setupSentimentTestDB(t)

	err := ensureSentimentScoresTable(db)
	require.NoError(t, err)

	err = ensureSentimentScoresTable(db)
	require.NoError(t, err)

	var tableName string
	err = db.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_name = 'sentiment_scores'",
	).Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "sentiment_scores", tableName)
}

func TestStoreSentimentResults(t *testing.T) {
	db := setupSentimentTestDB(t)

	results := []political.SentimentResult{
		{
			ArticleID: "art-001",
			Title:     "Meloni promette riforme importanti",
			Date:      "2025-05-20",
			Party:     "Fratelli d'Italia",
			Score:     0.75,
			Coalition: "CDX",
			Category:  "news",
			Source:    "ansa",
		},
		{
			ArticleID: "art-001",
			Title:     "Meloni promette riforme importanti",
			Date:      "2025-05-20",
			Party:     "Partito Democratico",
			Score:     -0.25,
			Coalition: "CSX",
			Category:  "news",
			Source:    "ansa",
		},
		{
			ArticleID: "art-002",
			Title:     "Schlein attacca il governo sulla sanità",
			Date:      "2025-05-21",
			Party:     "Partito Democratico",
			Score:     -0.50,
			Coalition: "CSX",
			Category:  "politics",
			Source:    "repubblica",
		},
		{
			ArticleID: "art-003",
			Title:     "Salvini e la Lega in crescita nei sondaggi",
			Date:      "2025-05-22",
			Party:     "Lega",
			Score:     0.60,
			Coalition: "CDX",
			Category:  "polls",
			Source:    "corriere",
		},
		{
			ArticleID: "art-004",
			Title:     "Conte presenta il nuovo programma M5S",
			Date:      "2025-05-23",
			Party:     "Movimento 5 Stelle",
			Score:     0.40,
			Coalition: "CSX",
			Category:  "politics",
			Source:    "fanpage",
		},
	}

	err := StoreSentimentResults(db, results)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentiment_scores").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestStoreSentimentDeduplication(t *testing.T) {
	db := setupSentimentTestDB(t)

	first := []political.SentimentResult{
		{
			ArticleID: "art-001",
			Title:     "Original Title",
			Date:      "2025-05-20",
			Party:     "Fratelli d'Italia",
			Score:     0.50,
			Coalition: "CDX",
			Category:  "news",
			Source:    "ansa",
		},
	}
	err := StoreSentimentResults(db, first)
	require.NoError(t, err)

	second := []political.SentimentResult{
		{
			ArticleID: "art-001",
			Title:     "Updated Title",
			Date:      "2025-05-20",
			Party:     "Fratelli d'Italia",
			Score:     0.75,
			Coalition: "CDX",
			Category:  "politics",
			Source:    "ansa",
		},
	}
	err = StoreSentimentResults(db, second)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentiment_scores").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should deduplicate on (article_id, party)")

	var title, category string
	var score float64
	err = db.QueryRow(
		"SELECT title, score, category FROM sentiment_scores WHERE article_id = 'art-001'",
	).Scan(&title, &score, &category)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", title, "upsert should update title")
	assert.Equal(t, 0.75, score, "upsert should update score")
	assert.Equal(t, "politics", category, "upsert should update category")
}

func TestStoreSentimentEmptyResults(t *testing.T) {
	db := setupSentimentTestDB(t)

	err := StoreSentimentResults(db, nil)
	require.NoError(t, err)

	err = StoreSentimentResults(db, []political.SentimentResult{})
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentiment_scores").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestStoreSentimentMultiPartySameArticle(t *testing.T) {
	db := setupSentimentTestDB(t)

	results := []political.SentimentResult{
		{ArticleID: "art-001", Title: "T", Date: "2025-01-01", Party: "Fratelli d'Italia", Score: 0.8, Coalition: "CDX", Category: "", Source: "src"},
		{ArticleID: "art-001", Title: "T", Date: "2025-01-01", Party: "Partito Democratico", Score: -0.3, Coalition: "CSX", Category: "", Source: "src"},
		{ArticleID: "art-001", Title: "T", Date: "2025-01-01", Party: "Lega", Score: 0.5, Coalition: "CDX", Category: "", Source: "src"},
	}
	err := StoreSentimentResults(db, results)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentiment_scores WHERE article_id = 'art-001'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}
