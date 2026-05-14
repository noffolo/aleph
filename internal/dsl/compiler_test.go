package dsl

import (
	"testing"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestCompileObject(t *testing.T) {
	input := `
		object AppaltoPubblico
		from dataset anac_bandi_cig
		id cig
		property cig type identifier from CIG
		property tipo_procedura type enum from TIPO_SCELTA_CONTRAENTE
			map "01" to "aperta"
			map "02" to "ristretta"
	`
	dataRoot := "/tmp/aleph-data"
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("AppaltoPubblico")

	assert.NoError(t, err)
	assert.Contains(t, sql, "SELECT")
	assert.Contains(t, sql, `"AppaltoPubblico"."CIG" AS "cig"`)
	assert.Contains(t, sql, `CASE "AppaltoPubblico"."TIPO_SCELTA_CONTRAENTE" WHEN '01' THEN 'aperta' WHEN '02' THEN 'ristretta' END AS "tipo_procedura"`)
	assert.Contains(t, sql, "read_parquet('/tmp/aleph-data/anac_bandi_cig/latest/*.parquet')")
}

func TestCompileWithRelation(t *testing.T) {
	input := `
		object Appalto
		from dataset d1
		id id1
		property id1 type text

		object Azienda
		from dataset d2
		id id2
		property id2 type text

		relation Fornitore from Appalto to Azienda on p_id equals id2
	`
	dataRoot := "/data"
	program, _ := Parse(input)
	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.Contains(t, sql, `LEFT JOIN read_parquet('/data/d2/latest/*.parquet') AS "Azienda"`)
	assert.Contains(t, sql, `ON "Appalto"."p_id" = "Azienda"."id2"`)
}

func TestCompileWithFilter(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
		property importo type float
		filter importo gt 10000
	`
	dataRoot := "/tmp/aleph-data"
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.Contains(t, sql, `WHERE "Appalto"."importo" > 10000`)
	assert.NotContains(t, sql, "GROUP BY")
}

func TestCompileWithAggregate(t *testing.T) {
	input := `
		object Statistiche
		from dataset bandi
		id cig
		property tipo type text
		aggregate sum(importo) as totale_importo
	`
	dataRoot := "/tmp/aleph-data"
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("Statistiche")

	assert.NoError(t, err)
	assert.Contains(t, sql, `SUM("Statistiche"."importo") AS "totale_importo"`)
	assert.Contains(t, sql, `GROUP BY "Statistiche"."tipo"`)
}

func TestSQLInjection_ObjectName(t *testing.T) {
	tests := []struct {
		name    string
		objName string
	}{
		{"semicolon", "obj; DROP TABLE"},
		{"single_quote", "obj'table"},
		{"double_quote", `obj"table`},
		{"sql_keyword", "DROP TABLE users"},
		{"comment_attempt", "obj--"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataRoot := "/tmp/aleph-data"
			obj := &ObjectDefinition{
				Name:       "SafeObj",
				FromSource: "safe_source",
				ID:         "id",
				Properties: []*Property{
					{Name: "col", Type: "text", From: "col"},
				},
			}
			prog := &Program{
				Statements: []*Statement{
					{Object: obj},
				},
			}
			compiler := NewCompiler(prog, dataRoot)
			sql, err := compiler.CompileObject(tt.objName)
			if err != nil {
				return
			}
			assert.NotContains(t, sql, ";")
			assert.NotContains(t, sql, "'")
			assert.NotContains(t, sql, `"`+tt.objName+`"`)
		})
	}
}

func TestSQLInjection_PropertyFrom(t *testing.T) {
	tests := []struct {
		name       string
		propName   string
		propFrom   string
		expectErr  bool
	}{
		{"valid_normal", "cig", "CIG", false},
		{"valid_underscore", "my_col", "my_col", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataRoot := "/tmp/aleph-data"
			obj := &ObjectDefinition{
				Name:       "TestObj",
				FromSource: "test_source",
				ID:         "id",
				Properties: []*Property{
					{Name: tt.propName, Type: "text", From: tt.propFrom},
				},
			}
			prog := &Program{
				Statements: []*Statement{
					{Object: obj},
				},
			}
			compiler := NewCompiler(prog, dataRoot)
			sql, err := compiler.CompileObject("TestObj")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, sql, `"TestObj"."`+tt.propFrom+`"`)
			}
		})
	}
}

func TestSQLInjection_FilterField(t *testing.T) {
	tests := []struct {
		name      string
		filterOn  string
		expectErr bool
	}{
		{"valid_field", "importo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &Program{
				Statements: []*Statement{
					{Object: &ObjectDefinition{
						Name:       "SafeObj",
						FromSource: "src",
						ID:         "id",
						Properties: []*Property{{Name: "importo", Type: "float"}},
						Filters: []*FilterDefinition{
							{Field: tt.filterOn, Op: "gt", Value: "100"},
						},
					}},
				},
			}
			compiler := NewCompiler(prog, "/d")
			sql, err := compiler.CompileObject("SafeObj")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, sql, `"SafeObj"."`+tt.filterOn+`"`)
			}
		})
	}
}

func TestSQLInjection_FilterValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid_string", "aperta", false},
		{"valid_number", "12345", false},
		{"valid_percent", "test%", false},
		{"invalid_semicolon", "val; DROP", true},
		{"invalid_quote", "val'test", true},
		{"invalid_traversal", "../etc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &Program{
				Statements: []*Statement{
					{Object: &ObjectDefinition{
						Name:       "SafeObj",
						FromSource: "src",
						ID:         "id",
						Properties: []*Property{{Name: "col", Type: "text"}},
						Filters: []*FilterDefinition{
							{Field: "col", Op: "eq", Value: tt.value},
						},
					}},
				},
			}
			compiler := NewCompiler(prog, "/d")
			_, err := compiler.CompileObject("SafeObj")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLInjection_FromSource(t *testing.T) {
	tests := []struct {
		name       string
		fromSource string
		expectErr  bool
	}{
		{"valid_source", "bandi", false},
		{"valid_path_chars", "my_dataset_2024", false},
		{"path_traversal", "../etc/passwd", true},
		{"shell_injection", "dataset;rm -rf", true},
		{"single_quote", "test'table", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &Program{
				Statements: []*Statement{
					{Object: &ObjectDefinition{
						Name:       "SafeObj",
						FromSource: tt.fromSource,
						ID:         "id",
						Properties: []*Property{{Name: "col", Type: "text"}},
					}},
				},
			}
			compiler := NewCompiler(prog, "/d")
			_, err := compiler.CompileObject("SafeObj")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLInjection_AggregateField(t *testing.T) {
	tests := []struct {
		name      string
		aggField  string
		aggAlias  string
		expectErr bool
	}{
		{"valid", "importo", "totale", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &Program{
				Statements: []*Statement{
					{Object: &ObjectDefinition{
						Name:       "SafeObj",
						FromSource: "src",
						ID:         "id",
						Properties: []*Property{{Name: "col", Type: "text"}},
						Aggregates: []*AggregateDefinition{
							{Function: "sum", Field: tt.aggField, Alias: tt.aggAlias},
						},
					}},
				},
			}
			compiler := NewCompiler(prog, "/d")
			sql, err := compiler.CompileObject("SafeObj")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, sql, `SUM("SafeObj".`)
			}
		})
	}
}

func TestSQLInjection_RelationFields(t *testing.T) {
	prog := &Program{
		Statements: []*Statement{
			{Object: &ObjectDefinition{
				Name: "Left", FromSource: "s1", ID: "id1",
				Properties: []*Property{{Name: "id1", Type: "text"}},
			}},
			{Object: &ObjectDefinition{
				Name: "Right", FromSource: "s2", ID: "id2",
				Properties: []*Property{{Name: "id2", Type: "text"}},
			}},
			{Relation: &RelationDefinition{
				Name: "Rel", From: "Left", To: "Right",
				LeftOn: "id1", RightOn: "id2",
			}},
		},
	}
	compiler := NewCompiler(prog, "/d")
	sql, err := compiler.CompileObject("Left")
	assert.NoError(t, err)
	assert.Contains(t, sql, `LEFT JOIN`)
	assert.Contains(t, sql, `read_parquet('/d/s2/latest/*.parquet')`)
}

func TestCompileWithFilterAndAggregate(t *testing.T) {
	input := `
		object Statistiche
		from dataset bandi
		id cig
		property tipo type text
		filter tipo eq "aperta"
		aggregate sum(importo) as totale_importo
	`
	dataRoot := "/tmp/aleph-data"
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("Statistiche")

	assert.NoError(t, err)
	assert.Contains(t, sql, `WHERE "Statistiche"."tipo" = 'aperta'`)
	assert.Contains(t, sql, `SUM("Statistiche"."importo") AS "totale_importo"`)
	assert.Contains(t, sql, `GROUP BY "Statistiche"."tipo"`)
}

func TestCompileDDL_SingleObject(t *testing.T) {
	input := `
		object Bandi
		from dataset anac_bandi
		id cig
		property cig type text from CIG
		property importo type float
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	ddls, err := compiler.CompileDDL()

	assert.NoError(t, err)
	assert.Len(t, ddls, 1)
	ddl := ddls[0]
	assert.Contains(t, ddl, `CREATE OR REPLACE VIEW "Bandi" AS`)
	assert.Contains(t, ddl, `read_parquet('/data/anac_bandi/latest/*.parquet')`)
	assert.Contains(t, ddl, `source."CIG" AS "cig"`)
}

func TestCompileDDL_MultipleObjects(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		object Azienda
		from dataset imprese
		id piva
		property piva type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	ddls, err := compiler.CompileDDL()

	assert.NoError(t, err)
	assert.Len(t, ddls, 2)
}

func TestCompileDDL_SkipsNonObjects(t *testing.T) {
	input := `
		dataset raw_data version 1 from source_x

		object Metrics
		from dataset analytics
		id metric_id
		property metric_id type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	ddls, err := compiler.CompileDDL()

	assert.NoError(t, err)
	assert.Len(t, ddls, 1)
}

func TestCompileDDL_InvalidFromSource(t *testing.T) {
	prog := &Program{
		Statements: []*Statement{
			{Object: &ObjectDefinition{
				Name:       "BadObj",
				FromSource: "../etc/passwd",
				ID:         "id",
				Properties: []*Property{{Name: "col", Type: "text"}},
			}},
		},
	}
	compiler := NewCompiler(prog, "/data")
	_, err := compiler.CompileDDL()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data source")
}

func TestCompileActions_Single(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		action send_alert on Appalto
			property message type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	actions, err := compiler.CompileActions()

	assert.NoError(t, err)
	assert.Len(t, actions, 1)
	action := actions[0]
	assert.Equal(t, "function", action["type"])

	fn := action["function"].(map[string]interface{})
	assert.Equal(t, "send_alert", fn["name"])
	assert.Contains(t, fn["description"], "Appalto")
}

func TestCompileActions_NoActions(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	actions, err := compiler.CompileActions()

	assert.NoError(t, err)
	assert.Empty(t, actions)
}

func TestCompileActions_MultipleActions(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		action alert on Appalto
			property msg type text

		action export on Appalto
			property format type text
			property dest type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	actions, err := compiler.CompileActions()

	assert.NoError(t, err)
	assert.Len(t, actions, 2)
}

func TestCompileObject_UseViews(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	compiler.SetUseViews(true)
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.Contains(t, sql, `FROM "Appalto"`)
	assert.NotContains(t, sql, "read_parquet")
}

func TestCompileObject_UseViews_WithRelation(t *testing.T) {
	input := `
		object Appalto
		from dataset d1
		id id1
		property id1 type text

		object Azienda
		from dataset d2
		id id2
		property id2 type text

		relation R from Appalto to Azienda on id1 equals id2
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	compiler.SetUseViews(true)
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.Contains(t, sql, `LEFT JOIN "Azienda"`)
	assert.NotContains(t, sql, "read_parquet")
}

func TestCompileObject_NotFound(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	_, err = compiler.CompileObject("NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCompileObject_WithPredict(t *testing.T) {
	input := `
		object Forecast
		from dataset metrics
		id ts
		property value type float predict
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Forecast")

	assert.NoError(t, err)
	assert.Contains(t, sql, `"value"_probability`)
	assert.Contains(t, sql, `"value"_vector`)
}

func TestCompileObject_WithFactors(t *testing.T) {
	input := `
		object Analysis
		from dataset metrics
		id ts
		property value type float
		factor seasonality type time from ts
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Analysis")

	assert.NoError(t, err)
	assert.Contains(t, sql, `"_factor_seasonality"`)
}

func TestCompileObject_MultipleFilters(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
		property importo type float
		property tipo type text
		filter importo gt 10000
		filter tipo eq "aperta"
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.Contains(t, sql, `"Appalto"."importo" > 10000`)
	assert.Contains(t, sql, `"Appalto"."tipo" = 'aperta'`)
	assert.Contains(t, sql, " AND ")
}

func TestCompileObject_StringFilterLike(t *testing.T) {
	input := `
		object Products
		from dataset catalog
		id sku
		property sku type text
		property name type text
		filter name like "test%"
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Products")

	assert.NoError(t, err)
	assert.Contains(t, sql, `LIKE 'test%'`)
}

func TestCompileObject_NumericFilterNoQuotes(t *testing.T) {
	input := `
		object Stats
		from dataset metrics
		id id
		property id type int
		property score type float
		filter score lte 0.5
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Stats")

	assert.NoError(t, err)
	assert.Contains(t, sql, `<= 0.5`)
	assert.NotContains(t, sql, `'0.5'`)
}

func TestCompileObject_RelationToMissingObject(t *testing.T) {
	input := `
		object Appalto
		from dataset d1
		id id1
		property id1 type text

		relation R from Appalto to MissingObj on id1 equals id2
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	compiler := NewCompiler(program, "/data")
	sql, err := compiler.CompileObject("Appalto")

	assert.NoError(t, err)
	assert.NotContains(t, sql, "LEFT JOIN")
}

func TestValidateDSLInput_Valid(t *testing.T) {
	err := ValidateDSLInput("object Appalto from dataset bandi id cig")
	assert.NoError(t, err)
}

func TestValidateDSLInput_TooLarge(t *testing.T) {
	largeInput := make([]byte, maxDSLInputSize+1)
	for i := range largeInput {
		largeInput[i] = 'x'
	}
	err := ValidateDSLInput(string(largeInput))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestValidateDSLInput_NullBytes(t *testing.T) {
	err := ValidateDSLInput("object Appalto\x00from dataset")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "null bytes")
}

func TestValidateDSLInput_AtLimit(t *testing.T) {
	atLimit := make([]byte, maxDSLInputSize)
	for i := range atLimit {
		atLimit[i] = 'x'
	}
	err := ValidateDSLInput(string(atLimit))
	assert.NoError(t, err)
}

func TestValidateDSLInput_SQLInjectionPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"drop_table", "DROP TABLE users"},
		{"drop_schema", "DROP SCHEMA public"},
		{"truncate", "TRUNCATE users"},
		{"create_table", "CREATE TABLE hack"},
		{"alter_table", "ALTER TABLE users ADD COLUMN hack"},
		{"insert_into", "INSERT INTO users VALUES"},
		{"update_set", "UPDATE users SET admin=1"},
		{"delete_from", "DELETE FROM users"},
		{"grant", "GRANT ALL ON users"},
		{"revoke", "REVOKE ALL ON users"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDSLInput(tt.input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "forbidden SQL pattern")
		})
	}
}

func TestValidateDSLInput_CaseInsensitive(t *testing.T) {
	err := ValidateDSLInput("drop table users")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden SQL pattern")
}

func TestValidateTool_Valid(t *testing.T) {
	def := &ToolDefinition{
		Name:        "my_tool",
		Description: "A test tool",
		Inputs: []*ToolParam{
			{Name: "input1", Type: "string", Required: true},
		},
		Outputs: []*ToolParam{
			{Name: "output1", Type: "string"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "HandleMyTool",
		},
	}
	errs := ValidateTool(def)
	assert.Empty(t, errs)
}

func TestValidateTool_EmptyName(t *testing.T) {
	def := &ToolDefinition{
		Name: "",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "must not be empty")
}

func TestValidateTool_InvalidName(t *testing.T) {
	def := &ToolDefinition{
		Name: "123invalid",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "must match")
}

func TestValidateTool_InvalidInputType(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Inputs: []*ToolParam{
			{Name: "bad", Type: "unknown_type"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "unsupported type")
}

func TestValidateTool_InvalidOutputType(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Outputs: []*ToolParam{
			{Name: "bad", Type: "array"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "unsupported type")
}

func TestValidateTool_DuplicateInputName(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Inputs: []*ToolParam{
			{Name: "x", Type: "string"},
			{Name: "x", Type: "int"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "duplicate input")
}

func TestValidateTool_DuplicateOutputName(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Outputs: []*ToolParam{
			{Name: "result", Type: "string"},
			{Name: "result", Type: "int"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "duplicate output")
}

func TestValidateTool_InvalidLanguage(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Handler: &ToolHandler{
			Language:   "rust",
			EntryPoint: "main",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "language must be")
}

func TestValidateTool_EmptyEntryPoint(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "",
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "entry point must not be empty")
}

func TestValidateTool_MissingHandler(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "handler block")
}

func TestValidateTool_SelfReferencingDep(t *testing.T) {
	def := &ToolDefinition{
		Name: "my_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: []*ToolDep{
			{Name: "my_tool", Type: "library"},
		},
	}
	errs := ValidateTool(def)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "cannot reference the tool itself")
}

func TestValidateTool_AllValidTypes(t *testing.T) {
	def := &ToolDefinition{
		Name: "typed_tool",
		Inputs: []*ToolParam{
			{Name: "s", Type: "string"},
			{Name: "f", Type: "float"},
			{Name: "i", Type: "int"},
			{Name: "b", Type: "bool"},
		},
		Outputs: []*ToolParam{
			{Name: "out", Type: "string"},
		},
		Handler: &ToolHandler{
			Language:   "python",
			EntryPoint: "main",
		},
	}
	errs := ValidateTool(def)
	assert.Empty(t, errs)
}

func TestSecurityScan_Clean(t *testing.T) {
	def := &ToolDefinition{
		Name: "clean_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "HandleClean",
		},
		Deps: []*ToolDep{
			{Name: "json", Type: "library"},
			{Name: "http", Type: "library"},
		},
	}
	issues := SecurityScan(def)
	assert.Empty(t, issues)
}

func TestSecurityScan_NilHandler(t *testing.T) {
	def := &ToolDefinition{
		Name: "no_handler",
	}
	issues := SecurityScan(def)
	assert.Empty(t, issues)
}

func TestSecurityScan_SuspiciousExecDep(t *testing.T) {
	def := &ToolDefinition{
		Name: "bad_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: []*ToolDep{
			{Name: "os_exec", Type: "library"},
		},
	}
	issues := SecurityScan(def)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0], "suspicious dependency name")
}

func TestSecurityScan_ShellDep(t *testing.T) {
	def := &ToolDefinition{
		Name: "bad_tool",
		Handler: &ToolHandler{
			Language:   "python",
			EntryPoint: "main",
		},
		Deps: []*ToolDep{
			{Name: "shell_exec", Type: "library"},
		},
	}
	issues := SecurityScan(def)
	assert.NotEmpty(t, issues)
}

func TestSecurityScan_SystemDep(t *testing.T) {
	def := &ToolDefinition{
		Name: "bad_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: []*ToolDep{
			{Name: "os_system", Type: "library"},
		},
	}
	issues := SecurityScan(def)
	assert.NotEmpty(t, issues)
}

func TestSecurityScan_ExecEntryPoint(t *testing.T) {
	def := &ToolDefinition{
		Name: "bad_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "exec_command",
		},
	}
	issues := SecurityScan(def)
	assert.NotEmpty(t, issues)
}

func TestSecurityScan_EvalEntryPoint(t *testing.T) {
	def := &ToolDefinition{
		Name: "bad_tool",
		Handler: &ToolHandler{
			Language:   "python",
			EntryPoint: "evaluate_input",
		},
	}
	issues := SecurityScan(def)
	assert.NotEmpty(t, issues)
}

func TestSecurityScan_ExecDepNonLibrary(t *testing.T) {
	def := &ToolDefinition{
		Name: "ok_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: []*ToolDep{
			{Name: "os_exec", Type: "binary"},
		},
	}
	issues := SecurityScan(def)
	assert.Empty(t, issues)
}

func TestValidateToolRecord_ValidBuiltin(t *testing.T) {
	rec := &repository.ToolRecord{
		ID:         "tool_1",
		Name:       "test",
		SourceType: "builtin",
	}
	err := ValidateToolRecord(rec)
	assert.NoError(t, err)
}

func TestValidateToolRecord_ValidUser(t *testing.T) {
	rec := &repository.ToolRecord{
		ID:         "tool_1",
		Name:       "test",
		SourceType: "user",
	}
	err := ValidateToolRecord(rec)
	assert.NoError(t, err)
}

func TestValidateToolRecord_ValidMcp(t *testing.T) {
	rec := &repository.ToolRecord{
		ID:         "tool_1",
		Name:       "test",
		SourceType: "mcp",
	}
	err := ValidateToolRecord(rec)
	assert.NoError(t, err)
}

func TestValidateToolRecord_ValidPackage(t *testing.T) {
	rec := &repository.ToolRecord{
		ID:         "tool_1",
		Name:       "test",
		SourceType: "package",
	}
	err := ValidateToolRecord(rec)
	assert.NoError(t, err)
}

func TestValidateToolRecord_Nil(t *testing.T) {
	err := ValidateToolRecord(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool record is nil")
}

func TestValidateToolRecord_InvalidSourceType(t *testing.T) {
	rec := &repository.ToolRecord{
		ID:         "tool_1",
		Name:       "test",
		SourceType: "invalid_type",
	}
	err := ValidateToolRecord(rec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source_type")
}

func TestGoType(t *testing.T) {
	tests := []struct {
		dslType  string
		expected string
	}{
		{"string", "string"},
		{"float", "float64"},
		{"int", "int"},
		{"bool", "bool"},
		{"unknown", "string"},
	}
	for _, tt := range tests {
		t.Run(tt.dslType, func(t *testing.T) {
			assert.Equal(t, tt.expected, goType(tt.dslType))
		})
	}
}

func TestPythonType(t *testing.T) {
	tests := []struct {
		dslType  string
		expected string
	}{
		{"string", "str"},
		{"float", "float"},
		{"int", "int"},
		{"bool", "bool"},
		{"unknown", "str"},
	}
	for _, tt := range tests {
		t.Run(tt.dslType, func(t *testing.T) {
			assert.Equal(t, tt.expected, pythonType(tt.dslType))
		})
	}
}

func TestProtoType(t *testing.T) {
	tests := []struct {
		dslType  string
		expected string
	}{
		{"string", "string"},
		{"float", "double"},
		{"int", "int64"},
		{"bool", "bool"},
		{"unknown", "string"},
	}
	for _, tt := range tests {
		t.Run(tt.dslType, func(t *testing.T) {
			assert.Equal(t, tt.expected, protoType(tt.dslType))
		})
	}
}

func TestSelectTemplate(t *testing.T) {
	tests := []struct {
		name     string
		expected ToolTemplate
	}{
		{"api_fetcher", TemplateAPIConnector},
		{"http_gateway", TemplateAPIConnector},
		{"fetch_data", TemplateAPIConnector},
		{"analyze_trends", TemplateAnalyzer},
		{"score_risk", TemplateAnalyzer},
		{"audit_logs", TemplateAnalyzer},
		{"data_processor", TemplateDataProcessor},
		{"default_tool", TemplateDataProcessor},
		{"random_name", TemplateDataProcessor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, selectTemplate(tt.name))
		})
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"world", "World"},
		{"a", "A"},
		{"", ""},
		{"ALREADY", "ALREADY"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, title(tt.input))
		})
	}
}

func TestCompileToolDefinition_DataProcessor(t *testing.T) {
	def := &ToolDefinition{
		Name:        "data_aggregator",
		Description: "Aggregates data",
		Inputs: []*ToolParam{
			{Name: "source", Type: "string", Required: true},
			{Name: "limit", Type: "int", Required: false},
		},
		Outputs: []*ToolParam{
			{Name: "summary", Type: "string"},
			{Name: "count", Type: "int"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "AggregateData",
		},
	}
	gt, err := CompileToolDefinition(def)
	assert.NoError(t, err)
	assert.NotNil(t, gt)
	assert.Equal(t, "data_aggregator", gt.Name)
	assert.Equal(t, TemplateDataProcessor, gt.Template)
	assert.Contains(t, gt.GoCode, "Handledata_aggregator")
	assert.Contains(t, gt.PythonCode, "handledata_aggregator")
	assert.Contains(t, gt.ProtoDef, "syntax = \"proto3\"")
	assert.Contains(t, gt.ProtoDef, "data_aggregator")
	assert.Contains(t, gt.TestCode, "TestHandledata_aggregator")
}

func TestCompileToolDefinition_APIConnector(t *testing.T) {
	def := &ToolDefinition{
		Name:        "api_fetcher",
		Description: "Fetches data from API",
		Inputs: []*ToolParam{
			{Name: "url", Type: "string", Required: true},
		},
		Outputs: []*ToolParam{
			{Name: "data", Type: "string"},
		},
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "FetchURL",
		},
	}
	gt, err := CompileToolDefinition(def)
	assert.NoError(t, err)
	assert.NotNil(t, gt)
	assert.Equal(t, TemplateAPIConnector, gt.Template)
	assert.Contains(t, gt.GoCode, "ssrf.NewClient()")
}

func TestCompileToolDefinition_Analyzer(t *testing.T) {
	def := &ToolDefinition{
		Name:        "score_analyzer",
		Description: "Scores and analyzes data",
		Inputs: []*ToolParam{
			{Name: "data", Type: "string", Required: true},
		},
		Outputs: []*ToolParam{
			{Name: "score", Type: "float"},
		},
		Handler: &ToolHandler{
			Language:   "python",
			EntryPoint: "analyze",
		},
	}
	gt, err := CompileToolDefinition(def)
	assert.NoError(t, err)
	assert.NotNil(t, gt)
	assert.Equal(t, TemplateAnalyzer, gt.Template)
	assert.Contains(t, gt.PythonCode, "handlescore_analyzer")
}

func TestCompileToolDefinition_Nil(t *testing.T) {
	_, err := CompileToolDefinition(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestCompileToolDefinition_EmptyTool(t *testing.T) {
	def := &ToolDefinition{
		Name: "minimal_tool",
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
	}
	gt, err := CompileToolDefinition(def)
	assert.NoError(t, err)
	assert.NotNil(t, gt)
	assert.Contains(t, gt.GoCode, "minimal_tool")
}

func TestCompileToolDefinition_PythonHandler(t *testing.T) {
	def := &ToolDefinition{
		Name: "python_tool",
		Inputs: []*ToolParam{
			{Name: "text", Type: "string", Required: false},
			{Name: "threshold", Type: "float", Required: false},
			{Name: "enabled", Type: "bool", Required: false},
		},
		Outputs: []*ToolParam{
			{Name: "result", Type: "string"},
		},
		Handler: &ToolHandler{
			Language:   "python",
			EntryPoint: "main",
		},
	}
	gt, err := CompileToolDefinition(def)
	assert.NoError(t, err)
	assert.NotNil(t, gt)
	assert.Contains(t, gt.PythonCode, "text: str = \"\"")
	assert.Contains(t, gt.PythonCode, "threshold: float = 0.0")
	assert.Contains(t, gt.PythonCode, "enabled: bool = False")
}

func TestNegotiationStatus_String(t *testing.T) {
	tests := []struct {
		status   NegotiationStatus
		expected string
	}{
		{NegotiationPending, "pending"},
		{NegotiationAccepted, "accepted"},
		{NegotiationRejected, "rejected"},
		{NegotiationModified, "modified"},
		{NegotiationSuperseded, "superseded"},
		{NegotiationStatus(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}
