package storage

import (
	"context"
	"fmt"
	"regexp"
)

var validProjectID = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,128}$`)

type schemaCtxKey string

const schemaKey schemaCtxKey = "duckdb_schema"

// ContextWithSchema returns a context containing the schema name for
// DuckDB operations. Used by schema-aware query methods to SET schema
// before execution.
// WARNING: schema must be validated via SanitizeProjectID before calling this.
func ContextWithSchema(ctx context.Context, schema string) context.Context {
	return context.WithValue(ctx, schemaKey, schema)
}

// SchemaFromContext extracts the schema name from the context, if set.
func SchemaFromContext(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(schemaKey).(string)
	return s, ok
}

// SanitizeProjectID validates that a projectID matches the allowed pattern
// [a-zA-Z0-9_.:-]{1,128} and returns an error if it doesn't.
// This prevents SQL injection via schema names in SET schema, CREATE SCHEMA,
// and other schema-qualified queries that cannot use parameterized queries.
func SanitizeProjectID(projectID string) error {
	if !validProjectID.MatchString(projectID) {
		return fmt.Errorf("invalid projectID: must match [a-zA-Z0-9_.:-]{1,128}")
	}
	return nil
}

// EnsureProjectSchema creates a DuckDB schema for the given project if it doesn't exist.
// The schema name must already be validated via SanitizeProjectID.
func EnsureProjectSchema(d *DuckDB, schema string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))
	return err
}

// scopeQuery wraps a query with a SET schema statement from context.
// If no schema is set in context, the original query is returned as-is.
// The schema from context is assumed to already be validated.
func scopeQuery(ctx context.Context, query string) string {
	schema, ok := SchemaFromContext(ctx)
	if !ok || schema == "" {
		return query
	}
	return fmt.Sprintf("SET schema = '%s'; %s", schema, query)
}
