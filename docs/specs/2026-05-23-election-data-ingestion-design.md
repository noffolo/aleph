# Election Data Ingestion — Design Document

**Date:** 2026-05-23
**Status:** Draft

## 1. Overview

Add a new `source_type="election"` to aleph's ingestion engine that fetches Italian
election results from the official Eligendo API (`eleapi.interno.gov.it`), covering
all election types from 2000 to present. Data is stored at comune-level granularity
with dual-write: raw API output + normalized party mapping.

## 2. API: Eligendo (Ministero dell'Interno)

### 2.1 Base URL & Headers

```
Base: https://eleapi.interno.gov.it/siel/PX/
Headers:
  Origin: https://elezioni.interno.gov.it
  Referer: https://elezioni.interno.gov.it/
  Accept: application/json
```

### 2.2 Endpoints

| Endpoint | Description |
|---|---|
| `getentiFI/DE/{YYYYMMDD}/TE/{tipo}` | Get entity tree (regioni → province → comuni) |
| `scrutiniFI/DE/{data}/TE/{te}/RE/{reg}/PR/{prv}/CM/{com}` | Get results for entity |
| `votantiFI/DE/{data}/TE/{te}/SK/01` | Get voter turnout |

### 2.3 TE Codes (Election Types)

| Code | Type | config value |
|---|---|---|
| 01 | Camera dei Deputati | `camera` |
| 02 | Senato della Repubblica | `senato` |
| 03 | Parlamento Europeo | `europee` |
| 04 | Regionali | `regionali` |
| 05 | Comunali | `comunali` |
| 09 | Referendum | `referendum` |

### 2.4 Rate Limiting

- **1 request/second** (matches ondata/referendum-download tool default).
- Retry with exponential backoff (1s, 2s, 4s) on 429/5xx.
- HTTP timeout: 30s.
- Uses existing `fetcher.go` with SSRF protection.

### 2.5 Pagination

No pagination — each `scrutiniFI` call returns all lists for that entity in a
single JSON response.

### 2.6 Multi-turn Elections

Comunali with ballottaggio require two separate calls: first round date and
ballottaggio date. Tagged with `turno = "primo"` or `turno = "secondo"`.

## 3. Data Model (Dual-Write)

### 3.1 `election_results_raw` — Raw API Data

| Column | Type | Description |
|---|---|---|
| `election_type` | VARCHAR | camera/senato/europee/regionali/comunali/referendum |
| `election_date` | DATE | Election date (YYYY-MM-DD) |
| `turno` | VARCHAR | primo/secondo (comunali ballottaggio) |
| `ente_cod` | VARCHAR | Eligendo entity code |
| `ente_desc` | VARCHAR | Human-readable entity name |
| `istat_cod` | VARCHAR | ISTAT code (mapped from cod_eligendo) |
| `lista_cod` | INTEGER | Eligendo list code |
| `lista_desc` | VARCHAR | Original list name from API |
| `voti` | INTEGER | Vote count |
| `voti_pct` | DOUBLE | Vote percentage |
| `coalizione_cod` | INTEGER | Coalition code (if aggregated) |
| `coalizione_desc` | VARCHAR | Coalition name |
| `candidato_cod` | INTEGER | Candidate code (regionali/comunali) |
| `candidato_desc` | VARCHAR | Candidate name |
| `quesito_cod` | INTEGER | Referendum question code |
| `voti_si` | INTEGER | Referendum yes votes |
| `voti_no` | INTEGER | Referendum no votes |
| `scraped_at` | TIMESTAMP | When scraped |

### 3.2 `election_results` — Normalized Data

Same schema as raw, plus:

| Column | Type | Description |
|---|---|---|
| `canonical_party` | VARCHAR | Canonical party name (e.g. "Fratelli d'Italia") |
| `confidence` | DOUBLE | Mapping confidence (0.0–1.0) |
| `mapping_source` | VARCHAR | "auto" (fuzzy match) or "manual" (override) |

### 3.3 `party_mapping` — Manual Override Table

| Column | Type | Description |
|---|---|---|
| `lista_desc` | VARCHAR | Exact list name from API |
| `canonical_party` | VARCHAR | Canonical party name |
| `created_at` | TIMESTAMP | When mapping was created |

## 4. Normalization Engine

### 4.1 Phase 1 — Fuzzy Auto-Matching

For each `lista_desc` from raw API data:

1. **Preprocessing**: uppercase → strip suffixes (`- LISTA CIVICA`, `- ITALIA ...`,
   candidate name suffixes) → remove punctuation → trim.
2. **Jaro-Winkler** distance against canonical party name list.
3. **Threshold**: score > 0.85 → auto-map (`confidence: score`, `mapping_source: "auto"`).
4. **Sub-threshold**: `canonical_party: NULL`, `confidence: score`, `mapping_source: NULL`.

### 4.2 Phase 2 — Manual Override

- User populates `party_mapping` with explicit entries.
- Manual overrides take **absolute priority** over fuzzy matching.
- Mapped entries get `confidence: 1.0` and `mapping_source: "manual"`.

### 4.3 Canonical Party List

Hardcoded in `election.go` with ~50 major Italian parties from 2000–present:
Fratelli d'Italia, Partito Democratico, Movimento 5 Stelle, Lega, Forza Italia,
Italia Viva, Azione, +Europa, Alleanza Verdi e Sinistra, Noi Moderati, etc.

## 5. Config JSON Schema

```json
{
  "election_type": "camera",
  "level": "comune",
  "start_date": "2018-01-01",
  "end_date": "2024-12-31"
}
```

| Field | Required | Description |
|---|---|---|
| `election_type` | Yes | camera/senato/europee/regionali/comunali/referendum |
| `level` | No | comune (default)/provincia/regione — output granularity |
| `start_date` | No | Filter by election_date (ISO date) |
| `end_date` | No | Filter by election_date (ISO date) |

`start_date`/`end_date` use the existing `DateRangeConfig` from `datefilter.go`.

## 6. Engine Integration

### 6.1 Switch Case

```go
// engine.go — new case in source dispatch
case "election":
    return e.runElection(ctx, task)
```

### 6.2 `runElection()` Flow

1. Parse `configJson` → extract `election_type`, `level`, date range.
2. Look up TE code from election_type.
3. Build `ElectionFetcher` with 1 req/s rate limiter.
4. Call `getentiFI` → get region/province/comune tree.
5. For each entity: call `scrutiniFI` → parse JSON.
6. For each list: write to `election_results_raw`, apply normalization,
   write to `election_results`.
7. `updateProgress()` after each entity.

### 6.3 Rate Limiter

```go
type electionRateLimiter struct {
    lastCall time.Time
    mu       sync.Mutex
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
```

### 6.4 ISTAT Code Mapping

`eligendo_codes.json` (static, go:embed) provides `cod_eligendo → cod_istat` lookup
table at 100% coverage (source: ondata association).

## 7. Frontend Integration

### 7.1 DataSourceFormSlideOver.tsx

New fields in the election source form:

- **Dropdown `election_type`**: Camera, Senato, Europee, Regionali, Comunali, Referendum
- **Dropdown `level`**: Comune (default), Provincia, Regione
- **Date range**: start_date / end_date (already existing, inherited from date filter)

### 7.2 RunTask Dialog

Date overrides via `configOverrides` (already implemented in config_overrides feature):
user can set start/end date before running any election ingestion task.

## 8. Execution Time Estimates

| Scope | Entities | Time (1 req/s) |
|---|---|---|
| National election | ~8,000 comuni | ~2h 13min |
| Single region | ~300 comuni | ~5 min |
| Single comune | 1 | ~1 sec |

## 9. Out of Scope

- **Party funding data** (`party_funding` source type) — separate task, uses
  `ondata/liberiamoli-tutti` CSV + Camera portal scraper.
- **Real-time election night data** — the API serves definitive results, not live
  counting.
- **Census/demographic cross-reference** — ISTAT population data for per-capita
  analysis is a future feature.
