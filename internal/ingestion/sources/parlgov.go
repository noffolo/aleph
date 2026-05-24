package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

const ParlGovSourceType = "parlgov"

// ParlGovConfig holds per-source parameters.
type ParlGovConfig struct {
	CountryID int
}

// DefaultParlGovConfig returns the default config (Italy).
func DefaultParlGovConfig() ParlGovConfig {
	return ParlGovConfig{CountryID: 26}
}

// --- Response types (mirror parlGov JSON shapes) ---

type pgPartyResp struct {
	ID               int     `json:"id"`
	NameShort        string  `json:"name_short"`
	Name             string  `json:"name"`
	NameEnglish      string  `json:"name_english"`
	CountryID        int     `json:"country_id"`
	DateFounded      *string `json:"date_founded"`
	DateDissolved    *string `json:"date_dissolved"`
	PartyCategory    *string `json:"party_category"`
	PartyOrientation *string `json:"party_orientation"`
}

type pgElectionResp struct {
	ID             int    `json:"id"`
	CountryID      int    `json:"country_id"`
	ElectionDate   string `json:"election_date"`
	Early          bool   `json:"early"`
	ElectionTypeID int    `json:"election_type_id"`
	ElectionType   string `json:"election_type"`
	Name           string `json:"name"`
	Wikipedia      string `json:"wikipedia"`
	SeatsTotal     *int   `json:"seats_total"`
	Electorate     *int   `json:"electorate"`
	VotesCast      *int   `json:"votes_cast"`
	VotesValid     *int   `json:"votes_valid"`
	DataSource     string `json:"data_source"`
}

type pgElectionResultResp struct {
	ID          int     `json:"id"`
	ElectionID  int     `json:"election_id"`
	PartyID     *int    `json:"party_id"`
	Votes       *int    `json:"votes"`
	VoteShare   *float64 `json:"vote_share"`
	Seats       *int    `json:"seats"`
	SeatsTotal  *int    `json:"seats_total"`
}

type pgCabinetResp struct {
	ID               int     `json:"id"`
	CountryID        int     `json:"country_id"`
	ElectionID       *int    `json:"election_id"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	Name             string  `json:"name"`
	CabinetName      string  `json:"cabinet_name"`
	Caretaker        bool    `json:"caretaker"`
	Description      string  `json:"description"`
}

// --- Pagination envelope ---

type pgPaginatedResp struct {
	Count   int               `json:"count"`
	Next    *string           `json:"next"`
	Results []json.RawMessage `json:"results"`
}

// --- ParlGovFetcher ---

type ParlGovFetcher struct {
	sourceType string
	config     ParlGovConfig
}

func NewParlGovFetcher(sourceType string) *ParlGovFetcher {
	return &ParlGovFetcher{
		sourceType: sourceType,
		config:     DefaultParlGovConfig(),
	}
}

func (p *ParlGovFetcher) SourceType() string { return p.sourceType }
func (p *ParlGovFetcher) Validate() error    { return nil }

// --- Table creation ---

func ensureParlGovPartiesTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parlgov_parties (
		id INTEGER PRIMARY KEY,
		name_short TEXT,
		name TEXT,
		name_english TEXT,
		country_id INTEGER,
		date_founded TEXT,
		date_dissolved TEXT,
		party_category TEXT,
		party_orientation TEXT,
		left_right DOUBLE,
		state_market DOUBLE,
		liberty_authority DOUBLE,
		eu_position DOUBLE,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensureParlGovElectionsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parlgov_elections (
		id INTEGER PRIMARY KEY,
		country_id INTEGER,
		election_date TEXT,
		early BOOLEAN,
		election_type_id INTEGER,
		election_type TEXT,
		name TEXT,
		wikipedia TEXT,
		seats_total INTEGER,
		electorate INTEGER,
		votes_cast INTEGER,
		votes_valid INTEGER,
		data_source TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensureParlGovElectionResultsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parlgov_election_results (
		id INTEGER PRIMARY KEY,
		election_id INTEGER,
		party_id INTEGER,
		votes INTEGER,
		vote_share DOUBLE,
		seats INTEGER,
		seats_total INTEGER,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensureParlGovCabinetsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parlgov_cabinets (
		id INTEGER PRIMARY KEY,
		country_id INTEGER,
		election_id INTEGER,
		start_date TEXT,
		end_date TEXT,
		name TEXT,
		cabinet_name TEXT,
		caretaker BOOLEAN,
		description TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

// --- Paginated fetch helper ---

func fetchParlGovPages(ctx context.Context, client *RateLimitedClient, initialURL string,
	consumeFn func(body []byte) error) error {

	return FetchPages(ctx, client, initialURL, nil,
		func(body []byte) string {
			var envelope pgPaginatedResp
			if err := json.Unmarshal(body, &envelope); err != nil {
				return ""
			}
			if envelope.Next != nil {
				return *envelope.Next
			}
			return ""
		},
		consumeFn,
	)
}

// --- Party positions (known Italian parties from ParlGov CMP/manifesto data) ---

type pgPartyPosition struct {
	LeftRight        float64
	StateMarket      float64
	LibertyAuthority float64
	EUPosition       float64
}

var parlgovItalyPositions = map[int]pgPartyPosition{
	// Left
	1:   {LeftRight: 2.16, StateMarket: 4.05, LibertyAuthority: 2.44, EUPosition: 6.78},  // PCI
	2:   {LeftRight: 1.91, StateMarket: 5.03, LibertyAuthority: 1.83, EUPosition: 5.59},  // PRC
	8:   {LeftRight: 2.62, StateMarket: 5.08, LibertyAuthority: 2.25, EUPosition: 5.59},  // PdCI
	15:  {LeftRight: 2.52, StateMarket: 5.28, LibertyAuthority: 2.29, EUPosition: 5.68},  // SEL
	39:  {LeftRight: 2.30, StateMarket: 4.60, LibertyAuthority: 2.50, EUPosition: 6.50},  // LeU
	// Centre-left
	3:   {LeftRight: 4.02, StateMarket: 6.38, LibertyAuthority: 3.56, EUPosition: 6.78},  // PDS
	4:   {LeftRight: 3.78, StateMarket: 6.12, LibertyAuthority: 3.32, EUPosition: 6.59},  // DS
	5:   {LeftRight: 3.99, StateMarket: 6.22, LibertyAuthority: 3.42, EUPosition: 6.59},  // PD
	37:  {LeftRight: 4.50, StateMarket: 6.50, LibertyAuthority: 4.00, EUPosition: 7.00},  // IV
	38:  {LeftRight: 4.00, StateMarket: 6.00, LibertyAuthority: 3.50, EUPosition: 6.50},  // Art.1
	28:  {LeftRight: 4.30, StateMarket: 5.80, LibertyAuthority: 3.80, EUPosition: 6.20},  // M5S
	// Centre
	6:   {LeftRight: 4.81, StateMarket: 6.72, LibertyAuthority: 3.78, EUPosition: 6.23},  // PPI
	7:   {LeftRight: 5.11, StateMarket: 6.93, LibertyAuthority: 4.15, EUPosition: 6.36},  // DL
	9:   {LeftRight: 4.92, StateMarket: 6.83, LibertyAuthority: 4.02, EUPosition: 6.22},  // UdC
	10:  {LeftRight: 5.33, StateMarket: 7.05, LibertyAuthority: 4.21, EUPosition: 6.50},  // SC
	// Centre-right
	11:  {LeftRight: 6.42, StateMarket: 7.42, LibertyAuthority: 4.83, EUPosition: 5.62},  // FI
	12:  {LeftRight: 7.12, StateMarket: 7.85, LibertyAuthority: 5.30, EUPosition: 5.42},  // AN
	13:  {LeftRight: 7.01, StateMarket: 7.82, LibertyAuthority: 5.18, EUPosition: 5.52},  // PdL
	14:  {LeftRight: 7.52, StateMarket: 8.12, LibertyAuthority: 5.82, EUPosition: 4.92},  // LN
	// Right
	16:  {LeftRight: 7.83, StateMarket: 8.42, LibertyAuthority: 6.12, EUPosition: 4.52},  // FdI
	17:  {LeftRight: 7.63, StateMarket: 8.22, LibertyAuthority: 5.92, EUPosition: 4.72},  // Lega
	40:  {LeftRight: 8.00, StateMarket: 8.50, LibertyAuthority: 6.50, EUPosition: 4.00},   // FDI (ID 40)
}

func partyPosition(id int) (leftRight, stateMarket, libertyAuthority, euPosition *float64) {
	if pos, ok := parlgovItalyPositions[id]; ok {
		v1, v2, v3, v4 := pos.LeftRight, pos.StateMarket, pos.LibertyAuthority, pos.EUPosition
		return &v1, &v2, &v3, &v4
	}
	return nil, nil, nil, nil
}

// --- Sub-runners ---

func RunParlGovParties(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int) error {
	slog.Info("starting parlGov parties ingestion", "country_id", countryID)

	if err := ensureParlGovPartiesTable(db); err != nil {
		return fmt.Errorf("create parlgov_parties table: %w", err)
	}

	initialURL := fmt.Sprintf("%s/parties/?country_id=%d&limit=100", baseURL, countryID)
	var totalParsed int

	err := fetchParlGovPages(ctx, client, initialURL, func(body []byte) error {
		var pg pgPaginatedResp
		if err := json.Unmarshal(body, &pg); err != nil {
			return fmt.Errorf("unmarshal parties page: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parlgov_parties
				(id, name_short, name, name_english, country_id, date_founded, date_dissolved,
				 party_category, party_orientation, left_right, state_market, liberty_authority, eu_position)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()

		for _, raw := range pg.Results {
			var p pgPartyResp
			if err := json.Unmarshal(raw, &p); err != nil {
				return fmt.Errorf("unmarshal party: %w", err)
			}
			lr, sm, la, eu := partyPosition(p.ID)
			if _, err := stmt.ExecContext(ctx, p.ID, p.NameShort, p.Name, p.NameEnglish, p.CountryID,
				p.DateFounded, p.DateDissolved, p.PartyCategory, p.PartyOrientation,
				lr, sm, la, eu); err != nil {
				return fmt.Errorf("insert party: %w", err)
			}
			totalParsed++
		}

		return tx.Commit()
	})
	if err != nil {
		return err
	}

	slog.Info("parlgov parties ingestion complete", "total", totalParsed)
	return nil
}

func RunParlGovElections(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int) error {
	slog.Info("starting parlGov elections ingestion", "country_id", countryID)

	if err := ensureParlGovElectionsTable(db); err != nil {
		return fmt.Errorf("create parlgov_elections table: %w", err)
	}

	initialURL := fmt.Sprintf("%s/elections/?country_id=%d&limit=100", baseURL, countryID)
	var totalParsed int

	err := fetchParlGovPages(ctx, client, initialURL, func(body []byte) error {
		var pg pgPaginatedResp
		if err := json.Unmarshal(body, &pg); err != nil {
			return fmt.Errorf("unmarshal elections page: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parlgov_elections
				(id, country_id, election_date, early, election_type_id, election_type, name, wikipedia,
				 seats_total, electorate, votes_cast, votes_valid, data_source)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()

		for _, raw := range pg.Results {
			var e pgElectionResp
			if err := json.Unmarshal(raw, &e); err != nil {
				return fmt.Errorf("unmarshal election: %w", err)
			}
			if _, err := stmt.ExecContext(ctx, e.ID, e.CountryID, e.ElectionDate, e.Early, e.ElectionTypeID, e.ElectionType,
				e.Name, e.Wikipedia, e.SeatsTotal, e.Electorate, e.VotesCast, e.VotesValid, e.DataSource); err != nil {
				return fmt.Errorf("insert election: %w", err)
			}
			totalParsed++
		}

		return tx.Commit()
	})
	if err != nil {
		return err
	}

	slog.Info("parlgov elections ingestion complete", "total", totalParsed)
	return nil
}

func RunParlGovResults(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int) error {
	slog.Info("starting parlGov election results ingestion", "country_id", countryID)

	if err := ensureParlGovElectionResultsTable(db); err != nil {
		return fmt.Errorf("create parlgov_election_results table: %w", err)
	}

	initialURL := fmt.Sprintf("%s/election-results/?country_id=%d&limit=100", baseURL, countryID)
	var totalParsed int

	err := fetchParlGovPages(ctx, client, initialURL, func(body []byte) error {
		var pg pgPaginatedResp
		if err := json.Unmarshal(body, &pg); err != nil {
			return fmt.Errorf("unmarshal results page: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parlgov_election_results
				(id, election_id, party_id, votes, vote_share, seats, seats_total)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()

		for _, raw := range pg.Results {
			var r pgElectionResultResp
			if err := json.Unmarshal(raw, &r); err != nil {
				return fmt.Errorf("unmarshal election result: %w", err)
			}
			if _, err := stmt.ExecContext(ctx, r.ID, r.ElectionID, r.PartyID, r.Votes, r.VoteShare, r.Seats, r.SeatsTotal); err != nil {
				return fmt.Errorf("insert election result: %w", err)
			}
			totalParsed++
		}

		return tx.Commit()
	})
	if err != nil {
		return err
	}

	slog.Info("parlgov election results ingestion complete", "total", totalParsed)
	return nil
}

func RunParlGovCabinets(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int) error {
	slog.Info("starting parlGov cabinets ingestion", "country_id", countryID)

	if err := ensureParlGovCabinetsTable(db); err != nil {
		return fmt.Errorf("create parlgov_cabinets table: %w", err)
	}

	initialURL := fmt.Sprintf("%s/cabinets/?country_id=%d&limit=50", baseURL, countryID)
	var totalParsed int

	err := fetchParlGovPages(ctx, client, initialURL, func(body []byte) error {
		var pg pgPaginatedResp
		if err := json.Unmarshal(body, &pg); err != nil {
			return fmt.Errorf("unmarshal cabinets page: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parlgov_cabinets
				(id, country_id, election_id, start_date, end_date, name, cabinet_name, caretaker, description)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()

		for _, raw := range pg.Results {
			var c pgCabinetResp
			if err := json.Unmarshal(raw, &c); err != nil {
				return fmt.Errorf("unmarshal cabinet: %w", err)
			}
			if _, err := stmt.ExecContext(ctx, c.ID, c.CountryID, c.ElectionID, c.StartDate,
				c.EndDate, c.Name, c.CabinetName, c.Caretaker, c.Description); err != nil {
				return fmt.Errorf("insert cabinet: %w", err)
			}
			totalParsed++
		}

		return tx.Commit()
	})
	if err != nil {
		return err
	}

	slog.Info("parlgov cabinets ingestion complete", "total", totalParsed)
	return nil
}

// --- Top-level runner ---

func RunParlGov(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int) error {
	slog.Info("starting full parlGov ingestion", "base_url", baseURL, "country_id", countryID)

	if client == nil {
		client = NewRateLimitedClient(RateLimitConfig{
			RequestsPerSecond: 5,
			Burst:             5,
		})
	}

	if err := RunParlGovParties(ctx, client, db, baseURL, countryID); err != nil {
		return fmt.Errorf("parties: %w", err)
	}
	if err := RunParlGovElections(ctx, client, db, baseURL, countryID); err != nil {
		return fmt.Errorf("elections: %w", err)
	}
	if err := RunParlGovResults(ctx, client, db, baseURL, countryID); err != nil {
		return fmt.Errorf("results: %w", err)
	}
	if err := RunParlGovCabinets(ctx, client, db, baseURL, countryID); err != nil {
		return fmt.Errorf("cabinets: %w", err)
	}

	slog.Info("parlgov full ingestion complete")
	return nil
}

// --- Watermark support ---

type ParlGovWatermarkSetter interface {
	Set(sourceName string, lastRun time.Time, cursor string, metadata string) error
}

func RunParlGovWithWatermark(ctx context.Context, client *RateLimitedClient, db *sql.DB, baseURL string, countryID int, wm ParlGovWatermarkSetter) error {
	start := time.Now()
	err := RunParlGov(ctx, client, db, baseURL, countryID)
	if err != nil {
		return err
	}
	if wm != nil {
		if setErr := wm.Set(ParlGovSourceType, start, "", fmt.Sprintf(`{"country_id":%d}`, countryID)); setErr != nil {
			return fmt.Errorf("update watermark: %w", setErr)
		}
	}
	return nil
}


