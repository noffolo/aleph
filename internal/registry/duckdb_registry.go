package registry

import (
	"database/sql"
	"log/slog"
	"time"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/google/uuid"
)

type ComponentMetadata struct {
	ID                 string
	Name               string
	Description        string
	Version            string
	Type               string
	Category           string
	Source             string
	Status             string
	ApprovalStatus     string
	CreationTimestamp  time.Time
}

type DuckDBRegistry struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewDuckDBRegistry(dbPath string, logger *slog.Logger) (*DuckDBRegistry, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil { return nil, err }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS components (id VARCHAR PRIMARY KEY, name VARCHAR, description VARCHAR, version VARCHAR, type VARCHAR, category VARCHAR, source VARCHAR, status VARCHAR, approval_status VARCHAR, creation_timestamp TIMESTAMP)`)
	return &DuckDBRegistry{db: db, logger: logger}, err
}

func (r *DuckDBRegistry) RegisterComponent(meta ComponentMetadata) (string, error) {
	if meta.ID == "" { meta.ID = uuid.New().String() }
	_, err := r.db.Exec("INSERT INTO components VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", 
        meta.ID, meta.Name, meta.Description, meta.Version, meta.Type, meta.Category, meta.Source, meta.Status, meta.ApprovalStatus, time.Now())
	return meta.ID, err
}

func (r *DuckDBRegistry) ListComponents(filter map[string]string) ([]ComponentMetadata, error) {
	rows, err := r.db.Query("SELECT id, name, type FROM components")
	if err != nil { return nil, err }
	defer rows.Close()
	var comps []ComponentMetadata
	for rows.Next() {
		var c ComponentMetadata
		rows.Scan(&c.ID, &c.Name, &c.Type)
		comps = append(comps, c)
	}
	return comps, nil
}
