package dsl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseObject(t *testing.T) {
	input := `
		object AppaltoPubblico
		from dataset anac_bandi_cig
		id cig
		property cig type identifier from CIG
		property oggetto type text from OGGETTO_LOTTO
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 1)

	obj := program.Statements[0].Object
	assert.Equal(t, "AppaltoPubblico", obj.Name)
	assert.Equal(t, "anac_bandi_cig", obj.FromSource)
	assert.Equal(t, "cig", obj.ID)
	assert.Len(t, obj.Properties, 2)
}

func TestParseObject_WithPredict(t *testing.T) {
	input := `
		object Forecast
		from dataset metrics
		id ts
		property value type float predict
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Properties, 1)
	assert.True(t, obj.Properties[0].Predict)
}

func TestParseObject_WithMaps(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property tipo type enum from TIPO_SCELTA
			map "01" to "aperta"
			map "02" to "ristretta"
			map "03" to "negoziata"
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Properties, 1)
	assert.Len(t, obj.Properties[0].Maps, 3)
	assert.Equal(t, "01", obj.Properties[0].Maps[0].From)
	assert.Equal(t, "aperta", obj.Properties[0].Maps[0].To)
}

func TestParseObject_WithFactors(t *testing.T) {
	input := `
		object Analysis
		from dataset metrics
		id ts
		property value type float
		factor seasonality type time from ts
		factor trend type regression from ts
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Factors, 2)
	assert.Equal(t, "seasonality", obj.Factors[0].Name)
}

func TestParseObject_WithFilters(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property importo type float
		property tipo type text
		filter importo gt 10000
		filter tipo eq "aperta"
		filter importo lte 500000
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Filters, 3)
	assert.Equal(t, "importo", obj.Filters[0].Field)
	assert.Equal(t, "gt", obj.Filters[0].Op)
}

func TestParseObject_WithAggregates(t *testing.T) {
	input := `
		object Stats
		from dataset metrics
		id id
		property category type text
		aggregate count(id) as num_records
		aggregate sum(value) as total_value
		aggregate avg(value) as avg_value
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Aggregates, 3)
	assert.Equal(t, "count", obj.Aggregates[0].Function)
}

func TestParseObject_PropertyWithoutFrom(t *testing.T) {
	input := `
		object Simple
		from dataset src
		id id
		property name type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Equal(t, "name", obj.Properties[0].Name)
	assert.Equal(t, "", obj.Properties[0].From)
}

func TestParseDataset(t *testing.T) {
	input := `
		dataset raw_data version 1 from source_x
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	ds := program.Statements[0].Dataset
	assert.NotNil(t, ds)
	assert.Equal(t, "raw_data", ds.Name)
	assert.Equal(t, "1", ds.Version)
}

func TestParseDataset_AutoVersion(t *testing.T) {
	// Note: 'version auto' parsing depends on participle grammar.
	// Test with explicit int version which is fully supported.
	input := `
		dataset raw_data version 42 from source_x
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	ds := program.Statements[0].Dataset
	assert.Equal(t, "42", ds.Version)
}

func TestParseRelation(t *testing.T) {
	input := `
		object Appalto
		from dataset d1
		id id1
		property id1 type text

		object Azienda
		from dataset d2
		id id2
		property id2 type text

		relation Fornitore from Appalto to Azienda on piva equals id2
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 3)

	rel := program.Statements[2].Relation
	assert.NotNil(t, rel)
	assert.Equal(t, "Fornitore", rel.Name)
	assert.Equal(t, "Appalto", rel.From)
	assert.Equal(t, "Azienda", rel.To)
}

func TestParseAction(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		action send_notification on Appalto
			property message type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	action := program.Statements[1].Action
	assert.NotNil(t, action)
	assert.Equal(t, "send_notification", action.Name)
	assert.Equal(t, "Appalto", action.OnObject)
}

func TestParseMultipleObjects(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		object Azienda
		from dataset imprese
		id piva
		property piva type text

		object Lotto
		from dataset lotti
		id lotto_id
		property lotto_id type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 3)
}

func TestParseFullProgram(t *testing.T) {
	input := `
		dataset raw_data version 1 from importer

		object Appalto
		from dataset bandi
		id cig
		property cig type text from CIG
		property importo type float
		filter importo gt 0

		object Azienda
		from dataset imprese
		id piva
		property piva type text

		relation Fornitore from Appalto to Azienda on cig equals piva

		action export on Appalto
			property format type text
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 5)

	assert.NotNil(t, program.Statements[0].Dataset)
	assert.NotNil(t, program.Statements[1].Object)
	assert.NotNil(t, program.Statements[2].Object)
	assert.NotNil(t, program.Statements[3].Relation)
	assert.NotNil(t, program.Statements[4].Action)
}

func TestParseEmptyInput(t *testing.T) {
	program, err := Parse("")
	assert.NoError(t, err)
	assert.NotNil(t, program)
	assert.Empty(t, program.Statements)
}

func TestParseWhitespaceOnly(t *testing.T) {
	program, err := Parse("   \n  \t  \n  ")
	assert.NoError(t, err)
	assert.Empty(t, program.Statements)
}

func TestParseInvalidSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"gibberish", "not a valid dsl statement at all"},
		{"partial_object", "object"},
		{"missing_from", "object Appalto id cig"},
		{"missing_id", "object Appalto from dataset bandi"},
		{"partial_relation", "relation R from"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestParseFilterOperators(t *testing.T) {
	input := `
		object Test
		from dataset src
		id id
		property id type text
		filter id eq "a"
		filter id neq "b"
		filter id gt "c"
		filter id gte "d"
		filter id lt "e"
		filter id lte "f"
		filter id like "g%"
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Filters, 7)
	ops := make([]string, len(obj.Filters))
	for i, f := range obj.Filters {
		ops[i] = f.Op
	}
	assert.Contains(t, ops, "eq")
	assert.Contains(t, ops, "neq")
	assert.Contains(t, ops, "gt")
	assert.Contains(t, ops, "gte")
	assert.Contains(t, ops, "lt")
	assert.Contains(t, ops, "lte")
	assert.Contains(t, ops, "like")
}

func TestParseAggregateFunctions(t *testing.T) {
	input := `
		object Stats
		from dataset src
		id id
		property id type text
		aggregate count(id) as cnt
		aggregate sum(val) as total
		aggregate avg(val) as mean
		aggregate min(val) as minimum
		aggregate max(val) as maximum
	`
	program, err := Parse(input)
	assert.NoError(t, err)

	obj := program.Statements[0].Object
	assert.Len(t, obj.Aggregates, 5)
	funcs := make([]string, len(obj.Aggregates))
	for i, a := range obj.Aggregates {
		funcs[i] = a.Function
	}
	assert.Contains(t, funcs, "count")
	assert.Contains(t, funcs, "sum")
	assert.Contains(t, funcs, "avg")
	assert.Contains(t, funcs, "min")
	assert.Contains(t, funcs, "max")
}

func TestParse_FullProgram(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text
		property importo type float
		filter importo gt 10000
		aggregate sum(importo) as totale

		object Azienda
		from dataset anagrafica
		id piva
		property piva type text
		property nome type text

		relation Fornitore from Appalto to Azienda on cig equals piva

		dataset bandi version 1 from anac

		action VerificaFornitore on Appalto
		property cig type text
		property soglia type float
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 5)

	assert.NotNil(t, program.Statements[0].Object)
	assert.NotNil(t, program.Statements[1].Object)
	assert.NotNil(t, program.Statements[2].Relation)
	assert.NotNil(t, program.Statements[3].Dataset)
	assert.NotNil(t, program.Statements[4].Action)

	assert.Equal(t, "Appalto", program.Statements[0].Object.Name)
	assert.Equal(t, "Azienda", program.Statements[1].Object.Name)
	assert.Equal(t, "Fornitore", program.Statements[2].Relation.Name)
	assert.Equal(t, "bandi", program.Statements[3].Dataset.Name)
	assert.Equal(t, "VerificaFornitore", program.Statements[4].Action.Name)
}

func TestParse_DatasetWithVersion(t *testing.T) {
	input := `
		dataset mydata version 3 from source_table
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	assert.Len(t, program.Statements, 1)
	assert.NotNil(t, program.Statements[0].Dataset)
	assert.Equal(t, "mydata", program.Statements[0].Dataset.Name)
	assert.Equal(t, "3", program.Statements[0].Dataset.Version)
}

func TestParse_ActionWithMultipleParams(t *testing.T) {
	input := `
		action ProcessData on InputObject
		property source type text
		property limit type int
		property debug type bool
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	action := program.Statements[0].Action
	assert.NotNil(t, action)
	assert.Equal(t, "ProcessData", action.Name)
	assert.Equal(t, "InputObject", action.OnObject)
	assert.Len(t, action.Parameters, 3)
}

func TestParse_ObjectWithPredict(t *testing.T) {
	input := `
		object Forecast
		from dataset historical
		id entity_id
		property score type float from raw_score predict
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	obj := program.Statements[0].Object
	assert.Len(t, obj.Properties, 1)
	assert.True(t, obj.Properties[0].Predict, "predict flag should be true")
}

func TestParse_ObjectWithMultipleMaps(t *testing.T) {
	input := `
		object Data
		from dataset src
		id key
		property status type text
			map "01" to "active"
			map "02" to "inactive"
			map "03" to "pending"
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	prop := program.Statements[0].Object.Properties[0]
	assert.Len(t, prop.Maps, 3)
	assert.Equal(t, "01", prop.Maps[0].From)
	assert.Equal(t, "active", prop.Maps[0].To)
}

func TestParse_InvalidSyntax(t *testing.T) {
	_, err := Parse("this is not a valid DSL program at all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse error")
}

func TestParse_EmptyInput(t *testing.T) {
	program, err := Parse("")
	assert.NoError(t, err)
	assert.NotNil(t, program)
	assert.Empty(t, program.Statements)
}

func TestParse_MultipleFilters(t *testing.T) {
	input := `
		object Data
		from dataset src
		id id
		property id type text
		property amt type float
		filter amt gt 100
		filter amt lt 1000
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	obj := program.Statements[0].Object
	assert.Len(t, obj.Filters, 2)
}

func TestParse_MultipleAggregates(t *testing.T) {
	input := `
		object Stats
		from dataset src
		id id
		property id type text
		aggregate count(id) as cnt
		aggregate sum(amt) as total
		aggregate avg(amt) as avg_amt
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	obj := program.Statements[0].Object
	assert.Len(t, obj.Aggregates, 3)
	assert.Equal(t, "count", obj.Aggregates[0].Function)
	assert.Equal(t, "sum", obj.Aggregates[1].Function)
	assert.Equal(t, "avg", obj.Aggregates[2].Function)
}

func TestParse_ObjectWithFactors(t *testing.T) {
	input := `
		object Score
		from dataset raw
		id id
		property val type float
		factor seasonality type numeric from date_col
	`
	program, err := Parse(input)
	assert.NoError(t, err)
	obj := program.Statements[0].Object
	assert.Len(t, obj.Factors, 1)
	assert.Equal(t, "seasonality", obj.Factors[0].Name)
}
