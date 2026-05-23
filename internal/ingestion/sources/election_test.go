package sources

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestISTATLookup(t *testing.T) {
	codes := NewISTATLookup()
	// Roma
	name, found := codes.Lookup("058091")
	assert.True(t, found)
	assert.Contains(t, name, "Roma")
	// Nonexistent
	_, found = codes.Lookup("999999")
	assert.False(t, found)
}

func TestPartyMappingExactMatch(t *testing.T) {
	mapper := NewPartyMapper()
	mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
	mapper.AddAlias("FRATELLI D'ITALIA - GIORGIA MELONI", "fratelli-italia")
	mapper.AddAlias("PARTITO DEMOCRATICO", "partito-democratico")
	mapper.AddAlias("PD", "partito-democratico")

	// Exact match
	canonical, found := mapper.Lookup("PARTITO DEMOCRATICO")
	require.True(t, found)
	assert.Equal(t, "partito-democratico", canonical)

	// Alias match
	canonical, found = mapper.Lookup("PD")
	require.True(t, found)
	assert.Equal(t, "partito-democratico", canonical)

	// No match
	_, found = mapper.Lookup("LISTA INESISTENTE XYZ")
	assert.False(t, found)
}

func TestPartyMappingManualOverride(t *testing.T) {
	mapper := NewPartyMapper()
	mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
	mapper.SetOverride("FRATELLI D'ITALIA - ROMA", "fratelli-italia-roma")

	// Manual override takes priority over alias match
	canonical, found := mapper.Lookup("FRATELLI D'ITALIA - ROMA")
	require.True(t, found)
	assert.Equal(t, "fratelli-italia-roma", canonical)

	// Without override, uses alias table
	canonical, found = mapper.Lookup("FRATELLI D'ITALIA")
	require.True(t, found)
	assert.Equal(t, "fratelli-italia", canonical)
}

func TestElectionConfigValidation(t *testing.T) {
	valid := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "comune", Year: 2022}
	assert.NoError(t, valid.Validate())

	invalidType := ElectionConfig{ElectionType: "fantasia", ElectionDate: "20220925", Level: "comune", Year: 2022}
	assert.ErrorContains(t, invalidType.Validate(), "invalid election_type")

	invalidLevel := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "quartiere", Year: 2022}
	assert.ErrorContains(t, invalidLevel.Validate(), "invalid level")

	invalidYear := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "comune", Year: 1990}
	assert.ErrorContains(t, invalidYear.Validate(), "year before 2000")
}

func TestElectionFetcherRateLimit(t *testing.T) {
	var gotAcceptHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAcceptHeader = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"intestazione":{},"enti":{"ente":[]}}`))
	}))
	defer srv.Close()

	fetcher := NewElectionFetcher(srv.URL, 1.0)
	cfg := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "comune", Year: 2022}
	ctx := context.Background()

	// First call should be fast (token available immediately)
	start := time.Now()
	_, err := fetcher.GetEntities(ctx, cfg)
	firstDuration := time.Since(start)
	require.NoError(t, err)
	assert.Equal(t, "application/json", gotAcceptHeader)
	assert.Less(t, firstDuration, 200*time.Millisecond, "first call should return immediately")

	// Second call should wait (rate limited: 1 req/s, burst=1)
	start = time.Now()
	_, err = fetcher.GetEntities(ctx, cfg)
	secondDuration := time.Since(start)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, secondDuration, 800*time.Millisecond, "second call should be rate-limited")
}

func TestElectionFetcherTEMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"intestazione":{"te":"TE01"},"enti":{"ente":[{"cod":"058091","desc":"ROMA"}]}}`))
	}))
	defer srv.Close()

	fetcher := NewElectionFetcher(srv.URL, 5.0)
	cfg := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "comune", Year: 2022}
	ctx := context.Background()

	entities, err := fetcher.GetEntities(ctx, cfg)
	require.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Equal(t, "058091", entities[0].Cod)
	assert.Equal(t, "ROMA", entities[0].Desc)
}

func setupTestDuckDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRunElectionFullPipeline(t *testing.T) {
	var callCount atomic.Int64

	getentiBody := `{"intestazione":{"te":"TE01"},"enti":{"ente":[{"cod":"058091","desc":"ROMA"},{"cod":"015146","desc":"MILANO"}]}}`
	scrutiniBody := `{"intestazione":{"cod":"058091"},"liste":{"lista":[{"desc":"PARTITO DEMOCRATICO","voti":50000,"perc":30.5,"seggi":10}]},"datiGenerali":{"elettori":200000,"votanti":165000}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/getentiFI/") {
			w.Write([]byte(getentiBody))
			return
		}
		w.Write([]byte(scrutiniBody))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	mapper := NewPartyMapper()
	mapper.AddAlias("PARTITO DEMOCRATICO", "partito-democratico")

	rawDir := t.TempDir()
	cfg := ElectionConfig{ElectionType: "politiche", ElectionDate: "20220925", Level: "comune", Year: 2022}
	ctx := context.Background()

	results, err := RunElection(ctx, db, srv.URL, cfg, mapper, rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.GreaterOrEqual(t, callCount.Load(), int64(3))

	_, statErr := os.Stat(filepath.Join(rawDir, "election", "2022-politiche-comune", "getenti.json"))
	assert.NoError(t, statErr)

	var rowCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM election_results WHERE year = 2022").Scan(&rowCount))
	assert.Greater(t, rowCount, 0)

	var canonical string
	require.NoError(t, db.QueryRow("SELECT party_canonical FROM election_results WHERE lista = 'PARTITO DEMOCRATICO' LIMIT 1").Scan(&canonical))
	assert.Equal(t, "partito-democratico", canonical)
}

func TestElectionDateMap(t *testing.T) {
	assert.Equal(t, "20220925", GetElectionDate("politiche", 2022))
	assert.Equal(t, "", GetElectionDate("politiche", 2000))   // no data
	assert.Equal(t, "", GetElectionDate("comunali", 2023))    // empty map
	assert.Equal(t, "", GetElectionDate("fantasia", 2022))    // unknown type
}
