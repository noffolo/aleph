package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/ff3300/aleph-v2/internal/political"
)

func main() {
	dbPath := flag.String("db", "data/aleph.duckdb", "DuckDB database path")
	outDir := flag.String("out", "./analysis_data", "Output directory for JSON exports")
	flag.Parse()

	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping: %v", err)
	}

	os.MkdirAll(*outDir, 0755)

	log.Println("Fetching articles from rss_corriere...")
	rows, err := db.Query(`SELECT title, description, pub_date, link, category, source
		FROM rss_corriere
		WHERE title IS NOT NULL AND title != ''
		AND pub_date IS NOT NULL AND pub_date != ''
		ORDER BY pub_date DESC`)
	if err != nil {
		log.Fatalf("query rss_corriere: %v", err)
	}
	defer rows.Close()

	extractor := political.NewEntityExtractor(political.DefaultPartyKeywords)

	type article struct {
		title       string
		description string
		pubDate     string
		link        string
		category    string
		source      string
	}

	var articles []article
	for rows.Next() {
		var a article
		var desc, category, source sql.NullString
		if err := rows.Scan(&a.title, &desc, &a.pubDate, &a.link, &category, &source); err != nil {
			log.Printf("WARN: scan row: %v", err)
			continue
		}
		a.description = desc.String
		a.category = category.String
		a.source = source.String
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows iteration: %v", err)
	}
	log.Printf("Loaded %d articles", len(articles))

	var results []political.SentimentResult
	seen := make(map[string]bool)

	for _, a := range articles {
		mentions := extractor.ExtractEntities(a.title, a.description)
		if len(mentions) == 0 {
			continue
		}

		combinedText := strings.ToLower(a.title + " " + a.description)
		for _, m := range mentions {
			score := political.ScoreSentiment(combinedText)
			coalition := political.CoalitionFor(m.Party)
			dateStr := formatDate(a.pubDate)

			key := dateStr + "|" + m.Party
			if seen[key] {
				continue
			}
			seen[key] = true

			results = append(results, political.SentimentResult{
				ArticleID: a.link,
				Title:     a.title,
				Date:      dateStr,
				Party:     m.Party,
				Score:     score,
				Coalition: coalition,
				Category:  a.category,
				Source:    a.source,
			})
		}
	}

	log.Printf("Found %d party mentions across %d articles", len(results), len(articles))

	if len(results) == 0 {
		log.Println("No party mentions found. Exiting.")
		os.Exit(0)
	}

	log.Println("Storing sentiment results...")
	if err := political.StoreSentiment(db, results); err != nil {
		log.Fatalf("store sentiment: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM political_sentiment").Scan(&count)
	log.Printf("Stored %d rows in political_sentiment", count)

	log.Println("Exporting weekly analytics...")
	if err := political.ExportWeeklySentiment(db, *outDir); err != nil {
		log.Fatalf("export: %v", err)
	}

	log.Printf("Analysis complete. JSON files written to %s", *outDir)

	// Print summary
	csxCount := 0
	cdxCount := 0
	otherCount := 0
	for _, r := range results {
		switch r.Coalition {
		case "CSX":
			csxCount++
		case "CDX":
			cdxCount++
		default:
			otherCount++
		}
	}
	log.Printf("Summary: CSX=%d CDX=%d Other=%d", csxCount, cdxCount, otherCount)
}

var dateFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func formatDate(raw string) string {
	for _, f := range dateFormats {
		t, err := time.Parse(f, strings.TrimSpace(raw))
		if err == nil {
			return t.Format("2006-01-02")
		}
	}
	return raw
}
