package sources

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const anacSourceType = "public_contracts"

func RunANAC(ctx context.Context, baseURL string, db *sql.DB, wm interface {
	Set(sourceName string, lastRun time.Time, cursor string, metadata string) error
}, anno int, rawDir string) error {
	url := fmt.Sprintf("%s/CIG_%d.csv", baseURL, anno)
	slog.Info("downloading ANAC CSV", "url", url, "year", anno)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create ANAC request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download ANAC CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ANAC CSV download failed: HTTP %d", resp.StatusCode)
	}

	tmpDir := filepath.Join(rawDir, anacSourceType)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("CIG_%d.csv", anno))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read CSV body: %w", err)
	}

	if !utf8.Valid(body) {
		decoder := charmap.ISO8859_1.NewDecoder()
		utf8Body, err := decoder.Bytes(body)
		if err != nil {
			return fmt.Errorf("convert ISO-8859-1 to UTF-8: %w", err)
		}
		body = utf8Body
	}

	if err := os.WriteFile(tmpPath, body, 0644); err != nil {
		return fmt.Errorf("write raw CSV: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS public_contracts (
		cig TEXT PRIMARY KEY,
		anno INTEGER,
		importo REAL,
		stazione_appaltante TEXT,
		aggiudicatario TEXT,
		partecipanti TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	query := fmt.Sprintf(`INSERT OR REPLACE INTO public_contracts (cig, anno, importo, stazione_appaltante, aggiudicatario, partecipanti)
		SELECT CIG, %[1]d, CAST(Importo AS REAL), StazioneAppaltante, Aggiudicatario, Partecipanti
		FROM read_csv_auto('%[2]s', delim=';', header=true, all_varchar=true)`, anno, tmpPath)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("load CSV into DuckDB: %w", err)
	}

	if err := wm.Set(anacSourceType, time.Now(), fmt.Sprintf("%d", anno), `{"status":"ok"}`); err != nil {
		return fmt.Errorf("update watermark: %w", err)
	}

	slog.Info("ANAC ingestion complete", "year", anno, "file", tmpPath)
	return nil
}

func RunANACDryRun(baseURL string, anno int) error {
	url := fmt.Sprintf("%s/CIG_%d.csv", baseURL, anno)
	resp, err := http.DefaultClient.Head(url)
	if err != nil {
		return fmt.Errorf("ANAC source unreachable: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ANAC CSV %d not found: HTTP %d", anno, resp.StatusCode)
	}
	return nil
}
