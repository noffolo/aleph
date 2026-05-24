package ingestion

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type Migration struct {
	Version int
	Name    string
	Up      string
}

type MigrationManager struct {
	db         *sql.DB
	migrations map[int]Migration
}

func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db, migrations: make(map[int]Migration)}
}

func (m *MigrationManager) Register(mig Migration) {
	m.migrations[mig.Version] = mig
}

func (m *MigrationManager) ensureTable() error {
	_, err := m.db.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INT PRIMARY KEY, name TEXT, applied_at TIMESTAMP)`,
	)
	return err
}

func (m *MigrationManager) Up() error {
	if err := m.ensureTable(); err != nil {
		return err
	}
	versions := make([]int, 0, len(m.migrations))
	for v := range m.migrations {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	for _, v := range versions {
		var exists int
		row := m.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", v)
		if err := row.Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		mig := m.migrations[v]
		if _, err := m.db.Exec(mig.Up); err != nil {
			return fmt.Errorf("migration v%d (%s) failed: %w", v, mig.Name, err)
		}
		if _, err := m.db.Exec(
			"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
			v, mig.Name, time.Now(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (m *MigrationManager) CurrentVersion() (int, error) {
	if err := m.ensureTable(); err != nil {
		return 0, err
	}
	var v sql.NullInt64
	row := m.db.QueryRow("SELECT MAX(version) FROM schema_migrations")
	if err := row.Scan(&v); err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}
