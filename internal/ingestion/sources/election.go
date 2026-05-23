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

type ElectionConfig struct {
	ElectionType string `json:"election_type"`
	Level        string `json:"level"`
	Year         int    `json:"year"`
}

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

type ElectionResult struct {
	ElectionType   string
	Level          string
	Year           int
	Comune         string
	ComuneISTAT    string
	Lista          string
	PartyCanonical string
	Voti           int64
	Percentuale    float64
	Seggi          int
	Elettori       int64
	Votanti        int64
}

type ISTATLookup struct {
	byCode map[string]string
	byName map[string]string
}

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

func (l *ISTATLookup) Lookup(code string) (string, bool) {
	name, ok := l.byCode[code]
	return name, ok
}

func (l *ISTATLookup) LookupByName(name string) (string, bool) {
	code, ok := l.byName[strings.ToLower(name)]
	return code, ok
}

type PartyMapper struct {
	aliases   map[string]string
	overrides map[string]string
	mu        sync.RWMutex
}

func NewPartyMapper() *PartyMapper {
	pm := &PartyMapper{
		aliases:   make(map[string]string),
		overrides: make(map[string]string),
	}
	pm.loadBuiltinAliases()
	return pm
}

func (pm *PartyMapper) loadBuiltinAliases() {
	var raw map[string]string
	if err := json.Unmarshal(partyAliasesRaw, &raw); err != nil {
		return
	}
	for rawName, canonical := range raw {
		pm.AddAlias(rawName, canonical)
	}
}

func (pm *PartyMapper) AddAlias(rawName string, canonical string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.aliases[normalizePartyName(rawName)] = canonical
}

func (pm *PartyMapper) SetOverride(rawName string, canonical string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.overrides[rawName] = canonical
}

func (pm *PartyMapper) Lookup(rawName string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if canonical, ok := pm.overrides[rawName]; ok {
		return canonical, true
	}
	canonical, ok := pm.aliases[normalizePartyName(rawName)]
	return canonical, ok
}

func (pm *PartyMapper) GetOverride(rawName string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	canonical, ok := pm.overrides[rawName]
	return canonical, ok
}

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
