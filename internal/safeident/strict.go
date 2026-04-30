// Package safeident provides strict SQL identifier validation and file path
// sanitization to prevent SQL injection in dynamically-constructed queries.
//
// ValidateStrictIdentifier is an alias for ValidateIdentifier — it enforces
// the same rules (identifier pattern, max 64 chars, SQL keyword blacklist,
// no special characters). It exists as a separately-named function so that
// callers in ingestion and query handlers can explicitly document the use
// of the strictest validation level.
//
// ValidateStrictIdentifier MUST be called before any identifier is
// interpolated into a SQL string (even when using QuoteIdentifier), because
// SQL identifiers cannot be parameterized.
package safeident

// ValidateStrictIdentifier validates that a string is a safe SQL identifier.
//
// Rules are identical to ValidateIdentifier:
//   - Must match ^[a-zA-Z_][a-zA-Z0-9_]*$
//   - Maximum 64 characters
//   - Must not be a SQL keyword (case-insensitive)
//   - Must not contain SQL special characters (;, --, ', ", /*, */)
//
// Returns nil if valid, or a descriptive error if not.
func ValidateStrictIdentifier(id string) error {
	return ValidateIdentifier(id)
}
