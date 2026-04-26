package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/google/uuid"
	"github.com/ff3300/aleph-v2/internal/storage"
)

type ComponentMetadata struct {
	ID                   string
	Name                 string
	Description          string
	Version              string
	Type                 string
	Category             string
	Source               string
	Status               string
	ApprovalStatus       string
	ConfigSchemaJSON     string
	ExecutionCommand     string
	DependenciesJSON     string
	InputSchemaJSON      string
	OutputSchemaJSON     string
	PromptTemplate       string
	ToolIdsJSON          string
	AvgCpuUsage          float64
	AvgMemoryMb          float64
	AvgExecTimeMs        float64
	AvgBrierScore        float64
	AvgLatencyMs         float64
	TrustScore           float64
	CreatedByAgentId     string
	CreationTimestamp    time.Time
	LastUpdatedTimestamp time.Time
}

type DuckDBRegistry struct {
	db     *sql.DB
	logger *slog.Logger
}

const createTableSQL = `CREATE TABLE IF NOT EXISTS components (
	id VARCHAR PRIMARY KEY,
	name VARCHAR,
	description VARCHAR,
	version VARCHAR,
	type VARCHAR,
	category VARCHAR,
	source VARCHAR,
	status VARCHAR,
	approval_status VARCHAR,
	config_schema_json VARCHAR,
	execution_command VARCHAR,
	dependencies_json VARCHAR,
	input_schema_json VARCHAR,
	output_schema_json VARCHAR,
	prompt_template VARCHAR,
	tool_ids_json VARCHAR,
	avg_cpu_usage DOUBLE DEFAULT 0,
	avg_memory_mb DOUBLE DEFAULT 0,
	avg_exec_time_ms DOUBLE DEFAULT 0,
	avg_brier_score DOUBLE DEFAULT 0,
	avg_latency_ms DOUBLE DEFAULT 0,
	trust_score DOUBLE DEFAULT 0,
	created_by_agent_id VARCHAR,
	creation_timestamp TIMESTAMP,
	last_updated_timestamp TIMESTAMP
)`

func NewDuckDBRegistry(dbPath string, logger *slog.Logger) (*DuckDBRegistry, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil { return nil, err }
	// Table managed by migrations/000001_init_schema.up.sql
	// _, err = db.Exec(createTableSQL)
	return &DuckDBRegistry{db: db, logger: logger}, err
}

func NewDuckDBRegistryFromDB(db *sql.DB, logger *slog.Logger) (*DuckDBRegistry, error) {
	// Table managed by migrations/000001_init_schema.up.sql
	// _, err := db.Exec(createTableSQL)
	return &DuckDBRegistry{db: db, logger: logger}, nil
}

func NewDuckDBRegistryFromDuckDB(d *storage.DuckDB, logger *slog.Logger) (*DuckDBRegistry, error) {
	return NewDuckDBRegistryFromDB(d.DB(), logger)
}

func (r *DuckDBRegistry) RegisterComponent(meta ComponentMetadata) (string, error) {
	if meta.ID == "" { meta.ID = uuid.New().String() }
	now := time.Now()
	if meta.CreationTimestamp.IsZero() { meta.CreationTimestamp = now }
	if meta.LastUpdatedTimestamp.IsZero() { meta.LastUpdatedTimestamp = now }
	_, err := r.db.Exec(`INSERT INTO components VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		meta.ID, meta.Name, meta.Description, meta.Version, meta.Type, meta.Category, meta.Source, meta.Status, meta.ApprovalStatus,
		meta.ConfigSchemaJSON, meta.ExecutionCommand, meta.DependenciesJSON, meta.InputSchemaJSON, meta.OutputSchemaJSON,
		meta.PromptTemplate, meta.ToolIdsJSON,
		meta.AvgCpuUsage, meta.AvgMemoryMb, meta.AvgExecTimeMs, meta.AvgBrierScore, meta.AvgLatencyMs,
		meta.TrustScore, meta.CreatedByAgentId, meta.CreationTimestamp, meta.LastUpdatedTimestamp)
	return meta.ID, err
}

func (r *DuckDBRegistry) GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error) {
	var c ComponentMetadata
	err := r.db.QueryRowContext(ctx, `SELECT id, name, description, version, type, category, source, status, approval_status,
		config_schema_json, execution_command, dependencies_json, input_schema_json, output_schema_json,
		prompt_template, tool_ids_json,
		avg_cpu_usage, avg_memory_mb, avg_exec_time_ms, avg_brier_score, avg_latency_ms,
		trust_score, created_by_agent_id, creation_timestamp, last_updated_timestamp
		FROM components WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.Version, &c.Type, &c.Category, &c.Source, &c.Status, &c.ApprovalStatus,
		&c.ConfigSchemaJSON, &c.ExecutionCommand, &c.DependenciesJSON, &c.InputSchemaJSON, &c.OutputSchemaJSON,
		&c.PromptTemplate, &c.ToolIdsJSON,
		&c.AvgCpuUsage, &c.AvgMemoryMb, &c.AvgExecTimeMs, &c.AvgBrierScore, &c.AvgLatencyMs,
		&c.TrustScore, &c.CreatedByAgentId, &c.CreationTimestamp, &c.LastUpdatedTimestamp)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *DuckDBRegistry) UpdateComponentStatus(id string, status string) error {
	_, err := r.db.Exec("UPDATE components SET status = ?, last_updated_timestamp = ? WHERE id = ?", status, time.Now(), id)
	return err
}

func (r *DuckDBRegistry) ListComponents(filter map[string]string) ([]ComponentMetadata, error) {
	rows, err := r.db.Query(`SELECT id, name, description, version, type, category, source, status, approval_status,
		config_schema_json, execution_command, dependencies_json, input_schema_json, output_schema_json,
		prompt_template, tool_ids_json,
		avg_cpu_usage, avg_memory_mb, avg_exec_time_ms, avg_brier_score, avg_latency_ms,
		trust_score, created_by_agent_id, creation_timestamp, last_updated_timestamp
		FROM components`)
	if err != nil { return nil, err }
	defer rows.Close()
	var comps []ComponentMetadata
	for rows.Next() {
		var c ComponentMetadata
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.Version, &c.Type, &c.Category, &c.Source, &c.Status, &c.ApprovalStatus,
			&c.ConfigSchemaJSON, &c.ExecutionCommand, &c.DependenciesJSON, &c.InputSchemaJSON, &c.OutputSchemaJSON,
			&c.PromptTemplate, &c.ToolIdsJSON,
			&c.AvgCpuUsage, &c.AvgMemoryMb, &c.AvgExecTimeMs, &c.AvgBrierScore, &c.AvgLatencyMs,
			&c.TrustScore, &c.CreatedByAgentId, &c.CreationTimestamp, &c.LastUpdatedTimestamp); err != nil {
			continue
		}
		comps = append(comps, c)
	}
	return comps, nil
}

func ParseToolIdsJSON(jsonStr string) []string {
	if jsonStr == "" { return nil }
	var ids []string
	json.Unmarshal([]byte(jsonStr), &ids)
	return ids
}
