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

const opdmSourceType = "opdm"

const opdmMaxRate = 10000.0 / 86400.0

type opdmItem struct {
	PersonID   int    `json:"person_id"`
	OrgID      int    `json:"org_id"`
	Role       string `json:"role"`
	PersonName string `json:"person_name"`
	StartDate  string `json:"start_date,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
}

type opdmResponse struct {
	Data []opdmItem `json:"data"`
	Next string     `json:"next"`
}

type opdmRateLimiter struct {
	tokenCh chan struct{}
}

func newOPDMRateLimiter(burst int) *opdmRateLimiter {
	if burst <= 0 {
		burst = 5
	}
	rl := &opdmRateLimiter{
		tokenCh: make(chan struct{}, burst),
	}
	for i := 0; i < burst; i++ {
		rl.tokenCh <- struct{}{}
	}
	go func() {
		interval := time.Duration(86400.0 / 10000.0 * float64(time.Second))
		if interval < time.Second {
			interval = time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rl.tokenCh <- struct{}{}:
			default:
			}
		}
	}()
	return rl
}

func (r *opdmRateLimiter) Wait() {
	<-r.tokenCh
}

type opdmWatermarkSetter interface {
	Set(sourceName string, lastRun time.Time, cursor string, metadata string) error
}

func ensureOPDMTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS opdm_memberships (
		person_id INTEGER,
		org_id INTEGER,
		role TEXT,
		person_name TEXT,
		start_date TEXT DEFAULT '',
		end_date TEXT DEFAULT '',
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (person_id, org_id, role)
	)`)
	return err
}

func RunOPDM(ctx context.Context, baseURL string, db *sql.DB, wm opdmWatermarkSetter, apiKey string, rawDir string) error {
	slog.Info("starting OPDM ingestion", "base_url", baseURL)

	if err := ensureOPDMTable(db); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	rl := newOPDMRateLimiter(5)
	rawPath := filepath.Join(rawDir, opdmSourceType)
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	page := 0
	nextCursor := baseURL
	lastCursor := baseURL

	for {
		rl.Wait()

		req, err := http.NewRequestWithContext(ctx, "GET", nextCursor, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http get: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("opdm API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 500)]))
		}

		page++
		rawFile := filepath.Join(rawPath, fmt.Sprintf("page_%d.json", page))
		if err := os.WriteFile(rawFile, body, 0644); err != nil {
			slog.Warn("failed to save raw response", "page", page, "error", err)
		}

		var parsed opdmResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return fmt.Errorf("decode page %d: %w", page, err)
		}

		if len(parsed.Data) > 0 {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("begin tx: %w", err)
			}

			stmt, err := tx.PrepareContext(ctx,
				`INSERT OR REPLACE INTO opdm_memberships (person_id, org_id, role, person_name, start_date, end_date)
				 VALUES (?, ?, ?, ?, ?, ?)`,
			)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("prepare stmt: %w", err)
			}

			for _, item := range parsed.Data {
				if _, err := stmt.ExecContext(ctx, item.PersonID, item.OrgID, item.Role, item.PersonName, item.StartDate, item.EndDate); err != nil {
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

		if parsed.Next == "" {
			if err := wm.Set(opdmSourceType, time.Now(), lastCursor, fmt.Sprintf(`{"pages_fetched":%d}`, page)); err != nil {
				return fmt.Errorf("update watermark: %w", err)
			}
			break
		}

		lastCursor = nextCursor
		nextCursor = parsed.Next
	}

	slog.Info("OPDM ingestion complete", "pages", page, "source", opdmSourceType)
	return nil
}
