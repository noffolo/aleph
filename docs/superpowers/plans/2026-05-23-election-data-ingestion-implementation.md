# Election Data Ingestion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `source_type="election"` to aleph's ingestion engine, fetching Italian election results from Eligendo API with dual-write (raw + normalized party mapping) and frontend support.

**Architecture:** New source handler `election.go` with `ElectionFetcher` (1 req/s rate limiting), fuzzy matching normalization engine, dual DuckDB write. Reuses existing `datefilter.go`, `fetcher.go`. No proto changes (source_type is plain string).

**Tech Stack:** Go 1.26, DuckDB, testify, Jaro-Winkler (`github.com/xrash/smetrics`)

---

### Task 1: Election helper types + ISTAT code lookup

**Files:**
- Create: `internal/ingestion/sources/election.go`
- Create: `internal/ingestion/sources/eligendo_codes.json`

- [ ] **Step 1: Create eligibility_codes.json with go:embed support**

```json
{
  "regioni": {
    "01": {"cod_istat": "01", "desc": "Piemonte"},
    "02": {"cod_istat": "02", "desc": "Valle d'Aosta"},
    "03": {"cod_istat": "03", "desc": "Lombardia"}
  }
}
```

- [ ] **Step 2: Create election.go with TE codes, canonical parties, and types**

```go
package sources

import (
    "embed"
    "encoding/json"
    "time"
)

//go:embed eligibility_codes.json
var eligibilityCodesData embed.FS

// TE codes mapping
var teCodes = map[string]string{
    "camera":     "01",
    "senato":     "02",
    "europee":    "03",
    "regionali":  "04",
    "comunali":   "05",
    "referendum": "09",
}

// Canonical party list for fuzzy matching
var canonicalParties = []string{
    "Fratelli d'Italia",
    "Partito Democratico",
    "Movimento 5 Stelle",
    "Lega",
    "Forza Italia",
    "Italia Viva",
    "Azione",
    "+Europa",
    "Alleanza Verdi e Sinistra",
    "Noi Moderati",
    "Unione Popolare",
    "Italexit",
    "Südtiroler Volkspartei",
    "Rifondazione Comunista",
    "Sinistra Italiana",
    "Verdi",
    "Liberi e Uguali",
    "Potere al Popolo",
    "+Europa con Emma Bonino",
    "Noi con l'Italia",
    "Centro Democratico",
    "Scelta Civica",
    "Il Popolo della Libertà",
    "La Destra",
    "Fiamma Tricolore",
    "Italia dei Valori",
    "Radicali Italiani",
    "Federazione dei Verdi",
    "Partito Socialista Italiano",
    "UDC",
}

type ElectionConfig struct {
    ElectionType string `json:"election_type"`
    Level        string `json:"level"`
}

type ElectionResult struct {
    ElectionType   string
    ElectionDate   time.Time
    Turno          string
    EnteCod        string
    EnteDesc       string
    IstatCod       string
    ListaCod        int64
    ListaDesc       string
    Voti            int64
    VotiPct         float64
    CoalizioneCod   int64
    CoalizioneDesc  string
    CandidatoCod    int64
    CandidatoDesc  string
    QuesitoCod      int64
    VotiSi          int64
    VotiNo          int64
    CanonicalParty  string
    Confidence      float64
    MappingSource   string
}

type eligibilityCodeMap struct {
    CodIstat string `json:"cod_istat"`
    Desc     string `json:"desc"`
}

func loadEligendoCodes() (map[string]eligendoCodeMap, error) {
    data, err := eligibilityCodesData.ReadFile("eligendo_codes.json")
    if err != nil {
        return nil, err
    }
    var raw map[string]map[string]eligendoCodeMap
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }
    return raw["comuni"], nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/ingestion/sources/`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/eligendo_codes.json
git commit -m "feat(election): add election types, TE codes, canonical party list, ISTAT lookup"
```

---

### Task 2: ElectionFetcher with rate limiting + API calls

**Files:**
- Modify: `internal/ingestion/sources/election.go`

- [ ] **Step 1: Write failing test for rate limiter**

Create `internal/ingestion/sources/election_test.go`:

```go
package sources

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestElectionRateLimiter_WaitsAtLeastOneSecond(t *testing.T) {
    rl := newElectionRateLimiter()
    start := time.Now()
    rl.Wait()
    rl.Wait()
    elapsed := time.Since(start)
    assert.GreaterOrEqual(t, elapsed, 1*time.Second)
}

func TestTEcodeMapping(t *testing.T) {
    assert.Equal(t, "01", teCodes["camera"])
    assert.Equal(t, "09", teCodes["referendum"])
    _, ok := teCodes["invalid"]
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ingestion/sources/ -run TestElectionRateLimiter -v`
Expected: FAIL (rate limiter not defined yet)

- [ ] **Step 3: Implement rate limiter and ElectionFetcher**

```go
import (
    "sync"
    "net/http"
    "fmt"
    "context"
)

type electionRateLimiter struct {
    lastCall time.Time
    mu       sync.Mutex
}

func newElectionRateLimiter() *electionRateLimiter {
    return &electionRateLimiter{}
}

func (r *electionRateLimiter) Wait() {
    r.mu.Lock()
    defer r.mu.Unlock()
    elapsed := time.Since(r.lastCall)
    if elapsed < time.Second {
        time.Sleep(time.Second - elapsed)
    }
    r.lastCall = time.Now()
}

type ElectionFetcher struct {
    client      *http.Client
    baseURL     string
    rateLimiter *electionRateLimiter
}

func NewElectionFetcher(client *http.Client) *ElectionFetcher {
    return &ElectionFetcher{
        client:      client,
        baseURL:     "https://eleapi.interno.gov.it/siel/PX",
        rateLimiter: newElectionRateLimiter(),
    }
}

func (f *ElectionFetcher) fetch(ctx context.Context, path string) ([]byte, error) {
    f.rateLimiter.Wait()
    url := fmt.Sprintf("%s/%s", f.baseURL, path)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Origin", "https://elezioni.interno.gov.it")
    req.Header.Set("Referer", "https://elezioni.interno.gov.it/")
    req.Header.Set("Accept", "application/json")
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    // ... read body, handle non-200 ...
    return io.ReadAll(resp.Body)
}

func (f *ElectionFetcher) GetEntities(ctx context.Context, date string, teCode string) ([]Entity, error) {
    // GET getentiFI/DE/{date}/TE/{teCode} → parse entity tree
}

func (f *ElectionFetcher) GetScrutini(ctx context.Context, date, te, reg, prv, com string) (*ScrutiniResponse, error) {
    // GET scrutiniFI/DE/{date}/TE/{te}/RE/{reg}/PR/{prv}/CM/{com} → parse results
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ingestion/sources/ -run "TestElectionRateLimiter|TestTEcodeMapping" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go
git commit -m "feat(election): add ElectionFetcher with rate limiting and Eligendo API client"
```

---

### Task 3: Fuzzy matching normalization engine

**Files:**
- Modify: `internal/ingestion/sources/election.go`
- Modify: `internal/ingestion/sources/election_test.go`

- [ ] **Step 1: Write failing tests for normalization**

```go
func TestNormalizePartyName_ExactMatch(t *testing.T) {
    result := normalizePartyName("PARTITO DEMOCRATICO")
    assert.Equal(t, "Partito Democratico", result.CanonicalParty)
    assert.Greater(t, result.Confidence, 0.9)
    assert.Equal(t, "auto", result.MappingSource)
}

func TestNormalizePartyName_FuzzyMatch(t *testing.T) {
    result := normalizePartyName("FRATELLI D'ITALIA CON GIORGIA MELONI")
    assert.Equal(t, "Fratelli d'Italia", result.CanonicalParty)
    assert.Greater(t, result.Confidence, 0.85)
    assert.Equal(t, "auto", result.MappingSource)
}

func TestNormalizePartyName_NoMatch(t *testing.T) {
    result := normalizePartyName("LISTA CIVICA XYXZZZ MAI ESISTITA 1999")
    assert.Empty(t, result.CanonicalParty)
    assert.Less(t, result.Confidence, 0.85)
    assert.Empty(t, result.MappingSource)
}

func TestNormalizePartyName_ManualOverride(t *testing.T) {
    mappings["IL POPOLO DELLA LIBERTA'"] = "Forza Italia"
    result := normalizePartyName("IL POPOLO DELLA LIBERTA'")
    assert.Equal(t, "Forza Italia", result.CanonicalParty)
    assert.Equal(t, 1.0, result.Confidence)
    assert.Equal(t, "manual", result.MappingSource)
    delete(mappings, "IL POPOLO DELLA LIBERTA'")
}

func TestPreprocessListName(t *testing.T) {
    tests := []struct{ name, input, expected string }{
        {"suffix removal", "PARTITO DEMOCRATICO - ITALIA DEMOCRATICA E PROGRESSISTA", "PARTITO DEMOCRATICO"},
        {"already clean", "LEGA", "LEGA"},
        {"punctuation", "MOVIMENTO 5 STELLE!!!", "MOVIMENTO 5 STELLE"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, preprocessListName(tt.input))
        })
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ingestion/sources/ -run "TestNormalize|TestPreprocess" -v`
Expected: FAIL (functions not defined)

- [ ] **Step 3: Implement normalization**

```go
import (
    "strings"
    "github.com/xrash/smetrics"
    "unicode"
)

var mappings = make(map[string]string)

type NormalizedParty struct {
    CanonicalParty string
    Confidence      float64
    MappingSource   string
}

func preprocessListName(name string) string {
    name = strings.ToUpper(name)
    name = strings.TrimSpace(name)
    suffixes := []string{
        " - ITALIA DEMOCRATICA E PROGRESSISTA",
        " - LISTA CIVICA",
        " CON GIORGIA MELONI",
        " - LEGA SALVINI PREMIER",
        " - BERLUSCONI PRESIDENTE",
        " CON MATTEO RENZI",
        " - ITALIA VIVA",
    }
    for _, s := range suffixes {
        if idx := strings.Index(name, s); idx > 0 {
            name = name[:idx]
        }
    }
    name = strings.Map(func(r rune) rune {
        if unicode.IsPunct(r) && r != '\'' {
            return -1
        }
        return r
    }, name)
    return strings.TrimSpace(name)
}

func normalizePartyName(raw string) NormalizedParty {
    if mapped, ok := mappings[raw]; ok {
        return NormalizedParty{
            CanonicalParty: mapped,
            Confidence:      1.0,
            MappingSource:   "manual",
        }
    }
    clean := preprocessListName(raw)
    bestScore := 0.0
    bestParty := ""
    for _, cp := range canonicalParties {
        score := smetrics.JaroWinkler(clean, strings.ToUpper(cp), 0.7, 4)
        if score > bestScore {
            bestScore = score
            bestParty = cp
        }
    }
    if bestScore > 0.85 {
        return NormalizedParty{
            CanonicalParty: bestParty,
            Confidence:      bestScore,
            MappingSource:   "auto",
        }
    }
    return NormalizedParty{
        Confidence:    bestScore,
        MappingSource: "",
    }
}
```

- [ ] **Step 4: Add `github.com/xrash/smetrics` dependency**

Run: `cd /tmp/opencode/aleph && go get github.com/xrash/smetrics`

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ingestion/sources/ -run "TestNormalize|TestPreprocess" -v`
Expected: all 5 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ingestion/sources/election.go internal/ingestion/sources/election_test.go go.mod go.sum
git commit -m "feat(election): add fuzzy matching normalization engine with manual override"
```

---

### Task 4: runElection() in engine.go

**Files:**
- Modify: `internal/ingestion/engine.go`
- Create: `internal/ingestion/election_test.go` (integration test with mock API)

- [ ] **Step 1: Write failing integration test**

```go
package ingestion

import (
    "testing"
    "net/http"
    "net/http/httptest"
    "github.com/stretchr/testify/require"
)

func TestRunElection_ParsesConfig(t *testing.T) {
    engine := setupTestEngine(t)
    task := &IngestionTask{
        SourceType: "election",
        ConfigJSON: json.RawMessage(`{"election_type":"camera"}`),
    }
    config, err := parseElectionConfig(task.ConfigJSON)
    require.NoError(t, err)
    require.Equal(t, "camera", config.ElectionType)
    require.Equal(t, "comune", config.Level) // default
}

func TestRunElection_InvalidType(t *testing.T) {
    engine := setupTestEngine(t)
    task := &IngestionTask{
        SourceType: "election",
        ConfigJSON: json.RawMessage(`{"election_type":"invalid"}`),
    }
    err := engine.runElection(context.Background(), task)
    require.Error(t, err)
    require.Contains(t, err.Error(), "unknown election_type")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ingestion/ -run "TestRunElection" -v`
Expected: FAIL (runElection not defined)

- [ ] **Step 3: Add case to engine.go switch + parse function + runElection**

```go
// In engine.go, add to the switch at ~line 180:
case "election":
    return e.runElection(ctx, task)

// New function in engine.go:
func parseElectionConfig(raw json.RawMessage) (sources.ElectionConfig, error) {
    var config sources.ElectionConfig
    if err := json.Unmarshal(raw, &config); err != nil {
        return config, fmt.Errorf("invalid election config: %w", err)
    }
    if config.Level == "" {
        config.Level = "comune"
    }
    if _, ok := sources.TECodes[config.ElectionType]; !ok {
        return config, fmt.Errorf("unknown election_type: %s", config.ElectionType)
    }
    return config, nil
}

func (e *Engine) runElection(ctx context.Context, task *IngestionTask) error {
    config, err := parseElectionConfig(task.ConfigJSON)
    if err != nil {
        return err
    }
    dr, err := sources.ParseDateRangeFromConfig(task.ConfigJSON)
    if err != nil {
        return err
    }
    fetcher := sources.NewElectionFetcher(e.httpClient)
    teCode := sources.TECodes[config.ElectionType]

    // Get available election dates
    // For MVP: use known dates from a hardcoded list (2024-06-08 europee, 2022-09-25 politiche, etc.)
    // Full implementation: call getentiFI to discover available dates

    dates := getElectionDates(config.ElectionType, dr)

    for _, date := range dates {
        entities, err := fetcher.GetEntities(ctx, date, teCode)
        if err != nil {
            return fmt.Errorf("getenti for %s: %w", date, err)
        }
        for _, ent := range entities {
            if e.isDone(ctx) {
                return ctx.Err()
            }
            results, err := fetcher.GetScrutini(ctx, date, teCode, ent.Reg, ent.Prv, ent.Com)
            if err != nil {
                e.log.Warn("scrutini failed for ente %s: %v", ent.Desc, err)
                continue
            }
            for _, r := range results.Liste {
                raw := sources.ElectionResult{
                    ElectionType:  config.ElectionType,
                    ElectionDate:  date,
                    EnteCod:       ent.Cod,
                    EnteDesc:      ent.Desc,
                    IstatCod:      lookUpIstat(ent.Cod),
                    ListaCod:       r.Cod,
                    ListaDesc:      r.Desc,
                    Voti:           r.Voti,
                    VotiPct:        r.Pct,
                    CoalizioneCod:  r.CoalCod,
                    CoalizioneDesc: r.CoalDesc,
                }
                // Write raw
                if err := e.writeElectionRaw(raw); err != nil {
                    return fmt.Errorf("write raw: %w", err)
                }
                // Normalize and write
                norm := sources.NormalizePartyName(raw.ListaDesc)
                raw.CanonicalParty = norm.CanonicalParty
                raw.Confidence = norm.Confidence
                raw.MappingSource = norm.MappingSource
                if err := e.writeElectionResult(raw); err != nil {
                    return fmt.Errorf("write normalized: %w", err)
                }
            }
            e.updateProgress(task.ID, progress)
        }
    }
    return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ingestion/ -run "TestRunElection" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/engine.go internal/ingestion/election_test.go
git commit -m "feat(election): add runElection() with Eligendo API integration and dual DuckDB write"
```

---

### Task 5: Frontend — DataSourceForm election fields

**Files:**
- Modify: `web/src/components/DataSourceFormSlideOver.tsx`
- Modify: `web/src/components/DataSourceForm.tsx`

- [ ] **Step 1: Add election_type dropdown to DataSourceForm**

In `DataSourceForm.tsx`, add after the source_type dropdown:

```tsx
{sourceType === 'election' && (
  <>
    <div className="space-y-2">
      <Label>Tipo Elezione</Label>
      <Select
        value={config.election_type || ''}
        onValueChange={(v) => setConfig({ ...config, election_type: v })}
      >
        <SelectTrigger>
          <SelectValue placeholder="Seleziona tipo elezione" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="camera">Camera dei Deputati</SelectItem>
          <SelectItem value="senato">Senato della Repubblica</SelectItem>
          <SelectItem value="europee">Parlamento Europeo</SelectItem>
          <SelectItem value="regionali">Regionali</SelectItem>
          <SelectItem value="comunali">Comunali</SelectItem>
          <SelectItem value="referendum">Referendum</SelectItem>
        </SelectContent>
      </Select>
    </div>
    <div className="space-y-2">
      <Label>Livello di dettaglio</Label>
      <Select
        value={config.level || 'comune'}
        onValueChange={(v) => setConfig({ ...config, level: v })}
      >
        <SelectTrigger>
          <SelectValue placeholder="Livello" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="comune">Comune</SelectItem>
          <SelectItem value="provincia">Provincia</SelectItem>
          <SelectItem value="regione">Regione</SelectItem>
        </SelectContent>
      </Select>
    </div>
  </>
)}
```

- [ ] **Step 2: Apply same changes to DataSourceFormSlideOver.tsx**

Same election_type and level dropdowns.

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add web/src/components/DataSourceForm.tsx web/src/components/DataSourceFormSlideOver.tsx
git commit -m "feat(election): add election_type and level dropdowns to DataSourceForm"
```
