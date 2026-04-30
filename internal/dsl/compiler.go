package dsl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Compiler struct {
	Program  *Program
	DataRoot string
	// UseViews, when true, generates DuckDB views (CREATE OR REPLACE VIEW) instead
	// of inline read_parquet() calls. Set at construction or via SetUseViews().
	UseViews bool
}

var validFilterValue = regexp.MustCompile(`^[a-zA-Z0-9 _\-.%]+$`)

func NewCompiler(p *Program, dataRoot string) *Compiler {
	return &Compiler{Program: p, DataRoot: dataRoot}
}

// SetUseViews controls whether CompileObject generates view-based SQL.
func (c *Compiler) SetUseViews(useViews bool) {
	c.UseViews = useViews
}

func (c *Compiler) CompileObject(objName string) (string, error) {
	var obj *ObjectDefinition
	for _, stmt := range c.Program.Statements {
		if stmt.Object != nil && stmt.Object.Name == objName {
			obj = stmt.Object
			break
		}
	}

	if obj == nil {
		return "", fmt.Errorf("object %s not found", objName)
	}

	var selectClauses []string
	for _, prop := range obj.Properties {
		sourceField := prop.From
		if sourceField == "" {
			sourceField = prop.Name
		}

		safeSource := fmt.Sprintf("\"%s\".\"%s\"", objName, sourceField)
		clause := fmt.Sprintf("%s AS \"%s\"", safeSource, prop.Name)

		// Predictive AI Enhancement: If 'predict' is set, add probability and embedding placeholders
		if prop.Predict {
			selectClauses = append(selectClauses, fmt.Sprintf("0.0 AS \"%s_probability\"", prop.Name))
			selectClauses = append(selectClauses, fmt.Sprintf("NULL AS \"%s_vector\"", prop.Name))
		}

		// Phase 2: Factor Decomposition Support
		for _, f := range obj.Factors {
			selectClauses = append(selectClauses, fmt.Sprintf("0.0 AS \"_factor_%s\"", f.Name))
		}

		if len(prop.Maps) > 0 {
			caseExpr := "CASE " + safeSource
			for _, m := range prop.Maps {
				safeFrom := strings.ReplaceAll(m.From, "'", "''")
				safeTo := strings.ReplaceAll(m.To, "'", "''")
				caseExpr += fmt.Sprintf(" WHEN '%s' THEN '%s'", safeFrom, safeTo)
			}
			caseExpr += " END"
			clause = fmt.Sprintf("%s AS \"%s\"", caseExpr, prop.Name)
		}

		selectClauses = append(selectClauses, clause)
	}

	var joinClauses []string
	for _, stmt := range c.Program.Statements {
		if stmt.Relation != nil && stmt.Relation.From == objName {
			var targetObj *ObjectDefinition
			for _, s := range c.Program.Statements {
				if s.Object != nil && s.Object.Name == stmt.Relation.To {
					targetObj = s.Object
					break
				}
			}
			if targetObj != nil {
				var source string
				if c.UseViews {
					source = fmt.Sprintf("\"%s\"", stmt.Relation.To)
				} else {
					source = fmt.Sprintf("read_parquet('%s/%s/latest/*.parquet') AS \"%s\"",
						c.DataRoot, targetObj.FromSource, stmt.Relation.To)
				}
				joinClauses = append(joinClauses, fmt.Sprintf(
					" LEFT JOIN %s ON \"%s\".\"%s\" = \"%s\".\"%s\"",
					source,
					objName, stmt.Relation.LeftOn, stmt.Relation.To, stmt.Relation.RightOn,
				))
			}
		}
	}

	aggregateFields := make(map[string]bool)
	for _, agg := range obj.Aggregates {
		aggregateFields[agg.Field] = true
		sqlFunc := strings.ToUpper(agg.Function)
		selectClauses = append(selectClauses, fmt.Sprintf("%s(\"%s\".\"%s\") AS \"%s\"", sqlFunc, objName, agg.Field, agg.Alias))
	}

	var groupByClauses []string
	if len(obj.Aggregates) > 0 {
		for _, prop := range obj.Properties {
			if !aggregateFields[prop.Name] {
				groupByClauses = append(groupByClauses, fmt.Sprintf("\"%s\".\"%s\"", objName, prop.Name))
			}
		}
	}

	var whereClauses []string
	for _, f := range obj.Filters {
		opMap := map[string]string{
			"eq": "=", "neq": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<=", "like": "LIKE",
		}
		sqlOp := opMap[f.Op]
		val := f.Value
		if !isNumeric(val) {
			if !validFilterValue.MatchString(val) {
				return "", fmt.Errorf("invalid filter value: %q contains disallowed characters", val)
			}
			val = fmt.Sprintf("'%s'", strings.ReplaceAll(val, "'", "''"))
		}
		whereClauses = append(whereClauses, fmt.Sprintf("\"%s\".\"%s\" %s %s", objName, f.Field, sqlOp, val))
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	groupByClause := ""
	if len(groupByClauses) > 0 {
		groupByClause = " GROUP BY " + strings.Join(groupByClauses, ", ")
	}

	var fromSource string
	if c.UseViews {
		fromSource = fmt.Sprintf("\"%s\"", objName)
	} else {
		fromSource = fmt.Sprintf("read_parquet('%s/%s/latest/*.parquet') AS \"%s\"",
			c.DataRoot, obj.FromSource, objName)
	}

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s%s",
		strings.Join(selectClauses, ", "),
		fromSource,
		strings.Join(joinClauses, " "),
		whereClause,
		groupByClause,
	)

	return sql, nil
}

// CompileDDL generates CREATE OR REPLACE VIEW statements for all objects
// in the program, replacing inline read_parquet() calls with persistent views.
// Returns a slice of DDL statements that can be executed against DuckDB.
func (c *Compiler) CompileDDL() ([]string, error) {
	var ddls []string
	for _, stmt := range c.Program.Statements {
		if stmt.Object == nil {
			continue
		}
		obj := stmt.Object
		var selectClauses []string
		for _, prop := range obj.Properties {
			sourceField := prop.From
			if sourceField == "" {
				sourceField = prop.Name
			}
			safeSource := fmt.Sprintf("source.\"%s\"", sourceField)
			clause := fmt.Sprintf("%s AS \"%s\"", safeSource, prop.Name)
			selectClauses = append(selectClauses, clause)
		}
		ddl := fmt.Sprintf("CREATE OR REPLACE VIEW \"%s\" AS SELECT %s FROM read_parquet('%s/%s/latest/*.parquet') AS source",
			obj.Name,
			strings.Join(selectClauses, ", "),
			c.DataRoot,
			obj.FromSource,
		)
		ddls = append(ddls, ddl)
	}
	return ddls, nil
}

func (c *Compiler) CompileActions() ([]map[string]interface{}, error) {
	var actionTools []map[string]interface{}
	for _, stmt := range c.Program.Statements {
		if stmt.Action != nil {
			params := make(map[string]interface{})
			var required []string
			for _, p := range stmt.Action.Parameters {
				params[p.Name] = map[string]interface{}{
					"type": "string", // Semplificazione per LLM Tooling
				}
				required = append(required, p.Name)
			}
			
			tool := map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        stmt.Action.Name,
					"description": fmt.Sprintf("Execute action %s on object %s", stmt.Action.Name, stmt.Action.OnObject),
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": params,
						"required":   required,
					},
				},
			}
			actionTools = append(actionTools, tool)
		}
	}
	return actionTools, nil
}

func isNumeric(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}
