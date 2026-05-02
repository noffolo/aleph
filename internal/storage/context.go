package storage

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ff3300/aleph-v2/internal/safeident"
)

var validProjectID = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// SchemaIdentity is a validated schema identifier for DuckDB operations.
// Always construct via NewSchemaIdentity which enforces the identifier policy.
type SchemaIdentity struct {
	schema string
}

// NewSchemaIdentity validates and wraps a raw schema string into a
// SchemaIdentity. Returns an error if the schema name is invalid.
func NewSchemaIdentity(schema string) (SchemaIdentity, error) {
	if err := SanitizeProjectID(schema); err != nil {
		return SchemaIdentity{}, fmt.Errorf("schema identity: %w", err)
	}
	return SchemaIdentity{schema: schema}, nil
}

// MustNewSchemaIdentity panics on invalid schema — use only in tests.
func MustNewSchemaIdentity(schema string) SchemaIdentity {
	si, err := NewSchemaIdentity(schema)
	if err != nil {
		panic(err)
	}
	return si
}

func (si SchemaIdentity) String() string { return si.schema }

func (si SchemaIdentity) Quoted() string { return safeident.QuoteIdentifier(si.schema) }

func (si SchemaIdentity) Validate() error { return SanitizeProjectID(si.schema) }

type schemaCtxKey string

const schemaKey schemaCtxKey = "duckdb_schema"

// ContextWithSchema returns a context containing the schema name for
// DuckDB operations. Accepts a validated SchemaIdentity so that callers
// cannot inject unvalidated strings into the context.
func ContextWithSchema(ctx context.Context, si SchemaIdentity) context.Context {
	return context.WithValue(ctx, schemaKey, si.schema)
}

// SchemaFromContext extracts the schema name from the context, if set.
func SchemaFromContext(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(schemaKey).(string)
	return s, ok
}

// SanitizeProjectID validates that a projectID matches the allowed pattern
// [a-zA-Z_][a-zA-Z0-9_-]* and is at most 128 characters.
// This prevents SQL injection via schema names in SET schema, CREATE SCHEMA,
// and other schema-qualified queries that cannot use parameterized queries.
func SanitizeProjectID(projectID string) error {
	if len(projectID) > 128 {
		return fmt.Errorf("invalid projectID: too long (%d chars, max 128)", len(projectID))
	}
	if !validProjectID.MatchString(projectID) {
		return fmt.Errorf("invalid projectID: must match [a-zA-Z_][a-zA-Z0-9_-]*")
	}
	return nil
}

// EnsureProjectSchema creates a DuckDB schema for the given project if it doesn't exist.
// Serialized via writeMu with other DDL operations.
// Accepts a pre-validated SchemaIdentity.
// Uses context.TODO() — DDL operations at this level are self-contained and
// should be refactored to accept a caller-provided context.
func EnsureProjectSchema(d *DuckDB, si SchemaIdentity) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_, err := d.db.ExecContext(context.TODO(), fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, si.Quoted()))
	if err != nil {
		return fmt.Errorf("ensureProjectSchema: %w", err)
	}
	return nil
}

// DropProjectSchema drops the DuckDB schema for a project.
// Serialized via writeMu with other DDL operations.
// Accepts a pre-validated SchemaIdentity.
// Uses context.TODO() — see EnsureProjectSchema note.
func DropProjectSchema(d *DuckDB, si SchemaIdentity) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_, err := d.db.ExecContext(context.TODO(), fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, si.Quoted()))
	if err != nil {
		return fmt.Errorf("dropProjectSchema: %w", err)
	}
	return nil
}

// scopeQuery wraps a query with a SET schema statement from context.
// If no schema is set in context, the original query is returned as-is.
// The schema from context is re-validated before interpolation as a
// defense-in-depth measure.
func scopeQuery(ctx context.Context, query string) string {
	schema, ok := SchemaFromContext(ctx)
	if !ok || schema == "" {
		return query
	}
	if err := SanitizeProjectID(schema); err != nil {
		return query
	}
	return fmt.Sprintf("SET schema = %s; %s", safeident.QuoteIdentifier(schema), query)
}
