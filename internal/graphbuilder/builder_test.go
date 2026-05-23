package graphbuilder

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (
		donation_amount REAL, donation_year INTEGER,
		recipient_party TEXT, donor_type TEXT, donor_name TEXT
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO party_funding VALUES (50000, 2022, 'Partito Democratico', 'Persona Fisica', 'Mario Rossi')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO party_funding VALUES (100000, 2023, 'Partito Democratico', 'Societa', 'ACME SpA')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO party_funding VALUES (25000, 2022, 'Fratelli d Italia', 'Persona Fisica', 'Luigi Verdi')`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS election_results_2022_camera (
		codice TEXT, descrizione TEXT, voti_validi REAL, perc REAL
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO election_results_2022_camera VALUES ('001', 'PARTITO DEMOCRATICO', 5000000, 19.0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO election_results_2022_camera VALUES ('002', 'FRATELLI D ITALIA', 7500000, 26.0)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (
		id TEXT, name TEXT, country TEXT, position TEXT, party TEXT
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pep_entities VALUES ('p1', 'Mario Rossi', 'IT', 'Deputato', 'Partito Democratico')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pep_entities VALUES ('p2', 'Giorgia Bianchi', 'IT', 'Senatore', 'Fratelli d Italia')`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS opdm_memberships (
		person_id TEXT, person_name TEXT, org_name TEXT, role TEXT
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO opdm_memberships VALUES ('m1', 'Mario Rossi', 'Partito Democratico', 'Membro')`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS parliament_votes (
		id TEXT, votazione TEXT, data TEXT, titolo TEXT, esito TEXT, deputato TEXT, gruppo TEXT
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS election_metadata_2022_camera (
		"CODICE ISTAT" TEXT, "COMUNE" TEXT, "REGIONE" TEXT, "PROVINCIA" TEXT
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO election_metadata_2022_camera VALUES ('001', 'Roma', 'Lazio', 'RM')`)
	require.NoError(t, err)

	return db
}

func TestBuildGraph(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewPoliticalGraphBuilder(db)
	err := builder.Build()
	require.NoError(t, err)

	assert.Greater(t, builder.Graph.NumNodes(), 0, "graph should have nodes")
	assert.Greater(t, builder.Graph.NumEdges(), 0, "graph should have edges")
}

func TestAnalyzeTrends(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Build first so we have the builder context (though trends uses its own queries)
	builder := NewPoliticalGraphBuilder(db)

	report, err := builder.AnalyzeTrends()
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, len(report.FundingConcentration), 0, "should have funding concentration data")
}

func TestExportGraph(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewPoliticalGraphBuilder(db)
	err := builder.Build()
	require.NoError(t, err)

	export := builder.ExportGraph()
	assert.Greater(t, len(export.Nodes), 0, "export should have nodes")
	assert.Greater(t, len(export.Edges), 0, "export should have edges")
}

func TestTrainGNN(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewPoliticalGraphBuilder(db)
	err := builder.Build()
	require.NoError(t, err)

	result, err := builder.TrainGNN(32, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result.LossHistory), 0, "should have loss history")
	assert.NotZero(t, result.EpochsRun)
	assert.GreaterOrEqual(t, result.AUC, 0.0)
	assert.LessOrEqual(t, result.AUC, 1.0)
}

func TestBuildGraphEdgeCases(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (donation_amount REAL, donation_year INTEGER, recipient_party TEXT, donor_type TEXT, donor_name TEXT)`)
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS election_results_2022_camera (codice TEXT, descrizione TEXT, voti_validi REAL, perc REAL)`)
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (id TEXT, name TEXT, country TEXT, position TEXT, party TEXT)`)
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS opdm_memberships (person_id TEXT, person_name TEXT, org_name TEXT, role TEXT)`)
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS parliament_votes (id TEXT, votazione TEXT, data TEXT, titolo TEXT, esito TEXT, deputato TEXT, gruppo TEXT)`)
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS election_metadata_2022_camera ("CODICE ISTAT" TEXT, "COMUNE" TEXT, "REGIONE" TEXT, "PROVINCIA" TEXT)`)
		require.NoError(t, err)

		builder := NewPoliticalGraphBuilder(db)
		err = builder.Build()
		require.NoError(t, err)
		assert.Equal(t, 0, builder.Graph.NumNodes())

		// TrainGNN should handle empty graph gracefully
		result, err := builder.TrainGNN(32, 10)
		assert.Error(t, err, "training on empty graph should return error")
		assert.Nil(t, result)
	})

	t.Run("normalized node IDs", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		builder := NewPoliticalGraphBuilder(db)
		err := builder.Build()
		require.NoError(t, err)

		// Check that node IDs are normalized to lowercase-with-hyphens
		for id := range builder.Graph.Nodes {
			for _, ch := range string(id) {
				// Uppercase letters should not appear
				assert.False(t, ch >= 'A' && ch <= 'Z', "node ID %s contains uppercase letters", id)
			}
		}
	})
}

func TestExportPredictions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewPoliticalGraphBuilder(db)
	err := builder.Build()
	require.NoError(t, err)

	result, err := builder.TrainGNN(32, 10)
	require.NoError(t, err)

	model := result.Model
	embeddings := model.Forward()
	nodeIndex := builder.Graph.BuildNodeIndex()

	predictions := builder.ExportPredictions(model, embeddings, nodeIndex, result.AUC, result.MRR)
	assert.NotNil(t, predictions)
	assert.NotNil(t, predictions.Predictions)
	// With a small test graph, we should get some predictions
	assert.Greater(t, len(predictions.Predictions), 0)
	assert.Equal(t, result.AUC, predictions.ModelMetadata.AUC)
	assert.Equal(t, result.MRR, predictions.ModelMetadata.MRR)
}

func TestTrendNormalization(t *testing.T) {
	// Test that party name normalization works for matching across tables
	db := setupTestDB(t)
	defer db.Close()

	builder := NewPoliticalGraphBuilder(db)
	report, err := builder.AnalyzeTrends()
	require.NoError(t, err)

	// The VoteFundingRatio query should be able to join despite different case
	// in election_results vs party_funding (e.g., "PARTITO DEMOCRATICO" vs "Partito Democratico")
	for _, vfr := range report.VoteFundingRatio {
		assert.NotZero(t, vfr.CostPerVote) // If matched, cost should be calculated
	}
}
