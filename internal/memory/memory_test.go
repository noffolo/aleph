package memory

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

// openDuckDB opens a DuckDB database at the given path (use ":memory:" for in-memory).
func openDuckDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}
	// In-memory DuckDB databases are per-connection; limit to single connection
	if dbPath == "" || dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}
	return db, nil
}

// inMemoryDB opens an in-memory DuckDB connection suitable for testing.
// Uses the same driver as the rest of the project (go-duckdb).
func inMemoryDB(t *testing.T) *MemoryStore {
	t.Helper()
	return inMemoryDBDim(t, 4)
}

func inMemoryDBDim(t *testing.T, dim int) *MemoryStore {
	t.Helper()
	db, ms, err := newTestStore("", dim)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return ms
}

func newTestStore(schema string, dim int) (interface{ Close() error }, *MemoryStore, error) {
	// Use database/sql directly with the duckdb driver name.
	// We open the driver by its registered name.
	return newTestStoreSQL(schema, dim)
}

func newTestStoreSQL(schema string, dim int) (interface{ Close() error }, *MemoryStore, error) {
	db, err := openDuckDB(":memory:")
	if err != nil {
		return nil, nil, err
	}
	ms, err := NewMemoryStore(db, schema, dim)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	return db, ms, nil
}

func TestStoreAndGet(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	err := ms.Store(ctx, "key1", []byte("hello world"), embed)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	val, ok := ms.Get(ctx, "key1")
	if !ok {
		t.Fatal("Get: expected key1 to exist")
	}
	if string(val) != "hello world" {
		t.Errorf("Get: expected 'hello world', got %q", string(val))
	}

	// Non-existent key
	_, ok = ms.Get(ctx, "nonexistent")
	if ok {
		t.Fatal("Get: expected nonexistent key to return false")
	}
}

func TestStoreReplace(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	_ = ms.Store(ctx, "replace_me", []byte("original"), embed)
	_ = ms.Store(ctx, "replace_me", []byte("updated"), embed)

	val, ok := ms.Get(ctx, "replace_me")
	if !ok {
		t.Fatal("Get after replace: expected key to exist")
	}
	if string(val) != "updated" {
		t.Errorf("Get after replace: expected 'updated', got %q", string(val))
	}
}

func TestSearchVector(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	entries := []struct {
		key   string
		value string
		emb   []float32
	}{
		{"a", "apple", []float32{1.0, 0.0, 0.0, 0.0}},
		{"b", "banana", []float32{0.0, 1.0, 0.0, 0.0}},
		{"c", "cherry", []float32{0.0, 0.0, 1.0, 0.0}},
	}
	for _, e := range entries {
		if err := ms.Store(ctx, e.key, []byte(e.value), e.emb); err != nil {
			t.Fatalf("Store %s: %v", e.key, err)
		}
	}

	// Search for the most similar to [1.0, 0.0, 0.0, 0.0] — should be "a" first
	results, err := ms.Search(ctx, []float32{1.0, 0.0, 0.0, 0.0}, 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Search: expected 3 results, got %d", len(results))
	}
	if results[0].Key != "a" {
		t.Errorf("Search: expected first result key 'a', got %q", results[0].Key)
	}
	if results[0].Score <= 0 {
		t.Errorf("Search: expected positive score, got %f", results[0].Score)
	}
}

func TestSearchText(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	_ = ms.Store(ctx, "alpha", []byte("this is some data"), embed)
	_ = ms.Store(ctx, "beta", []byte("completely different"), embed)
	_ = ms.Store(ctx, "gamma", []byte("some other stuff"), embed)

	// Search for "some" in value — matches alpha and gamma
	results, err := ms.SearchText(ctx, "some", 10)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("SearchText: expected 2 results, got %d: %+v", len(results), results)
	}

	// Search for "alpha" in key — matches alpha
	results, err = ms.SearchText(ctx, "alpha", 10)
	if err != nil {
		t.Fatalf("SearchText key: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("SearchText key: expected 1 result, got %d", len(results))
	}
	if string(results[0].Value) != "this is some data" {
		t.Errorf("SearchText key: wrong value %q", string(results[0].Value))
	}

	// No match
	results, err = ms.SearchText(ctx, "zzzznotfound", 10)
	if err != nil {
		t.Fatalf("SearchText no match: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchText no match: expected 0 results, got %d", len(results))
	}
}

func TestDelete(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	_ = ms.Store(ctx, "todelete", []byte("delete me"), embed)

	// Verify exists
	_, ok := ms.Get(ctx, "todelete")
	if !ok {
		t.Fatal("Delete setup: key should exist")
	}

	// Delete
	if err := ms.Delete(ctx, "todelete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify gone
	_, ok = ms.Get(ctx, "todelete")
	if ok {
		t.Fatal("Delete: key should be gone")
	}

	// Delete non-existent should not error
	if err := ms.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete nonexistent: %v", err)
	}
}

func TestList(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i)) // "a", "b", "c", "d", "e"
		_ = ms.Store(ctx, key, []byte("value_"+key), embed)
	}

	// List with limit
	entries, err := ms.List(ctx, 3, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("List: expected 3 entries, got %d", len(entries))
	}
	if entries[0].Key != "a" || entries[1].Key != "b" || entries[2].Key != "c" {
		t.Errorf("List: expected keys a,b,c in order, got %+v", entries)
	}

	// List with offset
	entries, err = ms.List(ctx, 3, 3)
	if err != nil {
		t.Fatalf("List offset: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List offset: expected 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "d" || entries[1].Key != "e" {
		t.Errorf("List offset: expected keys d,e, got %+v", entries)
	}

	// List with empty results
	entries, err = ms.List(ctx, 10, 100)
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List empty: expected 0, got %d", len(entries))
	}
}

func TestNewMemoryStore_NilDB(t *testing.T) {
	_, err := NewMemoryStore(nil, "", 4)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewMemoryStore_ZeroDim(t *testing.T) {
	db, err := openDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = NewMemoryStore(db, "", 0)
	if err == nil {
		t.Fatal("expected error for zero dim")
	}
}

func TestStoreRoundTrip_EmbeddingPreserved(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	embed := []float32{0.5, 0.6, 0.7, 0.8}
	_ = ms.Store(ctx, "roundtrip", []byte("test"), embed)

	entries, err := ms.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List: expected 1 entry, got %d", len(entries))
	}

	if len(entries[0].Embedding) != 4 {
		t.Fatalf("List: expected embedding len 4, got %d", len(entries[0].Embedding))
	}
	for i := range embed {
		if entries[0].Embedding[i] != embed[i] {
			t.Errorf("List embedding[%d]: expected %f, got %f", i, embed[i], entries[0].Embedding[i])
		}
	}
}
