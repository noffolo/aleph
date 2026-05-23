package ingestion

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupViewsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (id TEXT, name TEXT, party TEXT, position TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS opdm_memberships (person_id INTEGER, org_id INTEGER, role TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS parliament_votes (id TEXT, data TEXT, titolo TEXT, esito TEXT, deputato TEXT, gruppo TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS public_contracts (cig TEXT, aggiudicatario TEXT, importo REAL)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (recipient_party TEXT, donation_amount TEXT, donation_year INTEGER)`)
	require.NoError(t, err)

	return db
}

func TestCrossReferenceViews(t *testing.T) {
	db := setupViewsTestDB(t)

	err := RegisterCrossReferenceViews(db)
	require.NoError(t, err)

	err = RegisterCrossReferenceViews(db)
	require.NoError(t, err)

	var views []string
	rows, err := db.Query("SELECT view_name FROM duckdb_views() WHERE view_name LIKE 'v_%'")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		views = append(views, v)
	}
	assert.GreaterOrEqual(t, len(views), 3)
	assert.Contains(t, views, "v_politician_full_profile")
	assert.Contains(t, views, "v_contract_party_link")
	assert.Contains(t, views, "v_funding_timeline")
}

func TestCrossReferenceViewsQueryable(t *testing.T) {
	db := setupViewsTestDB(t)
	require.NoError(t, RegisterCrossReferenceViews(db))

	_, err := db.Exec(`INSERT INTO pep_entities VALUES ('1', 'Mario Rossi', 'FDI', 'Deputato')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO opdm_memberships VALUES (1, 10, 'Membro')`)
	require.NoError(t, err)

	var name string
	err = db.QueryRow("SELECT name FROM v_politician_full_profile LIMIT 1").Scan(&name)
	require.NoError(t, err)
	assert.Contains(t, name, "Mario")
}

func TestCrossReferenceViewsEmptyTables(t *testing.T) {
	db := setupViewsTestDB(t)
	require.NoError(t, RegisterCrossReferenceViews(db))

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM v_politician_full_profile").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = db.QueryRow("SELECT COUNT(*) FROM v_funding_timeline").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
