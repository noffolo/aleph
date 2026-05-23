package political

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testPartyKeywords = map[string][]string{
	"Fratelli d'Italia":  {"fratelli d'italia", "fdi", "meloni", "giorgia meloni"},
	"Partito Democratico": {"partito democratico", "pd", "schlein", "elly schlein", "democratico"},
	"Lega":                {"lega", "salvini", "lega nord", "matteo salvini"},
	"Movimento 5 Stelle":  {"movimento 5 stelle", "m5s", "cinque stelle", "grillo", "conte"},
}

func TestExtractParty(t *testing.T) {
	extractor := NewEntityExtractor(testPartyKeywords)

	mentions := extractor.ExtractEntities(
		"Meloni ha annunciato nuove riforme economiche",
		"La premier Giorgia Meloni ha presentato il piano",
	)

	require.Len(t, mentions, 1)
	assert.Equal(t, "Fratelli d'Italia", mentions[0].Party)
}

func TestExtractParty_NoMatch(t *testing.T) {
	extractor := NewEntityExtractor(testPartyKeywords)

	mentions := extractor.ExtractEntities(
		"Nuovo modello Ferrari presentato al salone",
		"La nuova auto sportiva raggiunge i 300 km/h",
	)

	assert.Empty(t, mentions)
}

func TestExtractParty_Multiple(t *testing.T) {
	extractor := NewEntityExtractor(testPartyKeywords)

	mentions := extractor.ExtractEntities(
		"Salvini e Meloni insieme per la riforma",
		"I leader della Lega e di Fratelli d'Italia collaborano",
	)

	require.Len(t, mentions, 2)
	parties := make(map[string]bool)
	for _, m := range mentions {
		parties[m.Party] = true
	}
	assert.True(t, parties["Lega"])
	assert.True(t, parties["Fratelli d'Italia"])
}

func TestScoreSentiment_Positive(t *testing.T) {
	score := ScoreSentiment("crescita economica e sviluppo positivo")
	assert.Greater(t, score, 0.0)
}

func TestScoreSentiment_Negative(t *testing.T) {
	score := ScoreSentiment("crisi e fallimento del governo")
	assert.Less(t, score, 0.0)
}

func TestScoreSentiment_Neutral(t *testing.T) {
	score := ScoreSentiment("l'incontro si è tenuto martedì")
	assert.InDelta(t, 0.0, score, 0.15)
}

func TestCoalition(t *testing.T) {
	assert.Equal(t, "CSX", CoalitionFor("Partito Democratico"))
	assert.Equal(t, "CDX", CoalitionFor("Fratelli d'Italia"))
	assert.Equal(t, "", CoalitionFor("Unknown Party"))
}

func TestStoreAndExport(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	// Create test results
	results := []SentimentResult{
		{
			ArticleID: "art-1",
			Title:     "Schlein propone riforma",
			Date:      "2024-01-15",
			Party:     "Partito Democratico",
			Score:     0.5,
			Coalition: "CSX",
			Category:  "politica",
			Source:    "Corriere della Sera",
		},
		{
			ArticleID: "art-2",
			Title:     "Meloni critica l'opposizione",
			Date:      "2024-01-15",
			Party:     "Fratelli d'Italia",
			Score:     -0.3,
			Coalition: "CDX",
			Category:  "politica",
			Source:    "Corriere della Sera",
		},
		{
			ArticleID: "art-3",
			Title:     "Nuova proposta PD",
			Date:      "2024-01-22",
			Party:     "Partito Democratico",
			Score:     0.7,
			Coalition: "CSX",
			Category:  "politica",
			Source:    "Corriere della Sera",
		},
	}

	err = StoreSentiment(db, results)
	require.NoError(t, err)

	// Verify rows exist
	rows, err := db.Query("SELECT COUNT(*) FROM political_sentiment")
	require.NoError(t, err)
	defer rows.Close()

	rows.Next()
	var count int
	err = rows.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Export JSON
	outDir := t.TempDir()
	err = ExportWeeklySentiment(db, outDir)
	require.NoError(t, err)

	// Verify weekly_sentiment.json exists and has valid JSON
	weeklyData, err := os.ReadFile(outDir + "/weekly_sentiment.json")
	require.NoError(t, err)
	var weekly interface{}
	err = json.Unmarshal(weeklyData, &weekly)
	require.NoError(t, err)

	// Verify coalition_weekly.json exists and has valid JSON
	coalitionData, err := os.ReadFile(outDir + "/coalition_weekly.json")
	require.NoError(t, err)
	var coalition interface{}
	err = json.Unmarshal(coalitionData, &coalition)
	require.NoError(t, err)
}
