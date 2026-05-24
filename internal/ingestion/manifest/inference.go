package manifest

import (
	"sort"
	"strings"
)

type entityInferrer struct {
	cfg DomainConfig
}

// NewEntityInferrer creates a new EntityInferrer with the given domain configuration.
func NewEntityInferrer(cfg DomainConfig) EntityInferrer {
	return &entityInferrer{cfg: cfg}
}

// Infer groups classified columns into entity definitions, skipping junction
// tables (2+FK, 0 PK, ≤3 cols), ignore tables, and tables with <2 columns.
// Table names are singularized using Italian plural→singular rules and results
// are sorted deterministically by entity Name.
func (e *entityInferrer) Infer(tables []TableSchema) ([]Entity, error) {
	ignoreSet := make(map[string]bool, len(e.cfg.IgnoreTables))
	for _, t := range e.cfg.IgnoreTables {
		ignoreSet[t] = true
	}

	var entities []Entity

	for _, table := range tables {
		if ignoreSet[table.Name] {
			continue
		}
		if len(table.Columns) < 2 {
			continue
		}
		if isJunctionTable(table.Columns) {
			continue
		}

		keyCol := firstPKColumn(table.Columns)
		labelCol := firstLabelColumn(table.Columns)
		if labelCol == "" {
			labelCol = firstNonPKVarchar(table.Columns)
		}

		entities = append(entities, Entity{
			Name:        singularize(table.Name),
			Table:       table.Name,
			KeyColumn:   keyCol,
			LabelColumn: labelCol,
			Properties:  collectProperties(table.Columns),
		})
	}

	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Name < entities[j].Name
	})

	return entities, nil
}

func isJunctionTable(cols []ColumnSchema) bool {
	if len(cols) > 3 {
		return false
	}
	fkCount, pkCount := 0, 0
	for _, col := range cols {
		if col.IsFK {
			fkCount++
		}
		if col.IsPK {
			pkCount++
		}
	}
	return fkCount >= 2 && pkCount == 0
}

func firstPKColumn(cols []ColumnSchema) string {
	for _, col := range cols {
		if col.Class == PrimaryKey {
			return col.Name
		}
	}
	return ""
}

func firstLabelColumn(cols []ColumnSchema) string {
	for _, col := range cols {
		if col.Class == Label {
			return col.Name
		}
	}
	return ""
}

func firstNonPKVarchar(cols []ColumnSchema) string {
	for _, col := range cols {
		if col.IsPK {
			continue
		}
		upper := strings.ToUpper(col.Type)
		if strings.Contains(upper, "VARCHAR") ||
			strings.Contains(upper, "TEXT") ||
			strings.Contains(upper, "CHAR") ||
			strings.Contains(upper, "STRING") {
			return col.Name
		}
	}
	return ""
}

func collectProperties(cols []ColumnSchema) []ColumnSchema {
	var props []ColumnSchema
	for _, col := range cols {
		switch col.Class {
		case Category, Temporal, Boolean, Coordinate:
			props = append(props, col)
		}
	}
	return props
}

// singularize applies Italian plural→singular rules to a table name.
//
//	-i suffix → -o  (deputati→deputato, partiti→partito, sondaggi→sondaggio)
//	-e suffix → -a  (tasse→tassa)
//	otherwise → unchanged
func singularize(name string) string {
	if name == "" {
		return name
	}
	if strings.HasSuffix(name, "ggi") {
		return name[:len(name)-1] + "o"
	}
	if strings.HasSuffix(name, "ti") {
		return name[:len(name)-1] + "o"
	}
	if strings.HasSuffix(name, "i") {
		return name[:len(name)-1] + "o"
	}
	if strings.HasSuffix(name, "e") {
		return name[:len(name)-1] + "a"
	}
	return name
}
