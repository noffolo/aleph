package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"sort"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
)

func main() {
	dbPath := flag.String("db", "elections.duckdb", "Path to DuckDB database")
	rawDir := flag.String("raw", "./raw", "Directory for raw API dumps")
	years := flag.String("years", "", "Comma-separated years (default: all available)")
	types := flag.String("types", "", "Election types (default: all)")
	baseURL := flag.String("base-url", "https://eleapi.interno.gov.it/siel/PX", "Eligendo API base URL")
	flag.Parse()

	db, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=READ_WRITE", *dbPath))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	mapper := sources.NewPartyMapper()

	allTypes := flagTypes(*types)
	filterYears := flagYears(*years)

	for _, et := range allTypes {
		yearsForType, ok := sources.ElectionDateMap[et]
		if !ok {
			slog.Info("skipping unknown type", "type", et)
			continue
		}
		sorted := sortedKeys(yearsForType)
		for _, year := range sorted {
			if len(filterYears) > 0 && !contains(filterYears, year) {
				continue
			}

			dateStr := sources.GetElectionDate(et, year)
			if dateStr == "" {
				continue
			}

			cfg := sources.ElectionConfig{
				ElectionType: et,
				Level:        "comune",
				Year:         year,
				ElectionDate: dateStr,
			}

			slog.Info("processing election", "type", et, "year", year, "date", dateStr)
			results, err := sources.RunElection(context.Background(), db, *baseURL, cfg, mapper, *rawDir)
			if err != nil {
				slog.Error("election failed", "type", et, "year", year, "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			slog.Info("election complete", "type", et, "year", year, "results", len(results))
		}
	}
}

func flagTypes(s string) []string {
	var types []string
	for k := range sources.ElectionDateMap {
		types = append(types, k)
	}
	sort.Strings(types)

	if s == "" {
		return types
	}
	selected := splitAndTrim(s)
	var out []string
	for _, t := range selected {
		if _, ok := sources.ElectionDateMap[t]; ok {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		log.Fatalf("no valid election types specified (valid: %v)", strings.Join(types, ", "))
	}
	return out
}

// flagYears parses a comma-separated year list.
func flagYears(s string) []int {
	if s == "" {
		return nil
	}
	parts := splitAndTrim(s)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var y int
		if _, err := fmt.Sscanf(p, "%d", &y); err != nil {
			log.Fatalf("invalid year: %q", p)
		}
		out = append(out, y)
	}
	return out
}

func sortedKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func splitAndTrim(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
