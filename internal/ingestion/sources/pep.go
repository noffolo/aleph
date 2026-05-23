package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// --- FollowTheMoney Entity ---

type FtMEntity struct {
	ID        string `json:"id"`
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	Country   string `json:"country"`
	BirthDate string `json:"birthDate"`
	Position  string `json:"position"`
	Dataset   string `json:"dataset"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

func ParseFtMEntity(data []byte) (FtMEntity, error) {
	var raw struct {
		ID         string   `json:"id"`
		Schema     string   `json:"schema"`
		Properties struct {
			Name      []string `json:"name"`
			Country   []string `json:"country"`
			BirthDate []string `json:"birthDate"`
			Position  []string `json:"position"`
		} `json:"properties"`
		Datasets  []string `json:"datasets"`
		FirstSeen string   `json:"first_seen"`
		LastSeen  string   `json:"last_seen"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return FtMEntity{}, err
	}
	return FtMEntity{
		ID:        raw.ID,
		Schema:    raw.Schema,
		Name:      firstPEP(raw.Properties.Name),
		Country:   firstPEP(raw.Properties.Country),
		BirthDate: firstPEP(raw.Properties.BirthDate),
		Position:  firstPEP(raw.Properties.Position),
		Dataset:   firstPEP(raw.Datasets),
		FirstSeen: raw.FirstSeen,
		LastSeen:  raw.LastSeen,
	}, nil
}

func firstPEP(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// --- Watermarker interface (avoids import cycle with ingestion package) ---

type WatermarkSetter interface {
	Set(sourceName string, lastRun time.Time, cursor string, metadata string) error
}

// --- PEP Fetcher ---

const pepSourceType = "pep"

type PEPFetcher struct {
	sourceType string
}

func (p *PEPFetcher) SourceType() string { return p.sourceType }
func (p *PEPFetcher) Validate() error    { return nil }

// --- Ingestion ---

func ensurePEPTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pep_entities (
		id TEXT PRIMARY KEY,
		schema_name TEXT DEFAULT '',
		name TEXT,
		country TEXT,
		birth_date TEXT,
		position TEXT,
		dataset TEXT,
		first_seen TEXT,
		last_seen TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

type ftmResponse struct {
	Entities []json.RawMessage `json:"entities"`
}

func RunPEP(ctx context.Context, baseURL string, db *sql.DB, wm WatermarkSetter, rawDir string) error {
	slog.Info("starting PEP ingestion", "url", baseURL)

	if err := ensurePEPTable(db); err != nil {
		return fmt.Errorf("ensure pep_entities table: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch PEP data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PEP API returned HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 500)]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read PEP response: %w", err)
	}

	// Save raw response
	rawDir = filepath.Join(rawDir, pepSourceType)
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}
	rawPath := filepath.Join(rawDir, fmt.Sprintf("pep_%d.json", time.Now().Unix()))
	if err := os.WriteFile(rawPath, body, 0644); err != nil {
		slog.Warn("failed to save raw PEP data", "path", rawPath, "error", err)
	}

	var response ftmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse PEP response: %w", err)
	}

	var lastID string
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO pep_entities
		(id, schema_name, name, country, birth_date, position, dataset, first_seen, last_seen, ingested_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, raw := range response.Entities {
		entity, parseErr := ParseFtMEntity(raw)
		if parseErr != nil {
			slog.Warn("skipping unparseable PEP entity", "error", parseErr)
			continue
		}
		lastID = entity.ID
		if _, execErr := stmt.Exec(entity.ID, entity.Schema, entity.Name, entity.Country, entity.BirthDate,
			entity.Position, entity.Dataset, entity.FirstSeen, entity.LastSeen); execErr != nil {
			return fmt.Errorf("insert entity %s: %w", entity.ID, execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("PEP ingestion complete", "entities", len(response.Entities), "raw_file", rawPath)

	if err := wm.Set(pepSourceType, time.Now(), lastID, fmt.Sprintf(`{"entities":%d}`, len(response.Entities))); err != nil {
		return fmt.Errorf("update watermark: %w", err)
	}

	return nil
}
