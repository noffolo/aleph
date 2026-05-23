package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	valid := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022}
	assert.NoError(t, valid.Validate())

	invalidType := ElectionConfig{ElectionType: "fantasia", Level: "comune", Year: 2022}
	assert.ErrorContains(t, invalidType.Validate(), "invalid election_type")

	invalidLevel := ElectionConfig{ElectionType: "politiche", Level: "quartiere", Year: 2022}
	assert.ErrorContains(t, invalidLevel.Validate(), "invalid level")

	invalidYear := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 1990}
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
	cfg := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022}
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
	cfg := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022}
	ctx := context.Background()

	entities, err := fetcher.GetEntities(ctx, cfg)
	require.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Equal(t, "058091", entities[0].Cod)
	assert.Equal(t, "ROMA", entities[0].Desc)
}
