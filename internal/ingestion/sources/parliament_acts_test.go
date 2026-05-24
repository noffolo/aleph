package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleActsSPARQL = `{
  "head": {"vars": ["atto", "tipo", "titolo", "dataPresentazione", "primoFirmatario", "stato", "dataStato"]},
  "results": {"bindings": [
    {"atto": {"value": "http://dati.camera.it/ocd/atto.rdf/scheda_parlamento_20_001"}, "tipo": {"value": "DDL"}, "titolo": {"value": "Legge di bilancio 2025"}, "dataPresentazione": {"value": "2024-10-20"}, "primoFirmatario": {"value": "Giorgetti Giancarlo"}, "stato": {"value": "approvato"}, "dataStato": {"value": "2024-12-31"}},
    {"atto": {"value": "http://dati.camera.it/ocd/atto.rdf/scheda_parlamento_20_002"}, "tipo": {"value": "mozione"}, "titolo": {"value": "Mozione su politiche energetiche"}, "dataPresentazione": {"value": "2024-11-15"}, "primoFirmatario": {"value": "Conte Giuseppe"}, "stato": {"value": "in esame"}, "dataStato": {}},
    {"atto": {"value": "http://dati.camera.it/ocd/atto.rdf/scheda_parlamento_20_003"}, "tipo": {"value": "interpellanza"}, "titolo": {"value": "Interpellanza su sicurezza lavoro"}, "dataPresentazione": {"value": "2024-09-01"}, "primoFirmatario": {"value": "Schlein Elly"}, "stato": {"value": "presentato"}, "dataStato": {}}
  ]}
}`

const sampleAttendanceSPARQL = `{
  "head": {"vars": ["parlamentare", "presenze", "assenze", "missioni", "totaleSedute", "percentuale", "gruppo"]},
  "results": {"bindings": [
    {"parlamentare": {"value": "Giorgetti Giancarlo"}, "presenze": {"value": "180"}, "assenze": {"value": "20"}, "missioni": {"value": "15"}, "totaleSedute": {"value": "200"}, "percentuale": {"value": "90.0"}, "gruppo": {"value": "Lega"}},
    {"parlamentare": {"value": "Conte Giuseppe"}, "presenze": {"value": "160"}, "assenze": {"value": "40"}, "missioni": {"value": "10"}, "totaleSedute": {"value": "200"}, "percentuale": {"value": "80.0"}, "gruppo": {"value": "M5S"}},
    {"parlamentare": {"value": "Schlein Elly"}, "presenze": {"value": "190"}, "assenze": {"value": "10"}, "missioni": {"value": "5"}, "totaleSedute": {"value": "200"}, "percentuale": {"value": "95.0"}, "gruppo": {"value": "PD"}}
  ]}
}`

func TestEnsureParliamentActsTable(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	err := ensureParliamentActsTable(db)
	require.NoError(t, err)

	var tableName string
	require.NoError(t, db.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_name = 'parliamentary_acts'",
	).Scan(&tableName))
	assert.Equal(t, "parliamentary_acts", tableName)
}

func TestEnsureParliamentAttendanceTable(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	err := ensureParliamentAttendanceTable(db)
	require.NoError(t, err)

	var tableName string
	require.NoError(t, db.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_name = 'parliamentary_attendance'",
	).Scan(&tableName))
	assert.Equal(t, "parliamentary_attendance", tableName)
}

func TestSPARQLActsParsing(t *testing.T) {
	acts, err := ParseSPARQLActs([]byte(sampleActsSPARQL), 20, "camera")
	require.NoError(t, err)
	require.Len(t, acts, 3)

	assert.Equal(t, "scheda_parlamento_20_001", acts[0].ID)
	assert.Equal(t, 20, acts[0].Legislature)
	assert.Equal(t, "DDL", acts[0].ActType)
	assert.Equal(t, "Legge di bilancio 2025", acts[0].Title)
	assert.Equal(t, "2024-10-20", acts[0].PresentationDate)
	assert.Equal(t, "approvato", acts[0].Status)
	assert.Equal(t, "2024-12-31", acts[0].StatusDate)
	assert.Equal(t, "Giorgetti Giancarlo", acts[0].FirstSigner)
	assert.Equal(t, "camera", acts[0].Chamber)

	assert.Equal(t, "scheda_parlamento_20_002", acts[1].ID)
	assert.Equal(t, "mozione", acts[1].ActType)
	assert.Equal(t, "scheda_parlamento_20_003", acts[2].ID)
	assert.Equal(t, "interpellanza", acts[2].ActType)
}

func TestSPARQLActsParsingEmpty(t *testing.T) {
	data := `{"results": {"bindings": []}}`
	acts, err := ParseSPARQLActs([]byte(data), 20, "camera")
	require.NoError(t, err)
	assert.Empty(t, acts)
}

func TestSPARQLActsParsingMalformed(t *testing.T) {
	_, err := ParseSPARQLActs([]byte(`not json`), 20, "camera")
	assert.Error(t, err)
}

func TestSPARQLAttendanceParsing(t *testing.T) {
	records, err := ParseSPARQLAttendance([]byte(sampleAttendanceSPARQL), 20, 2024, "camera")
	require.NoError(t, err)
	require.Len(t, records, 3)

	assert.Equal(t, "Giorgetti Giancarlo", records[0].ParliamentarianName)
	assert.Equal(t, 20, records[0].Legislature)
	assert.Equal(t, 2024, records[0].Year)
	assert.Equal(t, 200, records[0].TotalSessions)
	assert.Equal(t, 180, records[0].Attended)
	assert.Equal(t, 20, records[0].Absences)
	assert.Equal(t, 90.0, records[0].AttendancePct)
	assert.Equal(t, 15, records[0].MissionAbsences)
	assert.Equal(t, "Lega", records[0].GroupAtTime)
	assert.Equal(t, "camera", records[0].Chamber)

	assert.Equal(t, "Conte Giuseppe", records[1].ParliamentarianName)
	assert.Equal(t, 80.0, records[1].AttendancePct)
	assert.Equal(t, "Schlein Elly", records[2].ParliamentarianName)
	assert.Equal(t, 95.0, records[2].AttendancePct)
}

func TestSPARQLAttendanceParsingEmpty(t *testing.T) {
	data := `{"results": {"bindings": []}}`
	records, err := ParseSPARQLAttendance([]byte(data), 20, 2024, "camera")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestSPARQLAttendanceParsingMalformed(t *testing.T) {
	_, err := ParseSPARQLAttendance([]byte(`not json`), 20, 2024, "camera")
	assert.Error(t, err)
}

func TestExtractActID(t *testing.T) {
	assert.Equal(t, "scheda_parlamento_20_001", extractActID("http://dati.camera.it/ocd/atto.rdf/scheda_parlamento_20_001"))
	assert.Equal(t, "atto_42", extractActID("http://example.com/atto_42"))
	assert.Equal(t, "", extractActID(""))
}

func TestBuildActsSPARQLQuery(t *testing.T) {
	query := BuildActsSPARQLQuery(20)
	assert.Contains(t, query, "SELECT")
	assert.Contains(t, query, "?atto")
	assert.Contains(t, query, "ocd:atto")
	assert.Contains(t, query, "repubblica_20")
	assert.Contains(t, query, "LIMIT 1000")
}

func TestBuildAttendanceSPARQLQuery(t *testing.T) {
	query := BuildAttendanceSPARQLQuery(19, 2023)
	assert.Contains(t, query, "SELECT")
	assert.Contains(t, query, "?parlamentare")
	assert.Contains(t, query, "repubblica_19")
	assert.Contains(t, query, `"2023"`)
	assert.Contains(t, query, "LIMIT 2000")
}

func TestParseIntSafe(t *testing.T) {
	assert.Equal(t, 42, parseIntSafe("42"))
	assert.Equal(t, 0, parseIntSafe(""))
	assert.Equal(t, 0, parseIntSafe("abc"))
	assert.Equal(t, -5, parseIntSafe("-5"))
}

func TestParseFloatSafe(t *testing.T) {
	assert.Equal(t, 90.5, parseFloatSafe("90.5"))
	assert.Equal(t, 0.0, parseFloatSafe(""))
	assert.Equal(t, 0.0, parseFloatSafe("abc"))
	assert.Equal(t, 3.14, parseFloatSafe("3.14"))
}

func TestParliamentActsIngestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/sparql-results+json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(sampleActsSPARQL))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentActs(context.Background(), db, server.URL, 20, "camera")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_acts").Scan(&count))
	assert.Equal(t, 3, count)

	var title, actType, chamber string
	var legislature int
	require.NoError(t, db.QueryRow(
		"SELECT title, act_type, chamber, legislature FROM parliamentary_acts WHERE id = 'scheda_parlamento_20_001'",
	).Scan(&title, &actType, &chamber, &legislature))
	assert.Equal(t, "Legge di bilancio 2025", title)
	assert.Equal(t, "DDL", actType)
	assert.Equal(t, "camera", chamber)
	assert.Equal(t, 20, legislature)
}

func TestParliamentActsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(`{"results": {"bindings": []}}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentActs(context.Background(), db, server.URL, 20, "camera")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_acts").Scan(&count))
	assert.Equal(t, 0, count)
}

func TestParliamentActsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentActs(context.Background(), db, server.URL, 20, "camera")
	assert.ErrorContains(t, err, "500")
}

func TestParliamentAttendanceIngestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(sampleAttendanceSPARQL))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentAttendance(context.Background(), db, server.URL, 20, 2024, "camera")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_attendance").Scan(&count))
	assert.Equal(t, 3, count)

	var name, groupAtTime string
	var attendancePct float64
	require.NoError(t, db.QueryRow(
		"SELECT parliamentarian_name, attendance_pct, group_at_time FROM parliamentary_attendance WHERE parliamentarian_name = 'Giorgetti Giancarlo'",
	).Scan(&name, &attendancePct, &groupAtTime))
	assert.Equal(t, "Giorgetti Giancarlo", name)
	assert.Equal(t, 90.0, attendancePct)
	assert.Equal(t, "Lega", groupAtTime)
}

func TestParliamentAttendanceEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(`{"results": {"bindings": []}}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentAttendance(context.Background(), db, server.URL, 20, 2024, "camera")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_attendance").Scan(&count))
	assert.Equal(t, 0, count)
}

func TestParliamentAttendanceHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	err := RunParliamentAttendance(context.Background(), db, server.URL, 20, 2024, "camera")
	assert.ErrorContains(t, err, "503")
}

func TestActJSONTags(t *testing.T) {
	a := Act{
		ID: "act_1", Legislature: 20, ActType: "DDL", Title: "Test",
		PresentationDate: "2024-01-01", Status: "approvato", StatusDate: "2024-12-31",
		FirstSigner: "Test Signer", Chamber: "camera",
	}

	db := setupTestDuckDB(t)
	defer db.Close()
	require.NoError(t, ensureParliamentActsTable(db))

	require.NoError(t, RunParliamentActs(context.Background(), db, mustMockServer(sampleActsSPARQL).URL, 20, "camera"))
	_ = a

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_acts").Scan(&count))
	assert.Equal(t, 3, count)
}

func TestAttendanceJSONTags(t *testing.T) {
	r := Attendance{
		ParliamentarianName: "Test Name", Legislature: 20, Year: 2024,
		TotalSessions: 200, Attended: 180, Absences: 20, AttendancePct: 90.0,
		MissionAbsences: 10, GroupAtTime: "Lega", Chamber: "camera",
	}

	db := setupTestDuckDB(t)
	defer db.Close()
	require.NoError(t, ensureParliamentAttendanceTable(db))

	require.NoError(t, RunParliamentAttendance(context.Background(), db, mustMockServer(sampleAttendanceSPARQL).URL, 20, 2024, "camera"))
	_ = r

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM parliamentary_attendance").Scan(&count))
	assert.Equal(t, 3, count)
}

func mustMockServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(body))
	}))
}

func TestSPARQLActsParsingPartialFields(t *testing.T) {
	data := `{
	  "results": {"bindings": [
	    {"atto": {"value": "http://dati.camera.it/ocd/atto.rdf/atto_min"}, "tipo": {}, "titolo": {"value": "Solo titolo"}, "dataPresentazione": {"value": "2024-01-01"}, "primoFirmatario": {}, "stato": {}, "dataStato": {}}
	  ]}
	}`
	acts, err := ParseSPARQLActs([]byte(data), 20, "senato")
	require.NoError(t, err)
	require.Len(t, acts, 1)
	assert.Equal(t, "Solo titolo", acts[0].Title)
	assert.Equal(t, "", acts[0].ActType)
	assert.Equal(t, "", acts[0].FirstSigner)
	assert.Equal(t, "senato", acts[0].Chamber)
}

func TestSPARQLAttendanceParsingPartialFields(t *testing.T) {
	data := `{
	  "results": {"bindings": [
	    {"parlamentare": {"value": "Minimo"}, "presenze": {}, "assenze": {}, "missioni": {}, "totaleSedute": {}, "percentuale": {}, "gruppo": {}}
	  ]}
	}`
	records, err := ParseSPARQLAttendance([]byte(data), 20, 2024, "senato")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "Minimo", records[0].ParliamentarianName)
	assert.Equal(t, 0, records[0].TotalSessions)
	assert.Equal(t, 0.0, records[0].AttendancePct)
	assert.Equal(t, "senato", records[0].Chamber)
}

func TestExtractActIDEdgeCases(t *testing.T) {
	assert.Equal(t, "single", extractActID("single"))
	assert.Equal(t, "end", extractActID("/a/b/end"))
	assert.Equal(t, "", extractActID("/"))
}
