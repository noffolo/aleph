package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	_ "github.com/marcboeker/go-duckdb"
)

type DonorLink struct {
	DonorName         string
	CandidateFullName string
	Party             string
	DonationAmount    float64
	DonationYear      int
	MatchType         string
}

// tokenizeName splits a normalized full name into cognome and nome.
// Assumes Italian convention: "Name Surname" where surname is the last token.
func tokenizeName(normalized string) (cognome, nome string) {
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return strings.ToUpper(parts[0]), ""
	}
	// NormalizeName output is Title Cased — store upcased for matching
	nome = strings.ToUpper(strings.Join(parts[:len(parts)-1], " "))
	cognome = strings.ToUpper(parts[len(parts)-1])
	return cognome, nome
}

func stripPunct(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\'' || r == '-' || r == ' ' {
			return r
		}
		if r < 'A' || r > 'Z' {
			return ' '
		}
		return r
	}, strings.ToUpper(s))
}

func matchTypeForNames(donorNome, candNome string) string {
	dn := strings.TrimSpace(donorNome)
	cn := strings.TrimSpace(candNome)
	if dn == "" {
		return "cognome"
	}
	if dn == cn {
		return "exact"
	}
	dnClean := strings.TrimSpace(strings.ReplaceAll(stripPunct(dn), "  ", " "))
	cnClean := strings.TrimSpace(strings.ReplaceAll(stripPunct(cn), "  ", " "))
	if dnClean != "" && cnClean != "" && (strings.Contains(cnClean, dnClean) || strings.Contains(dnClean, cnClean)) {
		return "fuzzy"
	}
	return ""
}

func CrossRefDonors(db *sql.DB) ([]DonorLink, error) {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS donor_candidate_links (
		donor_name VARCHAR,
		candidate_full_name VARCHAR,
		party VARCHAR,
		donation_amount REAL,
		donation_year INTEGER,
		match_type VARCHAR
	)`); err != nil {
		return nil, fmt.Errorf("create donor_candidate_links table: %w", err)
	}

	rows, err := db.Query(`SELECT donor_name, recipient_party, donation_amount, donation_year
	FROM party_funding
	WHERE donor_type IN ('Parlamentare o membro del Governo', 'Persona')`)
	if err != nil {
		return nil, fmt.Errorf("query party_funding: %w", err)
	}
	defer rows.Close()

	var links []DonorLink

	for rows.Next() {
		var donorName, party string
		var amount float64
		var year int

		if err := rows.Scan(&donorName, &party, &amount, &year); err != nil {
			return nil, fmt.Errorf("scan party_funding: %w", err)
		}

		party = strings.TrimSpace(strings.ToUpper(party))
		normalized := sources.NormalizeName(donorName)
		cognome, nome := tokenizeName(normalized)

		if cognome == "" {
			continue
		}

		candidateRows, err := db.Query(
			`SELECT full_name, cognome, nome, party FROM candidates_normalized
			 WHERE cognome = ? AND party = ?`, cognome, party)
		if err != nil {
			return nil, fmt.Errorf("query candidates_normalized: %w", err)
		}

		for candidateRows.Next() {
			var candFullName, candCognome, candNome, candParty string
			if err := candidateRows.Scan(&candFullName, &candCognome, &candNome, &candParty); err != nil {
				candidateRows.Close()
				return nil, fmt.Errorf("scan candidates_normalized: %w", err)
			}

			matchType := matchTypeForNames(nome, candNome)
			if matchType == "" {
				continue
			}

			link := DonorLink{
				DonorName:         donorName,
				CandidateFullName: candFullName,
				Party:             candParty,
				DonationAmount:    amount,
				DonationYear:      year,
				MatchType:         matchType,
			}
			links = append(links, link)

			if _, err := db.Exec(
				`INSERT INTO donor_candidate_links (donor_name, candidate_full_name, party, donation_amount, donation_year, match_type)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				link.DonorName, link.CandidateFullName, link.Party,
				link.DonationAmount, link.DonationYear, link.MatchType,
			); err != nil {
				candidateRows.Close()
				return nil, fmt.Errorf("insert donor_candidate_links: %w", err)
			}
		}
		candidateRows.Close()

		if err := candidateRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate candidates: %w", err)
		}
	}

	return links, rows.Err()
}

func runCLI() {
	dbPath := flag.String("db", "data/aleph.duckdb", "path to DuckDB database")
	flag.Parse()

	db, err := sql.Open("duckdb", *dbPath+"?access_mode=READ_WRITE")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	links, err := CrossRefDonors(db)
	if err != nil {
		log.Fatalf("crossref: %v", err)
	}

	fmt.Printf("Created %d donor-candidate links\n", len(links))

	counts := make(map[string]int)
	for _, l := range links {
		counts[l.MatchType]++
	}
	fmt.Printf("  exact: %d\n  fuzzy: %d\n", counts["exact"], counts["fuzzy"])
}

func main() {
	runCLI()
}
