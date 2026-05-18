package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OntologyRepository struct {
	db *sql.DB
}

func NewOntologyRepository(db *sql.DB) *OntologyRepository {
	return &OntologyRepository{db: db}
}

type OntologyVersionRecord struct {
	VersionID         string
	ProjectID         string
	ParentVersionID   sql.NullString
	DiffJSON          string
	CoreAlephSnapshot string
	Status            string
	SourceDescription sql.NullString
	Rationale         sql.NullString
	Confidence        sql.NullFloat64
	CreatedAt         time.Time
	ModifiedAt        sql.NullTime
}

func (r *OntologyRepository) ProposeOntologyDiff(ctx context.Context, projectID, parentVersionID, diffJSON, snapshot, sourceDesc, rationale string, confidence float64) (versionID string, err error) {
	versionID = uuid.New().String()
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO ontology_versions (version_id, project_id, parent_version_id, diff_json, core_aleph_snapshot, status, source_description, rationale, confidence) VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8)",
		versionID, projectID, parentVersionID, diffJSON, snapshot, sourceDesc, rationale, confidence,
	)
	if err != nil {
		return "", fmt.Errorf("proposeOntologyDiff: %w", err)
	}
	return versionID, nil
}

func (r *OntologyRepository) AcceptDiff(ctx context.Context, versionID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE ontology_versions SET status = 'accepted', modified_at = CURRENT_TIMESTAMP WHERE version_id = $1",
		versionID,
	)
	if err != nil {
		return fmt.Errorf("acceptDiff: %w", err)
	}
	return nil
}

func (r *OntologyRepository) RejectDiff(ctx context.Context, versionID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE ontology_versions SET status = 'rejected', rationale = $1, modified_at = CURRENT_TIMESTAMP WHERE version_id = $2",
		reason, versionID,
	)
	if err != nil {
		return fmt.Errorf("rejectDiff: %w", err)
	}
	return nil
}

func (r *OntologyRepository) ListVersions(ctx context.Context, projectID string, limit int) ([]OntologyVersionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		"SELECT version_id, project_id, parent_version_id, diff_json, core_aleph_snapshot, status, source_description, rationale, confidence, created_at, modified_at FROM ontology_versions WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2",
		projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listVersions: %w", err)
	}
	defer rows.Close()
	var versions []OntologyVersionRecord
	for rows.Next() {
		var v OntologyVersionRecord
		if err := rows.Scan(&v.VersionID, &v.ProjectID, &v.ParentVersionID, &v.DiffJSON, &v.CoreAlephSnapshot, &v.Status, &v.SourceDescription, &v.Rationale, &v.Confidence, &v.CreatedAt, &v.ModifiedAt); err != nil {
			return nil, fmt.Errorf("listVersions scan: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, nil
}
