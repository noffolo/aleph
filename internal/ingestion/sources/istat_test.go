package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	require.NoError(t, ensureISTATEmploymentTable(db))

	var tables []string
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'istat_%' ORDER BY table_name")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	assert.Equal(t, []string{"istat_employment", "istat_income", "istat_population"}, tables)
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

func TestISTATEmploymentTableCreation(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	require.NoError(t, ensureISTATEmploymentTable(db))

	var tableExists int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'istat_employment'",
	).Scan(&tableExists))
	assert.Equal(t, 1, tableExists)
}

func TestISTATEmploymentParsing(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	// Create a temp JSONL file with sample employment data
	f, err := os.CreateTemp("", "istat-employment-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	samples := []employmentJSONLine{
		{ComuneISTAT: "058091", ComuneNome: "Roma", Anno: 2022, TassoOccupazione: 65.5, TassoDisoccupazione: 8.2, TassoInattivita: 26.3, AddettiTotali: 850000, ULTotale: 42000},
		{ComuneISTAT: "001001", ComuneNome: "Agliè", Anno: 2022, TassoOccupazione: 62.1, TassoDisoccupazione: 6.5, TassoInattivita: 31.4, AddettiTotali: 340, ULTotale: 25},
		{ComuneISTAT: "015146", ComuneNome: "Milano", Anno: 2022, TassoOccupazione: 70.3, TassoDisoccupazione: 5.1, TassoInattivita: 24.6, AddettiTotali: 920000, ULTotale: 55000},
	}
	enc := json.NewEncoder(f)
	for _, s := range samples {
		require.NoError(t, enc.Encode(s))
	}
	f.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{
		BaseURL:              "https://esploradati.istat.it/SDMXWS/rest",
		StartYear:            2011,
		EndYear:              2022,
		NumObservations:      10,
		EmploymentJSONLPath:  f.Name(),
	}

	err = RunISTATEmployment(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_employment").Scan(&count)
	assert.Equal(t, 3, count)

	var tassoOcc float64
	var addetti int
	db.QueryRow("SELECT tasso_occupazione, addetti_totali FROM istat_employment WHERE comune_istat = '058091' AND year = 2022").Scan(&tassoOcc, &addetti)
	assert.InDelta(t, 65.5, tassoOcc, 0.01)
	assert.Equal(t, 850000, addetti)

	db.QueryRow("SELECT tasso_occupazione, addetti_totali FROM istat_employment WHERE comune_istat = '015146' AND year = 2022").Scan(&tassoOcc, &addetti)
	assert.InDelta(t, 70.3, tassoOcc, 0.01)
	assert.Equal(t, 920000, addetti)
}

func TestISTATEmploymentEmptyFile(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	// Empty JSONL path → graceful skip
	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{
		BaseURL:             "https://esploradati.istat.it/SDMXWS/rest",
		StartYear:           2011,
		EndYear:             2022,
		NumObservations:     10,
		EmploymentJSONLPath: "",
	}

	err := RunISTATEmployment(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_employment").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestISTATEmploymentInvalidJSONL(t *testing.T) {
	db := setupTestDuckDB(t)
	defer db.Close()

	f, err := os.CreateTemp("", "istat-employment-bad-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	// Write a mix of valid and invalid lines
	f.WriteString("not valid json\n")
	f.WriteString(`{"comune_istat":"058091","comune_nome":"Roma","anno":2022,"tasso_occupazione":65.5,"tasso_disoccupazione":8.2,"tasso_inattivita":26.3,"addetti_totali":850000,"ul_totali":42000}` + "\n")
	f.WriteString("\n") // empty line, skipped
	f.WriteString("[this,is,not,an,object]\n")
	f.Close()

	client := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	client.client = &http.Client{}

	cfg := ISTATConfig{
		BaseURL:             "https://esploradati.istat.it/SDMXWS/rest",
		StartYear:           2011,
		EndYear:             2022,
		NumObservations:     10,
		EmploymentJSONLPath: f.Name(),
	}

	err = RunISTATEmployment(context.Background(), client, db, cfg)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM istat_employment").Scan(&count)
	assert.Equal(t, 1, count) // only one valid row inserted
}
