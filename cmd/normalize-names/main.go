package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

type Candidate struct {
	Codice   string `json:"codice"`
	Cognome  string `json:"cognome"`
	Nome     string `json:"nome"`
	FullName string `json:"full_name"`
	Party    string `json:"party"`
	RawParty string `json:"raw_party"`
	Source   string `json:"source"`
}

func main() {
	cameraPath := flag.String("camera", "data/raw/elections/politiche2022/camera-italia-comune.csv", "Camera CSV file")
	senatoPath := flag.String("senato", "data/raw/elections/politiche2022/senato-italia-comune.csv", "Senato CSV file")
	dbPath := flag.String("db", "data/aleph.duckdb", "DuckDB database path")
	outPath := flag.String("out", "export_data/candidates_normalized.json", "Output JSON path")
	flag.Parse()

	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		slog.Error("failed to open DuckDB", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping DuckDB", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to DuckDB", "path", *dbPath)

	seen := make(map[string]*Candidate)

	if err := processCSV(*cameraPath, "camera", seen); err != nil {
		slog.Error("processing camera CSV", "error", err)
		os.Exit(1)
	}

	if err := processCSV(*senatoPath, "senato", seen); err != nil {
		slog.Error("processing senato CSV", "error", err)
		os.Exit(1)
	}

	slog.Info("deduplicated candidates", "count", len(seen))

	candidates := make([]Candidate, 0, len(seen))
	for _, c := range seen {
		candidates = append(candidates, *c)
	}

	if err := createAndInsert(db, candidates); err != nil {
		slog.Error("database operation", "error", err)
		os.Exit(1)
	}

	if err := writeJSON(*outPath, candidates); err != nil {
		slog.Error("writing JSON", "error", err)
		os.Exit(1)
	}
	slog.Info("export complete", "path", *outPath, "candidates", len(candidates))
}

func processCSV(path, source string, seen map[string]*Candidate) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ','
	r.LazyQuotes = true

	if _, err := r.Read(); err != nil {
		return fmt.Errorf("read header %s: %w", path, err)
	}

	slog.Info("processing CSV", "path", filepath.Base(path), "source", source)

	for {
		record, err := r.Read()
		if err != nil {
			break
		}

		if len(record) < 12 {
			continue
		}

		codice := record[0]
		cogn := record[1]
		nome := record[2]
		descLis := record[11]

		cognNorm := sources.NormalizeName(cogn)
		nomeNorm := sources.NormalizeName(nome)
		fullName := sources.NormalizeFullName(cogn, nome)
		partyNorm := sources.NormalizeName(descLis)

		key := cognNorm + "|" + nomeNorm + "|" + partyNorm
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = &Candidate{
			Codice:   codice,
			Cognome:  cognNorm,
			Nome:     nomeNorm,
			FullName: fullName,
			Party:    partyNorm,
			RawParty: descLis,
			Source:   source,
		}
	}
	return nil
}

func createAndInsert(db *sql.DB, candidates []Candidate) error {
	if _, err := db.Exec(`DROP TABLE IF EXISTS candidates_normalized`); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	ddl := `CREATE TABLE IF NOT EXISTS candidates_normalized (
		codice VARCHAR,
		cognome VARCHAR,
		nome VARCHAR,
		full_name VARCHAR,
		party VARCHAR,
		raw_party VARCHAR,
		source VARCHAR,
		PRIMARY KEY (codice, party, source)
	)`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	slog.Info("table ready", "table", "candidates_normalized")

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO candidates_normalized
		(codice, cognome, nome, full_name, party, raw_party, source)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, c := range candidates {
		if _, err := stmt.Exec(c.Codice, c.Cognome, c.Nome, c.FullName, c.Party, c.RawParty, c.Source); err != nil {
			return fmt.Errorf("exec row %s: %w", c.Codice, err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("inserted rows", "count", count)
	return nil
}

func writeJSON(path string, candidates []Candidate) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(candidates)
}
