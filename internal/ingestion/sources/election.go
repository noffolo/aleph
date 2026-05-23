package sources

import (
	"compress/gzip"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed electiondata/eligendo_codes.json
var eligendoCodesRaw []byte

var istatCache *ISTATLookup
var istatOnce sync.Once

//go:embed electiondata/party_aliases.json
var partyAliasesRaw []byte

// ElectionConfig holds the parameters used to identify a specific election to process.
type ElectionConfig struct {
	ElectionType string `json:"election_type"`
	ElectionDate string `json:"election_date"` // YYYYMMDD format, used for path-based API URLs
	Level        string `json:"level"`
	Year         int    `json:"year"`
}

// Validate checks that the election configuration contains valid known values
// and that the year is 2000 or later.
func (c ElectionConfig) Validate() error {
	validTypes := map[string]bool{"politiche": true, "europee": true, "regionali": true, "comunali": true, "provinciali": true, "referendum": true}
	validLevels := map[string]bool{"comune": true, "provincia": true, "regione": true}
	if !validTypes[c.ElectionType] {
		return fmt.Errorf("invalid election_type: %s", c.ElectionType)
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid level: %s", c.Level)
	}
	if c.Year < 2000 {
		return errors.New("year before 2000 not supported")
	}
	return nil
}

// ElectionResult represents a single election result row for a party in a comune.
type ElectionResult struct {
	ElectionType   string  `json:"election_type"`
	Level          string  `json:"level"`
	Year           int     `json:"year"`
	Comune         string  `json:"comune"`
	ComuneISTAT    string  `json:"comune_istat"`
	Lista          string  `json:"lista"`
	PartyCanonical string  `json:"party_canonical"`
	Voti           int64   `json:"voti"`
	Percentuale    float64 `json:"percentuale"`
	Seggi          int     `json:"seggi"`
	Elettori       int64   `json:"elettori"`
	Votanti        int64   `json:"votanti"`
}

// ISTATLookup provides bidirectional lookup between ISTAT codes and comune names.
type ISTATLookup struct {
	byCode map[string]string
	byName map[string]string
}

// NewISTATLookup returns a singleton ISTATLookup, loading data from the embedded
// eligendo codes file on first call.
func NewISTATLookup() *ISTATLookup {
	istatOnce.Do(func() {
		istatCache = &ISTATLookup{byCode: make(map[string]string), byName: make(map[string]string)}
		var raw map[string]string
		if err := json.Unmarshal(eligendoCodesRaw, &raw); err != nil {
			return
		}
		for code, name := range raw {
			istatCache.byCode[code] = name
			istatCache.byName[strings.ToLower(name)] = code
		}
	})
	return istatCache
}

// Lookup returns the comune name for the given ISTAT code.
func (l *ISTATLookup) Lookup(code string) (string, bool) {
	name, ok := l.byCode[code]
	return name, ok
}

// LookupByName returns the ISTAT code for the given comune name.
func (l *ISTATLookup) LookupByName(name string) (string, bool) {
	code, ok := l.byName[strings.ToLower(name)]
	return code, ok
}

// PartyMapper maps raw party names to canonical names using a built-in alias
// table and user-configurable overrides. It is safe for concurrent use.
type PartyMapper struct {
	aliases   map[string]string
	overrides map[string]string
	mu        sync.RWMutex
}

// NewPartyMapper creates a PartyMapper pre-loaded with built-in party aliases
// from the embedded party_aliases.json file.
func NewPartyMapper() *PartyMapper {
	pm := &PartyMapper{
		aliases:   make(map[string]string),
		overrides: make(map[string]string),
	}
	pm.loadBuiltinAliases()
	return pm
}

// loadBuiltinAliases populates the alias table from the embedded party_aliases.json.
// It is only called from NewPartyMapper, before the pointer escapes, so locking is not needed.
func (pm *PartyMapper) loadBuiltinAliases() {
	var raw map[string]string
	if err := json.Unmarshal(partyAliasesRaw, &raw); err != nil {
		return
	}
	for rawName, canonical := range raw {
		pm.aliases[normalizePartyName(rawName)] = canonical
	}
}

// AddAlias registers a mapping from rawName to its canonical party name.
func (pm *PartyMapper) AddAlias(rawName string, canonical string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.aliases[normalizePartyName(rawName)] = canonical
}

// SetOverride sets a manual override mapping that takes precedence over built-in aliases.
func (pm *PartyMapper) SetOverride(rawName string, canonical string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.overrides[rawName] = canonical
}

// Lookup resolves a raw party name to its canonical name, checking overrides
// before the built-in alias table.
func (pm *PartyMapper) Lookup(rawName string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if canonical, ok := pm.overrides[rawName]; ok {
		return canonical, true
	}
	canonical, ok := pm.aliases[normalizePartyName(rawName)]
	return canonical, ok
}

// GetOverride returns the manual override for rawName, if one exists.
func (pm *PartyMapper) GetOverride(rawName string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	canonical, ok := pm.overrides[rawName]
	return canonical, ok
}

// AllOverrides returns a copy of all manual override mappings.
func (pm *PartyMapper) AllOverrides() map[string]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make(map[string]string, len(pm.overrides))
	for k, v := range pm.overrides {
		result[k] = v
	}
	return result
}

func normalizePartyName(raw string) string {
	return strings.TrimSpace(strings.ToUpper(raw))
}

// ElectionFetcher wraps HTTP requests to the Eligendo API with rate limiting.
type ElectionFetcher struct {
	baseURL     string
	rateLimiter *tokenBucketLimiter
	httpClient  *http.Client
}

type tokenBucketLimiter struct {
	rate       float64
	burst      int
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucketLimiter(ratePerSecond float64, burst int) *tokenBucketLimiter {
	return &tokenBucketLimiter{
		rate:       ratePerSecond,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

func (rl *tokenBucketLimiter) Wait() error {
	rl.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		rl.mu.Unlock()
		return nil
	}
	needed := 1 - rl.tokens
	waitDuration := time.Duration(needed / rl.rate * float64(time.Second))
	rl.mu.Unlock()

	time.Sleep(waitDuration)

	rl.mu.Lock()
	elapsed = time.Since(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.lastRefill = time.Now()

	if rl.tokens >= 1 {
		rl.tokens--
	} else {
		rl.tokens = 0
	}
	rl.mu.Unlock()
	return nil
}

// NewElectionFetcher creates an ElectionFetcher with the given base URL and rate limit.
func NewElectionFetcher(baseURL string, ratePerSecond float64) *ElectionFetcher {
	return &ElectionFetcher{
		baseURL:     baseURL,
		rateLimiter: newTokenBucketLimiter(ratePerSecond, 1),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EligendoEntity represents a single entity (e.g., comune) returned by the Eligendo API.
type EligendoEntity struct {
	Cod  string `json:"cod"`
	Desc string `json:"desc"`
}

// getentiFIResponse is the response envelope for the getentiFI Eligendo endpoint.
type getentiFIResponse struct {
	Intestazione struct {
		TE string `json:"te"`
	} `json:"intestazione"`
	Enti struct {
		Ente []EligendoEntity `json:"ente"`
	} `json:"enti"`
}

// teCode returns the TE code for the election type in this config.
func (c ElectionConfig) teCode() string {
	switch c.ElectionType {
	case "politiche", "camera":
		return "TE01"
	case "europee", "senato":
		return "TE02"
	case "regionali":
		return "TE03"
	case "provinciali":
		return "TE04"
	case "comunali":
		return "TE05"
	case "referendum":
		return "TE09"
	default:
		return "TE01"
	}
}

// GetEntities fetches entities (comuni, province, etc.) from the Eligendo API.
func (f *ElectionFetcher) GetEntities(ctx context.Context, cfg ElectionConfig) ([]EligendoEntity, error) {
	if err := f.rateLimiter.Wait(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/getentiFI/DE/%s/TE/%s", f.baseURL, cfg.ElectionDate, cfg.teCode())
	resp, err := f.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed getentiFIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode eligendo response: %w", err)
	}
	return parsed.Enti.Ente, nil
}

// doGet performs an HTTP GET request with standard headers.
func (f *ElectionFetcher) doGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://elezioni.interno.gov.it")
	req.Header.Set("Referer", "https://elezioni.interno.gov.it/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Aleph/1.0; +https://github.com/ff3300/aleph-v2)")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("eligendo api returned %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

// --- ElectionSource ---
//
// Registration with GlobalRegistry is handled externally (in engine adapter) to
// avoid an import cycle between the sources and ingestion packages.

const electionSourceType = "election"

type ElectionSource struct {
	sourceType string
	config     ElectionConfig
	db         *sql.DB
	dataDir    string
	mapper     *PartyMapper
}

// SourceType returns the source type identifier.
func (s *ElectionSource) SourceType() string {
	return s.sourceType
}

// Validate checks that the source's election configuration is valid.
func (s *ElectionSource) Validate() error {
	return s.config.Validate()
}

// scrutiniFIResponse is the Eligendo API response envelope for the scrutiniFI endpoint.
type scrutiniFIResponse struct {
	Intestazione struct {
		Cod string `json:"cod"`
	} `json:"intestazione"`
	Liste struct {
		Lista []struct {
			Desc  string  `json:"desc"`
			Voti  int64   `json:"voti"`
			Perc  float64 `json:"perc"`
			Seggi int     `json:"seggi"`
		} `json:"lista"`
	} `json:"liste"`
	DatiGenerali struct {
		Elettori int64 `json:"elettori"`
		Votanti  int64 `json:"votanti"`
	} `json:"datiGenerali"`
}

// RunElection executes the full pipeline: getenti → scrutini per ente → raw gzip save → normalize party → write DuckDB.
func RunElection(ctx context.Context, db *sql.DB, baseURL string, cfg ElectionConfig, mapper *PartyMapper, rawDir string) ([]ElectionResult, error) {
	fetcher := NewElectionFetcher(baseURL, 1.0)

	entities, err := fetcher.GetEntities(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("getenti: %w", err)
	}

	rawPath := filepath.Join(rawDir, electionSourceType, fmt.Sprintf("%d-%s-%s", cfg.Year, cfg.ElectionType, cfg.Level))
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return nil, fmt.Errorf("create raw dir: %w", err)
	}

	entiPayload := map[string]interface{}{
		"enti": entities,
	}
	if err := saveRawJSON(filepath.Join(rawPath, "getenti.json"), entiPayload); err != nil {
		slog.Warn("failed to save getenti raw", "error", err)
	}

	var results []ElectionResult
	for _, ent := range entities {
		if err := fetcher.rateLimiter.Wait(); err != nil {
			return nil, fmt.Errorf("rate limiter wait: %w", err)
		}

		reg := ent.Cod[:2]
		prv := ent.Cod[2:5]
		com := ent.Cod[5:]
		url := fmt.Sprintf("%s/scrutiniFI/DE/%s/TE/%s/RE/%s/PR/%s/CM/%s", baseURL, cfg.ElectionDate, cfg.teCode(), reg, prv, com)
		resp, err := fetcher.doGet(ctx, url)
		if err != nil {
			slog.Error("scrutini fetch failed", "entity", ent.Cod, "error", err)
			continue
		}

		var parsed scrutiniFIResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			resp.Body.Close()
			slog.Error("scrutini decode failed", "entity", ent.Cod, "error", err)
			continue
		}
		resp.Body.Close()

		if err := saveRawJSON(filepath.Join(rawPath, fmt.Sprintf("scrutini_%s.json", ent.Cod)), parsed); err != nil {
			slog.Warn("failed to save scrutini raw", "entity", ent.Cod, "error", err)
		}

		for _, lista := range parsed.Liste.Lista {
			canonical, found := mapper.Lookup(lista.Desc)
			if !found {
				canonical = ""
				slog.Info("unmapped party", "raw", lista.Desc, "entity", ent.Cod, "entity_name", ent.Desc)
			}

			results = append(results, ElectionResult{
				ElectionType:   cfg.ElectionType,
				Level:          cfg.Level,
				Year:           cfg.Year,
				Comune:         ent.Desc,
				ComuneISTAT:    ent.Cod,
				Lista:          lista.Desc,
				PartyCanonical: canonical,
				Voti:           lista.Voti,
				Percentuale:    lista.Perc,
				Seggi:          lista.Seggi,
				Elettori:       parsed.DatiGenerali.Elettori,
				Votanti:        parsed.DatiGenerali.Votanti,
			})
		}
	}

	if err := writeElectionResults(db, results); err != nil {
		return results, fmt.Errorf("write results: %w", err)
	}

	slog.Info("election pipeline complete",
		"election_type", cfg.ElectionType,
		"entities", len(entities),
		"results", len(results),
		"raw_dir", rawPath,
	)

	return results, nil
}

// writeElectionResults inserts or replaces election results into DuckDB.
func writeElectionResults(db *sql.DB, results []ElectionResult) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS election_results (
			election_type   TEXT,
			level           TEXT,
			year            INTEGER,
			comune          TEXT,
			comune_istat    TEXT,
			lista           TEXT,
			party_canonical TEXT,
			voti            INTEGER,
			percentuale     REAL,
			seggi           INTEGER,
			elettori        INTEGER,
			votanti         INTEGER,
			ingested_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(election_type, level, year, comune_istat, lista)
		)
	`)
	if err != nil {
		return fmt.Errorf("create election_results table: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO election_results
			(election_type, level, year, comune, comune_istat, lista, party_canonical, voti, percentuale, seggi, elettori, votanti)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, r := range results {
		_, err := stmt.Exec(
			r.ElectionType, r.Level, r.Year, r.Comune, r.ComuneISTAT,
			r.Lista, r.PartyCanonical, r.Voti, r.Percentuale, r.Seggi,
			r.Elettori, r.Votanti,
		)
		if err != nil {
			return fmt.Errorf("insert row: %w", err)
		}
	}

	return tx.Commit()
}

// saveRawJSON writes data as gzip-compressed JSON to the given path.
func saveRawJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	return json.NewEncoder(gz).Encode(data)
}
