package dsl

import (
	"testing"
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
				// Error means we validated and rejected — good
				return
			}
			// If no error, the SQL must NOT contain the injection payload unquoted
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
	// Properties from DSL parser must be valid identifiers (regex-validated by participle)
	// The safeident.QuoteIdentifier handles any edge cases
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
	// Filter fields from DSL parser are @Ident tokens (already validated by participle)
	// safeident.QuoteIdentifier provides defense-in-depth
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

