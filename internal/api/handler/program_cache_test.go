package handler

import (
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/dsl"
)

func TestNewProgramCache(t *testing.T) {
	cache := newProgramCache()
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.maxEntries != defaultMaxEntries {
		t.Fatalf("expected maxEntries=%d, got %d", defaultMaxEntries, cache.maxEntries)
	}
	if cache.ttl != defaultTTL {
		t.Fatalf("expected ttl=%v, got %v", defaultTTL, cache.ttl)
	}
	if len(cache.entries) != 0 {
		t.Fatal("expected empty entries map")
	}
}

func TestProgramCache_Get_Miss(t *testing.T) {
	cache := newProgramCache()
	if got := cache.Get("nonexistent"); got != nil {
		t.Fatal("expected nil for cache miss")
	}
}

func programWithStatements(n int) *dsl.Program {
	prog := &dsl.Program{Statements: make([]*dsl.Statement, n)}
	for i := range prog.Statements {
		prog.Statements[i] = &dsl.Statement{}
	}
	return prog
}

func TestProgramCache_Set_And_Get(t *testing.T) {
	cache := newProgramCache()
	prog := programWithStatements(1)
	cache.Set("key1", prog)

	if got := cache.Get("key1"); got == nil {
		t.Fatal("expected non-nil for cache hit")
	} else if len(got.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(got.Statements))
	}
}

func TestProgramCache_Get_ExpiredTTL(t *testing.T) {
	cache := newProgramCache()
	cache.ttl = 1 * time.Millisecond
	cache.Set("key1", programWithStatements(1))

	time.Sleep(2 * time.Millisecond)

	if got := cache.Get("key1"); got != nil {
		t.Fatal("expected nil for expired cache entry")
	}
}

func TestProgramCache_Set_Eviction(t *testing.T) {
	cache := newProgramCache()
	cache.maxEntries = 3

	cache.Set("a", programWithStatements(1))
	time.Sleep(1 * time.Millisecond)
	cache.Set("b", programWithStatements(2))
	time.Sleep(1 * time.Millisecond)
	cache.Set("c", programWithStatements(3))
	time.Sleep(1 * time.Millisecond)
	cache.Set("d", programWithStatements(4))

	if cache.Get("a") != nil {
		t.Fatal("expected oldest entry 'a' to be evicted")
	}
	if cache.Get("d") == nil {
		t.Fatal("expected new entry 'd' to be present")
	}
}

func TestProgramCache_Get_DifferentKeys(t *testing.T) {
	cache := newProgramCache()
	cache.Set("x", programWithStatements(1))
	cache.Set("y", programWithStatements(2))

	x := cache.Get("x")
	y := cache.Get("y")
	if x == nil || y == nil {
		t.Fatal("expected both entries to be present")
	}
	if len(x.Statements) != 1 || len(y.Statements) != 2 {
		t.Fatal("wrong program returned")
	}
}

func TestProgramCache_Set_Overwrite(t *testing.T) {
	cache := newProgramCache()
	cache.Set("key", programWithStatements(1))
	cache.Set("key", programWithStatements(2))

	if got := cache.Get("key"); got == nil || len(got.Statements) != 2 {
		t.Fatal("expected overwritten value with 2 statements")
	}
}
