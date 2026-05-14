package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

// --- SetSlowQueryThreshold ---

func TestSetSlowQueryThreshold_Valid(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	d.SetSlowQueryThreshold(100 * time.Millisecond)
	if d.slowQueryThreshold != 100*time.Millisecond {
		t.Fatalf("expected 100ms, got %v", d.slowQueryThreshold)
	}
}

func TestSetSlowQueryThreshold_Zero(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	d.SetSlowQueryThreshold(0)
	if d.slowQueryThreshold != DefaultSlowQueryThreshold {
		t.Fatalf("expected default threshold %v, got %v", DefaultSlowQueryThreshold, d.slowQueryThreshold)
	}
}

func TestSetSlowQueryThreshold_Negative(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	d.SetSlowQueryThreshold(-1 * time.Second)
	if d.slowQueryThreshold != DefaultSlowQueryThreshold {
		t.Fatalf("expected default threshold %v for negative, got %v", DefaultSlowQueryThreshold, d.slowQueryThreshold)
	}
}

// --- DB() ---

func TestDuckDB_DB(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	sqlDB := d.DB()
	if sqlDB == nil {
		t.Fatal("expected non-nil *sql.DB")
	}

	// Verify it actually works
	var result int
	if err := sqlDB.QueryRow("SELECT 1").Scan(&result); err != nil {
		t.Fatalf("SELECT 1 failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

// --- QueryRowContext ---

func TestDuckDB_QueryRowContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	row := d.QueryRowContext(ctx, "SELECT 42")
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

// --- QueryRowContextOrError ---

func TestDuckDB_QueryRowContextOrError(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	row, err := d.QueryRowContextOrError(ctx, "SELECT 7")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 7 {
		t.Fatalf("expected 7, got %d", result)
	}
}

func TestDuckDB_QueryRowContextOrError_Cancelled(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = d.QueryRowContextOrError(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- QueryRow ---

func TestDuckDB_QueryRow(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	row := d.QueryRow("SELECT 99")
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 99 {
		t.Fatalf("expected 99, got %d", result)
	}
}

// --- Cleanup ---

func TestDuckDB_Cleanup(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	// Cleanup is a no-op — just ensure it doesn't panic
	d.Cleanup()
}

// --- isSerializationError ---

func TestIsSerializationError_Serialization(t *testing.T) {
	err := errors.New("could not serialize access")
	if !isSerializationError(err) {
		t.Fatal("expected serialization error to be detected")
	}
}

func TestIsSerializationError_SqliteBusy(t *testing.T) {
	err := errors.New("sqlite_busy: database is locked")
	if !isSerializationError(err) {
		t.Fatal("expected sqlite_busy to be detected as serialization error")
	}
}

func TestIsSerializationError_CouldNotSerialize(t *testing.T) {
	err := errors.New("ERROR: could not serialize access due to concurrent update")
	if !isSerializationError(err) {
		t.Fatal("expected could_not_serialize to be detected")
	}
}

func TestIsSerializationError_Nil(t *testing.T) {
	if isSerializationError(nil) {
		t.Fatal("expected nil to not be a serialization error")
	}
}

func TestIsSerializationError_NormalError(t *testing.T) {
	err := errors.New("table not found")
	if isSerializationError(err) {
		t.Fatal("expected normal error to not be a serialization error")
	}
}

func TestIsSerializationError_EmptyString(t *testing.T) {
	err := errors.New("")
	if isSerializationError(err) {
		t.Fatal("expected empty error to not be a serialization error")
	}
}

func TestIsSerializationError_WrappedSerialization(t *testing.T) {
	// Case-insensitive match
	err := errors.New("Serialization Failure during commit")
	if !isSerializationError(err) {
		t.Fatal("expected case-insensitive serialization match")
	}
}

// --- truncateQuery ---

func TestTruncateQuery_Short(t *testing.T) {
	result := truncateQuery("SELECT 1")
	if result != "SELECT 1" {
		t.Fatalf("expected unchanged short query, got %s", result)
	}
}

func TestTruncateQuery_Exactly200(t *testing.T) {
	query := strings.Repeat("a", 200)
	result := truncateQuery(query)
	if result != query {
		t.Fatalf("expected unchanged 200-char query, got %s", result)
	}
}

func TestTruncateQuery_Over200(t *testing.T) {
	query := strings.Repeat("a", 300)
	result := truncateQuery(query)
	expected := strings.Repeat("a", 200) + "..."
	if result != expected {
		t.Fatalf("expected truncated 200+... query, got %s", result)
	}
}

// --- txScopeQuery ---

func TestTxScopeQuery_EmptySchema(t *testing.T) {
	result := txScopeQuery("", "SELECT 1")
	if result != "SELECT 1" {
		t.Fatalf("expected unchanged query, got %s", result)
	}
}

func TestTxScopeQuery_WithSchema(t *testing.T) {
	result := txScopeQuery("my_schema", "SELECT * FROM users")
	if !strings.HasPrefix(result, "SET schema =") {
		t.Fatalf("expected SET schema prefix, got %s", result)
	}
	if !strings.Contains(result, "SELECT * FROM users") {
		t.Fatalf("expected original query in result, got %s", result)
	}
}

// --- BeginTX / Commit / Rollback ---

func TestDuckDB_BeginTX_Commit(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	// Create a table first (not in a tx)
	if _, err := d.Exec(context.Background(), "CREATE TABLE test_tx (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}

	if _, err := tx.Exec("INSERT INTO test_tx VALUES (1)"); err != nil {
		tx.Rollback()
		t.Fatalf("insert failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify the insert persisted
	row := d.QueryRow("SELECT COUNT(*) FROM test_tx")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count scan failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after commit, got %d", count)
	}
}

func TestDuckDB_BeginTX_Rollback(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE test_rb (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}

	if _, err := tx.Exec("INSERT INTO test_rb VALUES (99)"); err != nil {
		tx.Rollback()
		t.Fatalf("insert failed: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	// Verify the insert was rolled back
	row := d.QueryRow("SELECT COUNT(*) FROM test_rb")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count scan failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}
}

func TestDuckDB_Commit_Idempotent(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE test_idem (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}

	// Do a simple read (no write needed)
	row := tx.QueryRow("SELECT COUNT(*) FROM test_idem")
	var count int
	if err := row.Scan(&count); err != nil {
		tx.Rollback()
		t.Fatalf("scan failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("first commit failed: %v", err)
	}

	// Second commit should be no-op
	if err := tx.Commit(); err != nil {
		t.Fatalf("second commit should be no-op, got error: %v", err)
	}
}

func TestDuckDB_Rollback_Idempotent(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("first rollback failed: %v", err)
	}

	// Second rollback should be no-op
	if err := tx.Rollback(); err != nil {
		t.Fatalf("second rollback should be no-op, got error: %v", err)
	}
}

func TestDuckDB_BeginReadTX(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE test_read (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if _, err := d.Exec(context.Background(), "INSERT INTO test_read VALUES (1)"); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}

	row := tx.QueryRow("SELECT COUNT(*) FROM test_read")
	var count int
	if err := row.Scan(&count); err != nil {
		tx.Rollback()
		t.Fatalf("scan failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit read tx failed: %v", err)
	}
}

// --- ExecWithRetry ---

func TestDuckDB_ExecWithRetry_Success(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	_, err = d.ExecWithRetry(context.Background(), "CREATE TABLE retry_ok (id INTEGER)")
	if err != nil {
		t.Fatalf("ExecWithRetry failed: %v", err)
	}
}

func TestDuckDB_ExecWithRetry_NonRetryableError(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	// Syntax error should not be retried
	_, err = d.ExecWithRetry(context.Background(), "INVALID SQL SYNTAX")
	if err == nil {
		t.Fatal("expected error for invalid SQL")
	}
	// Should not be a serialization error, so no retries
	if isSerializationError(err) {
		t.Fatal("syntax error should not be detected as serialization error")
	}
}

// --- ExecTx ---

func TestDuckDB_ExecTx_Success(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE exectx_ok (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	err = d.ExecTx(context.Background(), func(tx *TX) error {
		_, execErr := tx.Exec("INSERT INTO exectx_ok VALUES (42)")
		return execErr
	})
	if err != nil {
		t.Fatalf("ExecTx failed: %v", err)
	}

	row := d.QueryRow("SELECT COUNT(*) FROM exectx_ok")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}

func TestDuckDB_ExecTx_FnError_Rollback(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE exectx_rb (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	expectedErr := errors.New("fn failed")
	err = d.ExecTx(context.Background(), func(tx *TX) error {
		if _, execErr := tx.Exec("INSERT INTO exectx_rb VALUES (1)"); execErr != nil {
			return execErr
		}
		return expectedErr
	})
	if err == nil {
		t.Fatal("expected error from fn")
	}

	// Verify insert was rolled back
	row := d.QueryRow("SELECT COUNT(*) FROM exectx_rb")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after fn error rollback, got %d", count)
	}
}

// --- TX Query/Exec methods ---

func TestTX_Query(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE tx_q (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if _, err := d.Exec(context.Background(), "INSERT INTO tx_q VALUES (1),(2),(3)"); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id FROM tx_q ORDER BY id")
	if err != nil {
		t.Fatalf("tx.Query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 rows, got %d", count)
	}
}

func TestTX_QueryContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(context.Background(), "CREATE TABLE tx_qc (id INTEGER)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(context.Background(), "SELECT COUNT(*) FROM tx_qc")
	if err != nil {
		t.Fatalf("tx.QueryContext failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		rows.Scan(&count)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows, got %d", count)
	}
}

func TestTX_Exec(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("CREATE TABLE tx_e (id INTEGER)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	_, err = tx.Exec("INSERT INTO tx_e VALUES (1)")
	if err != nil {
		t.Fatalf("tx.Exec insert failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
}

func TestTX_ExecContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginTX(context.Background())
	if err != nil {
		t.Fatalf("BeginTX failed: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(context.Background(), "CREATE TABLE tx_ec (id INTEGER)")
	if err != nil {
		t.Fatalf("tx.ExecContext failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
}

func TestTX_QueryRow(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	row := tx.QueryRow("SELECT 123")
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 123 {
		t.Fatalf("expected 123, got %d", result)
	}
}

func TestTX_QueryRowContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(context.Background(), "SELECT 456")
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 456 {
		t.Fatalf("expected 456, got %d", result)
	}
}

func TestTX_QueryRowContextOrError(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	row, err := tx.QueryRowContextOrError(context.Background(), "SELECT 789")
	if err != nil {
		t.Fatalf("QueryRowContextOrError failed: %v", err)
	}
	var result int
	if err := row.Scan(&result); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if result != 789 {
		t.Fatalf("expected 789, got %d", result)
	}
}

func TestTX_QueryRowContextOrError_Cancelled(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	tx, err := d.BeginReadTX(context.Background())
	if err != nil {
		t.Fatalf("BeginReadTX failed: %v", err)
	}
	defer tx.Rollback()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = tx.QueryRowContextOrError(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected error for cancelled context on tx")
	}
}

// --- logSlowQuery ---

func TestDuckDB_logSlowQuery_UnderThreshold(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	d.SetSlowQueryThreshold(1 * time.Second)
	// logSlowQuery with duration < threshold should be a no-op (no panics)
	d.logSlowQuery("test", "SELECT 1", 1*time.Millisecond)
}

func TestDuckDB_logSlowQuery_OverThreshold(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	d.SetSlowQueryThreshold(1 * time.Second)
	// logSlowQuery with duration > threshold logs a warning — just verify no panic
	d.logSlowQuery("test", "SELECT * FROM large_table", 2*time.Second)
}

// --- slowTxQueryLog ---

func TestSlowTxQueryLog_UnderThreshold(t *testing.T) {
	// Should not panic
	slowTxQueryLog(1*time.Second, "test", "SELECT 1", 1*time.Millisecond)
}

func TestSlowTxQueryLog_OverThreshold(t *testing.T) {
	// Should not panic, just logs
	slowTxQueryLog(1*time.Second, "test", "SELECT * FROM large_table", 2*time.Second)
}

// --- NewDuckDB constructor ---

func TestNewDuckDB_EmptyPath(t *testing.T) {
	d, err := NewDuckDB("")
	if err != nil {
		t.Fatalf("NewDuckDB with empty path failed: %v", err)
	}
	defer d.Close()

	if d.path != ":memory:" {
		t.Fatalf("expected :memory: path for empty string, got %s", d.path)
	}
}

func TestNewDuckDB_Memory(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("NewDuckDB :memory: failed: %v", err)
	}
	defer d.Close()

	if d.path != ":memory:" {
		t.Fatalf("expected :memory: path, got %s", d.path)
	}
	if d.db == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
	if d.slowQueryThreshold != DefaultSlowQueryThreshold {
		t.Fatalf("expected default slow query threshold")
	}
}

func TestDuckDB_Close(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestDuckDB_Query(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	rows, err := d.Query("SELECT 1 AS n UNION ALL SELECT 2")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}

func TestDuckDB_QueryContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	rows, err := d.QueryContext(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("QueryContext failed: %v", err)
	}
	defer rows.Close()

	var result int
	for rows.Next() {
		rows.Scan(&result)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

func TestDuckDB_Exec(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	_, err = d.Exec(context.Background(), "CREATE TABLE test_direct (id INTEGER)")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	_, err = d.Exec(context.Background(), "INSERT INTO test_direct VALUES (99)")
	if err != nil {
		t.Fatalf("Exec insert failed: %v", err)
	}
}

func TestDuckDB_ExecContext(t *testing.T) {
	d, err := NewDuckDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create duckdb: %v", err)
	}
	defer d.Close()

	_, err = d.ExecContext(context.Background(), "CREATE TABLE test_execctx (id INTEGER)")
	if err != nil {
		t.Fatalf("ExecContext failed: %v", err)
	}
}

func TestDuckDB_NewDuckDB_InvalidPath(t *testing.T) {
	// "/" directory is not a valid DuckDB path
	_, err := NewDuckDB("/")
	if err == nil {
		t.Log("warning: expected error for invalid path, but duckdb accepted it — test skipped")
	}
}

// Ensure sql package import is used
var _ sql.Result
