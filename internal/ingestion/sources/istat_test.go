package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestISTATTableCreation(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	require.NoError(t, ensureISTATPopulationTable(db))
	require.NoError(t, ensureISTATIncomeTable(db))

	var tables []string
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'istat_%' ORDER BY table_name")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	assert.Equal(t, []string{"istat_income", "istat_population"}, tables)
}

func TestISTATPopulationParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "startperiod=2011")
		assert.Contains(t, r.URL.RawQuery, "endperiod=2021")
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(`DATAFLOW,REF_AREA,FREQ,AGE,SEX,ETATOCIVILE,OBS_VALUE,TIME_PERIOD
IT1:DCIS_POPRES1(1.2),001001,A,JAN,9,99,1523,2022
IT1:DCIS_POPRES1(1.2),001002,A,JAN,9,99,845,2022
IT1:DCIS_POPRES1(1.2),058091,A,JAN,9,99,2761632,2022
`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{BaseURL: srv.URL, StartYear: 2011, EndYear: 2022, NumObservations: 10}

	err := RunISTATPopulation(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_population").Scan(&count)
	assert.Equal(t, 3, count)

	var pop int
	db.QueryRow("SELECT popolazione_residente FROM istat_population WHERE comune_istat = '058091' AND year = 2022").Scan(&pop)
	assert.Equal(t, 2761632, pop)

	db.QueryRow("SELECT popolazione_residente FROM istat_population WHERE comune_istat = '001001' AND year = 2022").Scan(&pop)
	assert.Equal(t, 1523, pop)
}

func TestISTATIncomeParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "lastNObservations=10")
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(`DATAFLOW,REF_AREA,TIME_PERIOD,OBS_VALUE,NUMERO_CONTRIBUENTI,IMPORTO_TOTALE
IT1:MEF_REDDITIIRPEF_COM(1.0),001001,2022,24567.50,1200,29481000.00
IT1:MEF_REDDITIIRPEF_COM(1.0),001002,2022,18750.25,890,16687722.50
IT1:MEF_REDDITIIRPEF_COM(1.0),058091,2022,31200.75,12500,390009375.00
`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{BaseURL: srv.URL, StartYear: 2011, EndYear: 2022, NumObservations: 10}

	err := RunISTATIncome(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_income").Scan(&count)
	assert.Equal(t, 3, count)

	var redditoMedio float64
	var contribuenti int
	db.QueryRow("SELECT reddito_medio, contribuenti FROM istat_income WHERE comune_istat = '058091' AND year = 2022").Scan(&redditoMedio, &contribuenti)
	assert.InDelta(t, 31200.75, redditoMedio, 0.01)
	assert.Equal(t, 12500, contribuenti)

	db.QueryRow("SELECT reddito_medio, contribuenti FROM istat_income WHERE comune_istat = '001001' AND year = 2022").Scan(&redditoMedio, &contribuenti)
	assert.InDelta(t, 24567.50, redditoMedio, 0.01)
	assert.Equal(t, 1200, contribuenti)
}

func TestISTATEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(`DATAFLOW,REF_AREA,OBS_VALUE,TIME_PERIOD
`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{BaseURL: srv.URL, StartYear: 2011, EndYear: 2022, NumObservations: 10}

	err := RunISTATPopulation(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_population").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestISTATHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{BaseURL: srv.URL, StartYear: 2011, EndYear: 2022, NumObservations: 10}

	err := RunISTATPopulation(context.Background(), client, db, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestISTATEndPeriodBug(t *testing.T) {
	var actualEndPeriod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualEndPeriod = r.URL.Query().Get("endperiod")
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(`DATAFLOW,REF_AREA,FREQ,AGE,SEX,ETATOCIVILE,OBS_VALUE,TIME_PERIOD
IT1:DCIS_POPRES1(1.2),001001,A,JAN,9,99,1523,2022
`))
	}))
	defer srv.Close()

	db := setupTestDuckDB(t)
	defer db.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{BaseURL: srv.URL, StartYear: 2011, EndYear: 2023, NumObservations: 10}

	err := RunISTATPopulation(context.Background(), client, db, cfg)
	require.NoError(t, err)

	assert.Equal(t, "2022", actualEndPeriod)
}

func TestISTATDefaultConfig(t *testing.T) {
	cfg := DefaultISTATConfig()
	assert.Equal(t, 2011, cfg.StartYear)
	assert.Equal(t, 10, cfg.NumObservations)
	assert.GreaterOrEqual(t, cfg.EndYear, 2024)
}
