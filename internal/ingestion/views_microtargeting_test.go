package ingestion

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMicrotargetingTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS election_results (
			election_type TEXT, year INTEGER, level TEXT,
			comune TEXT, comune_istat TEXT, lista TEXT,
			party_canonical TEXT, voti INTEGER, percentuale FLOAT,
			seggi INTEGER, elettori INTEGER, votanti INTEGER,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS istat_population (
			comune_istat TEXT, year INTEGER, popolazione_residente INTEGER,
			eta_media DOUBLE, indice_vecchiaia DOUBLE,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS istat_income (
			comune_istat TEXT, year INTEGER,
			reddito_medio DOUBLE, contribuenti INTEGER, importo_totale DOUBLE,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS istat_employment (
			comune_istat TEXT, year INTEGER,
			tasso_occupazione DOUBLE, tasso_disoccupazione DOUBLE,
			tasso_inattivita DOUBLE, addetti_totali INTEGER, ul_totali INTEGER,
			ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sentiment_scores (
			article_id TEXT, party TEXT, date TEXT, score DOUBLE,
			source TEXT, category TEXT,
			UNIQUE(article_id, party)
		)`,
		`CREATE TABLE IF NOT EXISTS polls (
			pollster TEXT, date TEXT, party_canonical TEXT,
			percentage DOUBLE, ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS parliamentary_acts (
			party_at_presentation TEXT, first_signer TEXT, legislature INTEGER,
			act_type TEXT, title TEXT
		)`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func seedMicrotargetingTestData(t *testing.T, db *sql.DB) {
	for _, stmt := range []string{
		`INSERT INTO election_results VALUES
		 ('politiche', 2022, 'comune', 'Roma', '058091', 'FDI',
		  'fratelli-italia', 10000, 35.0, 0, 50000, 30000, NOW()),
		 ('politiche', 2022, 'comune', 'Milano', '015146', 'PD',
		  'partito-democratico', 8000, 28.0, 0, 40000, 25000, NOW()),
		 ('politiche', 2018, 'comune', 'Roma', '058091', 'FDI',
		  'fratelli-italia', 5000, 18.0, 0, 50000, 28000, NOW())`,
		`INSERT INTO istat_population VALUES
		 ('058091', 2022, 2800000, 46.5, 180.0, NOW()),
		 ('015146', 2022, 1400000, 44.2, 160.0, NOW())`,
		`INSERT INTO istat_income VALUES
		 ('058091', 2022, 22000, 1200000, 2.6e10, NOW()),
		 ('015146', 2022, 28000, 800000, 2.2e10, NOW())`,
		`INSERT INTO istat_employment VALUES
		 ('058091', 2022, 62.0, 9.5, 28.5, 1200000, 240000, NOW()),
		 ('015146', 2022, 70.0, 6.0, 24.0, 900000, 180000, NOW())`,
		`INSERT INTO sentiment_scores VALUES
		 ('a1', 'fratelli-italia', '2022-06-01', 0.75, 'x', 'social'),
		 ('a2', 'partito-democratico', '2022-06-01', 0.60, 'x', 'social')`,
		`INSERT INTO polls VALUES
		 ('SWG', '2022-08-01', 'fratelli-italia', 28.5, NOW()),
		 ('SWG', '2022-08-01', 'partito-democratico', 22.0, NOW())`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}
}

func TestMicrotargetingViewsCompile(t *testing.T) {
	db := setupMicrotargetingTestDB(t)
	seedMicrotargetingTestData(t, db)

	// This will fail: RegisterMicrotargetingViews doesn't exist yet
	err := RegisterMicrotargetingViews(db)
	require.NoError(t, err)

	// Idempotent: second call must not error
	err = RegisterMicrotargetingViews(db)
	require.NoError(t, err)

	var viewNames []string
	rows, err := db.Query("SELECT view_name FROM duckdb_views() WHERE view_name LIKE 'v_%' AND view_name NOT IN (SELECT view_name FROM duckdb_views() WHERE view_name LIKE 'v_%' LIMIT 0)")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		viewNames = append(viewNames, v)
	}

	assert.Contains(t, viewNames, "v_comune_party_segments")
	assert.Contains(t, viewNames, "v_party_segment_strength")
	assert.Contains(t, viewNames, "v_swing_comuni")
	assert.Contains(t, viewNames, "v_party_intelligence_dashboard")
}

func TestComunePartySegmentsQueryable(t *testing.T) {
	db := setupMicrotargetingTestDB(t)
	seedMicrotargetingTestData(t, db)
	require.NoError(t, RegisterMicrotargetingViews(db))

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM v_comune_party_segments").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestPartySegmentStrengthQueryable(t *testing.T) {
	db := setupMicrotargetingTestDB(t)
	seedMicrotargetingTestData(t, db)
	require.NoError(t, RegisterMicrotargetingViews(db))

	var party string
	err := db.QueryRow("SELECT party_canonical FROM v_party_segment_strength LIMIT 1").Scan(&party)
	require.NoError(t, err)
	assert.NotEmpty(t, party)
}

func TestSwingComuniQueryable(t *testing.T) {
	db := setupMicrotargetingTestDB(t)
	seedMicrotargetingTestData(t, db)
	require.NoError(t, RegisterMicrotargetingViews(db))

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM v_swing_comuni").Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}

func TestPartyIntelligenceDashboardQueryable(t *testing.T) {
	db := setupMicrotargetingTestDB(t)
	seedMicrotargetingTestData(t, db)
	require.NoError(t, RegisterMicrotargetingViews(db))

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM v_party_intelligence_dashboard").Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}
