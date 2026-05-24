package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

const defaultCSVURL = "https://raw.githubusercontent.com/ondata/liberiamoli-tutti/main/soldi_e_politica/dati/political_finance.csv"

type simpleWatermark struct{}

func (s *simpleWatermark) Set(sourceName string, _ time.Time, _ string, _ string) error {
	slog.Info("watermark set", "source", sourceName)
	return nil
}

func main() {
	dbPath := flag.String("db", "funding.duckdb", "Path to DuckDB database")
	rawDir := flag.String("raw", "./raw", "Directory for raw CSV storage")
	csvURL := flag.String("csv-url", defaultCSVURL, "URL of the party funding CSV")
	csvPath := flag.String("csv-path", "", "Local path to CSV (alternative to download)")
	flag.Parse()

	db, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=READ_WRITE", *dbPath))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var localPath string
	if *csvPath != "" {
		localPath = *csvPath
		slog.Info("using local CSV", "path", localPath)
	} else {
		localPath = filepath.Join(*rawDir, "party_funding", "political_finance.csv")
		os.MkdirAll(filepath.Dir(localPath), 0755)

		slog.Info("downloading CSV", "url", *csvURL)
		resp, err := http.Get(*csvURL)
		if err != nil {
			log.Fatalf("download CSV: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Fatalf("HTTP %d downloading CSV", resp.StatusCode)
		}

		f, err := os.Create(localPath)
		if err != nil {
			log.Fatalf("create file: %v", err)
		}

		written, err := io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			log.Fatalf("write CSV: %v", err)
		}
		slog.Info("downloaded CSV", "bytes", written)
	}

	wm := &simpleWatermark{}
	if err := sources.ImportFundingCSV(context.Background(), db, wm, localPath, *rawDir); err != nil {
		log.Fatalf("import CSV: %v", err)
	}
	slog.Info("funding import complete")
}
