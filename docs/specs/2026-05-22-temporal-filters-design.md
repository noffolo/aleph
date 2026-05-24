# Date Range Filtering per l'Ingestion — Design Doc

> **Goal:** Aggiungere filtri temporali opzionali (start_date, end_date) a tutte le source type del sistema di ingestion di Aleph, permettendo di selezionare un intervallo di date per l'importazione dei dati.

**Architecture:** Approccio condiviso: una funzione `DateFilter` centrale più configurazioni per-source per l'estrazione della data dagli item. I filtri sono passati via `config_json` del task, senza modifiche a proto o schema DB.

**Tech Stack:** Go standard library (time.Time, json.Unmarshal), DuckDB per storage.

---

## 1. Config Format

Ogni task di ingestion può includere nel suo `config_json` due campi opzionali:

```json
{
  "url": "https://example.com/feed.xml",
  "start_date": "2025-05-22",
  "end_date": "2026-05-22"
}
```

**Regole:**
- `start_date` / `end_date` sono **opzionali** — se omessi, nessun filtro temporale viene applicato
- Formati supportati: `YYYY-MM-DD` (ISO 8601 data sola), RFC3339 (`2026-05-22T15:04:05Z`), Unix timestamp (secondi, intero)
- `start_date`: esclude item con data **precedente** a questa (inclusivo)
- `end_date`: esclude item con data **successiva** a questa (inclusivo)
- Se solo uno dei due è presente, si filtra solo da un lato
- **Nessuna modifica a proto né a schema DB** — i campi vivono solo nel `config_json`

## 2. DateFilter Component

Nuovo file: `internal/ingestion/sources/datefilter.go`

```go
package sources

import "time"

// DateRangeConfig holds optional temporal filter bounds.
type DateRangeConfig struct {
    StartDate *time.Time `json:"start_date,omitempty"`
    EndDate   *time.Time `json:"end_date,omitempty"`
}

// ParseDateRangeFromConfig extracts DateRangeConfig from a task's config_json.
// Returns zero-value DateRangeConfig (both nil) if neither field is present.
func ParseDateRangeFromConfig(configJSON json.RawMessage) (DateRangeConfig, error)

// IsInRange checks an extracted time against the filter bounds.
// If dr has no bounds set, returns true (no filter).
// If itemDate is nil (date not extractable), returns true (include anyway).
func (dr DateRangeConfig) IsInRange(itemDate *time.Time) bool
```

**Regola fondamentale:** Se la data di un item non è estraibile, l'item viene **incluso comunque** ("meglio dati in più che in meno").

## 3. Per-Source Date Extraction

Ogni source handler che produce item multipli deve implementare l'estrazione della data e il filtro. Source che producono un singolo item (es. `url`) non sono interessate.

| Source Type | Dove si estrae la data | Come |
|---|---|---|
| **sitemap** | `PageResult` → già presente in `URLEntry.LastMod` | Aggiungere campo `ParsedDate *time.Time` a `PageResult`, popolato durante il parsing XML. Filtrare i `PageResult` dopo il crawl ma prima di scrivere su DuckDB. |
| **rss** (via `runPrecompiled`) | RSS/Atom XML → `<pubDate>`, `<updated>`, `<published>` | Aggiungere parsing delle date nel flusso RSS dentro `runPrecompiled` o in un nuovo parser RSS. Filtrare dopo il parsing. |
| **jsonapi** | Dipende dalla risposta API | Config: `date_path` (JSON path alla data) + `date_format` (Go time layout) |
| **scrape** | Estratto dal contenuto HTML | Config: `date_selector` (CSS selector) + `date_format` (Go time layout) |
| **csv** | Colonna specificata | Config: `date_column` (nome colonna) + `date_format` |
| **github** | `committed_date` / `created_at` | Usare il campo data già presente nella risposta GitHub API |
| **email** | `Date` header dell'email | Parsare l'header Date con formati email standard (RFC5322) |

**Source NON interessate** (producono 0 o 1 item, o filtrano via SQL):
- `url`: un singolo URL, nessuna data da estrarre
- `copy`: copia raw di dati, non ci sono item
- `postgres`: il filtro va direttamente nella **WHERE clause SQL** (`start_date`/`end_date` → condizione sulla colonna timestamp)
- `custom_code`: l'utente gestisce il filtro internamente nel suo codice

## 4. Handler Flow (Modificato)

```
task.RunTask()
  |
  v
Parsing config: estrae DateRangeConfig + source-specific config
  |
  v
switch source_type:
  case "sitemap":
    1. Parse sitemap XML (con LastMod → ParsedDate)
    2. Crawl URLs
    3. Filter PageResult[] with DateRangeConfig.IsInRange(pageResult.ParsedDate)
    4. Write filtered rows to DuckDB
  
  case "rss" / "rest":
    1. Fetch e parse RSS/Atom feed
    2. Estrai data da ogni item (pubDate/updated/published)
    3. Filter items with DateRangeConfig.IsInRange(item.Date)
    4. Per ogni item filtrato, create task "url" separato (come oggi)
  
  case "jsonapi":
    1. Fetch API response
    2. Estrai data usando date_path + date_format
    3. Filter items
    
  case "scrape":
    1. Scrape pagine
    2. Estrai data usando date_selector + date_format
    3. Filter items
    
  case "csv":
    1. Carica CSV
    2. Estrai data usando date_column + date_format
    3. Filter rows via DuckDB o in-memory
    
  case "github":
    1. Fetch GitHub API
    2. Estrai committed_date / created_at
    3. Filter items
    
  case "email":
    1. Fetch emails
    2. Estrai Date header
    3. Filter items
    
  case "postgres":
    Aggiungi condizioni WHERE sul campo timestamp usando start_date/end_date
```

## 5. Priorità di Implementazione

1. **sitemap** — più semplice (LastMod già presente), necessario per backfill 24 mesi
2. **rss/rest** — pubDate standard, copre la maggior parte dei feed
3. **jsonapi, scrape, csv** — richiedono configurazione aggiuntiva
4. **github, email** — minore urgenza
5. **postgres** — query SQL, implementazione diversa

## 6. Considerazioni

- **Performance:** La data va estratta da tutti gli item prima di scrivere su DB. Per RSS con centinaia di item, il filtro in-memory è trascurabile rispetto al costo HTTP.
- **Formati data:** I feed RSS hanno formati data variabili. Usare `time.Parse` con fallback su formati multipli (RFC1123Z, RFC1123, RFC3339, `2006-01-02T15:04:05Z`, etc.)
- **Fusi orari:** Tutte le date sono trattate in UTC. Se una data non ha fuso, assumere UTC.
- **Backfill 24 mesi:** Con `start_date="2024-05-22"` + `end_date="2026-05-22"`, la sitemap produce centinaia di URL. Il filtro evita di scaricare pagine fuori intervallo. Il numero di fetch HTTP dipende dalla risoluzione temporale della sitemap (data di ultima modifica per URL).
