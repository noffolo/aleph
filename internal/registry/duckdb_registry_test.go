package registry

import (
	"context"
	"log/slog"
	"testing"
)

var testCtx = context.Background()

func setupRegistry(t *testing.T) *DuckDBRegistry {
	t.Helper()
	r, err := NewDuckDBRegistry(":memory:", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	// :memory: DuckDB needs explicit table creation (no migrations run)
	if _, err := r.db.Exec(createTableSQL); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDuckDBRegistry_RegisterAndList(t *testing.T) {
	r := setupRegistry(t)

	id, err := r.RegisterComponent(ComponentMetadata{
		Name:        "test-tool",
		Type:        "tool",
		Description: "A test tool",
		Version:     "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}

	comps, err := r.ListComponents(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", comps[0].Name)
	}
}

func TestDuckDBRegistry_GetComponentByID(t *testing.T) {
	r := setupRegistry(t)

	id, _ := r.RegisterComponent(ComponentMetadata{
		Name:   "my-skill",
		Type:   "skill",
		Status: "active",
	})

	meta, err := r.GetComponentByID(testCtx, id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "my-skill" {
		t.Errorf("expected my-skill, got %s", meta.Name)
	}
	if meta.Status != "active" {
		t.Errorf("expected active, got %s", meta.Status)
	}
}

func TestDuckDBRegistry_GetComponentByID_NotFound(t *testing.T) {
	r := setupRegistry(t)

	_, err := r.GetComponentByID(testCtx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent id")
	}
}

func TestDuckDBRegistry_UpdateComponentStatus(t *testing.T) {
	r := setupRegistry(t)

	id, _ := r.RegisterComponent(ComponentMetadata{Name: "test", Type: "tool"})

	err := r.UpdateComponentStatus(id, "approved")
	if err != nil {
		t.Fatal(err)
	}

	meta, _ := r.GetComponentByID(testCtx, id)
	if meta.Status != "approved" {
		t.Errorf("expected approved, got %s", meta.Status)
	}
}
