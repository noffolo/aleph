package sources

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

var pollPartyColumns = map[string]string{
	"Partito Democratico":      "partito-democratico",
	"Forza Italia":             "forza-italia",
	"Fratelli d'Italia":        "fratelli-italia",
	"Alleanza Verdi Sinistra":  "verdi-sinistra",
	"Lega":                     "lega",
	"Movimento 5 Stelle":       "movimento-5-stelle",
	"+Europa":                  "piu-europa",
	"Azione":                   "azione",
	"Italia Viva":              "italia-viva",
	"Stati Uniti d'Europa":     "stati-uniti-europa",
	"Pace Terra Dignità":       "pace-terra-dignita",
	"Azione - Italia Viva":     "azione-italia-viva",
	"Azione/+Europa":           "azione-piu-europa",
	"Sinistra Ecologia Libertà": "sinistra-ecologia-liberta",
	"Scelta Civica":            "scelta-civica",
	"Unione di Centro":         "unione-di-centro",
	"Sud Chiama Nord":          "sud-chiama-nord",
	"Unione Popolare":          "unione-popolare",
	"Altri":                    "altri",
}

func canonicalizePollParty(colName string) string {
	if id, ok := pollPartyColumns[colName]; ok {
		return id
	}
	return normalizePartyName(colName)
}

func ensurePollsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS polls (
		id VARCHAR PRIMARY KEY,
		pollster VARCHAR,
		date DATE,
		sample_size INTEGER,
		party VARCHAR,
		party_canonical VARCHAR,
		percentage DOUBLE,
		margin_error DOUBLE,
		source_url VARCHAR,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensurePollResultsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS poll_results (
		id VARCHAR PRIMARY KEY,
		poll_id VARCHAR,
		pollster VARCHAR,
		date DATE,
		party VARCHAR,
		percentage DOUBLE,
		trend_3m DOUBLE,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func RunPollIngestion(ctx context.Context, db *sql.DB, csvPath string) error {
	slog.Info("starting poll ingestion", "csv", csvPath)

	if err := ensurePollsTable(db); err != nil {
		return fmt.Errorf("create polls table: %w", err)
	}

	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read csv header: %w", err)
	}

	headerIdx := make(map[string]int)
	for i, h := range headers {
		headerIdx[strings.TrimSpace(h)] = i
	}

	metadataCols := map[string]bool{
		"Row": true, "Data Inserimento": true, "Realizzatore": true,
		"Committente": true, "Titolo": true, "text": true,
		"domanda": true, "national_poll_rationale": true, "national_poll": true,
	}
	var partyCols []string
	for _, h := range headers {
		h = strings.TrimSpace(h)
		if !metadataCols[h] {
			partyCols = append(partyCols, h)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO polls
			(id, pollster, date, sample_size, party, party_canonical, percentage, margin_error, source_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	lineNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read csv record at line %d: %w", lineNum+1, err)
		}
		lineNum++

		if getCSVField(record, headerIdx, "national_poll") != "1" {
			continue
		}

		pollster := strings.TrimSpace(getCSVField(record, headerIdx, "Realizzatore"))
		dateStr := strings.TrimSpace(getCSVField(record, headerIdx, "Data Inserimento"))
		if pollster == "" || dateStr == "" {
			slog.Debug("skipping row with missing pollster or date", "line", lineNum)
			continue
		}

		date, err := parseItalianDate(dateStr)
		if err != nil {
			slog.Warn("skipping row with unparseable date", "date", dateStr, "line", lineNum, "error", err)
			continue
		}

		sanitizedPollster := sanitizeID(pollster)

		for _, col := range partyCols {
			val := strings.TrimSpace(getCSVField(record, headerIdx, col))
			if val == "" {
				continue
			}

			percentage, err := parseItalianFloat(val)
			if err != nil {
				slog.Debug("skipping non-numeric party value", "col", col, "val", val, "line", lineNum)
				continue
			}

			canonical := canonicalizePollParty(col)
			id := fmt.Sprintf("%s-%s-%s", dateStr, sanitizedPollster, canonical)

			if _, err := stmt.ExecContext(ctx, id, pollster, date, nil, col, canonical, percentage, nil, "https://www.sondaggipoliticoelettorali.it/"); err != nil {
				return fmt.Errorf("insert poll row: %w", err)
			}
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	slog.Info("poll ingestion complete", "rows", inserted, "lines", lineNum)
	return nil
}

func getCSVField(record []string, headerIdx map[string]int, col string) string {
	idx, ok := headerIdx[col]
	if !ok || idx >= len(record) {
		return ""
	}
	return record[idx]
}

func parseItalianDate(s string) (time.Time, error) {
	return time.Parse("02/01/2006", s)
}

func parseItalianFloat(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

func sanitizeID(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' || r == '.' {
			return '_'
		}
		if r >= 0x80 {
			return r
		}
		return '_'
	}, strings.ToLower(s))
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}
