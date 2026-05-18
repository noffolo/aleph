//go:build integration

package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func duckdbAvailable() bool {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		return false
	}
	_ = db.Close()
	return true
}

func newTestDB(t *testing.T) *DuckDB {
	t.Helper()
	db, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("NewDuckDB(:memory:): %v", err)
	}
	return db
}

// --- 1. Database creation and connection ---

func TestIntegration_DBCreation(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	if db.path != ":memory:" {
		t.Errorf("expected path :memory:, got %s", db.path)
	}

	// Verify we can ping / execute a basic query
	if err := db.DB().Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

// --- 2. Schema creation ---

func TestIntegration_CreateSchema(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS test_schema")
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Verify the schema exists by querying information_schema
	rows, err := db.Query("SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'test_schema'")
	if err != nil {
		t.Fatalf("query schemata: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected schema 'test_schema' to exist")
	}
	var name string
	if err := rows.Scan(&name); err != nil {
		t.Fatalf("scan schema name: %v", err)
	}
	if name != "test_schema" {
		t.Errorf("expected 'test_schema', got %q", name)
	}
}

// --- 3. Table creation within a schema ---

func TestIntegration_CreateTableInSchema(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS myschema")
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	_, err = db.Exec(context.Background(), "CREATE TABLE myschema.people (id INTEGER, name VARCHAR)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Verify the table exists
	rows, err := db.Query(
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'myschema' AND table_name = 'people'",
	)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected table 'people' in schema 'myschema'")
	}
}

// --- 4. INSERT and SELECT round-trip ---

func TestIntegration_InsertSelectRoundtrip(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE items (id INTEGER, label VARCHAR)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(context.Background(), "INSERT INTO items VALUES (1, 'alpha'), (2, 'beta'), (3, 'gamma')")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := db.Query("SELECT id, label FROM items ORDER BY id")
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close()

	type row struct {
		id    int
		label string
	}
	var results []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.label); err != nil {
			t.Fatalf("scan: %v", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(results))
	}
	if results[0] != (row{1, "alpha"}) {
		t.Errorf("row 0: %+v", results[0])
	}
	if results[1] != (row{2, "beta"}) {
		t.Errorf("row 1: %+v", results[1])
	}
	if results[2] != (row{3, "gamma"}) {
		t.Errorf("row 2: %+v", results[2])
	}
}

// --- 5. Query with parameters ---

func TestIntegration_QueryWithParams(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE params_test (id INTEGER, name VARCHAR, score FLOAT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO params_test VALUES (1, 'alice', 9.5), (2, 'bob', 7.2), (3, 'carol', 8.1)")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	t.Run("single param", func(t *testing.T) {
		rows, err := db.Query("SELECT name FROM params_test WHERE id = ?", 2)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Fatal("expected row")
		}
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name != "bob" {
			t.Errorf("expected bob, got %s", name)
		}
	})

	t.Run("multiple params", func(t *testing.T) {
		rows, err := db.Query(
			"SELECT id, name FROM params_test WHERE score > ? AND score < ? ORDER BY id",
			7.0, 9.0,
		)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		var ids []int
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
			t.Errorf("expected ids [2,3], got %v", ids)
		}
	})

	t.Run("string param", func(t *testing.T) {
		row := db.QueryRow("SELECT id, score FROM params_test WHERE name = ?", "alice")
		var id int
		var score float64
		if err := row.Scan(&id, &score); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if id != 1 || score != 9.5 {
			t.Errorf("expected (1, 9.5), got (%d, %f)", id, score)
		}
	})
}

// --- 6. Table with different column types ---

func TestIntegration_ColumnTypes(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE type_test (" +
		"txt TEXT, " +
		"num INTEGER, " +
		"val FLOAT, " +
		"flag BOOLEAN, " +
		"ts TIMESTAMP" +
		")")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	_, err = db.Exec(context.Background(), "INSERT INTO type_test VALUES (?, ?, ?, ?, ?)",
		"hello world", 42, 3.14159, true, now,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	row := db.QueryRow("SELECT txt, num, val, flag, ts FROM type_test")
	var (
		txt  string
		num  int
		val  float64
		flag bool
		ts   time.Time
	)
	if err := row.Scan(&txt, &num, &val, &flag, &ts); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if txt != "hello world" {
		t.Errorf("txt: expected 'hello world', got %q", txt)
	}
	if num != 42 {
		t.Errorf("num: expected 42, got %d", num)
	}
	if val < 3.14 || val > 3.15 {
		t.Errorf("val: expected ~3.14159, got %f", val)
	}
	if !flag {
		t.Errorf("flag: expected true, got %v", flag)
	}
	if !ts.Equal(now) {
		t.Errorf("ts: expected %v, got %v", now, ts)
	}
}

// --- 7. Multiple tables in same schema ---

func TestIntegration_MultipleTablesInSchema(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS multi")
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	for _, tbl := range []string{"users", "orders", "products"} {
		_, err = db.Exec(context.Background(), fmt.Sprintf(
			"CREATE TABLE multi.%s (id INTEGER PRIMARY KEY, name VARCHAR)", tbl,
		))
		if err != nil {
			t.Fatalf("create table %s: %v", tbl, err)
		}
	}

	_, err = db.Exec(context.Background(), "INSERT INTO multi.users VALUES (1, 'alice')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO multi.orders VALUES (100, 'order-a')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO multi.products VALUES (10, 'widget')")
	if err != nil {
		t.Fatal(err)
	}

	// Verify all three tables have their data independently
	tableIDs := map[string]int{"users": 1, "orders": 100, "products": 10}
	expected := map[string]string{"users": "alice", "orders": "order-a", "products": "widget"}

	for table, want := range expected {
		t.Run(table, func(t *testing.T) {
			row := db.QueryRow(
				fmt.Sprintf("SELECT name FROM multi.%s WHERE id = ?", table),
				tableIDs[table],
			)
			var name string
			if err := row.Scan(&name); err != nil {
				t.Fatalf("query %s: %v", table, err)
			}
			if name != want {
				t.Errorf("%s: expected %q, got %q", table, want, name)
			}
		})
	}
}

// --- 8. Transaction commit ---

func TestIntegration_TransactionCommit(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE txn_commit (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Use ExecTx (wraps BeginTX + fn + commit with retry)
	err = db.ExecTx(context.Background(), func(tx *TX) error {
		_, execErr := tx.Exec("INSERT INTO txn_commit VALUES (?, ?)", 1, "committed-via-tx")
		return execErr
	})
	if err != nil {
		t.Fatalf("exec tx: %v", err)
	}

	// Verify data persisted after commit
	row := db.QueryRow("SELECT value FROM txn_commit WHERE id = 1")
	var val string
	if err := row.Scan(&val); err != nil {
		t.Fatalf("scan after commit: %v", err)
	}
	if val != "committed-via-tx" {
		t.Errorf("expected 'committed-via-tx', got %q", val)
	}

	// Also test manual BeginTX / Commit
	tx, err := db.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, err = tx.Exec("INSERT INTO txn_commit VALUES (?, ?)", 2, "committed-manual")
	if err != nil {
		tx.Rollback()
		t.Fatalf("insert in tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	// Verify second row also persisted
	row = db.QueryRow("SELECT value FROM txn_commit WHERE id = 2")
	if err := row.Scan(&val); err != nil {
		t.Fatalf("scan second row: %v", err)
	}
	if val != "committed-manual" {
		t.Errorf("expected 'committed-manual', got %q", val)
	}
}

// --- 9. Transaction rollback ---

func TestIntegration_TransactionRollback(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE txn_rollback (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Insert a row outside a transaction as a baseline
	_, err = db.Exec(context.Background(), "INSERT INTO txn_rollback VALUES (1, 'baseline')")
	if err != nil {
		t.Fatalf("insert baseline: %v", err)
	}

	// Start a transaction, insert a row, then rollback
	tx, err := db.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, err = tx.Exec("INSERT INTO txn_rollback VALUES (2, 'rolled-back')")
	if err != nil {
		t.Fatalf("insert in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Verify baseline row is still there, but rolled-back row is NOT
	row := db.QueryRow("SELECT value FROM txn_rollback WHERE id = 1")
	var val string
	if err := row.Scan(&val); err != nil {
		t.Fatalf("scan baseline: %v", err)
	}
	if val != "baseline" {
		t.Errorf("expected 'baseline', got %q", val)
	}

	// Verify the rolled-back row does not exist
	row = db.QueryRow("SELECT value FROM txn_rollback WHERE id = 2")
	err = row.Scan(&val)
	if err == nil {
		t.Error("expected rollback'd row to not exist, but it was found")
	}
}

// --- 10. Cleanup (DROP TABLE, DROP SCHEMA) ---

func TestIntegration_Cleanup(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	// Create schema + table
	_, err := db.Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS cleanup_schema")
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	_, err = db.Exec(context.Background(), "CREATE TABLE cleanup_schema.temp_data (id INTEGER)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.Exec(context.Background(), "INSERT INTO cleanup_schema.temp_data VALUES (1)")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Drop the table first
	_, err = db.Exec(context.Background(), "DROP TABLE IF EXISTS cleanup_schema.temp_data")
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}

	// Verify table is gone
	rows, err := db.Query(
		"SELECT table_name FROM information_schema.tables " +
			"WHERE table_schema = 'cleanup_schema' AND table_name = 'temp_data'",
	)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	if rows.Next() {
		t.Error("table 'temp_data' should have been dropped but still exists")
	}
	rows.Close()

	// Drop the schema
	_, err = db.Exec(context.Background(), "DROP SCHEMA IF EXISTS cleanup_schema")
	if err != nil {
		t.Fatalf("drop schema: %v", err)
	}

	// Verify schema is gone
	rows, err = db.Query(
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'cleanup_schema'",
	)
	if err != nil {
		t.Fatalf("query schemata: %v", err)
	}
	if rows.Next() {
		t.Error("schema 'cleanup_schema' should have been dropped but still exists")
	}
	rows.Close()
}

// --- 11. Context cancellation handling ---

func TestIntegration_ContextCancellation(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE ctx_cancel (id INTEGER)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	t.Run("ExecContext pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := db.ExecContext(ctx, "INSERT INTO ctx_cancel VALUES (1)")
		if err == nil {
			t.Error("expected error with cancelled context, got nil")
		}
	})

	t.Run("QueryContext pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := db.QueryContext(ctx, "SELECT * FROM ctx_cancel")
		if err == nil {
			t.Error("expected error with cancelled context, got nil")
		}
	})

	t.Run("QueryRowContextOrError pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := db.QueryRowContextOrError(ctx, "SELECT 1")
		if err == nil {
			t.Error("expected error with cancelled context, got nil")
		}
	})

	t.Run("Timeout context", func(t *testing.T) {
		// Use a context with a very short deadline
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give the timeout time to fire
		time.Sleep(1 * time.Millisecond)

		_, err := db.ExecContext(ctx, "INSERT INTO ctx_cancel VALUES (2)")
		if err == nil {
			t.Error("expected error with expired deadline, got nil")
		}
	})

	t.Run("ExecWithRetry respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := db.ExecWithRetry(ctx, "SELECT 1")
		if err == nil {
			t.Error("expected error with cancelled context, got nil")
		}
	})
}

// --- 12. Thread safety ---

func TestIntegration_ThreadSafety(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE ts_test (id INTEGER PRIMARY KEY, goroutine_id INTEGER, value TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	const numGoroutines = 20
	const insertsPerGoroutine = 50

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < insertsPerGoroutine; i++ {
				id := gid*insertsPerGoroutine + i
				_, err := db.Exec(context.Background(),
					"INSERT INTO ts_test VALUES (?, ?, ?)",
					id, gid, fmt.Sprintf("g%d-i%d", gid, i),
				)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d insert %d: %w", gid, i, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	// Check for any errors from goroutines
	for err := range errCh {
		t.Error(err)
	}

	// Verify all expected rows exist
	totalExpected := numGoroutines * insertsPerGoroutine
	row := db.QueryRow("SELECT COUNT(*) FROM ts_test")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != totalExpected {
		t.Errorf("expected %d rows, got %d", totalExpected, count)
	}

	// Verify data integrity: each goroutine's rows should be present
	for g := 0; g < numGoroutines; g++ {
		row := db.QueryRow(
			"SELECT COUNT(*) FROM ts_test WHERE goroutine_id = ?", g,
		)
		var gCount int
		if err := row.Scan(&gCount); err != nil {
			t.Errorf("count goroutine %d: %v", g, err)
			continue
		}
		if gCount != insertsPerGoroutine {
			t.Errorf("goroutine %d: expected %d rows, got %d", g, insertsPerGoroutine, gCount)
		}
	}
}

// TestIntegration_ThreadSafetyReads ensures concurrent reads work without blocking.
func TestIntegration_ThreadSafetyReads(t *testing.T) {
	if !duckdbAvailable() {
		t.Skip("DuckDB driver not available")
	}

	db := newTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), "CREATE TABLE read_test (id INTEGER, val FLOAT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	for i := 0; i < 100; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO read_test VALUES (?, ?)", i, float64(i)*1.5)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	const numReaders = 30
	var wg sync.WaitGroup
	errCh := make(chan error, numReaders)

	for g := 0; g < numReaders; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				rows, err := db.Query("SELECT id, val FROM read_test WHERE id = ?", i%100)
				if err != nil {
					errCh <- err
					return
				}
				for rows.Next() {
					var id int
					var val float64
					if scanErr := rows.Scan(&id, &val); scanErr != nil {
						rows.Close()
						errCh <- scanErr
						return
					}
				}
				rows.Close()
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}
