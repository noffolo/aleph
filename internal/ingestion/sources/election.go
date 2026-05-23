package sources

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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
