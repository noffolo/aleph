package main

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS party_funding (
		donor_name VARCHAR,
		recipient_party VARCHAR,
		donation_amount REAL,
		donation_year INTEGER,
		donor_type VARCHAR
	)`)
	if err != nil {
		t.Fatalf("failed to create party_funding: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS candidates_normalized (
		full_name VARCHAR,
		cognome VARCHAR,
		nome VARCHAR,
		raw_party VARCHAR,
		party VARCHAR
	)`)
	if err != nil {
		t.Fatalf("failed to create candidates_normalized: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS donor_candidate_links (
		donor_name VARCHAR,
		candidate_full_name VARCHAR,
		party VARCHAR,
		donation_amount REAL,
		donation_year INTEGER,
		match_type VARCHAR
	)`)
	if err != nil {
		t.Fatalf("failed to create donor_candidate_links: %v", err)
	}

	return db
}

func TestCrossRefDonorsExactMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO party_funding VALUES
		('Mario Rossi', 'PARTITO A', 1000.0, 2023, 'Persona Fisica'),
		('ROSSI MARIO', 'PARTITO A', 500.0, 2024, 'Persona Fisica')`)
	if err != nil {
		t.Fatalf("failed to insert party_funding: %v", err)
	}

	_, err = db.Exec(`INSERT INTO candidates_normalized VALUES
		('Mario Rossi', 'ROSSI', 'MARIO', 'partito a', 'PARTITO A'),
		('Luigi Bianchi', 'BIANCHI', 'LUIGI', 'partito b', 'PARTITO B')`)
	if err != nil {
		t.Fatalf("failed to insert candidates_normalized: %v", err)
	}

	links, err := CrossRefDonors(db)
	if err != nil {
		t.Fatalf("CrossRefDonors failed: %v", err)
	}

	if len(links) == 0 {
		t.Fatal("expected at least one match, got zero")
	}

	foundMario := false
	for _, l := range links {
		if l.DonorName == "Mario Rossi" && l.CandidateFullName == "Mario Rossi" {
			foundMario = true
			if l.MatchType != "exact" {
				t.Errorf("expected exact match_type, got %q", l.MatchType)
			}
		}
	}
	if !foundMario {
		t.Error("expected Mario Rossi -> Mario Rossi link, not found")
	}
}

func TestCrossRefDonorsSkipsPersonaGiuridica(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO party_funding VALUES
		('Mario Rossi', 'PARTITO A', 1000.0, 2023, 'Persona Giuridica'),
		('Luigi Bianchi', 'PARTITO B', 500.0, 2023, 'Persona Fisica')`)
	if err != nil {
		t.Fatalf("failed to insert party_funding: %v", err)
	}

	_, err = db.Exec(`INSERT INTO candidates_normalized VALUES
		('Mario Rossi', 'ROSSI', 'MARIO', 'partito a', 'PARTITO A'),
		('Luigi Bianchi', 'BIANCHI', 'LUIGI', 'partito b', 'PARTITO B')`)
	if err != nil {
		t.Fatalf("failed to insert candidates_normalized: %v", err)
	}

	links, err := CrossRefDonors(db)
	if err != nil {
		t.Fatalf("CrossRefDonors failed: %v", err)
	}

	for _, l := range links {
		if l.DonorName == "Mario Rossi" {
			t.Errorf("Persona Giuridica donor %q should not be matched", l.DonorName)
		}
	}
}

func TestCrossRefDonorsFuzzyMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO party_funding VALUES
		('M. Bianchi', 'PARTITO B', 750.0, 2024, 'Persona Fisica')`)
	if err != nil {
		t.Fatalf("failed to insert party_funding: %v", err)
	}

	_, err = db.Exec(`INSERT INTO candidates_normalized VALUES
		('Mario Bianchi', 'BIANCHI', 'MARIO', 'partito b', 'PARTITO B')`)
	if err != nil {
		t.Fatalf("failed to insert candidates_normalized: %v", err)
	}

	links, err := CrossRefDonors(db)
	if err != nil {
		t.Fatalf("CrossRefDonors failed: %v", err)
	}

	foundFuzzy := false
	for _, l := range links {
		if l.DonorName == "M. Bianchi" && l.CandidateFullName == "Mario Bianchi" {
			foundFuzzy = true
			if l.MatchType != "fuzzy" {
				t.Errorf("expected fuzzy match_type, got %q", l.MatchType)
			}
		}
	}
	if !foundFuzzy {
		t.Error("expected fuzzy match M. Bianchi -> Mario Bianchi, not found")
	}
}
