// Package safeident provides strict SQL identifier validation and file path
// sanitization to prevent SQL injection in dynamically-constructed queries.
//
// DuckDB and most SQL engines do not support parameterized identifiers
// (table names, column names, schema names). When identifiers must be
// interpolated into SQL strings, they MUST be validated by
// ValidateIdentifier before use.
//
// File paths used in DuckDB functions like read_csv_auto() cannot be
// parameterized either. Use SanitizeFilePath before interpolation.
package safeident

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// identifierPattern matches valid SQL identifiers: start with a letter or
// underscore, followed by letters, digits, or underscores, max 64 chars.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)

// sqlKeywordBlacklist contains SQL keywords that must never be used as
// identifiers to prevent injection via keyword confusion.
var sqlKeywordBlacklist = map[string]bool{
	// DML/DDL
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
	"DROP": true, "CREATE": true, "ALTER": true, "TRUNCATE": true,
	"REPLACE": true, "RENAME": true,
	// DML clauses (common injection targets)
	"FROM": true, "WHERE": true, "SET": true, "INTO": true, "VALUES": true,
	// DCL
	"GRANT": true, "REVOKE": true,
	// Flow control
	"EXEC": true, "EXECUTE": true, "CALL": true,
	// DDL objects
	"TABLE": true, "VIEW": true, "INDEX": true, "DATABASE": true,
	"SCHEMA": true, "FUNCTION": true, "PROCEDURE": true,
	"TRIGGER": true, "SEQUENCE": true,
	// Utility / dangerous
	"COPY": true, "ATTACH": true, "DETACH": true,
	"LOAD": true, "INSTALL": true, "UNINSTALL": true,
	"EXPORT": true, "IMPORT": true,
	// Subquery / expression
	"UNION": true, "INTERSECT": true, "EXCEPT": true,
	// DuckDB-specific
	"PRAGMA": true, "SUMMARIZE": true, "DESCRIBE": true,
	// Transaction
	"COMMIT": true, "ROLLBACK": true, "BEGIN": true,
	// Boolean
	"TRUE": true, "FALSE": true, "NULL": true,
	// DuckDB extension
	"ORDER": true, "GROUP": true, "HAVING": true, "LIMIT": true,
}

// ValidateIdentifier checks that a string is a safe SQL identifier.
//
// Rules:
//   - Must match ^[a-zA-Z_][a-zA-Z0-9_]*$
//   - Maximum 64 characters
//   - Must not be a SQL keyword (case-insensitive)
//   - Must not contain SQL special characters (;, --, ', ", /*, */)
//
// Returns nil if valid, or a descriptive error if not.
func ValidateIdentifier(id string) error {
	if id == "" {
		return fmt.Errorf("identifier must not be empty")
	}
	if len(id) > 64 {
		return fmt.Errorf("identifier too long (%d chars, max 64): %q", len(id), id)
	}
	if !identifierPattern.MatchString(id) {
		return fmt.Errorf("identifier contains invalid characters: %q (must match ^[a-zA-Z_][a-zA-Z0-9_]*$)", id)
	}
	// Check for SQL special characters (defense-in-depth, the regex already
	// excludes most of these, but we check explicitly for clarity)
	if strings.Contains(id, ";") || strings.Contains(id, "--") ||
		strings.Contains(id, "'") || strings.Contains(id, "\"") ||
		strings.Contains(id, "/*") || strings.Contains(id, "*/") {
		return fmt.Errorf("identifier contains SQL special characters: %q", id)
	}
	// Keyword blacklist check (case-insensitive)
	upper := strings.ToUpper(id)
	if sqlKeywordBlacklist[upper] {
		return fmt.Errorf("identifier is a reserved SQL keyword: %q", id)
	}
	return nil
}

// ValidateColumnName is a slightly looser check for column names that may
// come from JSON keys. Same rules as ValidateIdentifier but does NOT
// reject SQL keywords (since column names like "table" are valid in
// quoted position, e.g. "table").
func ValidateColumnName(name string) error {
	if name == "" {
		return fmt.Errorf("column name must not be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("column name too long (%d chars, max 64): %q", len(name), name)
	}
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("column name contains invalid characters: %q", name)
	}
	if strings.Contains(name, ";") || strings.Contains(name, "--") ||
		strings.Contains(name, "'") || strings.Contains(name, "\"") ||
		strings.Contains(name, "/*") || strings.Contains(name, "*/") {
		return fmt.Errorf("column name contains SQL special characters: %q", name)
	}
	return nil
}

// SanitizeFilePath validates a file path for safe use in DuckDB functions
// like read_csv_auto(), read_json_auto(), read_parquet().
//
// Rules:
//   - Must not contain path traversal (..)
//   - Must be a clean path (filepath.Clean must not change it)
//   - Must not contain shell metacharacters: ; | & $ ` ( ) < > \n \r
//   - Must not contain single quotes (SQL string delimiter)
//   - Must be an absolute path or a relative path within the project
func SanitizeFilePath(p string) error {
	if p == "" {
		return fmt.Errorf("file path must not be empty")
	}
	cleaned := filepath.Clean(p)
	if cleaned != p {
		return fmt.Errorf("file path is not clean (contains . or redundant separators): %q", p)
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("file path contains traversal: %q", p)
	}
	for _, forbidden := range ";|&$`()<>'\n\r" {
		if strings.ContainsRune(p, forbidden) {
			return fmt.Errorf("file path contains forbidden character %q: %q", forbidden, p)
		}
	}
	return nil
}

// QuoteIdentifier wraps a pre-validated identifier in double quotes for safe
// SQL interpolation. Caller MUST call ValidateIdentifier first.
func QuoteIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

// QuoteStringLiteral wraps a pre-validated string in single quotes for safe
// SQL interpolation in DuckDB functions (read_csv_auto, etc.).
// Caller MUST call SanitizeFilePath first. Prefer parameterized queries.
func QuoteStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}