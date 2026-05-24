package ingestion

import (
	"database/sql"
	"errors"
	"time"
)

var ErrWatermarkNotFound = errors.New("watermark not found")

type Watermark struct {
	SourceName string
	LastRun    time.Time
	Cursor     string
	Metadata   string
}

type WatermarkManager struct{ db *sql.DB }

func NewWatermarkManager(db *sql.DB) *WatermarkManager {
	return &WatermarkManager{db: db}
}

func (w *WatermarkManager) ensureTable() error {
	_, err := w.db.Exec(`CREATE TABLE IF NOT EXISTS ingestion_watermark (
		source_name TEXT PRIMARY KEY,
		last_run TIMESTAMP NOT NULL,
		cursor TEXT DEFAULT '',
		metadata TEXT DEFAULT ''
	)`)
	return err
}

func (w *WatermarkManager) Get(sourceName string) (Watermark, error) {
	if err := w.ensureTable(); err != nil {
		return Watermark{}, err
	}
	var wm Watermark
	row := w.db.QueryRow(
		"SELECT source_name, last_run, COALESCE(cursor,''), COALESCE(metadata,'') FROM ingestion_watermark WHERE source_name = ?",
		sourceName,
	)
	err := row.Scan(&wm.SourceName, &wm.LastRun, &wm.Cursor, &wm.Metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return Watermark{}, ErrWatermarkNotFound
	}
	return wm, err
}

func (w *WatermarkManager) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	if err := w.ensureTable(); err != nil {
		return err
	}
	_, err := w.db.Exec(
		`INSERT OR REPLACE INTO ingestion_watermark (source_name, last_run, cursor, metadata) VALUES (?, ?, ?, ?)`,
		sourceName, lastRun, cursor, metadata,
	)
	return err
}

func (w *WatermarkManager) ListAll() ([]Watermark, error) {
	if err := w.ensureTable(); err != nil {
		return nil, err
	}
	rows, err := w.db.Query(
		"SELECT source_name, last_run, COALESCE(cursor,''), COALESCE(metadata,'') FROM ingestion_watermark ORDER BY last_run DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Watermark
	for rows.Next() {
		var wm Watermark
		if err := rows.Scan(&wm.SourceName, &wm.LastRun, &wm.Cursor, &wm.Metadata); err != nil {
			return nil, err
		}
		result = append(result, wm)
	}
	return result, rows.Err()
}
