package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const parliamentSourceType = "parliament"

type Vote struct {
	ID       string `json:"id"`
	Data     string `json:"data"`
	Titolo   string `json:"titolo"`
	Esito    string `json:"esito"`
	Deputato string `json:"deputato"`
	Gruppo   string `json:"gruppo"`
}

type parliamentWatermarkSetter interface {
	Set(sourceName string, lastRun time.Time, cursor string, metadata string) error
}

type sparqlBinding struct {
	Value string `json:"value"`
}

type sparqlResult struct {
	Votazione sparqlBinding `json:"votazione"`
	Data      sparqlBinding `json:"data"`
	Titolo    sparqlBinding `json:"titolo"`
	Esito     sparqlBinding `json:"esito"`
	Deputato  sparqlBinding `json:"deputato"`
	Gruppo    sparqlBinding `json:"gruppo"`
}

type sparqlResults struct {
	Bindings []sparqlResult `json:"bindings"`
}

type sparqlResponse struct {
	Head    interface{}   `json:"head"`
	Results sparqlResults `json:"results"`
}

func ParseSPARQLVotes(data []byte) ([]Vote, error) {
	var resp sparqlResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal sparql: %w", err)
	}
	votes := make([]Vote, 0, len(resp.Results.Bindings))
	for _, b := range resp.Results.Bindings {
		votes = append(votes, Vote{
			ID:       b.Votazione.Value,
			Data:     b.Data.Value,
			Titolo:   b.Titolo.Value,
			Esito:    b.Esito.Value,
			Deputato: b.Deputato.Value,
			Gruppo:   b.Gruppo.Value,
		})
	}
	return votes, nil
}

func BuildSPARQLQuery(legislatura string) (string, error) {
	if legislatura == "" {
		return "", fmt.Errorf("legislatura is required")
	}
	var legInt int
	if _, err := fmt.Sscanf(legislatura, "%d", &legInt); err != nil || legInt < 13 || legInt > 20 {
		return "", fmt.Errorf("invalid legislatura: %q (must be 13-20)", legislatura)
	}
	return fmt.Sprintf(`SELECT ?votazione ?data ?titolo ?esito ?deputato ?gruppo WHERE {
	  ?votazione a ocd:votazione ;
	             ocd:legislatura ocd:legislatura_%s ;
	             ocd:data ?data ;
	             ocd:titolo ?titolo .
	  OPTIONAL { ?votazione ocd:esito ?esito }
	  OPTIONAL { ?votazione ocd:votante [ ocd:deputato ?deputato ; ocd:gruppo ?gruppo ] }
	} LIMIT 1000`, legislatura), nil
}

func ensureParliamentTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parliament_votes (
		id TEXT,
		data TEXT,
		titolo TEXT,
		esito TEXT,
		deputato TEXT,
		gruppo TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func RunParliament(ctx context.Context, baseURL string, db *sql.DB, wm parliamentWatermarkSetter, rawDir string, legislatura string) error {
	slog.Info("starting parliament SPARQL ingestion", "endpoint", baseURL, "legislatura", legislatura)

	query, err := BuildSPARQLQuery(legislatura)
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	if err := ensureParliamentTable(db); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	rawPath := filepath.Join(rawDir, parliamentSourceType)
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}

	formData := url.Values{}
	formData.Set("query", query)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SPARQL endpoint returned %d: %s", resp.StatusCode, string(body[:min(len(body), 500)]))
	}

	rawFile := filepath.Join(rawPath, fmt.Sprintf("sparql_%s.json", legislatura))
	if err := os.WriteFile(rawFile, body, 0644); err != nil {
		slog.Warn("failed to save raw SPARQL response", "error", err)
	}

	votes, err := ParseSPARQLVotes(body)
	if err != nil {
		return fmt.Errorf("parse SPARQL: %w", err)
	}

	slog.Info("parsed SPARQL votes", "count", len(votes), "legislatura", legislatura)

	if len(votes) > 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO parliament_votes (id, data, titolo, esito, deputato, gruppo)
			 VALUES (?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepare stmt: %w", err)
		}

		for _, v := range votes {
			if _, err := stmt.ExecContext(ctx, v.ID, v.Data, v.Titolo, v.Esito, v.Deputato, v.Gruppo); err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("insert: %w", err)
			}
		}
		stmt.Close()

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	if err := wm.Set(parliamentSourceType, time.Now(), legislatura, fmt.Sprintf(`{"votes_loaded":%d}`, len(votes))); err != nil {
		return fmt.Errorf("update watermark: %w", err)
	}
	slog.Info("parliament ingestion complete", "votes", len(votes))
	return nil
}
