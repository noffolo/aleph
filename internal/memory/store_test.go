package memory

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewMemoryStore(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	if s == nil {
		t.Fatal("store is nil")
	}
}

func TestMemoryStore_InsertAndSearch(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	err = s.Insert(ctx, "mem1", "test-ns", "hello world", "chat", `{"key":"val"}`, []float32{1, 0, 0, 0})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	err = s.Insert(ctx, "mem2", "test-ns", "goodbye world", "chat", "{}", []float32{0, 1, 0, 0})
	if err != nil {
		t.Fatalf("Insert mem2: %v", err)
	}

	results, err := s.Search(ctx, "test-ns", []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].ID != "mem1" {
		t.Errorf("top result = %q, want %q", results[0].ID, "mem1")
	}
	if results[0].Score < 0.9 {
		t.Errorf("score too low: %f", results[0].Score)
	}
}

func TestMemoryStore_SearchEmptyNamespace(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	results, err := s.Search(t.Context(), "nonexistent", []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	s.Insert(ctx, "del1", "ns1", "delete me", "chat", "{}", []float32{1, 0, 0, 0})
	err = s.Delete(ctx, "del1", "ns1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, _ := s.Search(ctx, "ns1", []float32{1, 0, 0, 0}, 5)
	if len(results) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(results))
	}
}

func TestMemoryStore_DeleteNamespace(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	s.Insert(ctx, "a", "ns-del", "a", "chat", "{}", []float32{1, 0, 0, 0})
	s.Insert(ctx, "b", "ns-del", "b", "chat", "{}", []float32{0, 1, 0, 0})
	err = s.DeleteNamespace(ctx, "ns-del")
	if err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	c, err := s.Count(ctx, "ns-del")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if c != 0 {
		t.Errorf("expected 0 after DeleteNamespace, got %d", c)
	}
}

func TestMemoryStore_Count(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	s.Insert(ctx, "c1", "ns-count", "x", "chat", "{}", []float32{1, 0, 0, 0})
	s.Insert(ctx, "c2", "ns-count", "y", "chat", "{}", []float32{0, 1, 0, 0})

	c, err := s.Count(ctx, "ns-count")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if c != 2 {
		t.Errorf("expected 2, got %d", c)
	}

	total, err := s.Count(ctx, "")
	if err != nil {
		t.Fatalf("Count total: %v", err)
	}
	if total < 2 {
		t.Errorf("expected >=2 total, got %d", total)
	}
}

func TestMemoryStore_NamespaceIsolation(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	s.Insert(ctx, "ns_a", "ns-a", "data from A", "chat", "{}", []float32{1, 0, 0, 0})

	results, err := s.Search(ctx, "ns-b", []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search ns-b: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("namespace leak: got %d results from ns-b", len(results))
	}
}

func TestMemoryStore_InsertUpdate(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	err = s.Insert(ctx, "upd1", "ns-upd", "original", "chat", "{}", []float32{1, 0, 0, 0})
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err = s.Insert(ctx, "upd1", "ns-upd", "updated", "chat", "{}", []float32{1, 0, 0, 0})
	if err != nil {
		t.Fatalf("update insert: %v", err)
	}

	results, _ := s.Search(ctx, "ns-upd", []float32{1, 0, 0, 0}, 5)
	if len(results) == 0 {
		t.Fatal("no results after update")
	}
	if results[0].Content != "updated" {
		t.Errorf("content = %q, want %q", results[0].Content, "updated")
	}
}

func TestMemoryStore_ListNamespaces(t *testing.T) {
	db := openTestDB(t)
	s, err := NewMemoryStore(db, "main", 4)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	ctx := t.Context()
	s.Insert(ctx, "l1", "ns-list-1", "x", "chat", "{}", []float32{1, 0, 0, 0})
	s.Insert(ctx, "l2", "ns-list-2", "y", "chat", "{}", []float32{0, 1, 0, 0})

	ns, err := s.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces: %v", err)
	}
	if len(ns) < 2 {
		t.Errorf("expected >=2 namespaces, got %d", len(ns))
	}
}

func TestChunk(t *testing.T) {
	tests := []struct {
		input string
		min   int
	}{
		{"", 0},
		{"hello world", 1},
		{strings.Repeat("word ", 1000), 2},
	}
	for _, tt := range tests {
		got := Chunk(tt.input)
		if len(got) < tt.min {
			t.Errorf("Chunk(%q) = %d chunks, want >=%d", tt.input[:min(20, len(tt.input))], len(got), tt.min)
		}
	}
}