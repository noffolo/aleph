package migrate

import (
	"testing"
)

func TestRunDuckDBMigrations(t *testing.T) {
	// Use in-memory database for testing
	// migrations directory is at project root, so use relative path from internal/migrate
	err := RunDuckDBMigrations(":memory:", "../../migrations/duckdb")
	if err != nil {
		t.Errorf("RunDuckDBMigrations failed: %v", err)
	}

	// Run again (should be idempotent)
	err = RunDuckDBMigrations(":memory:", "../../migrations/duckdb")
	if err != nil {
		t.Errorf("Second RunDuckDBMigrations failed: %v", err)
	}
}