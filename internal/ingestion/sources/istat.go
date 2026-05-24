package sources

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	istatBaseURL   = "https://esploradati.istat.it/SDMXWS/rest"
	istatAcceptCSV = "text/csv"
)

// ISTAT rate limit: 5 req/min (= one request every 12s).
// Using a burst of 1 means the limiter enforces the exact interval.
func istatRateLimit() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 1.0 / 12.0,
		Burst:             1,
	}
}

// --- Table creation ---

func ensureISTATPopulationTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS istat_population (
		comune_istat VARCHAR,
		year INTEGER,
		popolazione_residente INTEGER,
		eta_media DOUBLE,
		indice_vecchiaia DOUBLE,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (comune_istat, year)
	)`)
	return err
}

func ensureISTATIncomeTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS istat_income (
		comune_istat VARCHAR,
		year INTEGER,
		reddito_medio DOUBLE,
		contribuenti INTEGER,
		importo_totale DOUBLE,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (comune_istat, year)
	)`)
	return err
}

func ensureISTATEmploymentTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS istat_employment (
		comune_istat VARCHAR,
		comune_nome VARCHAR,
		year INTEGER,
		tasso_occupazione DOUBLE,
		tasso_disoccupazione DOUBLE,
		tasso_inattivita DOUBLE,
		addetti_totali INTEGER,
		ul_totali INTEGER,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (comune_istat, year)
	)`)
	return err
}

// --- CSV parsing helpers ---

// parseCSVByHeader reads a CSV body and returns a slice of maps keyed by header names.
func parseCSVByHeader(body string) ([]map[string]string, error) {
	r := csv.NewReader(strings.NewReader(body))
	r.TrimLeadingSpace = true

	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}

	var rows []map[string]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv record: %w", err)
		}
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if i < len(record) {
				row[h] = strings.TrimSpace(record[i])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// --- Population ---

// RunISTATPopulation fetches ISTAT demographic data from the SDMX REST API and inserts
// it into the istat_population table. It fetches population for a range of years starting
// from startYear up to endYear-1 (inclusive) due to the known ISTAT endPeriod bug.
//
// The dataflow 22_289 uses structure DCIS_POPRES1: A=Annual, JAN=all ages, 9=both sexes, TOTAL=total, 99=all marital status.
func RunISTATPopulation(ctx context.Context, client *RateLimitedClient, db *sql.DB, cfg ISTATConfig) error {
	slog.Info("starting ISTAT population ingestion",
		"start_year", cfg.StartYear,
		"end_year", cfg.EndYear,
	)

	if err := ensureISTATPopulationTable(db); err != nil {
		return fmt.Errorf("create istat_population table: %w", err)
	}

	if client == nil {
		client = NewRateLimitedClient(istatRateLimit())
	}

	endPeriod := cfg.EndYear - 1

	url := fmt.Sprintf("%s/data/IT1,22_289/A..JAN.9.TOTAL.99/?startperiod=%d&endperiod=%d",
		cfg.baseURL(), cfg.StartYear, endPeriod)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create population request: %w", err)
	}
	req.Header.Set("Accept", istatAcceptCSV)

	slog.Info("fetching ISTAT population", "url", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch population: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read population body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("istat population API returned %d: %s",
			resp.StatusCode, string(body[:min(len(body), 500)]))
	}

	rows, err := parseCSVByHeader(string(body))
	if err != nil {
		return fmt.Errorf("parse population CSV: %w", err)
	}

	if len(rows) == 0 {
		slog.Info("no population data returned")
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO istat_population (comune_istat, year, popolazione_residente, eta_media, indice_vecchiaia)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare population insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, row := range rows {
		comuneISTAT := row["REF_AREA"]
		obsValue := row["OBS_VALUE"]
		timePeriod := row["TIME_PERIOD"]

		if comuneISTAT == "" || obsValue == "" || timePeriod == "" {
			continue
		}

		pop, err := strconv.Atoi(obsValue)
		if err != nil {
			slog.Warn("skipping non-integer population value", "ref_area", comuneISTAT, "obs_value", obsValue, "error", err)
			continue
		}

		year, err := strconv.Atoi(timePeriod)
		if err != nil {
			slog.Warn("skipping non-integer year", "ref_area", comuneISTAT, "time_period", timePeriod, "error", err)
			continue
		}

		etaMedia := parseOptionalFloat(row["ETA_MEDIA"])
		indiceVecchiaia := parseOptionalFloat(row["INDICE_VECCHIAIA"])

		if _, err := stmt.ExecContext(ctx, comuneISTAT, year, pop, etaMedia, indiceVecchiaia); err != nil {
			return fmt.Errorf("insert population row: %w", err)
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit population: %w", err)
	}

	slog.Info("ISTAT population ingestion complete", "rows", inserted)
	return nil
}

// --- Income ---

// RunISTATIncome fetches ISTAT IRPEF income data from the SDMX REST API
// and inserts it into the istat_income table.
//
// The dataflow 30_1008 uses structure MEF_REDDITIIRPEF_COM. It returns the last N observations
// across all Italian comuni.
func RunISTATIncome(ctx context.Context, client *RateLimitedClient, db *sql.DB, cfg ISTATConfig) error {
	slog.Info("starting ISTAT income ingestion",
		"num_observations", cfg.NumObservations,
	)

	if err := ensureISTATIncomeTable(db); err != nil {
		return fmt.Errorf("create istat_income table: %w", err)
	}

	if client == nil {
		client = NewRateLimitedClient(istatRateLimit())
	}

	numObs := cfg.NumObservations
	if numObs <= 0 {
		numObs = 10
	}

	url := fmt.Sprintf("%s/data/IT1,30_1008?lastNObservations=%d", cfg.baseURL(), numObs)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create income request: %w", err)
	}
	req.Header.Set("Accept", istatAcceptCSV)

	slog.Info("fetching ISTAT income", "url", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch income: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read income body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("istat income API returned %d: %s",
			resp.StatusCode, string(body[:min(len(body), 500)]))
	}

	rows, err := parseCSVByHeader(string(body))
	if err != nil {
		return fmt.Errorf("parse income CSV: %w", err)
	}

	if len(rows) == 0 {
		slog.Info("no income data returned")
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO istat_income (comune_istat, year, reddito_medio, contribuenti, importo_totale)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare income insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, row := range rows {
		comuneISTAT := row["REF_AREA"]
		timePeriod := row["TIME_PERIOD"]

		if comuneISTAT == "" || timePeriod == "" {
			continue
		}

		year, err := strconv.Atoi(timePeriod)
		if err != nil {
			slog.Warn("skipping non-integer year in income", "ref_area", comuneISTAT, "time_period", timePeriod, "error", err)
			continue
		}

		redditoMedio := parseOptionalFloat(row["OBS_VALUE"])
		contribuenti := parseOptionalInt(row["NUMERO_CONTRIBUENTI"])
		importoTotale := parseOptionalFloat(row["IMPORTO_TOTALE"])

		if _, err := stmt.ExecContext(ctx, comuneISTAT, year, redditoMedio, contribuenti, importoTotale); err != nil {
			return fmt.Errorf("insert income row: %w", err)
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit income: %w", err)
	}

	slog.Info("ISTAT income ingestion complete", "rows", inserted)
	return nil
}

// --- Employment ---

// employmentJSONLine is the parsed structure of one JSONL line from the ISTAT XLSX preprocessor.
// Fields match the "A misura di comune" Lavoro sheet column order.
type employmentJSONLine struct {
	ComuneISTAT       string  `json:"comune_istat"`
	ComuneNome        string  `json:"comune_nome"`
	Anno              int     `json:"anno"`
	TassoOccupazione  float64 `json:"tasso_occupazione"`
	TassoDisoccupazione float64 `json:"tasso_disoccupazione"`
	TassoInattivita   float64 `json:"tasso_inattivita"`
	AddettiTotali     int     `json:"addetti_totali"`
	ULTotale          int     `json:"ul_totali"`
}

// RunISTATEmployment ingests employment data from a JSONL file (generated by a Python XLSX
// preprocessor) into the istat_employment table. The JSONL path is set via cfg.EmploymentJSONLPath.
// If the path is empty, the function returns nil (graceful skip).
func RunISTATEmployment(ctx context.Context, client *RateLimitedClient, db *sql.DB, cfg ISTATConfig) error {
	if cfg.EmploymentJSONLPath == "" {
		slog.Info("no employment JSONL path configured, skipping ISTAT employment ingestion")
		return nil
	}

	slog.Info("starting ISTAT employment ingestion", "path", cfg.EmploymentJSONLPath)

	if err := ensureISTATEmploymentTable(db); err != nil {
		return fmt.Errorf("create istat_employment table: %w", err)
	}

	f, err := os.Open(cfg.EmploymentJSONLPath)
	if err != nil {
		return fmt.Errorf("open employment JSONL: %w", err)
	}
	defer f.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO istat_employment
			(comune_istat, comune_nome, year, tasso_occupazione, tasso_disoccupazione, tasso_inattivita, addetti_totali, ul_totali)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare employment insert: %w", err)
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(f)
	inserted := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec employmentJSONLine
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			slog.Warn("skipping invalid employment JSONL line", "line", line, "error", err)
			continue
		}

		if rec.ComuneISTAT == "" || rec.Anno == 0 {
			continue
		}

		if _, err := stmt.ExecContext(ctx,
			rec.ComuneISTAT, rec.ComuneNome, rec.Anno,
			rec.TassoOccupazione, rec.TassoDisoccupazione, rec.TassoInattivita,
			rec.AddettiTotali, rec.ULTotale,
		); err != nil {
			return fmt.Errorf("insert employment row (comune=%s, year=%d): %w", rec.ComuneISTAT, rec.Anno, err)
		}
		inserted++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read employment JSONL: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit employment: %w", err)
	}

	slog.Info("ISTAT employment ingestion complete", "rows", inserted)
	return nil
}

// --- Config ---

// ISTATConfig holds configuration for ISTAT data ingestion.
type ISTATConfig struct {
	// BaseURL overrides the default ISTAT SDMX REST API base URL (for testing).
	BaseURL string
	// StartYear is the first year to fetch population data for (default: 2011).
	StartYear int
	// EndYear is the last year of interest. Due to the endPeriod bug,
	// the API parameter will be EndYear-1.
	EndYear int
	// NumObservations is the number of income observations to fetch (default: 10).
	NumObservations int
	// EmploymentJSONLPath is the path to a JSONL file containing pre-parsed XLSX data
	// from ISTAT "A misura di comune" Lavoro sheet. If empty, ingestion is skipped.
	EmploymentJSONLPath string
}

func (c *ISTATConfig) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return istatBaseURL
}

// DefaultISTATConfig returns sensible defaults for ISTAT ingestion.
func DefaultISTATConfig() ISTATConfig {
	now := time.Now()
	return ISTATConfig{
		StartYear:       2011,
		EndYear:         now.Year(),
		NumObservations: 10,
	}
}

// --- Helpers ---

func parseOptionalFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseOptionalInt(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
