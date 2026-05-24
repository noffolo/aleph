package sources

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollsTableCreation(t *testing.T) {
	db := setupTestDuckDB(t)

	require.NoError(t, ensurePollsTable(db))
	require.NoError(t, ensurePollResultsTable(db))

	var tableCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM duckdb_tables() WHERE table_name IN ('polls', 'poll_results')").Scan(&tableCount))
	assert.Equal(t, 2, tableCount)
}

func TestPollsTableIdempotent(t *testing.T) {
	db := setupTestDuckDB(t)

	require.NoError(t, ensurePollsTable(db))
	require.NoError(t, ensurePollsTable(db))

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM polls").Scan(&count))
	assert.Equal(t, 0, count)
}

func TestRunPollIngestion(t *testing.T) {
	csvContent := `Row,Data Inserimento,Realizzatore,Committente,Titolo,text,domanda,national_poll_rationale,national_poll,Partito Democratico,Forza Italia,Fratelli d'Italia,Alleanza Verdi Sinistra,Lega,Movimento 5 Stelle,+Europa,Azione,Italia Viva,Stati Uniti d'Europa,Pace Terra Dignità,Azione - Italia Viva,Azione/+Europa,Sinistra Ecologia Libertà,Scelta Civica,Unione di Centro,Sud Chiama Nord,Unione Popolare,Altri
1,19/05/2026,SWG S.p.A.,La7,intenzioni di voto,text content,domanda text,rationale,1,22.2,7.6,28.5,6.7,6.0,12.5,1.4,3.5,2.4,,,,,,,,,,11.1
2,10/04/2026,Tecnè,Quarta Repubblica,intenzioni di voto,text content,domanda text,rationale,1,21.0,8.1,27.9,5.9,7.2,11.8,2.1,3.8,2.1,,,,,,,,,,10.1
3,15/01/2024,Euromedia Research,Porta a Porta,intenzioni di voto,text content,domanda text,rationale,1,19.8,9.5,28.2,4.5,8.5,14.2,,,,,,,,,,,,15.3
`

	tmpFile, err := os.CreateTemp("", "polls_test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	db := setupTestDuckDB(t)
	ctx := context.Background()

	err = RunPollIngestion(ctx, db, tmpFile.Name())
	require.NoError(t, err)

	var rowCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM polls").Scan(&rowCount))
	assert.Greater(t, rowCount, 0)

	var pdCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM polls WHERE party_canonical = 'partito-democratico'").Scan(&pdCount))
	assert.Equal(t, 3, pdCount)

	var canonical string
	require.NoError(t, db.QueryRow("SELECT party_canonical FROM polls WHERE party = 'Fratelli d''Italia' LIMIT 1").Scan(&canonical))
	assert.Equal(t, "fratelli-italia", canonical)

	require.NoError(t, db.QueryRow("SELECT party_canonical FROM polls WHERE party = 'Forza Italia' LIMIT 1").Scan(&canonical))
	assert.Equal(t, "forza-italia", canonical)
}

func TestRunPollIngestionSkipsNonNational(t *testing.T) {
	csvContent := `Row,Data Inserimento,Realizzatore,Committente,Titolo,text,domanda,national_poll_rationale,national_poll,Partito Democratico,Forza Italia
1,19/05/2026,SWG,La7,poll,text,domanda,rationale,0,22.2,7.6
2,10/04/2026,X,Quarta,poll,text,domanda,rationale,1,21.0,9.0
`

	tmpFile, err := os.CreateTemp("", "polls_nonat_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	db := setupTestDuckDB(t)
	ctx := context.Background()

	err = RunPollIngestion(ctx, db, tmpFile.Name())
	require.NoError(t, err)

	var rowCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM polls").Scan(&rowCount))
	assert.Equal(t, 2, rowCount)

	var canonical string
	require.NoError(t, db.QueryRow("SELECT party_canonical FROM polls WHERE party = 'Forza Italia' LIMIT 1").Scan(&canonical))
	assert.Equal(t, "forza-italia", canonical)
}

func TestRunPollIngestionEmptyFile(t *testing.T) {
	csvContent := `Row,Data Inserimento,Realizzatore,Committente,Titolo,text,domanda,national_poll_rationale,national_poll,Partito Democratico
`

	tmpFile, err := os.CreateTemp("", "polls_empty_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	db := setupTestDuckDB(t)
	ctx := context.Background()

	err = RunPollIngestion(ctx, db, tmpFile.Name())
	require.NoError(t, err)

	var rowCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM polls").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)
}

func TestCanonicalizePollParty(t *testing.T) {
	assert.Equal(t, "partito-democratico", canonicalizePollParty("Partito Democratico"))
	assert.Equal(t, "fratelli-italia", canonicalizePollParty("Fratelli d'Italia"))
	assert.Equal(t, "forza-italia", canonicalizePollParty("Forza Italia"))
	assert.Equal(t, "lega", canonicalizePollParty("Lega"))
	assert.Equal(t, "movimento-5-stelle", canonicalizePollParty("Movimento 5 Stelle"))
	assert.Equal(t, "verdi-sinistra", canonicalizePollParty("Alleanza Verdi Sinistra"))
	assert.Equal(t, "piu-europa", canonicalizePollParty("+Europa"))
	assert.Equal(t, "azione", canonicalizePollParty("Azione"))
	assert.Equal(t, "italia-viva", canonicalizePollParty("Italia Viva"))

	unknown := canonicalizePollParty("NON ESISTE AFFATTO")
	assert.Equal(t, "NON ESISTE AFFATTO", unknown)
}

func TestParseItalianDate(t *testing.T) {
	d, err := parseItalianDate("19/05/2026")
	require.NoError(t, err)
	assert.Equal(t, 2026, d.Year())
	assert.Equal(t, time.May, d.Month())
	assert.Equal(t, 19, d.Day())

	d, err = parseItalianDate("01/12/2023")
	require.NoError(t, err)
	assert.Equal(t, 2023, d.Year())

	_, err = parseItalianDate("2023-12-01")
	assert.Error(t, err)
}

func TestParseItalianFloat(t *testing.T) {
	v, err := parseItalianFloat("22,2")
	require.NoError(t, err)
	assert.InDelta(t, 22.2, v, 0.01)

	v, err = parseItalianFloat("15.7")
	require.NoError(t, err)
	assert.InDelta(t, 15.7, v, 0.01)

	v, err = parseItalianFloat("0")
	require.NoError(t, err)
	assert.Equal(t, 0.0, v)

	_, err = parseItalianFloat("abc")
	assert.Error(t, err)
}

func TestSanitizeID(t *testing.T) {
	assert.Equal(t, "swg_s_p_a", sanitizeID("SWG S.p.A."))
	assert.Equal(t, "tecnè", sanitizeID("Tecnè"))
	assert.Equal(t, "emg_srl", sanitizeID("EMG Srl"))
	assert.Equal(t, "euromedia_research", sanitizeID("Euromedia Research"))
}
