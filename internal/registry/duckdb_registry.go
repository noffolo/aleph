package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"log/slog"
	"strings"
	"sync"
	"time"
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
	mu     sync.RWMutex
}

func NewDuckDBRegistry(dbPath string, logger *slog.Logger) (*DuckDBRegistry, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}
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
	if meta.ID == "" {
		meta.ID = uuid.New().String()
	}
	now := time.Now()
	if meta.CreationTimestamp.IsZero() {
		meta.CreationTimestamp = now
	}
	if meta.LastUpdatedTimestamp.IsZero() {
		meta.LastUpdatedTimestamp = now
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var exists int
	err := r.db.QueryRow(`SELECT 1 FROM components WHERE id = ?`, meta.ID).Scan(&exists)
	if err == nil {
		return "", fmt.Errorf("registerComponent: duplicate id %s", meta.ID)
	}

	_, err = r.db.Exec(`INSERT INTO components VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		meta.ID, meta.Name, meta.Description, meta.Version, meta.Type, meta.Category, meta.Source, meta.Status, meta.ApprovalStatus,
		meta.ConfigSchemaJSON, meta.ExecutionCommand, meta.DependenciesJSON, meta.InputSchemaJSON, meta.OutputSchemaJSON,
		meta.PromptTemplate, meta.ToolIdsJSON,
		meta.AvgCpuUsage, meta.AvgMemoryMb, meta.AvgExecTimeMs, meta.AvgBrierScore, meta.AvgLatencyMs,
		meta.TrustScore, meta.CreatedByAgentId, meta.CreationTimestamp, meta.LastUpdatedTimestamp)
	return meta.ID, err
}

func (r *DuckDBRegistry) GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec("UPDATE components SET status = ?, last_updated_timestamp = ? WHERE id = ?", status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("updateComponentStatus: %w", err)
	}
	return nil
}

func (r *DuckDBRegistry) ListComponents(filter map[string]string) ([]ComponentMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var whereClauses []string
	var args []any
	for key, value := range filter {
		if value != "" {
			whereClauses = append(whereClauses, key+" = ?")
			args = append(args, value)
		}
	}

	query := `SELECT id, name, description, version, type, category, source, status, approval_status,
		config_schema_json, execution_command, dependencies_json, input_schema_json, output_schema_json,
		prompt_template, tool_ids_json,
		avg_cpu_usage, avg_memory_mb, avg_exec_time_ms, avg_brier_score, avg_latency_ms,
		trust_score, created_by_agent_id, creation_timestamp, last_updated_timestamp
		FROM components`
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
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

func (r *DuckDBRegistry) DeleteComponent(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.ExecContext(ctx, `DELETE FROM components WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleteComponent: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleteComponent: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("deleteComponent: component %s not found", id)
	}
	return nil
}

func (r *DuckDBRegistry) UpdateComponent(ctx context.Context, meta ComponentMetadata) error {
	if meta.ID == "" {
		return fmt.Errorf("updateComponent: id must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	result, err := r.db.ExecContext(ctx, `UPDATE components SET
		name = ?, description = ?, version = ?, type = ?, category = ?,
		source = ?, status = ?, approval_status = ?,
		config_schema_json = ?, execution_command = ?, dependencies_json = ?,
		input_schema_json = ?, output_schema_json = ?, prompt_template = ?,
		tool_ids_json = ?,
		avg_cpu_usage = ?, avg_memory_mb = ?, avg_exec_time_ms = ?,
		avg_brier_score = ?, avg_latency_ms = ?, trust_score = ?,
		created_by_agent_id = ?, last_updated_timestamp = ?
		WHERE id = ?`,
		meta.Name, meta.Description, meta.Version, meta.Type, meta.Category,
		meta.Source, meta.Status, meta.ApprovalStatus,
		meta.ConfigSchemaJSON, meta.ExecutionCommand, meta.DependenciesJSON,
		meta.InputSchemaJSON, meta.OutputSchemaJSON, meta.PromptTemplate,
		meta.ToolIdsJSON,
		meta.AvgCpuUsage, meta.AvgMemoryMb, meta.AvgExecTimeMs,
		meta.AvgBrierScore, meta.AvgLatencyMs, meta.TrustScore,
		meta.CreatedByAgentId, now,
		meta.ID,
	)
	if err != nil {
		return fmt.Errorf("updateComponent: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updateComponent: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("updateComponent: component %s not found", meta.ID)
	}
	return nil
}

type Registry interface {
	RegisterComponent(meta ComponentMetadata) (string, error)
	GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error)
	UpdateComponentStatus(id string, status string) error
	ListComponents(filter map[string]string) ([]ComponentMetadata, error)
	DeleteComponent(ctx context.Context, id string) error
	UpdateComponent(ctx context.Context, meta ComponentMetadata) error
}

func ParseToolIdsJSON(jsonStr string) []string {
	if jsonStr == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		slog.Warn("failed to parse tool IDs JSON", "error", err)
		return nil
	}
	return ids
}
