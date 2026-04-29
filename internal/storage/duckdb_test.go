package storage

import (
	"testing"
)

func TestNewDuckDB_InMemory(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE test (id INTEGER, name VARCHAR)")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO test VALUES (1, 'hello')")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("SELECT name FROM test WHERE id = ?", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var name string
	rows.Scan(&name)
	if name != "hello" {
		t.Errorf("expected hello, got %s", name)
	}
}

func TestNewDuckDB_VSSFlag(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_ = db.HasVSS
}

func TestDuckDB_VSSVectorSearch(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if !db.HasVSS {
		t.Skip("VSS extension not available")
	}

	_, err = db.Exec(`CREATE TABLE embeddings (
		id INTEGER PRIMARY KEY,
		vec FLOAT[3]
	)`)
	if err != nil {
		t.Fatalf("create embeddings table: %v", err)
	}

	_, err = db.Exec(`INSERT INTO embeddings VALUES
		(1, [0.1, 0.2, 0.3]),
		(2, [0.4, 0.5, 0.6]),
		(3, [0.9, 0.8, 0.7])`)
	if err != nil {
		t.Fatalf("insert embeddings: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_vec ON embeddings USING HNSW (vec) WITH (metric = 'cosine')`)
	if err != nil {
		t.Fatalf("create VSS index: %v", err)
	}

	rows, err := db.Query(`SELECT id, array_cosine_similarity(vec, [0.1, 0.2, 0.3]::FLOAT[3]) AS score
		FROM embeddings
		ORDER BY score DESC
		LIMIT 3`)
	if err != nil {
		t.Fatalf("vector search query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 results, got %d", count)
	}
}
