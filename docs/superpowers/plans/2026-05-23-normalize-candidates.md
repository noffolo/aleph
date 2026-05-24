# Normalizzazione Nomi Candidati — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Normalizzare i nomi dei candidati (MAIUSCOLO → capitalized italiano) dai CSV Camera/Senato 2022 e creare una tabella DuckDB per cross-reference con donatori PEP e sentiment.

**Architecture:** `cmd/normalize-names/` CLI che legge i CSV ondata, applica NormalizeName() con regole di capitalizzazione italiana (D', De, Mac, apostrofi, accenti), e popola `candidates_normalized` in DuckDB con export JSON.

**Tech Stack:** Go, DuckDB, CSV parsing

---

### Task 1: NormalizeName function + tests

**Files:**
- Create: `internal/ingestion/sources/normalize.go`
- Create: `internal/ingestion/sources/normalize_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestNormalizeName(t *testing.T) {
    tests := []struct{ input, expected string }{
        {"MAGI", "Magi"},
        {"RICCARDO", "Riccardo"},
        {"DE LUCIA", "De Lucia"},
        {"D'AGOSTO", "D'Agosto"},
        {"D'AMICO", "D'Amico"},
        {"MACCHIA", "Macchia"},
        {"BLO'", "Blo'"},
        {"CALABRO'", "Calabro'"},
        {"NICCOLO'", "Niccolo'"},
        {"VAN DEN BOSCH", "Van Den Bosch"}, // foreign name convention
        {"STEFANO  MARIA", "Stefano Maria"}, // double space
        {"SALVATORE ", "Salvatore"},         // trailing space
        {" ALESSANDRO", "Alessandro"},        // leading space
        {"DE ANGELIS", "De Angelis"},
        {"DELL'ANNA", "Dell'Anna"},
        {"DE ROSA", "De Rosa"},
        {"LA ROSA", "La Rosa"},
        {"DI GIOVANNI", "Di Giovanni"},
        {"DEL MONTE", "Del Monte"},
        {"DE LUCA", "De Luca"},
    }
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            assert.Equal(t, tt.expected, NormalizeName(tt.input))
        })
    }
}

func TestNormalizeFullName(t *testing.T) {
    assert.Equal(t, "Magi Riccardo", NormalizeFullName("MAGI", "RICCARDO"))
    assert.Equal(t, "De Lucia Maria", NormalizeFullName("DE LUCIA", "MARIA"))
    assert.Equal(t, "D'Agosto Luigi", NormalizeFullName("D'AGOSTO", "LUIGI"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestNormalizeName -v -count=1 2>&1
```
Expected: FAIL (function not defined)

- [ ] **Step 3: Implement NormalizeName**

```go
package sources

import "strings"

// NormalizeName applies Italian capitalization rules:
// - First letter uppercase, rest lowercase
// - Preserves apostrophe+next uppercase (D'Agosto, Dell'Anna)
// - Preserves Mac/Mc as is (Macchi, McDonald)
// - Preserves accents on last char (Calabrò → Calabro' stays)
// - Handles compound surnames (De Luca, Di Giovanni, Van Den Bosch)
func NormalizeName(raw string) string {
    raw = strings.TrimSpace(raw)
    raw = strings.Join(strings.Fields(raw), " ") // collapse whitespace

    words := strings.Split(raw, " ")
    for i, w := range words {
        if isItalianPrefix(w) {
            words[i] = capitalize(w)
        } else if hasApostrophePrefix(w) {
            // D'Agosto → D'Agosto (preserve D' + capitalize next)
            parts := strings.SplitN(w, "'", 2)
            if len(parts) == 2 {
                parts[0] = strings.ToUpper(parts[0])
                parts[1] = capitalize(parts[1])
                words[i] = strings.Join(parts, "'")
            }
        } else {
            words[i] = capitalize(w)
        }
    }
    return strings.Join(words, " ")
}

func capitalize(s string) string {
    if len(s) == 0 {
        return s
    }
    return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

var italianPrefixes = map[string]bool{
    "de": true, "del": true, "della": true, "delle": true, "dei": true, "degli": true,
    "di": true, "dal": true, "dallo": true, "dalla": true, "dai": true, "dagli": true,
    "la": true, "lo": true, "le": true, "li": true,
    "van": true, "von": true, "ten": true,
}

func isItalianPrefix(s string) bool {
    return italianPrefixes[strings.ToLower(s)]
}

func hasApostrophePrefix(s string) bool {
    parts := strings.SplitN(s, "'", 2)
    if len(parts) != 2 {
        return false
    }
    prefix := strings.ToUpper(parts[0])
    return prefix == "D" || prefix == "L" || prefix == "N" || prefix == "S" || prefix == "M" || prefix == "DELL"
}

// NormalizeFullName combines cognome and nome with proper capitalization.
func NormalizeFullName(cognome, nome string) string {
    return NormalizeName(cognome) + " " + NormalizeName(nome)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /tmp/opencode/aleph && go test ./internal/ingestion/sources/ -run TestNormalizeName -v -count=1 2>&1
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/sources/normalize.go internal/ingestion/sources/normalize_test.go
git commit -m "feat: add NormalizeName with Italian capitalization rules"
```

---

### Task 2: cmd/normalize-names CLI

**Files:**
- Create: `cmd/normalize-names/main.go`

- [ ] **Step 1: Write the CLI**

```go
package main

import (
    "database/sql"
    "encoding/csv"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "log/slog"
    "os"
    "path/filepath"
    "strings"

    _ "github.com/marcboeker/go-duckdb"
    "github.com/noffolo/aleph/internal/ingestion/sources"
)

type Candidate struct {
    Codice     string `json:"codice"`
    Cognome    string `json:"cognome"`
    Nome       string `json:"nome"`
    FullName   string `json:"full_name"`
    Party      string `json:"party"`
    RawParty   string `json:"raw_party"`
    Source     string `json:"source"` // "camera" | "senato"
}

func main() {
    cameraPath := flag.String("camera", "data/raw/elections/politiche2022/camera-italia-comune.csv", "Camera CSV path")
    senatoPath := flag.String("senato", "data/raw/elections/politiche2022/senato-italia-comune.csv", "Senato CSV path")
    dbPath := flag.String("db", "data/aleph.duckdb", "DuckDB database path")
    outPath := flag.String("out", "export_data/candidates_normalized.json", "JSON export path")
    flag.Parse()

    db, err := sql.Open("duckdb", *dbPath)
    if err != nil {
        slog.Error("open db", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    allCandidates := make(map[string]*Candidate) // key: cognome+nome+party

    // Camera CSV columns: codice,cogn,nome,a_nome,r_pos,voti,perc,eletto,voti_solo_can,l_pos,pos,desc_lis,perc_cand
    count := readCSV(*cameraPath, allCandidates, "camera", 0, 1, 2, 11)
    slog.Info("read camera CSV", "rows", count)

    // Senato CSV columns: codice,cogn,nome,a_nome,r_pos,voti,perc,eletto,voti_solo_can,l_pos,pos,desc_lis
    count = readCSV(*senatoPath, allCandidates, "senato", 0, 1, 2, 11)
    slog.Info("read senato CSV", "rows", count)

    slog.Info("unique candidates", "count", len(allCandidates))

    // Create DuckDB table
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS candidates_normalized (
        codice VARCHAR,
        cognome VARCHAR,
        nome VARCHAR,
        full_name VARCHAR,
        party VARCHAR,
        raw_party VARCHAR,
        source VARCHAR,
        UNIQUE(codice, party, source)
    )`)
    if err != nil {
        slog.Error("create table", "error", err)
        os.Exit(1)
    }

    // Batch insert
    tx, err := db.Begin()
    if err != nil {
        slog.Error("begin tx", "error", err)
        os.Exit(1)
    }
    stmt, err := tx.Prepare(`INSERT OR IGNORE INTO candidates_normalized (codice, cognome, nome, full_name, party, raw_party, source) VALUES (?, ?, ?, ?, ?, ?, ?)`)
    if err != nil {
        slog.Error("prepare", "error", err)
        os.Exit(1)
    }

    for _, c := range allCandidates {
        _, err := stmt.Exec(c.Codice, c.Cognome, c.Nome, c.FullName, c.Party, c.RawParty, c.Source)
        if err != nil {
            slog.Warn("insert", "error", err, "codice", c.Codice)
        }
    }
    if err := tx.Commit(); err != nil {
        slog.Error("commit", "error", err)
        os.Exit(1)
    }

    // Export JSON
    candidates := make([]*Candidate, 0, len(allCandidates))
    for _, c := range allCandidates {
        candidates = append(candidates, c)
    }
    out, err := json.MarshalIndent(candidates, "", "  ")
    if err != nil {
        slog.Error("marshal", "error", err)
        os.Exit(1)
    }
    if err := os.MkdirAll(filepath.Dir(*outPath), 0755); err != nil {
        slog.Error("mkdir", "error", err)
        os.Exit(1)
    }
    if err := os.WriteFile(*outPath, out, 0644); err != nil {
        slog.Error("write", "error", err)
        os.Exit(1)
    }
    slog.Info("export complete", "path", *outPath, "candidates", len(candidates))
}

func readCSV(path string, candidates map[string]*Candidate, source string, codiceIdx, cognIdx, nomeIdx, partyIdx int) int {
    f, err := os.Open(path)
    if err != nil {
        slog.Error("open csv", "path", path, "error", err)
        return 0
    }
    defer f.Close()

    r := csv.NewReader(f)
    r.Comma = ','
    r.LazyQuotes = true

    header, _ := r.Read() // skip header

    // Detect comma-vs-semicolumn
    if header != nil {
        // re-read since we consumed the header
    }
    r = csv.NewReader(f)
    r.Comma = ','
    r.LazyQuotes = true

    // Skip header by reading it
    r.Read()

    count := 0
    for {
        row, err := r.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            continue
        }
        if len(row) <= partyIdx {
            continue
        }

        cogn := strings.TrimSpace(row[cognIdx])
        nome := strings.TrimSpace(row[nomeIdx])
        party := strings.TrimSpace(row[partyIdx])
        codice := strings.TrimSpace(row[codiceIdx])

        if cogn == "" || nome == "" || party == "" {
            continue
        }

        fullName := sources.NormalizeFullName(cogn, nome)
        partyClean := sources.NormalizeName(party)
        key := cogn + "|" + nome + "|" + party

        if _, exists := candidates[key]; !exists {
            candidates[key] = &Candidate{
                Codice:   codice,
                Cognome:  sources.NormalizeName(cogn),
                Nome:     sources.NormalizeName(nome),
                FullName: fullName,
                Party:    partyClean,
                RawParty: party,
                Source:   source,
            }
        }
        count++
    }
    return count
}
```

- [ ] **Step 2: Build and run**

```bash
cd /tmp/opencode/aleph && go build ./cmd/normalize-names/ && go vet ./cmd/normalize-names/
```
Expected: Exit 0, no errors

- [ ] **Step 3: Run against real DuckDB**

```bash
cd /tmp/opencode/aleph && ./normalize-names -camera data/raw/elections/politiche2022/camera-italia-comune.csv -senato data/raw/elections/politiche2022/senato-italia-comune.csv -db data/aleph.duckdb
```
Expected: Reads ~235k rows, inserts unique candidates into DuckDB

- [ ] **Step 4: Verify output**

```bash
cd /tmp/opencode/aleph && go run ./cmd/sql-query/ -db data/aleph.duckdb -query "SELECT COUNT(*), source FROM candidates_normalized GROUP BY source ORDER BY source" 2>/dev/null || echo "no sql-query, using duckdb cli instead"
```
Expected: Counts per source type. Verify first few rows look right.

- [ ] **Step 5: Cleanup binary and commit**

```bash
rm -f normalize-names normalize-names.exe normalize-names.test normalize-names.test.exe
git add cmd/normalize-names/main.go export_data/candidates_normalized.json 2>/dev/null; git add cmd/normalize-names/main.go
git commit -m "feat: normalize candidate names from Camera/Senato CSVs into DuckDB"
```

---

### Task 3: Verify cross-reference with party_funding donors

**Files:** (read-only)

- [ ] **Step 1: Run DuckDB query to find donor-candidate matches**

Use `cmd/export-data` or direct tool to run:
```sql
SELECT DISTINCT pf.donor_name, cn.full_name, pf.recipient_party, cn.party
FROM party_funding pf
JOIN candidates_normalized cn ON LOWER(cn.full_name) = LOWER(pf.donor_name)
LIMIT 20;
```

- [ ] **Step 2: Report match statistics**

Count exact matches, partial matches (cognome match within same party), and no-match donors.
