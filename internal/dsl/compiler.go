package dsl

import (
	"fmt"
	"strings"
)

type Compiler struct {
	Program  *Program
	DataRoot string
}

func NewCompiler(p *Program, dataRoot string) *Compiler {
	return &Compiler{Program: p, DataRoot: dataRoot}
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

		// Full-Quote both table and field for absolute safety
		safeSource := fmt.Sprintf("\"%s\".\"%s\"", objName, sourceField)
		clause := fmt.Sprintf("%s AS \"%s\"", safeSource, prop.Name)

		// Predictive AI Enhancement: If 'predict' is set, add probability and embedding placeholders
		if prop.Predict {
			selectClauses = append(selectClauses, fmt.Sprintf("0.0 AS \"%s_probability\"", prop.Name))
			selectClauses = append(selectClauses, fmt.Sprintf("NULL AS \"%s_vector\"", prop.Name))
		}

		// Phase 2: Factor Decomposition Support
		for _, f := range obj.Factors {
			selectClauses = append(selectClauses, fmt.Sprintf("0.0 AS \"_factor_%s\"", f.Name)) // Placeholder
		}

		if len(prop.Maps) > 0 {
			caseExpr := "CASE " + safeSource
			for _, m := range prop.Maps {
				// Escape single quotes for SQL string literals
				safeFrom := strings.ReplaceAll(m.From, "'", "''")
				safeTo := strings.ReplaceAll(m.To, "'", "''")
				caseExpr += fmt.Sprintf(" WHEN '%s' THEN '%s'", safeFrom, safeTo)
			}
			caseExpr += " END"
			clause = fmt.Sprintf("%s AS \"%s\"", caseExpr, prop.Name)
		}

		selectClauses = append(selectClauses, clause)
	}

	// Basic Relation/JOIN support with full quoting
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
				joinClauses = append(joinClauses, fmt.Sprintf(
					" LEFT JOIN read_parquet('%s/%s/latest/*.parquet') AS \"%s\" ON \"%s\".\"%s\" = \"%s\".\"%s\"",
					c.DataRoot, targetObj.FromSource, stmt.Relation.To, 
					objName, stmt.Relation.LeftOn, stmt.Relation.To, stmt.Relation.RightOn,
				))
			}
		}
	}

	sql := fmt.Sprintf(
		"SELECT %s FROM read_parquet('%s/%s/latest/*.parquet') AS \"%s\" %s",
		strings.Join(selectClauses, ", "),
		c.DataRoot, obj.FromSource, objName,
		strings.Join(joinClauses, " "),
	)

	return sql, nil
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
