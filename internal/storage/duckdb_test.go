package storage

import "testing"

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
