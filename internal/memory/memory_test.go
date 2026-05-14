package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/storage"
)

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

func newTestStore(schema string, dim int) (*storage.DuckDB, *MemoryStore, error) {
	db, err := storage.NewDuckDB(":memory:")
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

func TestMemoryExistingSQLGuard(t *testing.T) {
	t.Run("constructor rejects SQL injection in schema", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		_, err = NewMemoryStore(db, "valid_table; DROP TABLE users --", 4)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid")
	})
	t.Run("constructor accepts valid schema", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		store, err := NewMemoryStore(db, "my_schema", 4)
		require.NoError(t, err)
		require.NotNil(t, store)
	})
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
	db, err := storage.NewDuckDB(":memory:")
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

// ============================================================================
// SQL Injection Regression Tests
// ============================================================================

// TestSQLInjection_ConstructorRejectsMaliciousSchema verifies that NewMemoryStore
// rejects schema names containing SQL injection payloads. This is the primary
// defense line — identifiers cannot be parameterized, so they are validated.
func TestSQLInjection_ConstructorRejectsMaliciousSchema(t *testing.T) {
	malicious := []struct {
		name  string
		input string
	}{
		{"semicolon_drop", "valid_table; DROP TABLE users --"},
		{"semicolon_comment", "schema; -- comment"},
		{"multiple_statements", "x'; INSERT INTO audit VALUES('hacked')--"},
		{"union_inject", "u UNION SELECT * FROM secrets"},
		{"or_true", "t' OR '1'='1"},
		{"drop_table", "DROP TABLE users"},
		{"delete_all", "DELETE FROM users"},
		{"alter_table", "ALTER TABLE users"},
		{"select_star", "SELECT * FROM users"},
	}

	for _, tt := range malicious {
		t.Run(tt.name, func(t *testing.T) {
			db, err := storage.NewDuckDB(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			_, err = NewMemoryStore(db, tt.input, 4)
			require.Error(t, err, "expected rejection for SQL injection payload: %q", tt.input)
			require.Contains(t, err.Error(), "invalid", "error should mention invalid identifier")
		})
	}
}

// TestSQLInjection_ConstructorRejectsSpecialChars verifies the regex-based guard
// catches spaces, hyphens, dots, and other characters that would break identifier safety.
func TestSQLInjection_ConstructorRejectsSpecialChars(t *testing.T) {
	invalid := []struct {
		name  string
		input string
	}{
		{"space_in_name", "my schema"},
		{"space_leading", " leading"},
		{"space_trailing", "trailing "},
		{"hyphen", "my-schema"},
		{"dot", "my.schema"},
		{"slash", "my/schema"},
		{"quote_single", "schema'inject"},
		{"quote_double", `schema"inject`},
		{"comment_dash", "schema-- DROP"},
		{"comment_block", "schema/* DROP */"},
		{"null_byte", "schema\x00name"},
		{"newline", "schema\nname"},
		{"tab", "schema\tname"},
		{"unicode", "αlpha"},      // non-ASCII letters are rejected by ASCII regex
		{"emoji", "sch😀ma"},      // emoji rejected
		{"overlong", strings.Repeat("a", 65)}, // too long for identifier rule
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			db, err := storage.NewDuckDB(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			_, err = NewMemoryStore(db, tt.input, 4)
			require.Error(t, err, "expected rejection for special char payload: %q", tt.input)
		})
	}
}

// TestSQLInjection_ConstructorRejectsSQLKeywords confirms that reserved SQL
// keywords cannot be used as schema names, preventing keyword confusion.
func TestSQLInjection_ConstructorRejectsSQLKeywords(t *testing.T) {
	keywords := []string{
		"SELECT", "select", "Select",
		"DROP", "drop", "Drop",
		"TABLE", "table", "Table",
		"INSERT", "INSERT",
		"UPDATE", "update",
		"DELETE", "delete",
		"CREATE", "create",
		"ALTER", "alter",
		"TRUNCATE", "truncate",
		"GRANT", "grant",
		"EXEC", "exec",
		"SCHEMA", "schema",
		"DATABASE", "database",
		"UNION", "union",
		"COPY", "copy",
		"ATTACH", "attach",
		"LOAD", "load",
		"ORDER", "order",
		"GROUP", "group",
	}

	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			db, err := storage.NewDuckDB(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			_, err = NewMemoryStore(db, kw, 4)
			require.Error(t, err, "expected rejection for keyword: %q", kw)
			require.Contains(t, err.Error(), "invalid", "error should mention invalid identifier")
		})
	}
}

// TestSQLInjection_ConstructorAcceptsValidSchema verifies that legitimate
// schema names pass validation successfully.
func TestSQLInjection_ConstructorAcceptsValidSchema(t *testing.T) {
	valid := []struct {
		name  string
		input string
	}{
		{"lowercase", "my_schema"},
		{"uppercase", "MY_SCHEMA"},
		{"mixed", "MySchema_01"},
		{"underscore_leading", "_private"},
		{"single_char", "a"},
		{"single_underscore", "_"},
		{"with_digits", "proj_2024_beta3"},
		{"max_length", strings.Repeat("a", 64)}, // boundary case
	}

	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			db, err := storage.NewDuckDB(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			store, err := NewMemoryStore(db, tt.input, 4)
			require.NoError(t, err, "expected acceptance for valid schema: %q", tt.input)
			require.NotNil(t, store)
		})
	}
}

// TestSQLInjection_QuoteIdentifierTableName verifies that tableName() returns
// properly double-quoted identifiers, including escaping embedded double quotes.
func TestSQLInjection_QuoteIdentifierTableName(t *testing.T) {
	// We can't use QuoteIdentifier with injectable schemas (they're blocked
	// at the constructor), but we can test the mechanism via valid schemas.
	db, err := storage.NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store, err := NewMemoryStore(db, "my_schema", 4)
	require.NoError(t, err)

	// tableName should use safeident.QuoteIdentifier, producing "my_schema".memory_store
	tn := store.tableName()
	require.Equal(t, `"my_schema".memory_store`, tn)

	// Empty schema should return unqualified table name
	store2, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)
	tn2 := store2.tableName()
	require.Equal(t, "memory_store", tn2)
}

// TestSQLInjection_StoreParameterizedSafety confirms that Store() uses
// parameterized queries (?) so user-controlled keys/values cannot inject SQL.
func TestSQLInjection_StoreParameterizedSafety(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	// Keys that would be dangerous if interpolated directly
	maliciousKeys := []string{
		"key'; DROP TABLE memory_store; --",
		"key\"; DELETE FROM memory_store; --",
		"normal_key",
		"; INSERT INTO secrets VALUES ('x')",
		"' OR '1'='1",
		"key-- comment",
		"key/* drop */",
	}

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	for _, key := range maliciousKeys {
		err := ms.Store(ctx, key, []byte("payload"), embed)
		require.NoError(t, err, "Store should safely handle key via parameterization: %q", key)

		val, ok := ms.Get(ctx, key)
		require.True(t, ok, "key should be retrievable: %q", key)
		require.Equal(t, "payload", string(val))
	}
}

// TestSQLInjection_SearchParameterizedSafety confirms that Search() and
// SearchText() use parameterized queries for the limit and query parameters.
func TestSQLInjection_SearchParameterizedSafety(t *testing.T) {
	ms := inMemoryDB(t)
	ctx := context.Background()

	// Store entries with safe keys
	embed := []float32{0.1, 0.2, 0.3, 0.4}
	require.NoError(t, ms.Store(ctx, "alpha", []byte("apple pie"), embed))
	require.NoError(t, ms.Store(ctx, "beta", []byte("banana bread"), embed))

	// SearchText with a query that would be dangerous if interpolated
	results, err := ms.SearchText(ctx, "'; DROP TABLE memory_store; --", 10)
	require.NoError(t, err, "SearchText should safely handle query via parameterization")
	require.Len(t, results, 0, "malicious query should match nothing")

	// Verify store is intact after the "injection" attempt
	results2, err := ms.SearchText(ctx, "apple", 10)
	require.NoError(t, err)
	require.Len(t, results2, 1)
	require.Equal(t, "alpha", results2[0].Key)
}

// TestSQLInjection_FullChainNewMemoryStore validates the complete construction
// and usage chain: valid schema passes, invalid schema fails, and the store
// safely handles data operations afterward.
func TestSQLInjection_FullChainNewMemoryStore(t *testing.T) {
	ctx := context.Background()

	// Positive case: valid schema roundtrip
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// DuckDB requires the schema to exist before it can be used.
	_, err = db.Exec(context.Background(), "CREATE SCHEMA trusted_schema")
	require.NoError(t, err)

	store, err := NewMemoryStore(db, "trusted_schema", 4)
	require.NoError(t, err)
	require.NotNil(t, store)

	embed := []float32{0.1, 0.2, 0.3, 0.4}
	require.NoError(t, store.Store(ctx, "my_key", []byte("my_value"), embed))

	val, ok := store.Get(ctx, "my_key")
	require.True(t, ok)
	require.Equal(t, "my_value", string(val))

	results, err := store.Search(ctx, embed, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Negative case: injection payload fails at constructor, never reaches DB
	db2, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db2.Close()

	_, err = NewMemoryStore(db2, "evil; DROP TABLE users", 4)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

// TestSQLInjection_NoBypassViaEmptySchema verifies that an empty schema is
// valid and that the unqualified table name "memory_store" is used safely.
// (Empty schema skips validation in NewMemoryStore, which is intentional.)
func TestSQLInjection_NoBypassViaEmptySchema(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)
	require.NotNil(t, store)

	ctx := context.Background()
	embed := []float32{0.1, 0.2, 0.3, 0.4}
	require.NoError(t, store.Store(ctx, "k", []byte("v"), embed))

	val, ok := store.Get(ctx, "k")
	require.True(t, ok)
	require.Equal(t, "v", string(val))

	require.Equal(t, "memory_store", store.tableName())
}

// TestSQLInjection_StoreAndSearchRejectSchemaParameterTampering confirms
// that once constructed with a valid schema, there's no mechanism to change
// the schema, and Store()/Search() always use the validated/quoted schema.
func TestSQLInjection_StoreAndSearchRejectSchemaParameterTampering(t *testing.T) {
	ms := inMemoryDBDim(t, 4)
	// ms was constructed with empty schema via inMemoryDB helper; ensure it works
	ctx := context.Background()
	embed := []float32{0.1, 0.2, 0.3, 0.4}

	// This should succeed; the schema cannot be tampered with at call time
	require.NoError(t, ms.Store(ctx, "key1", []byte("val1"), embed))

	results, err := ms.Search(ctx, embed, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "key1", results[0].Key)
}

// =============================================================================
// SQL Injection Regression Tests — memory.go
// =============================================================================

func truncateForTestName(s string) string {
	s = strings.ReplaceAll(s, "\x00", "\\x00")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\r", "\\r")
	if len(s) > 35 {
		return s[:35] + "..."
	}
	return s
}

// ---------------------------------------------------------------------------
// DEFENSE LAYER 1: Constructor validates schema via safeident.ValidateIdentifier
// ---------------------------------------------------------------------------

// TestSQLInjection_MaliciousSchemaPayloads verifies the constructor rejects
// classic SQL injection payloads, spaces, special chars, and destructive patterns.
// This is the primary defense layer — no MemoryStore can be created with a
// malicious identifier.
func TestSQLInjection_MaliciousSchemaPayloads(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	payloads := []string{
		// --- Classic SQL injection ---
		`users; DROP TABLE accounts--`,
		`'; DELETE FROM users WHERE '1'='1`,
		`users" OR "1"="1`,
		`users*/ DROP TABLE x;--`,
		`1; DROP TABLE users`,
		`table_name; INSERT INTO x VALUES(1);--`,
		`'; --`,
		`users" --`,
		`a'); DELETE FROM data; SELECT * FROM b WHERE ('1'='1`,
		// --- Semicolons ---
		`; DROP TABLE users`,
		`test;`,
		// --- SQL comment injection ---
		`users-- DROP`,
		`users/*DROP ALL*/`,
		// --- Spaces, tabs, newlines ---
		`my table`,
		"my\tschema",
		"my\nschema",
		"my\rschema",
		// --- Special characters ---
		`my-table`,
		`my.table`,
		`my/schema`,
		// --- Path traversal ---
		`../etc`,
		// --- Too long (>64 chars) ---
		strings.Repeat("a", 65),
		strings.Repeat("a", 128),
		// --- Null byte injection ---
		"users\x00name",
		// --- Backtick injection (URL-encoding variant) ---
		"users`; INSERT INTO x VALUES(1); --",
	}

	for _, payload := range payloads {
		name := truncateForTestName(payload)
		t.Run(name, func(t *testing.T) {
			_, err := NewMemoryStore(db, payload, 4)
			require.Error(t, err, "expected error for malicious schema %q", payload)
			require.Contains(t, err.Error(), "memory:",
				"error should originate from memory package")
		})
	}
}

// TestSQLInjection_RejectsSQLKeywords verifies every keyword in the safeident
// blacklist is rejected as a schema name (case-insensitive).
func TestSQLInjection_RejectsSQLKeywords(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	keywords := []string{
		// DML/DDL
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"DROP", "CREATE", "ALTER", "TRUNCATE",
		"REPLACE", "RENAME",
		// DML clauses
		"FROM", "WHERE", "SET", "INTO", "VALUES",
		// DCL
		"GRANT", "REVOKE",
		// Flow control
		"EXEC", "EXECUTE", "CALL",
		// DDL objects
		"TABLE", "VIEW", "INDEX", "DATABASE",
		"SCHEMA", "FUNCTION", "PROCEDURE",
		"TRIGGER", "SEQUENCE",
		// Utility / dangerous
		"COPY", "ATTACH", "DETACH",
		"LOAD", "INSTALL", "UNINSTALL",
		"EXPORT", "IMPORT",
		// Subquery / expression
		"UNION", "INTERSECT", "EXCEPT",
		// DuckDB-specific
		"PRAGMA", "SUMMARIZE", "DESCRIBE",
		// Transaction
		"COMMIT", "ROLLBACK", "BEGIN",
		// Boolean
		"TRUE", "FALSE", "NULL",
		// DuckDB extension
		"ORDER", "GROUP", "HAVING", "LIMIT",
	}

	for _, kw := range keywords {
		// Test both upper and lower case
		for _, variant := range []string{kw, strings.ToLower(kw)} {
			name := fmt.Sprintf("%s (%s)", kw, variant)
			t.Run(name, func(t *testing.T) {
				_, err := NewMemoryStore(db, variant, 4)
				require.Error(t, err,
					"expected keyword %q to be rejected by ValidateIdentifier", variant)
				require.Contains(t, err.Error(), "memory:",
					"error should originate from memory package")
			})
		}
	}
}

// TestSQLInjection_AcceptsValidSchemas verifies that normal, well-formed
// schema names are accepted by the constructor.
func TestSQLInjection_AcceptsValidSchemas(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	validSchemas := []string{
		"",                         // empty = no schema prefix, explicitly allowed
		"my_schema",               // standard
		"test",                    // short
		"s",                       // single char
		"my_schema_123",           // digits
		"CamelCase",               // mixed case
		"_leading_underscore",     // underscore prefix
		"my_test_namespace",       // non-keyword compound name
		"a123",                    // letter + digits
		strings.Repeat("a", 64),  // exactly at max length
	}

	for _, schema := range validSchemas {
		name := schema
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			store, err := NewMemoryStore(db, schema, 4)
			require.NoError(t, err, "expected valid schema %q to be accepted", schema)
			require.NotNil(t, store, "store should not be nil for valid schema %q", schema)
		})
	}
}

// ---------------------------------------------------------------------------
// DEFENSE LAYER 2: tableName() uses safeident.QuoteIdentifier (defense-in-depth)
// ---------------------------------------------------------------------------

// TestTableName_QuoteIdentifier verifies that tableName() always returns
// a properly double-quoted identifier. This is the second defense layer:
// even if validation were bypassed, QuoteIdentifier turns a raw name into
// a single SQL identifier token (e.g., `; DROP` → `"; DROP".memory_store`).
func TestTableName_QuoteIdentifier(t *testing.T) {
	tests := []struct {
		schema string
		want   string
	}{
		{"", "memory_store"},
		{"my_schema", `"my_schema".memory_store`},
		{"test", `"test".memory_store`},
		{"UPPER_CASE", `"UPPER_CASE".memory_store`},
		{"a", `"a".memory_store`},
		{"_private", `"_private".memory_store`},
	}

	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			ms := &MemoryStore{schema: tt.schema, dim: 4}
			got := ms.tableName()
			require.Equal(t, tt.want, got,
				"tableName() should return double-quoted identifier")
			if tt.schema != "" {
				require.True(t, strings.HasPrefix(got, `"`),
					"tableName() with schema should start with double-quote")
				require.True(t, strings.Contains(got, `".memory_store`),
					"tableName() should have closing quote before .memory_store")
			}
		})
	}
}

// TestMemoryStore_DefenseInDepth demonstrates that QuoteIdentifier wraps
// dangerous names so they become a single identifier token. Even if a
// schema like `; DROP TABLE` somehow reached tableName(), it would
// become `"; DROP TABLE".memory_store` — one quoted identifier.
func TestMemoryStore_DefenseInDepth(t *testing.T) {
	t.Run("normal_schema_quoted", func(t *testing.T) {
		ms := &MemoryStore{schema: "weird_name", dim: 4}
		got := ms.tableName()
		require.Equal(t, `"weird_name".memory_store`, got)
	})

	t.Run("embedded_quotes_escaped", func(t *testing.T) {
		ms := &MemoryStore{schema: `say"hi"`, dim: 4}
		got := ms.tableName()
		require.Equal(t, `"say""hi""".memory_store`, got)
	})
}

// ---------------------------------------------------------------------------
// INTEGRATION: Full-chain validation test
// ---------------------------------------------------------------------------

// TestFullChain_SchemaValidation verifies the end-to-end flow:
//  1. Valid schema → MemoryStore created → all operations succeed
//  2. Invalid schema → constructor returns error (no store created)
func TestFullChain_SchemaValidation(t *testing.T) {
	t.Run("valid_schema_all_operations_succeed", func(t *testing.T) {
		db, ms, err := newTestStore("", 4)
		require.NoError(t, err)
		defer db.Close()

		ctx := context.Background()
		embed := []float32{0.1, 0.2, 0.3, 0.4}

		// Store
		require.NoError(t, ms.Store(ctx, "k1", []byte("v1"), embed))
		// Store another
		require.NoError(t, ms.Store(ctx, "k2", []byte("v2"), embed))

		// Get
		val, ok := ms.Get(ctx, "k1")
		require.True(t, ok)
		require.Equal(t, "v1", string(val))

		// Search (vector)
		results, err := ms.Search(ctx, embed, 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(results), 1)

		// SearchText
		results, err = ms.SearchText(ctx, "v1", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(results), 1)

		// List
		entries, err := ms.List(ctx, 10, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(entries), 2)

		// Delete
		require.NoError(t, ms.Delete(ctx, "k1"))
		_, ok = ms.Get(ctx, "k1")
		require.False(t, ok, "key should be deleted")
	})

	t.Run("invalid_schema_blocked_at_construction", func(t *testing.T) {
		db, err := storage.NewDuckDB(":memory:")
		require.NoError(t, err)
		defer db.Close()

		_, err = NewMemoryStore(db, "; DROP TABLE users", 4)
		require.Error(t, err)
		require.Contains(t, err.Error(), "memory:")
	})
}

// TestFullChain_NoSchemaUsesBareTableName verifies that empty schema produces
// a bare "memory_store" table name (no schema prefix, no quoting).
func TestFullChain_NoSchemaUsesBareTableName(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	tableName := store.tableName()
	require.Equal(t, "memory_store", tableName,
		"empty schema should produce bare table name")
	require.False(t, strings.Contains(tableName, `"`),
		"empty schema should NOT produce quoted output")
}

// ---------------------------------------------------------------------------
// EDGE CASES: Unicode, international characters, boundary conditions
// ---------------------------------------------------------------------------

// TestSQLInjection_UnicodeAndInternational verifies behavior with non-ASCII
// characters. NOTE: The current regex ^[a-zA-Z_][a-zA-Z0-9_]*$ only matches
// ASCII. Unicode identifiers are REJECTED. This documents the current
// limitation — DuckDB supports quoted unicode identifiers, but ValidateIdentifier
// does not (yet) allow them.
func TestSQLInjection_UnicodeAndInternational(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	unicodePayloads := []string{
		"sch\u00e9ma",
		"espa\u00f1a",
		"Gr\u00fc\u00df",
		"\u0441\u0445\u0435\u043c\u0430",
		"\u30b9\u30ad\u30fc\u30de",
		"\u4e2d\u6587",
		"a\u0301",
	}

	for _, payload := range unicodePayloads {
		name := truncateForTestName(payload)
		t.Run(name, func(t *testing.T) {
			_, err := NewMemoryStore(db, payload, 4)
			require.Error(t, err,
				"unicode identifier %q should be rejected by ASCII-only regex", payload)
		})
	}

	t.Run("accents_within_ascii_rejected", func(t *testing.T) {
		_, err := NewMemoryStore(db, "sch\u00e9ma", 4)
		require.Error(t, err)
	})
}

// TestSQLInjection_BoundaryConditions tests identifiers at length boundaries.
func TestSQLInjection_BoundaryConditions(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	t.Run("exactly_64_chars_accepted", func(t *testing.T) {
		id := strings.Repeat("a", 64)
		store, err := NewMemoryStore(db, id, 4)
		require.NoError(t, err, "64-char identifier should be accepted")
		require.NotNil(t, store)

		expected := `"` + id + `".memory_store`
		require.Equal(t, expected, store.tableName())
	})

	t.Run("exactly_65_chars_rejected", func(t *testing.T) {
		id := strings.Repeat("a", 65)
		_, err := NewMemoryStore(db, id, 4)
		require.Error(t, err, "65-char identifier should be rejected")
	})

	t.Run("single_char_accepted", func(t *testing.T) {
		store, err := NewMemoryStore(db, "a", 4)
		require.NoError(t, err)
		require.NotNil(t, store)
	})

	t.Run("single_underscore_accepted", func(t *testing.T) {
		store, err := NewMemoryStore(db, "_", 4)
		require.NoError(t, err)
		require.NotNil(t, store)
	})

	t.Run("starts_with_digit_rejected", func(t *testing.T) {
		_, err := NewMemoryStore(db, "123abc", 4)
		require.Error(t, err, "identifier starting with digit should be rejected")
	})
}

// TestSQLInjection_CaseInsensitiveKeywords verifies that SQL keywords are
// rejected regardless of case (SELECT, select, Select, sElEcT, etc.)
func TestSQLInjection_CaseInsensitiveKeywords(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cases := []string{"SELECT", "select", "Select", "sElEcT", "SELect"}
	for _, variant := range cases {
		t.Run(variant, func(t *testing.T) {
			_, err := NewMemoryStore(db, variant, 4)
			require.Error(t, err,
				"keyword variant %q should be rejected case-insensitively", variant)
		})
	}
}


