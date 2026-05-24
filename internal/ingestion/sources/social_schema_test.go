package sources

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSocialTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRegisterSocialMediaTablesIdempotent(t *testing.T) {
	db := setupSocialTestDB(t)

	err := RegisterSocialMediaTables(db)
	require.NoError(t, err)

	// Second call must succeed (idempotent via IF NOT EXISTS)
	err = RegisterSocialMediaTables(db)
	require.NoError(t, err)
}

func TestSocialMediaTablesExist(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var tables []string
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'main'
		ORDER BY table_name
	`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	require.NoError(t, rows.Err())

	assert.Contains(t, tables, "posts_x")
	assert.Contains(t, tables, "posts_instagram")
	assert.Contains(t, tables, "posts_facebook")
	assert.Contains(t, tables, "politici")
}

func TestPostsXColumns(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var cols []string
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'posts_x' ORDER BY ordinal_position`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		cols = append(cols, c)
	}

	assert.Len(t, cols, 11)
	assert.Contains(t, cols, "id")
	assert.Contains(t, cols, "politico_id")
	assert.Contains(t, cols, "created_at")
	assert.Contains(t, cols, "text")
	assert.Contains(t, cols, "hashtags")
	assert.Contains(t, cols, "like_count")
	assert.Contains(t, cols, "retweet_count")
	assert.Contains(t, cols, "reply_count")
	assert.Contains(t, cols, "quote_count")
	assert.Contains(t, cols, "source")
	assert.Contains(t, cols, "ingested_at")
}

func TestPostsInstagramColumns(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var cols []string
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'posts_instagram' ORDER BY ordinal_position`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		cols = append(cols, c)
	}

	assert.Len(t, cols, 11)
	assert.Contains(t, cols, "shortcode")
	assert.Contains(t, cols, "politico_id")
	assert.Contains(t, cols, "taken_at")
	assert.Contains(t, cols, "caption")
	assert.Contains(t, cols, "hashtags")
	assert.Contains(t, cols, "like_count")
	assert.Contains(t, cols, "comments_count")
	assert.Contains(t, cols, "media_type")
	assert.Contains(t, cols, "media_url")
	assert.Contains(t, cols, "source")
	assert.Contains(t, cols, "ingested_at")
}

func TestPostsFacebookColumns(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var cols []string
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'posts_facebook' ORDER BY ordinal_position`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		cols = append(cols, c)
	}

	assert.Len(t, cols, 8)
	assert.Contains(t, cols, "post_id")
	assert.Contains(t, cols, "politico_id")
	assert.Contains(t, cols, "created_time")
	assert.Contains(t, cols, "message")
	assert.Contains(t, cols, "shares")
	assert.Contains(t, cols, "reactions")
	assert.Contains(t, cols, "source")
	assert.Contains(t, cols, "ingested_at")
}

func TestPoliticiColumns(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var cols []string
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'politici' ORDER BY ordinal_position`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		cols = append(cols, c)
	}

	assert.Len(t, cols, 7)
	assert.Contains(t, cols, "id")
	assert.Contains(t, cols, "full_name")
	assert.Contains(t, cols, "party")
	assert.Contains(t, cols, "screen_name_x")
	assert.Contains(t, cols, "username_ig")
	assert.Contains(t, cols, "page_id_fb")
	assert.Contains(t, cols, "created_at")
}

func TestPostsUnifiedViewExists(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var views []string
	rows, err := db.Query(`SELECT view_name FROM duckdb_views()`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		views = append(views, v)
	}
	require.NoError(t, rows.Err())

	assert.Contains(t, views, "v_posts_unified")
}

func TestPostsUnifiedViewQueryable(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	_, err := db.Exec(`INSERT INTO posts_x (id, politico_id, created_at, text, like_count, retweet_count, reply_count, quote_count) VALUES ('x-1', 'p1', '2024-01-01', 'Hello from X', 10, 2, 1, 0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO posts_instagram (shortcode, politico_id, taken_at, caption, like_count, comments_count, media_type, media_url) VALUES ('ig-1', 'p1', '2024-01-01', 'Hello from IG', 20, 5, 'IMAGE', 'https://example.com/img.jpg')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO posts_facebook (post_id, politico_id, created_time, message, shares) VALUES ('fb-1', 'p1', '2024-01-01', 'Hello from FB', 3)`)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM v_posts_unified").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify source column works
	var source string
	err = db.QueryRow("SELECT source FROM v_posts_unified WHERE post_id = 'x-1'").Scan(&source)
	require.NoError(t, err)
	assert.Equal(t, "x", source)

	err = db.QueryRow("SELECT source FROM v_posts_unified WHERE post_id = 'ig-1'").Scan(&source)
	require.NoError(t, err)
	assert.Equal(t, "instagram", source)

	err = db.QueryRow("SELECT source FROM v_posts_unified WHERE post_id = 'fb-1'").Scan(&source)
	require.NoError(t, err)
	assert.Equal(t, "facebook", source)
}

func TestPostsUnifiedViewEmpty(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM v_posts_unified").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPoliticiInsert(t *testing.T) {
	db := setupSocialTestDB(t)
	require.NoError(t, RegisterSocialMediaTables(db))

	_, err := db.Exec(`INSERT INTO politici (id, full_name, party, screen_name_x, username_ig, page_id_fb)
		VALUES ('p-meloni', 'Giorgia Meloni', 'FDI', '@GiorgiaMeloni', 'giorgiameloni', 'giorgiameloni')`)
	require.NoError(t, err)

	var name string
	err = db.QueryRow("SELECT full_name FROM politici WHERE id = 'p-meloni'").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Giorgia Meloni", name)
}
