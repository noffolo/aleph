package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParlGovTableCreation(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	require.NoError(t, ensureParlGovPartiesTable(db))
	require.NoError(t, ensureParlGovElectionsTable(db))
	require.NoError(t, ensureParlGovElectionResultsTable(db))
	require.NoError(t, ensureParlGovCabinetsTable(db))

	var tables []string
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'parlgov_%' ORDER BY table_name")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	assert.Equal(t, []string{"parlgov_cabinets", "parlgov_election_results", "parlgov_elections", "parlgov_parties"}, tables)
}

func TestParlGovPartiesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		if page <= 2 {
			nextURL := srv.URL + "/page/" + fmt.Sprint(page+1)
			w.Write([]byte(fmt.Sprintf(
				`{"count":3,"next":%q,"results":[
					{"id":%d,"name_short":"PD%d","name":"Partito Democratico %d","name_english":"Democratic Party %d","country_id":26,"date_founded":"2007-10-14","date_dissolved":null,"party_category":"social democratic","party_orientation":"centre-left"}
				]}`,
				nextURL, page, page, page, page,
			)))
		} else {
			w.Write([]byte(fmt.Sprintf(
				`{"count":3,"next":null,"results":[
					{"id":%d,"name_short":"FdI%d","name":"Fratelli dItalia %d","name_english":"Brothers of Italy %d","country_id":26,"date_founded":"2012-12-17","date_dissolved":null,"party_category":"national conservative","party_orientation":"right"}
				]}`,
				page, page, page, page,
			)))
		}
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovParties(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_parties").Scan(&count)
	assert.Equal(t, 3, count)

	var name string
	db.QueryRow("SELECT name FROM parlgov_parties WHERE id = 1").Scan(&name)
	assert.Equal(t, "Partito Democratico 1", name)
}

func TestParlGovElectionsSinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":1,"next":null,"results":[
			{"id":800,"country_id":26,"election_date":"2022-09-25","early":false,"election_type_id":1,"election_type":"parliament","name":"Camera dei deputati 2022","wikipedia":"https://it.wikipedia.org/wiki/Elezioni_politiche_italiane_del_2022","seats_total":400,"electorate":51000000,"votes_cast":35000000,"votes_valid":34000000,"data_source":"parlgov"}
		]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovElections(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_elections").Scan(&count)
	assert.Equal(t, 1, count)

	var ectype string
	var seats *int
	db.QueryRow("SELECT election_type, seats_total FROM parlgov_elections WHERE id = 800").Scan(&ectype, &seats)
	assert.Equal(t, "parliament", ectype)
	require.NotNil(t, seats)
	assert.Equal(t, 400, *seats)
}

func TestParlGovElectionResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":2,"next":null,"results":[
			{"id":5000,"election_id":800,"party_id":5,"votes":5000000,"vote_share":19.07,"seats":69,"seats_total":400},
			{"id":5001,"election_id":800,"party_id":16,"votes":7500000,"vote_share":26.00,"seats":119,"seats_total":400}
		]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovResults(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_election_results").Scan(&count)
	assert.Equal(t, 2, count)

	var votes int
	var share float64
	db.QueryRow("SELECT votes, vote_share FROM parlgov_election_results WHERE id = 5001").Scan(&votes, &share)
	assert.Equal(t, 7500000, votes)
	assert.InDelta(t, 26.0, share, 0.01)
}

func TestParlGovCabinets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":1,"next":null,"results":[
			{"id":700,"country_id":26,"election_id":null,"start_date":"2022-10-22","end_date":null,"name":"Meloni","cabinet_name":"Governo Meloni","caretaker":false,"description":"2022-10-22 – present, centre-right coalition"}
		]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovCabinets(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_cabinets").Scan(&count)
	assert.Equal(t, 1, count)

	var name string
	var caretaker bool
	db.QueryRow("SELECT name, caretaker FROM parlgov_cabinets WHERE id = 700").Scan(&name, &caretaker)
	assert.Equal(t, "Meloni", name)
	assert.False(t, caretaker)
}

func TestParlGovEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovParties(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_parties").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestParlGovHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovParties(context.Background(), client, db, srv.URL, 26)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestParlGovPartyPositions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":3,"next":null,"results":[
			{"id":5,"name_short":"PD","name":"Partito Democratico","name_english":"Democratic Party","country_id":26,"date_founded":"2007-10-14","date_dissolved":null,"party_category":"social democratic","party_orientation":"centre-left"},
			{"id":16,"name_short":"FdI","name":"Fratelli dItalia","name_english":"Brothers of Italy","country_id":26,"date_founded":"2012-12-17","date_dissolved":null,"party_category":"national conservative","party_orientation":"right"},
			{"id":999,"name_short":"UNK","name":"Unknown Party","name_english":null,"country_id":26,"date_founded":null,"date_dissolved":null,"party_category":null,"party_orientation":null}
		]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovParties(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	var leftRight *float64
	var euPosition *float64
	db.QueryRow("SELECT left_right, eu_position FROM parlgov_parties WHERE id = 5").Scan(&leftRight, &euPosition)
	require.NotNil(t, leftRight)
	require.NotNil(t, euPosition)
	assert.InDelta(t, 3.99, *leftRight, 0.01)
	assert.InDelta(t, 6.59, *euPosition, 0.01)

	db.QueryRow("SELECT left_right, eu_position FROM parlgov_parties WHERE id = 16").Scan(&leftRight, &euPosition)
	require.NotNil(t, leftRight)
	require.NotNil(t, euPosition)
	assert.InDelta(t, 7.83, *leftRight, 0.01)
	assert.InDelta(t, 4.52, *euPosition, 0.01)

	var nilLR *float64
	db.QueryRow("SELECT left_right FROM parlgov_parties WHERE id = 999").Scan(&nilLR)
	assert.Nil(t, nilLR)
}

func TestRunParlGov(t *testing.T) {
	var partyCalls, electionCalls, resultCalls, cabinetCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/parties/":
			partyCalls++
			w.Write([]byte(`{"count":1,"next":null,"results":[{"id":5,"name_short":"PD","name":"Partito Democratico","name_english":"Democratic Party","country_id":26,"date_founded":"2007-10-14","date_dissolved":null,"party_category":null,"party_orientation":null}]}`))
		case r.URL.Path == "/elections/":
			electionCalls++
			w.Write([]byte(`{"count":1,"next":null,"results":[{"id":800,"country_id":26,"election_date":"2022-09-25","early":false,"election_type_id":1,"election_type":"parliament","name":"Camera 2022","wikipedia":"","seats_total":400,"electorate":51000000,"votes_cast":35000000,"votes_valid":34000000,"data_source":"parlgov"}]}`))
		case r.URL.Path == "/election-results/":
			resultCalls++
			w.Write([]byte(`{"count":1,"next":null,"results":[{"id":5000,"election_id":800,"party_id":5,"votes":5000000,"vote_share":19.07,"seats":69,"seats_total":400}]}`))
		case r.URL.Path == "/cabinets/":
			cabinetCalls++
			w.Write([]byte(`{"count":1,"next":null,"results":[{"id":700,"country_id":26,"election_id":null,"start_date":"2022-10-22","end_date":null,"name":"Meloni","cabinet_name":"Governo Meloni","caretaker":false,"description":"2022-10-22 – present"}]}`))
		}
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGov(context.Background(), client, db, srv.URL, 26)
	require.NoError(t, err)

	assert.Equal(t, 1, partyCalls)
	assert.Equal(t, 1, electionCalls)
	assert.Equal(t, 1, resultCalls)
	assert.Equal(t, 1, cabinetCalls)

	var partyCount, electionCount, resultCount, cabinetCount int
	db.QueryRow("SELECT COUNT(*) FROM parlgov_parties").Scan(&partyCount)
	db.QueryRow("SELECT COUNT(*) FROM parlgov_elections").Scan(&electionCount)
	db.QueryRow("SELECT COUNT(*) FROM parlgov_election_results").Scan(&resultCount)
	db.QueryRow("SELECT COUNT(*) FROM parlgov_cabinets").Scan(&cabinetCount)
	assert.Equal(t, 1, partyCount)
	assert.Equal(t, 1, electionCount)
	assert.Equal(t, 1, resultCount)
	assert.Equal(t, 1, cabinetCount)
}

func TestParlGovFetcher(t *testing.T) {
	f := &ParlGovFetcher{
		sourceType: ParlGovSourceType,
		config:     DefaultParlGovConfig(),
	}
	assert.Equal(t, ParlGovSourceType, f.SourceType())
	assert.NoError(t, f.Validate())
	assert.Equal(t, 26, f.config.CountryID)
}

type parlgovMockWM struct {
	lastSource string
	lastMeta   string
}

func (m *parlgovMockWM) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	m.lastSource = sourceName
	m.lastMeta = metadata
	return nil
}

func TestParlGovWatermark(t *testing.T) {
	wm := &parlgovMockWM{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	err := RunParlGovWithWatermark(context.Background(), client, db, srv.URL, 26, wm)
	require.NoError(t, err)
	assert.Equal(t, ParlGovSourceType, wm.lastSource)
	assert.Contains(t, wm.lastMeta, `"country_id":26`)
}
