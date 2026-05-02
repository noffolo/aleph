# SPEC-04: Injection Hardening — SQL, DSL, DSN, Template Injection

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 1, tasks W1-1, W1-2, W1-9  
**Findings addressed**: I1-I6 (injection cluster)  
**Depends on**: `docs/specs/wave0-auth-spec.md` (context propagation from auth)  
**Related specs**: `docs/specs/wave1-sandbox-spec.md` (DSL validation feeds sandbox execution)  
**Status**: ✅ Approved — ready for execution

---

## 1. SQL Injection Vectors — Remediation Map

### Vector Matrix

| File | Line | Current Pattern | Risk | Fix |
|------|------|----------------|------|-----|
| `repository/metadata.go` | 709-726 | `fmt.Sprintf("DELETE FROM %s", table)` | CRITICAL | Hardcoded table name array |
| `storage/duckdb.go` | 159-170 | `scopeQuery` with fmt.Sprintf for schema | CRITICAL | Add schema name validation |
| `storage/context.go` | 32 | `SanitizeProjectID` — already validates | LOW | Verify callers |
| `ingestion/engine.go` | 796-797 | `resolveTableName(task.Id)` | HIGH | Validate with UUID parser |
| `ingestion/engine.go` | 808-809 | `fmt.Sprintf` for temp dir | LOW | Use `os.MkdirTemp` prefix |
| `repository/metadata.go` | 364 | `fmt.Sprintf` for cache key | LOW | Validate input |

### Fix Specification

#### V1: DeleteProjectCascade Table Names

```go
// BEFORE (vulnerable)
var tables = []string{
    "system_agents", "system_skills", "system_tasks",
    "system_api_keys", "system_notification_channels",
    "system_chat_history", "system_chat_sessions",
    "system_ontology_versions", "system_projects",
}
for _, table := range tables {
    query := fmt.Sprintf("DELETE FROM %s WHERE project_id = $1", table)
    // ⚠️ table name from variable, not validated
}

// AFTER (hardened)
type validatedTable struct {
    tableName string
}
var safeTables = []validatedTable{
    {"system_agents"}, {"system_skills"}, {"system_tasks"},
    {"system_api_keys"}, {"system_notification_channels"},
    {"system_chat_history"}, {"system_chat_sessions"},
    {"system_ontology_versions"}, {"system_projects"},
}

func (v validatedTable) validate() error {
    // All table names are hardcoded — they're safe by construction
    // No user input can reach this list
    return nil
}
for _, table := range safeTables {
    // Table name is compiler-verified constant
    query := "DELETE FROM " + table.tableName + " WHERE project_id = $1"
}
```

#### V2: scopeQuery Schema Validation

```go
// BEFORE 
func scopeQuery(query, projectID string) string {
    return fmt.Sprintf("FROM \"%s\".\"%s\"", projectID, tableName)
    // ⚠️ projectID from user input (though SanitizeProjectID should catch)
}

// AFTER
func scopeQuery(query, tableName string, schema SchemaIdentity) (string, error) {
    if err := schema.Validate(); err != nil {
        return "", fmt.Errorf("invalid schema: %w", err)
    }
    // schema.Name is guaranteed safe by Validate()
    return fmt.Sprintf("FROM \"%s\".\"%s\"", schema.Name, tableName), nil
}
```

#### V3: resolveTableName Validation

```go
// AFTER
func resolveTableName(taskID string) (string, error) {
    // Parse as UUID to prevent injection
    if _, err := uuid.Parse(taskID); err != nil {
        return "", fmt.Errorf("invalid task ID: must be UUID v4, got %q", taskID)
    }
    return "t_" + strings.ReplaceAll(taskID, "-", "_"), nil
}
```

---

## 2. DSL Injection Prevention

### Compiler Tool (`internal/dsl/compiler_tool.go`)

**Audit scope**: All template interpolation that reaches SQL execution.

#### Safe by Construction (no fix needed)

- Template variables (`__NAME__`, `__DESCRIPTION__`) are replaced before code generation
- These are validated during DSL parsing phase

#### Potentially Unsafe (must fix)

```go
// apiConnectorPythonTemplate: urllib.request import
// FIX: Remove from template. Generated tools must use SSRF-guarded HTTP client.
const apiConnectorPythonTemplate = `
# python
"""__DESCRIPTION__"""
import json
import sys
from typing import Any
# REMOVED: from urllib.request import Request, urlopen
# REMOVED: from urllib.error import URLError
# ADDED: Use safe_http_client (SSRF-guarded)
`
```

### Parser Validation (`internal/dsl/parser.go`)

Add input validation before parsing:

```go
func ValidateDSLInput(input string) error {
    // Max length
    if len(input) > 100_000 {
        return fmt.Errorf("DSL input exceeds max length (100KB)")
    }
    
    // No null bytes
    if strings.Contains(input, "\x00") {
        return fmt.Errorf("DSL input contains null byte")
    }
    
    // No raw SQL injection patterns
    dangerousPatterns := []string{
        "DROP TABLE", "DROP SCHEMA", "TRUNCATE",
        "CREATE TABLE", "ALTER TABLE",
        "INSERT INTO", "UPDATE ", "DELETE FROM",
        "GRANT ", "REVOKE ",
    }
    upper := strings.ToUpper(input)
    for _, p := range dangerousPatterns {
        if strings.Contains(upper, p) {
            return fmt.Errorf("DSL input contains forbidden pattern: %q", p)
        }
    }
    
    return nil
}
```

---

## 3. DSN Construction Security

### Current (Vulnerable)

```go
// internal/repository/metadata.go
dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
    user, password, host, port, dbname, sslmode)
db, err := sql.Open("postgres", dsn)
// ⚠️ If any variable contains URL special chars, DSN is mangled
```

### Fix: Parsed Config

```go
import "github.com/lib/pq"

func buildDSN(cfg DBConfig) (string, error) {
    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
    )
    
    // Validate by parsing
    _, err := pq.ParseURL(connStr)
    if err != nil {
        return "", fmt.Errorf("invalid DSN: %w", err)
    }
    
    return connStr, nil
}
```

---

## 4. Audit Checklist

All `fmt.Sprintf` calls constructing SQL must be audited:

```bash
# Find all sprintf + SQL patterns
grep -rn "fmt.Sprintf.*\(SELECT\|INSERT\|UPDATE\|DELETE\|CREATE\|DROP\|ALTER\|FROM\|WHERE\|SET\|SCHEMA\|TABLE\)" internal/ --include="*.go" | grep -v "_test.go"

# Expected result after fix: 0 matches
```

### Verified Safe Patterns

- `fmt.Sprintf("FROM \"%s\".\"%s\"", ...)` where first arg is from `SanitizeProjectID` ✅
- Hardcoded `DELETE FROM system_agents WHERE project_id = $1` ✅ 
- `fmt.Sprintf` for column names from DuckDB `information_schema` ✅ (system metadata, not user input)

### Verified Unsafe Patterns (must fix)

- Any `fmt.Sprintf` with user-supplied table name → parameterize or validate
- Any `fmt.Sprintf` with user-supplied schema name → use `SanitizeProjectID`
- Any template interpolation reaching SQL in DSL compiler → validate during parsing

---

## 5. Verification

### Test Coverage

- [ ] `duckdb_test.go`: 1000 random inputs against `SanitizeProjectID` → no bypass
- [ ] `injection_fuzz_test.go` (NEW): SQL injection payloads against all query paths → all rejected
- [ ] `dsl_validation_test.go` (NEW): DSL input with DROP TABLE → rejected; with null byte → rejected
- [ ] `dsn_test.go` (NEW): DSN with special characters → correctly parsed or rejected

### Manual Verification

```bash
# Project ID: SQL injection attempt
curl -X POST http://localhost:8080/api/v1/projects \
  -d '{"name":"test","project_id":"test\"; DROP TABLE system_projects;--"}'
# → 400 Bad Request (SanitizeProjectID rejects)

# DSL: SQL injection in description
# Submit tool definition with description containing DROP TABLE
# → Parser rejects during validation

# DSN: special chars in config
# Set DB_PASSWORD to "pass'word"
# → DSN is correctly escaped (not broken)
```
