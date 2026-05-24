package manifest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ff3300/aleph-v2/internal/storage"
)

// Scanner implements SchemaScanner via DuckDB PRAGMA/SUMMARIZE introspection.
type Scanner struct {
	cfg DomainConfig
}

// NewScanner creates a Scanner configured with the given DomainConfig.
func NewScanner(cfg DomainConfig) *Scanner {
	return &Scanner{cfg: cfg}
}

// Scan introspects a DuckDB database via PRAGMA_show_tables,
// PRAGMA table_info, and SUMMARIZE, producing []TableSchema.
// Tables in cfg.IgnoreTables are skipped. If SUMMARIZE fails,
// it falls back to per-column COUNT(DISTINCT).
func (s *Scanner) Scan(ctx context.Context, db storage.DBExecutor) ([]TableSchema, error) {
	ignoreSet := make(map[string]bool, len(s.cfg.IgnoreTables))
	for _, t := range s.cfg.IgnoreTables {
		ignoreSet[strings.ToLower(t)] = true
	}

	tableNames, err := s.listTables(ctx, db, ignoreSet)
	if err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}

	var result []TableSchema
	for _, name := range tableNames {
		schema, err := s.scanTable(ctx, db, name)
		if err != nil {
			slog.Warn("scanner: skipping table", "table", name, "error", err)
			continue
		}
		result = append(result, schema)
	}

	return result, nil
}

func (s *Scanner) listTables(ctx context.Context, db storage.DBExecutor, ignoreSet map[string]bool) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT table_name FROM duckdb_tables() WHERE schema_name = 'main'")
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		if ignoreSet[strings.ToLower(name)] {
			continue
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}
	return names, nil
}

func (s *Scanner) scanTable(ctx context.Context, db storage.DBExecutor, name string) (TableSchema, error) {
	var rowCount int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, name)).Scan(&rowCount); err != nil {
		return TableSchema{}, fmt.Errorf("count %s: %w", name, err)
	}

	colInfos, err := s.getTableInfo(ctx, db, name)
	if err != nil {
		return TableSchema{}, fmt.Errorf("table_info %s: %w", name, err)
	}

	// Detect UNIQUE/PRIMARY KEY constraints from duckdb_constraints()
	// (PRAGMA table_info only reports explicit PRIMARY KEY columns;
	//  duckdb_constraints also catches UNIQUE column groups).
	constraintPKs, constraintErr := s.getConstraintPKs(ctx, db, name)
	if constraintErr != nil {
		slog.Warn("scanner: constraint detection failed",
			"table", name, "error", constraintErr)
	} else {
		for i := range colInfos {
			if constraintPKs[colInfos[i].name] {
				colInfos[i].isPK = true
			}
		}
	}

	summarizeStats, summarizeErr := s.getSummarize(ctx, db, name)
	if summarizeErr != nil {
		slog.Warn("scanner: SUMMARIZE failed, using COUNT(DISTINCT) fallback",
			"table", name, "error", summarizeErr)
	}

	columns := make([]ColumnSchema, 0, len(colInfos))
	for _, info := range colInfos {
		col := ColumnSchema{
			Name:     info.name,
			Type:     info.colType,
			Nullable: info.nullable,
			IsPK:     info.isPK,
			RowCount: rowCount,
		}

		if stats, ok := summarizeStats[info.name]; ok {
			col.DistinctCount = stats.approxUnique
		} else if summarizeErr != nil && rowCount > 0 {
			col.DistinctCount = s.countDistinct(ctx, db, name, info.name)
		}

		col.SampleValues = s.sampleValues(ctx, db, name, info.name)
		columns = append(columns, col)
	}

	return TableSchema{
		Name:     name,
		Columns:  columns,
		RowCount: rowCount,
	}, nil
}

type columnInfo struct {
	name     string
	colType  string
	nullable bool
	isPK     bool
}

// getConstraintPKs queries duckdb_constraints() for UNIQUE and PRIMARY KEY
// constraints on a table. Returns a set of column names that participate in
// any UNIQUE or PRIMARY KEY index, so columns defined via UNIQUE(col1, col2, ...)
// are also marked as PK for classification purposes.
func (s *Scanner) getConstraintPKs(ctx context.Context, db storage.DBExecutor, tableName string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT constraint_type, constraint_column_names
		 FROM duckdb_constraints()
		 WHERE table_name = $1 AND schema_name = 'main'
		   AND constraint_type IN ('PRIMARY KEY', 'UNIQUE')`,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var constraintType string
		var colNames any
		if err := rows.Scan(&constraintType, &colNames); err != nil {
			return nil, fmt.Errorf("scan constraint row: %w", err)
		}
		for _, name := range parseConstraintColumns(colNames) {
			result[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate constraints: %w", err)
	}
	return result, nil
}

// parseConstraintColumns converts duckdb_constraints().constraint_column_names
// (which may arrive as []any from go-duckdb) into a string slice.
func parseConstraintColumns(raw any) []string {
	switch v := raw.(type) {
	case []any:
		out := make([]string, len(v))
		for i, item := range v {
			out[i] = fmt.Sprint(item)
		}
		return out
	case string:
		return []string{v}
	default:
		return nil
	}
}

// getTableInfo executes PRAGMA table_info('table'), which returns columns:
//
//	cid | name | type | notnull | dflt_value | pk
//
// pk is a BOOLEAN (true when the column is part of the primary key).
func (s *Scanner) getTableInfo(ctx context.Context, db storage.DBExecutor, tableName string) ([]columnInfo, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []columnInfo
	for rows.Next() {
		var cid int
		var colName, colType string
		var notnull, pk bool
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info row: %w", err)
		}

		result = append(result, columnInfo{
			name:     colName,
			colType:  colType,
			nullable: !notnull,
			isPK:     pk,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info: %w", err)
	}
	return result, nil
}

type summaryRow struct {
	approxUnique int64
}

// getSummarize executes SUMMARIZE "table", which returns columns:
//
//	column_names | column_types | min | max | approx_unique | avg | std |
//	q25 | q50 | q75 | count | null_percentage
//
// If SUMMARIZE is unavailable (e.g., older DuckDB), the caller uses
// per-column COUNT(DISTINCT) as a fallback.
func (s *Scanner) getSummarize(ctx context.Context, db storage.DBExecutor, tableName string) (map[string]summaryRow, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SUMMARIZE "%s"`, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]summaryRow)
	for rows.Next() {
		var colName, colType string
		var min, max, avgStr, stdStr, q25, q50, q75 sql.NullString
		var approxUnique, count sql.NullInt64
		var nullPct sql.NullFloat64

		if err := rows.Scan(&colName, &colType, &min, &max, &approxUnique,
			&avgStr, &stdStr, &q25, &q50, &q75, &count, &nullPct); err != nil {
			return nil, fmt.Errorf("scan SUMMARIZE row: %w", err)
		}

		if approxUnique.Valid {
			result[colName] = summaryRow{approxUnique: approxUnique.Int64}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SUMMARIZE: %w", err)
	}
	return result, nil
}

func (s *Scanner) countDistinct(ctx context.Context, db storage.DBExecutor, tableName, colName string) int64 {
	query := fmt.Sprintf(`SELECT COUNT(DISTINCT "%s") FROM "%s"`, colName, tableName)
	var count int64
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		slog.Warn("scanner: COUNT(DISTINCT) fallback failed",
			"table", tableName, "column", colName, "error", err)
		return 0
	}
	return count
}

func (s *Scanner) sampleValues(ctx context.Context, db storage.DBExecutor, tableName, colName string) []any {
	query := fmt.Sprintf(`SELECT DISTINCT "%s" FROM "%s" LIMIT 5`, colName, tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		slog.Warn("scanner: sample values query failed",
			"table", tableName, "column", colName, "error", err)
		return nil
	}
	defer rows.Close()

	var samples []any
	for rows.Next() {
		var val any
		if err := rows.Scan(&val); err != nil {
			slog.Warn("scanner: scan sample value failed",
				"table", tableName, "column", colName, "error", err)
			break
		}
		samples = append(samples, val)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("scanner: iterate sample values failed",
			"table", tableName, "column", colName, "error", err)
	}
	return samples
}
