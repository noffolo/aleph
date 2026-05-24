package sources

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const fundingSourceType = "party_funding"

func ImportFundingCSV(ctx context.Context, db *sql.DB, wm WatermarkSetter, csvPath string, rawDir string) error {
	slog.Info("importing party funding CSV", "path", csvPath)

	rawBytes, err := os.ReadFile(csvPath)
	if err != nil {
		return fmt.Errorf("read funding CSV: %w", err)
	}
	rawSavePath := filepath.Join(rawDir, fundingSourceType, "political_finance.csv")
	if err := os.MkdirAll(filepath.Dir(rawSavePath), 0755); err != nil {
		return fmt.Errorf("create raw dir: %w", err)
	}
	if err := os.WriteFile(rawSavePath, rawBytes, 0644); err != nil {
		slog.Warn("failed to save raw funding CSV", "error", err)
	}

	if _, err = db.ExecContext(ctx, `CREATE OR REPLACE TABLE party_funding (
		declaration_id TEXT,
		donation_amount REAL,
		donation_year INTEGER,
		recipient_party TEXT,
		donor_type TEXT,
		donor_name TEXT,
		source_name TEXT,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create party_funding table: %w", err)
	}

	query := fmt.Sprintf(`INSERT INTO party_funding (declaration_id, donation_amount, donation_year, recipient_party, donor_type, donor_name, source_name)
		SELECT declaration_id,
		       CAST(donation_amount AS REAL),
		       CAST(donation_year AS INTEGER),
		       recipient_party,
		       donor_type,
		       COALESCE(donor_name_01, donor_name_02, ''),
		       source_name
		FROM read_csv_auto('%s', header=true, all_varchar=true)`, csvPath)
	if _, err = db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("import funding CSV into DuckDB: %w", err)
	}

	if err := wm.Set(fundingSourceType, time.Now(), "", ""); err != nil {
		return fmt.Errorf("update watermark: %w", err)
	}

	slog.Info("party funding import complete", "path", csvPath)
	return nil
}
