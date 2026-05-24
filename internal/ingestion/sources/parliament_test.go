package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parliamentMockWatermark struct {
	lastSourceName string
}

func (m *parliamentMockWatermark) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	m.lastSourceName = sourceName
	return nil
}

const sampleSPARQL = `{
  "head": {"vars": ["votazione", "data", "titolo", "esito", "deputato", "gruppo"]},
  "results": {"bindings": [
    {"votazione": {"value": "123"}, "data": {"value": "2024-01-15"}, "titolo": {"value": "Fiducia"}, "esito": {"value": "APPROVATA"}, "deputato": {"value": "Mario Rossi"}, "gruppo": {"value": "FDI"}}
  ]}
}`

func TestSPARQLParsing(t *testing.T) {
	votes, err := ParseSPARQLVotes([]byte(sampleSPARQL))
	require.NoError(t, err)
	require.Len(t, votes, 1)
	assert.Equal(t, "123", votes[0].ID)
	assert.Equal(t, "2024-01-15", votes[0].Data)
	assert.Equal(t, "Fiducia", votes[0].Titolo)
	assert.Equal(t, "APPROVATA", votes[0].Esito)
	assert.Equal(t, "Mario Rossi", votes[0].Deputato)
	assert.Equal(t, "FDI", votes[0].Gruppo)
}

func TestSPARQLParsingMultipleVotes(t *testing.T) {
	data := `{
	  "results": {"bindings": [
	    {"votazione": {"value": "1"}, "data": {"value": "2024-01-01"}, "titolo": {"value": "Voto A"}, "esito": {"value": "APPROVATA"}, "deputato": {"value": "Anna"}, "gruppo": {"value": "PD"}},
	    {"votazione": {"value": "2"}, "data": {"value": "2024-01-02"}, "titolo": {"value": "Voto B"}, "esito": {"value": "RESPINTA"}, "deputato": {"value": "Luigi"}, "gruppo": {"value": "M5S"}}
	  ]}
	}`

	votes, err := ParseSPARQLVotes([]byte(data))
	require.NoError(t, err)
	require.Len(t, votes, 2)
	assert.Equal(t, "1", votes[0].ID)
	assert.Equal(t, "RESPINTA", votes[1].Esito)
}

func TestSPARQLParsingEmpty(t *testing.T) {
	data := `{"results": {"bindings": []}}`
	votes, err := ParseSPARQLVotes([]byte(data))
	require.NoError(t, err)
	assert.Empty(t, votes)
}

func TestSPARQLParsingMalformed(t *testing.T) {
	_, err := ParseSPARQLVotes([]byte(`not json`))
	assert.Error(t, err)
}

func TestParliamentIngestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(sampleSPARQL))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &parliamentMockWatermark{}

	err := RunParliament(context.Background(), server.URL, db, wm, t.TempDir(), "19")
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parliament_votes").Scan(&count)
	assert.Equal(t, 1, count)
	assert.Equal(t, "parliament", wm.lastSourceName)
}

func TestParliamentIngestionMultiplePages(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/sparql-results+json")
		if callCount == 1 {
			w.Write([]byte(`{"results": {"bindings": [
				{"votazione":{"value":"1"},"data":{"value":"2024-01-01"},"titolo":{"value":"Voto A"},"esito":{"value":"APPROVATA"},"deputato":{"value":"Anna"},"gruppo":{"value":"PD"}}
			]}}`))
		} else {
			w.Write([]byte(`{"results": {"bindings": []}}`))
		}
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &parliamentMockWatermark{}

	err := RunParliament(context.Background(), server.URL+"/page1", db, wm, t.TempDir(), "19")
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parliament_votes").Scan(&count)
	assert.Equal(t, 1, count)
}

func TestParliamentHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &parliamentMockWatermark{}

	err := RunParliament(context.Background(), server.URL, db, wm, t.TempDir(), "19")
	assert.ErrorContains(t, err, "500")
}

func TestParliamentInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`bad json`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &parliamentMockWatermark{}

	err := RunParliament(context.Background(), server.URL, db, wm, t.TempDir(), "19")
	assert.ErrorContains(t, err, "unmarshal")
}

func TestParliamentSPARQLQueryStructure(t *testing.T) {
	query, err := BuildSPARQLQuery("19")
	require.NoError(t, err)
	assert.Contains(t, query, "SELECT")
	assert.Contains(t, query, "?votazione")
	assert.Contains(t, query, "legislatura")
	assert.Contains(t, query, "19")
}

func TestParliamentLegislaturaValidation(t *testing.T) {
	_, err := BuildSPARQLQuery("")
	assert.ErrorContains(t, err, "legislatura")
	_, err = BuildSPARQLQuery("abc")
	assert.ErrorContains(t, err, "legislatura")
}

func TestVoteJSONTags(t *testing.T) {
	v := Vote{ID: "1", Data: "2024-01-15", Titolo: "test", Esito: "APPROVATA", Deputato: "Mario", Gruppo: "FDI"}
	data, err := json.Marshal(v)
	require.NoError(t, err)
	var decoded Vote
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, v.ID, decoded.ID)
}
