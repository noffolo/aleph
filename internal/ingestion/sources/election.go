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
	"strconv"
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

// ErrElectionUnavailable is returned when the Eligendo API returns HTTP 403,
// indicating the election data is not available (typically historical elections
// that predate the current API dataset).
var ErrElectionUnavailable = errors.New("election data not available from Eligendo API")

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

// teCode returns the TE code (numeric only) for the election type in this config.
// Used in path-based API URLs: /getentiFI/DE/{date}/TE/{code}
func (c ElectionConfig) teCode() string {
	switch c.ElectionType {
	case "politiche", "camera":
		return "01"
	case "europee":
		return "01"
	case "senato":
		return "02"
	case "regionali":
		return "03"
	case "provinciali":
		return "04"
	case "comunali":
		return "05"
	case "referendum":
		return "09"
	default:
		return "01"
	}
}

// endpointSuffix returns the API endpoint suffix for this election type.
// Different election types use different endpoint identifiers:
//   FI = nazionale/referendum, CI = camera, SI = senato,
//   EI = europee, R = regionali
func (c ElectionConfig) endpointSuffix() string {
	switch c.ElectionType {
	case "politiche":
		return "FI"
	case "camera":
		return "CI"
	case "senato":
		return "SI"
	case "europee":
		return "EI"
	case "regionali":
		return "R"
	case "comunali", "provinciali":
		return "FI"
	case "referendum":
		return "FI"
	default:
		return "FI"
	}
}

// GetEntities fetches entities (comuni, province, etc.) from the Eligendo API.
// Handles both flat array form ({"enti":[...]}) used by EI/CI/SI/R endpoints
// and nested form ({"enti":{"ente":[...]}}) used by FI endpoint.
func (f *ElectionFetcher) GetEntities(ctx context.Context, cfg ElectionConfig) ([]EligendoEntity, error) {
	if err := f.rateLimiter.Wait(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/getenti%s/DE/%s/TE/%s", f.baseURL, cfg.endpointSuffix(), cfg.ElectionDate, cfg.teCode())
	resp, err := f.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Try flat array form first (enti is a []EligendoEntity directly)
	var flat struct {
		Int  json.RawMessage `json:"int"`
		Enti []EligendoEntity `json:"enti"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && len(flat.Enti) > 0 {
		return flat.Enti, nil
	}

	// Fall back to nested form (enti is {"ente": [...]})
	var nested getentiFIResponse
	if err := json.Unmarshal(body, &nested); err != nil {
		return nil, fmt.Errorf("decode eligendo response as nested: %w", err)
	}
	return nested.Enti.Ente, nil
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

	if resp.StatusCode == http.StatusForbidden {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, ErrElectionUnavailable
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

// scrutiniParty holds a single party's results in a scrutini response.
type scrutiniParty struct {
	Desc string  `json:"desc"`
	Voti int64   `json:"voti"`
	Perc float64 `json:"-"`
	Seggi int    `json:"seggi"`
	DescLis string  `json:"desc_lis,omitempty"`
	VotiRaw float64 `json:"voti,omitempty"`
	// percRaw handles Italian locale (comma decimal) and standard float formats
	percRaw percString `json:"perc"`
}

// percString handles JSON unmarshaling of percentage values that may be
// in Italian locale format (e.g. "28,81") or standard numeric format.
type percString float64

func (p *percString) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == `""` {
		*p = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("parse perc %q: %w", s, err)
	}
	*p = percString(v)
	return nil
}

// scrutiniFIResponse is the Eligendo API response for the FI (nazionale) endpoint.
// Struct is nested: { "liste": { "lista": [...] }, "datiGenerali": { ... } }
type scrutiniFIResponse struct {
	Intestazione struct {
		Cod string `json:"cod"`
	} `json:"intestazione"`
	Liste struct {
		Lista []scrutiniParty `json:"lista"`
	} `json:"liste"`
	DatiGenerali struct {
		Elettori int64 `json:"elettori"`
		Votanti  int64 `json:"votanti"`
	} `json:"datiGenerali"`
}

// scrutiniFlatResponse is the response envelope for non-FI endpoints (EI, CI, SI, R).
// Liste is a flat array: { "liste": [...], "int": {...} }
type scrutiniFlatResponse struct {
	Liste []scrutiniParty `json:"liste"`
}

// extractParties extracts parties from a scrutini response, handling both nested and flat formats.
func extractParties(body []byte) ([]scrutiniParty, error) {
	var flat scrutiniFlatResponse
	if err := json.Unmarshal(body, &flat); err == nil && len(flat.Liste) > 0 {
		for i := range flat.Liste {
			if flat.Liste[i].Desc == "" && flat.Liste[i].DescLis != "" {
				flat.Liste[i].Desc = flat.Liste[i].DescLis
			}
			flat.Liste[i].Perc = float64(flat.Liste[i].percRaw)
		}
		return flat.Liste, nil
	}

	// Fall back to nested FI format
	var nested scrutiniFIResponse
	if err := json.Unmarshal(body, &nested); err != nil {
		return nil, fmt.Errorf("decode scrutini response: %w", err)
	}
	for i := range nested.Liste.Lista {
		nested.Liste.Lista[i].Perc = float64(nested.Liste.Lista[i].percRaw)
	}
	return nested.Liste.Lista, nil
}

// RunElection executes the full pipeline: getenti → raw save → scrutini per comune → raw save → normalize party → write DuckDB.
func RunElection(ctx context.Context, db *sql.DB, baseURL string, cfg ElectionConfig, mapper *PartyMapper, rawDir string) ([]ElectionResult, error) {
	fetcher := NewElectionFetcher(baseURL, 1.0)

	entities, err := fetcher.GetEntities(ctx, cfg)
	if err != nil {
		if errors.Is(err, ErrElectionUnavailable) {
			slog.Warn("election unavailable via Eligendo API", "type", cfg.ElectionType, "year", cfg.Year, "date", cfg.ElectionDate)
		}
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

	// Increase rate limit for batch ingestion (100 req/s instead of 1)
	fetcher.rateLimiter = newTokenBucketLimiter(100.0, 10)

	var results []ElectionResult

	// Europee endpoint returns all province data in a single call at TE level
	if cfg.ElectionType == "europee" {
		url := fmt.Sprintf("%s/scrutini%s/DE/%s/TE/%s", baseURL, cfg.endpointSuffix(), cfg.ElectionDate, cfg.teCode())
		if err := fetcher.rateLimiter.Wait(); err != nil {
			return nil, fmt.Errorf("rate limiter wait: %w", err)
		}
		resp, err := fetcher.doGet(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("europee scrutini fetch: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("europee scrutini read body: %w", err)
		}
		if err := os.WriteFile(filepath.Join(rawPath, "scrutini_all.json"), body, 0644); err != nil {
			slog.Warn("failed to save europee scrutini raw", "error", err)
		}
		parties, err := extractParties(body)
		if err != nil {
			slog.Warn("europee party extraction failed", "error", err)
		} else {
			for _, lista := range parties {
				canon, found := mapper.Lookup(lista.Desc)
				if !found {
					slog.Info("unmapped party", "raw", lista.Desc)
				}
				results = append(results, ElectionResult{
					ElectionType:   cfg.ElectionType,
					Level:          cfg.Level,
					Year:           cfg.Year,
					Comune:         "ITALIA",
					ComuneISTAT:    "1000000000",
					Lista:          lista.Desc,
					PartyCanonical: canon,
					Voti:           lista.Voti,
					Percentuale:    lista.Perc,
					Seggi:          lista.Seggi,
				})
			}
		}
	} else {
		for _, ent := range entities {
			if len(ent.Cod) < 6 {
				slog.Debug("skipping entity with short ISTAT code", "cod", ent.Cod, "desc", ent.Desc)
				continue
			}
			// 10-digit codes have a leading region digit that represents
			// the constituency for europee (RRRPPPCCCC not RRPPPCCCC).
			var reg, prv, com string
			if len(ent.Cod) >= 10 {
				reg = ent.Cod[:3]
				prv = ent.Cod[3:6]
				com = ent.Cod[6:]
			} else {
				reg = ent.Cod[:2]
				prv = ent.Cod[2:5]
				com = ent.Cod[5:]
			}
			if len(reg) < 2 || len(prv) < 2 || len(com) < 2 {
				slog.Debug("skipping entity with unparseable ISTAT code", "cod", ent.Cod, "desc", ent.Desc)
				continue
			}
			if err := fetcher.rateLimiter.Wait(); err != nil {
				return nil, fmt.Errorf("rate limiter wait: %w", err)
			}

			url := fmt.Sprintf("%s/scrutini%s/DE/%s/TE/%s/RE/%s/PR/%s/CM/%s", baseURL, cfg.endpointSuffix(), cfg.ElectionDate, cfg.teCode(), reg, prv, com)
			resp, err := fetcher.doGet(ctx, url)
			if err != nil {
				slog.Error("scrutini fetch failed", "entity", ent.Cod, "error", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				slog.Error("scrutini read body failed", "entity", ent.Cod, "error", err)
				continue
			}

			rawName := fmt.Sprintf("scrutini_%s.json", ent.Cod)
			if err := os.WriteFile(filepath.Join(rawPath, rawName), body, 0644); err != nil {
				slog.Warn("failed to save scrutini raw", "entity", ent.Cod, "error", err)
			}

			parties, err := extractParties(body)
			if err != nil {
				slog.Error("scrutini party extraction failed", "entity", ent.Cod, "error", err)
				continue
			}

			for _, lista := range parties {
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
				})
			}
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
