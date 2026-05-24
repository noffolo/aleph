package manifest

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func openMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func execSchema(t *testing.T, db *sql.DB, stmts ...string) {
	t.Helper()
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

func TestScannerBasic(t *testing.T) {
	db := openMemoryDB(t)
	execSchema(t, db,
		`CREATE TABLE candidates (codice VARCHAR PRIMARY KEY, cognome VARCHAR, nome VARCHAR)`,
		`CREATE TABLE parties (id INTEGER PRIMARY KEY, name VARCHAR, founded INTEGER)`,
		`CREATE TABLE donations (donor VARCHAR, amount FLOAT, party VARCHAR)`,
	)

	scanner := NewScanner(DefaultDomainConfig())
	ctx := context.Background()

	tables, err := scanner.Scan(ctx, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(tables) != 3 {
		t.Fatalf("expected 3 tables, got %d", len(tables))
	}

	tableNames := map[string]bool{}
	for _, tbl := range tables {
		tableNames[tbl.Name] = true
	}
	for _, want := range []string{"candidates", "parties", "donations"} {
		if !tableNames[want] {
			t.Errorf("missing table %q", want)
		}
	}

	candidates := findTable(tables, "candidates")
	if candidates.RowCount != 0 {
		t.Errorf("candidates.RowCount = %d, want 0", candidates.RowCount)
	}
	if !colHasPK(candidates, "codice") {
		t.Error("codice should be PK (explicit PRIMARY KEY)")
	}
}

func TestScannerPKDetection(t *testing.T) {
	db := openMemoryDB(t)
	execSchema(t, db,
		`CREATE TABLE election_results (
			election_type VARCHAR,
			level VARCHAR,
			year INTEGER,
			comune_istat VARCHAR,
			lista VARCHAR,
			voti INTEGER,
			UNIQUE(election_type, level, year, comune_istat, lista)
		)`,
		`INSERT INTO election_results VALUES
			('europee','comunale',2024,'001','Partito A', 1000),
			('europee','comunale',2024,'001','Partito B', 800)`,
		`CREATE TABLE with_pk (id INTEGER PRIMARY KEY, name VARCHAR)`,
		`INSERT INTO with_pk VALUES (1, 'test')`,
	)

	scanner := NewScanner(DefaultDomainConfig())
	ctx := context.Background()

	tables, err := scanner.Scan(ctx, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	er := findTable(tables, "election_results")
	for _, name := range []string{"election_type", "level", "year", "comune_istat", "lista"} {
		if !colHasPK(er, name) {
			t.Errorf("column %q should be detected as PK (part of UNIQUE constraint)", name)
		}
	}
	if colHasPK(er, "voti") {
		t.Error("voti should NOT be PK")
	}

	wp := findTable(tables, "with_pk")
	if !colHasPK(wp, "id") {
		t.Error("id should be PK (explicit PRIMARY KEY)")
	}
}

func TestFullPipeline(t *testing.T) {
	db := openMemoryDB(t)
	execSchema(t, db,
		`CREATE TABLE party (
			name VARCHAR PRIMARY KEY,
			founded INTEGER
		)`,
		`INSERT INTO party VALUES ('Partito A',1946),('Partito B',2009)`,
		`CREATE TABLE election_results (
			election_type VARCHAR,
			level VARCHAR,
			year INTEGER,
			comune VARCHAR,
			comune_istat VARCHAR,
			lista VARCHAR,
			party_canonical VARCHAR,
			voti INTEGER,
			percentuale FLOAT,
			seggi INTEGER,
			elettori INTEGER,
			votanti INTEGER,
			UNIQUE(election_type, level, year, comune_istat, lista)
		)`,
		`INSERT INTO election_results VALUES
			('europee','comunale',2024,'Roma','RM001','Partito A','Partito A',5000,35.2,3,15000,10000),
			('europee','comunale',2024,'Roma','RM001','Partito B','Partito B',8000,56.3,5,15000,10000)`,
		`CREATE TABLE candidates (
			codice VARCHAR PRIMARY KEY,
			cognome VARCHAR,
			nome VARCHAR,
			full_name VARCHAR,
			party VARCHAR
		)`,
		`INSERT INTO candidates VALUES ('C001','Rossi','Mario','Mario Rossi','Partito A')`,
		`CREATE TABLE party_funding (
			declaration_id VARCHAR PRIMARY KEY,
			donation_amount FLOAT,
			donation_year INTEGER,
			recipient_party VARCHAR,
			donor_name VARCHAR
		)`,
		`INSERT INTO party_funding VALUES ('D001',5000.0,2024,'Partito A','Donor X')`,
	)

	cfg := DefaultDomainConfig()
	scanner := NewScanner(cfg)
	classifier := NewClassifier(cfg)
	inferrer := NewEntityInferrer(cfg)
	discoverer := NewRelationDiscoverer(cfg)
	suggester := NewMetricSuggester(cfg)
	builder := NewGraphManifestBuilder()

	ctx := context.Background()
	tables, err := scanner.Scan(ctx, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(tables) != 4 {
		t.Fatalf("expected 4 tables, got %d", len(tables))
	}

	tables, err = classifier.Classify(tables)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	er := findTable(tables, "election_results")
	for _, name := range []string{"election_type", "level", "year", "comune_istat", "lista"} {
		if !colHasPK(er, name) {
			t.Errorf("election_results.%s should be PK (UNIQUE constraint)", name)
		}
		if colClass(er, name) != PrimaryKey {
			t.Errorf("election_results.%s class = %v, want PrimaryKey", name, colClass(er, name))
		}
	}

	entities, err := inferrer.Infer(tables)
	if err != nil {
		t.Fatalf("Infer: %v", err)
	}
	if len(entities) < 2 {
		t.Fatalf("expected at least 2 entities, got %d", len(entities))
	}

	relations, err := discoverer.Discover(entities, tables, db)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	metrics, err := suggester.Suggest(entities, tables)
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}

	graph := builder.Build(entities, relations, metrics)
	if len(graph.Entities) == 0 {
		t.Error("graph has no entities")
	}
}

func TestClassifierTypes(t *testing.T) {
	db := openMemoryDB(t)
	execSchema(t, db,
		`CREATE TABLE type_test (
			id VARCHAR PRIMARY KEY,
			category_col VARCHAR,
			amount FLOAT,
			event_date DATE,
			label_col VARCHAR
		)`,
		`INSERT INTO type_test VALUES ('001','A',100.0,'2024-06-15','Description text here')`,
		`INSERT INTO type_test VALUES ('002','B',200.0,'2024-06-16','Another description')`,
		`INSERT INTO type_test VALUES ('003','A',300.0,'2024-06-17','Third description')`,
	)

	cfg := DefaultDomainConfig()
	scanner := NewScanner(cfg)
	ctx := context.Background()

	tables, err := scanner.Scan(ctx, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	classifier := NewClassifier(cfg)
	tables, err = classifier.Classify(tables)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	tt := findTable(tables, "type_test")

	// label_col has 3 distinct values ≤ MaxCategoryDistinct(20), so it is
	// classified as Category (rule h) before the Label check (rule i).
	want := map[string]ColumnClass{
		"id":           PrimaryKey,
		"category_col": Category,
		"amount":       Measure,
		"event_date":   Temporal,
		"label_col":    Category,
	}
	for name, expectedClass := range want {
		got := colClass(tt, name)
		if got != expectedClass {
			t.Errorf("%s: got class %v, want %v", name, got, expectedClass)
		}
	}
}

func TestRelationDiscovery(t *testing.T) {
	db := openMemoryDB(t)
	execSchema(t, db,
		`CREATE TABLE party (
			id INTEGER PRIMARY KEY,
			name VARCHAR
		)`,
		`INSERT INTO party VALUES (1, 'Partito A'), (2, 'Partito B')`,
		`CREATE TABLE candidates (
			id INTEGER PRIMARY KEY,
			name VARCHAR,
			party_id INTEGER
		)`,
		`INSERT INTO candidates VALUES (1, 'Cand 1', 1), (2, 'Cand 2', 2)`,
	)

	cfg := DefaultDomainConfig()
	scanner := NewScanner(cfg)
	ctx := context.Background()

	tables, err := scanner.Scan(ctx, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	classifier := NewClassifier(cfg)
	tables, err = classifier.Classify(tables)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	inferrer := NewEntityInferrer(cfg)
	entities, err := inferrer.Infer(tables)
	if err != nil {
		t.Fatalf("Infer: %v", err)
	}
	if len(entities) < 2 {
		t.Fatalf("expected at least 2 entities, got %d", len(entities))
	}

	discoverer := NewRelationDiscoverer(cfg)
	relations, err := discoverer.Discover(entities, tables, db)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	found := false
	for _, rel := range relations {
		if rel.ViaColumn == "party_id" {
			found = true
			if rel.Confidence != 1.0 {
				t.Errorf("FK relation confidence = %f, want 1.0", rel.Confidence)
			}
		}
	}
	if !found {
		t.Error("party_id FK relation not discovered")
	}
}

func findTable(tables []TableSchema, name string) TableSchema {
	for _, tbl := range tables {
		if tbl.Name == name {
			return tbl
		}
	}
	return TableSchema{}
}

func colHasPK(tbl TableSchema, name string) bool {
	for _, col := range tbl.Columns {
		if col.Name == name {
			return col.IsPK
		}
	}
	return false
}

func colClass(tbl TableSchema, name string) ColumnClass {
	for _, col := range tbl.Columns {
		if col.Name == name {
			return col.Class
		}
	}
	return -1
}
