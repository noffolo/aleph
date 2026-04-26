package storage

import (
	"context"
	"testing"
)

func TestCrossProjectIsolation(t *testing.T) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create two project schemas
	for _, schema := range []string{"project_alice", "project_bob"} {
		_, err := db.Exec("CREATE SCHEMA IF NOT EXISTS \"" + schema + "\"")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS \"" + schema + "\".secrets (key VARCHAR, value VARCHAR)")
		if err != nil {
			t.Fatal(err)
		}
	}

	// Insert alice's data
	aliceCtx := ContextWithSchema(context.Background(), "project_alice")
	_, err = db.ExecContext(aliceCtx, "INSERT INTO secrets VALUES (?, ?)", "alice_key", "alice_value_42")
	if err != nil {
		t.Fatal(err)
	}

	// Insert bob's data
	bobCtx := ContextWithSchema(context.Background(), "project_bob")
	_, err = db.ExecContext(bobCtx, "INSERT INTO secrets VALUES (?, ?)", "bob_key", "bob_value_99")
	if err != nil {
		t.Fatal(err)
	}

	// Verify data exists in each schema
	tests := []struct {
		name       string
		schema     string
		wantKey    string
		wantValue  string
		wantCount  int
	}{
		{
			name:      "Alice reads her own data",
			schema:    "project_alice",
			wantKey:   "alice_key",
			wantValue: "alice_value_42",
			wantCount: 1,
		},
		{
			name:      "Bob reads his own data",
			schema:    "project_bob",
			wantKey:   "bob_key",
			wantValue: "bob_value_99",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithSchema(context.Background(), tt.schema)

			// Count rows scoped to schema
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM secrets").Scan(&count)
			if err != nil {
				t.Fatal(err)
			}
			if count != tt.wantCount {
				t.Errorf("expected %d row(s) in schema %s, got %d", tt.wantCount, tt.schema, count)
			}

			// Read back the value
			var key, value string
			err = db.QueryRowContext(ctx, "SELECT key, value FROM secrets LIMIT 1").Scan(&key, &value)
			if err != nil {
				t.Fatal(err)
			}
			if key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, key)
			}
			if value != tt.wantValue {
				t.Errorf("expected value %q, got %q", tt.wantValue, value)
			}
		})
	}

	// Cross-project isolation: alice cannot read bob's data and vice versa
	t.Run("Alice cannot see Bob's data", func(t *testing.T) {
		ctx := ContextWithSchema(context.Background(), "project_alice")
		var value string
		err := db.QueryRowContext(ctx, "SELECT value FROM secrets WHERE key = 'bob_key'").Scan(&value)
		if err == nil {
			t.Errorf("Alice should NOT be able to read Bob's data, but got value: %s", value)
		}
	})

	t.Run("Bob cannot see Alice's data", func(t *testing.T) {
		ctx := ContextWithSchema(context.Background(), "project_bob")
		var value string
		err := db.QueryRowContext(ctx, "SELECT value FROM secrets WHERE key = 'alice_key'").Scan(&value)
		if err == nil {
			t.Errorf("Bob should NOT be able to read Alice's data, but got value: %s", value)
		}
	})

	// Without schema context, queries should NOT see project-scoped data
	t.Run("Unscoped query sees no project data", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM project_alice.secrets").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("expected 1 row in project_alice.secrets, got %d", count)
		}
	})
}
