package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunDuckDBMigrations runs database migrations for DuckDB.
// dsn: DuckDB connection string (e.g., "aleph.db" or ":memory:")
// migrationsPath: path to migrations directory (e.g., "migrations")
func RunDuckDBMigrations(dsn string, migrationsPath string) error {
	log.Printf("Running DuckDB migrations on %s from %s", dsn, migrationsPath)
	
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	// Create schema_migrations table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version bigint PRIMARY KEY,
		dirty boolean NOT NULL DEFAULT false
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	var currentVersion int64
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE NOT dirty").Scan(&currentVersion)
	if err != nil {
		currentVersion = 0
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var highestVersion int64 = 0
	upFiles := make(map[int64]string)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.Split(name, "_")
			if len(parts) > 0 {
				var version int64
				_, err := fmt.Sscanf(parts[0], "%d", &version)
				if err == nil && version > currentVersion {
					upFiles[version] = filepath.Join(migrationsPath, name)
					if version > highestVersion {
						highestVersion = version
					}
				}
			}
		}
	}

	for version := currentVersion + 1; version <= highestVersion; version++ {
		if upFile, ok := upFiles[version]; ok {
			content, err := os.ReadFile(upFile)
			if err != nil {
				return fmt.Errorf("read migration file %s: %w", upFile, err)
			}

			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("begin transaction for version %d: %w", version, err)
			}

			_, err = tx.Exec(string(content))
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("execute migration version %d: %w", version, err)
			}

			_, err = tx.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, false)", version)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("record migration version %d: %w", version, err)
			}

			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("commit migration version %d: %w", version, err)
			}

			log.Printf("Applied migration version %d", version)
		}
	}

	if highestVersion > currentVersion {
		log.Printf("Database migrations complete. Applied %d migration(s)", highestVersion-currentVersion)
	} else {
		log.Printf("Database migrations up-to-date. No migrations applied.")
	}

	return nil
}

// RunPostgresMigrations runs database migrations for PostgreSQL.
// dsn: PostgreSQL connection string (e.g., "postgres://user:pass@localhost:5432/db")
// migrationsPath: path to migrations directory (e.g., "migrations/postgres")
func RunPostgresMigrations(dsn string, migrationsPath string) error {
	log.Printf("Running PostgreSQL migrations on %s from %s", dsn, migrationsPath)
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	// Create schema_migrations table if it doesn't exist (PostgreSQL version)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version bigint PRIMARY KEY,
		dirty boolean NOT NULL DEFAULT false
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	var currentVersion int64
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE NOT dirty").Scan(&currentVersion)
	if err != nil {
		currentVersion = 0
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var highestVersion int64 = 0
	upFiles := make(map[int64]string)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.Split(name, "_")
			if len(parts) > 0 {
				var version int64
				_, err := fmt.Sscanf(parts[0], "%d", &version)
				if err == nil && version > currentVersion {
					upFiles[version] = filepath.Join(migrationsPath, name)
					if version > highestVersion {
						highestVersion = version
					}
				}
			}
		}
	}

	for version := currentVersion + 1; version <= highestVersion; version++ {
		if upFile, ok := upFiles[version]; ok {
			content, err := os.ReadFile(upFile)
			if err != nil {
				return fmt.Errorf("read migration file %s: %w", upFile, err)
			}

			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("begin transaction for version %d: %w", version, err)
			}

			_, err = tx.Exec(string(content))
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("execute migration version %d: %w", version, err)
			}

			_, err = tx.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, false)", version)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("record migration version %d: %w", version, err)
			}

			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("commit migration version %d: %w", version, err)
			}

			log.Printf("Applied PostgreSQL migration version %d", version)
		}
	}

	if highestVersion > currentVersion {
		log.Printf("PostgreSQL migrations complete. Applied %d migration(s)", highestVersion-currentVersion)
	} else {
		log.Printf("PostgreSQL migrations up-to-date. No migrations applied.")
	}

	return nil
}

// RunAllMigrations runs migrations for both DuckDB and PostgreSQL.
// duckdbDSN: DuckDB connection string
// postgresDSN: PostgreSQL connection string
func RunAllMigrations(duckdbDSN, postgresDSN string) error {
	// Run DuckDB migrations first
	if err := RunDuckDBMigrations(duckdbDSN, "migrations/duckdb"); err != nil {
		log.Printf("Warning: DuckDB migrations failed: %v", err)
		// Continue without migrations for backward compatibility
	}

	// Run PostgreSQL migrations
	if err := RunPostgresMigrations(postgresDSN, "migrations/postgres"); err != nil {
		log.Printf("Warning: PostgreSQL migrations failed: %v", err)
		// Continue without migrations for backward compatibility
	}

	return nil
}