package storage

import (
	"context"
	"strings"
	"testing"
)

func TestSchemaIdentity_String(t *testing.T) {
	si := MustNewSchemaIdentity("test_project")
	if si.String() != "test_project" {
		t.Fatalf("expected test_project, got %s", si.String())
	}
}

func TestSchemaIdentity_Quoted(t *testing.T) {
	si := MustNewSchemaIdentity("test_project")
	quoted := si.Quoted()
	if !strings.Contains(quoted, "test_project") {
		t.Fatalf("expected quoted string to contain test_project, got %s", quoted)
	}
}

func TestSchemaIdentity_Validate_Valid(t *testing.T) {
	si := MustNewSchemaIdentity("valid_project_123")
	if err := si.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSchemaIdentity_Validate_Invalid(t *testing.T) {
	// Create via struct directly to bypass constructor validation
	si := SchemaIdentity{schema: "bad project!"}
	if err := si.Validate(); err == nil {
		t.Fatal("expected error for invalid schema, got nil")
	}
}

func TestNewSchemaIdentity_Valid(t *testing.T) {
	si, err := NewSchemaIdentity("my_project")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if si.String() != "my_project" {
		t.Fatalf("expected my_project, got %s", si.String())
	}
}

func TestNewSchemaIdentity_Invalid_Empty(t *testing.T) {
	_, err := NewSchemaIdentity("")
	if err == nil {
		t.Fatal("expected error for empty schema name")
	}
}

func TestNewSchemaIdentity_Invalid_StartsWithDigit(t *testing.T) {
	_, err := NewSchemaIdentity("123invalid")
	if err == nil {
		t.Fatal("expected error for schema starting with digit")
	}
}

func TestNewSchemaIdentity_Invalid_TooLong(t *testing.T) {
	longName := strings.Repeat("a", 129)
	_, err := NewSchemaIdentity(longName)
	if err == nil {
		t.Fatal("expected error for schema >128 chars")
	}
}

func TestNewSchemaIdentity_Exactly_128Chars(t *testing.T) {
	longName := strings.Repeat("a", 128)
	si, err := NewSchemaIdentity(longName)
	if err != nil {
		t.Fatalf("expected no error for 128-char valid name, got %v", err)
	}
	if len(si.String()) != 128 {
		t.Fatalf("expected 128 chars, got %d", len(si.String()))
	}
}

func TestMustNewSchemaIdentity_Valid(t *testing.T) {
	si := MustNewSchemaIdentity("safe_name")
	if si.String() != "safe_name" {
		t.Fatalf("expected safe_name, got %s", si.String())
	}
}

func TestMustNewSchemaIdentity_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid schema name")
		}
	}()
	MustNewSchemaIdentity("")
}

func TestSanitizeProjectID_Valid(t *testing.T) {
	validIDs := []string{
		"a",
		"Z",
		"_underscore",
		"project_123",
		"my-project-42",
		"Test_Mixed-Case_123",
	}
	for _, id := range validIDs {
		if err := SanitizeProjectID(id); err != nil {
			t.Fatalf("expected %q to be valid, got error: %v", id, err)
		}
	}
}

func TestSanitizeProjectID_Invalid(t *testing.T) {
	invalidIDs := []string{
		"",                // empty
		"123starts",       // starts with digit
		"has space",       // contains space
		"has.dot",         // contains dot
		"has/slash",       // contains slash
		"has;semicolon",   // contains semicolon
		"has'quote",       // contains quote
		"has\"dblquote",   // contains double quote
		"has`backtick",    // contains backtick
		"@special",         // contains @
		"#hash",            // contains #
	}
	for _, id := range invalidIDs {
		if err := SanitizeProjectID(id); err == nil {
			t.Fatalf("expected %q to be invalid, got nil error", id)
		}
	}
}

func TestSanitizeProjectID_TooLong(t *testing.T) {
	longName := strings.Repeat("a", 129)
	err := SanitizeProjectID(longName)
	if err == nil {
		t.Fatal("expected error for projectID >128 chars")
	}
}

func TestContextWithSchema_SchemaFromContext(t *testing.T) {
	ctx := context.Background()
	si := MustNewSchemaIdentity("ctx_project")

	ctx = ContextWithSchema(ctx, si)

	schema, ok := SchemaFromContext(ctx)
	if !ok {
		t.Fatal("expected schema to be found in context")
	}
	if schema != "ctx_project" {
		t.Fatalf("expected ctx_project, got %s", schema)
	}
}

func TestSchemaFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	_, ok := SchemaFromContext(ctx)
	if ok {
		t.Fatal("expected no schema in empty context")
	}
}

func TestScopeQuery_NoSchema(t *testing.T) {
	ctx := context.Background()
	result := scopeQuery(ctx, "SELECT 1")
	if result != "SELECT 1" {
		t.Fatalf("expected unchanged query, got %s", result)
	}
}

func TestScopeQuery_WithSchema(t *testing.T) {
	si := MustNewSchemaIdentity("my_schema")
	ctx := ContextWithSchema(context.Background(), si)

	result := scopeQuery(ctx, "SELECT * FROM users")
	if !strings.Contains(result, "SET schema =") {
		t.Fatalf("expected SET schema in result, got %s", result)
	}
	if !strings.Contains(result, "SELECT * FROM users") {
		t.Fatalf("expected original query in result, got %s", result)
	}
}

func TestScopeQuery_InvalidSchemaInContext(t *testing.T) {
	// Put an invalid schema directly in context (bypassing validation)
	ctx := context.WithValue(context.Background(), schemaKey, "bad schema!")
	result := scopeQuery(ctx, "SELECT 1")
	// Should fall back to original query due to validation failure
	if result != "SELECT 1" {
		t.Fatalf("expected fallback to original query, got %s", result)
	}
}

func TestEnsureProjectSchema_Creates(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory duckdb: %v", err)
	}
	defer d.Close()

	si := MustNewSchemaIdentity("test_ensure_schema")
	ctx := context.Background()

	if err := EnsureProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("EnsureProjectSchema failed: %v", err)
	}

	// Verify by querying information_schema
	rows, err := d.Query("SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'test_ensure_schema'")
	if err != nil {
		t.Fatalf("query schemata failed: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		if name == "test_ensure_schema" {
			found = true
		}
	}
	if !found {
		t.Fatal("schema was not created")
	}
}

func TestEnsureProjectSchema_Idempotent(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory duckdb: %v", err)
	}
	defer d.Close()

	si := MustNewSchemaIdentity("test_idempotent")
	ctx := context.Background()

	// Create twice — should not error
	if err := EnsureProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("first EnsureProjectSchema failed: %v", err)
	}
	if err := EnsureProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("second EnsureProjectSchema failed: %v", err)
	}
}

func TestDropProjectSchema(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory duckdb: %v", err)
	}
	defer d.Close()

	si := MustNewSchemaIdentity("test_drop")
	ctx := context.Background()

	// Create then drop
	if err := EnsureProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("EnsureProjectSchema failed: %v", err)
	}
	if err := DropProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("DropProjectSchema failed: %v", err)
	}

	// Verify it's gone
	rows, err := d.Query("SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'test_drop'")
	if err != nil {
		t.Fatalf("query schemata failed: %v", err)
	}
	defer rows.Close()

	if rows.Next() {
		t.Fatal("schema should have been dropped")
	}
}

func TestDropProjectSchema_Idempotent(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory duckdb: %v", err)
	}
	defer d.Close()

	si := MustNewSchemaIdentity("test_drop_idempotent")
	ctx := context.Background()

	// Drop non-existent schema — should not error
	if err := DropProjectSchema(ctx, d, si); err != nil {
		t.Fatalf("DropProjectSchema on non-existent schema failed: %v", err)
	}
}
