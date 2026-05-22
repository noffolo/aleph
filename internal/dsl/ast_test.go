package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAST_ConstructFullObject verifies construction of a complete
// ObjectDefinition with all optional fields (properties, factors,
// filters, aggregates) populated correctly.
func TestAST_ConstructFullObject(t *testing.T) {
	obj := &ObjectDefinition{
		Name:       "AppaltoCompleto",
		FromSource: "anac_bandi",
		ID:         "cig",
		Properties: []*Property{
			{
				Name:    "cig",
				Type:    "identifier",
				From:    "CIG",
				Predict: false,
				Maps:    nil,
			},
			{
				Name:    "tipo",
				Type:    "enum",
				From:    "TIPO_SCELTA",
				Predict: false,
				Maps: []*Map{
					{From: "01", To: "aperta"},
					{From: "02", To: "ristretta"},
				},
			},
			{
				Name:    "score",
				Type:    "float",
				From:    "raw_score",
				Predict: true,
				Maps:    nil,
			},
		},
		Factors: []*Factor{
			{Name: "seasonality", Type: "time", From: "data_col"},
			{Name: "trend", Type: "regression", From: "data_col"},
		},
		Filters: []*FilterDefinition{
			{Field: "importo", Op: "gt", Value: "10000"},
			{Field: "tipo", Op: "eq", Value: "aperta"},
		},
		Aggregates: []*AggregateDefinition{
			{Function: "sum", Field: "importo", Alias: "totale"},
			{Function: "count", Field: "cig", Alias: "num_record"},
		},
	}

	assert.Equal(t, "AppaltoCompleto", obj.Name)
	assert.Equal(t, "anac_bandi", obj.FromSource)
	assert.Equal(t, "cig", obj.ID)
	assert.Len(t, obj.Properties, 3)
	assert.Len(t, obj.Factors, 2)
	assert.Len(t, obj.Filters, 2)
	assert.Len(t, obj.Aggregates, 2)

	assert.Equal(t, "cig", obj.Properties[0].Name)
	assert.Equal(t, "identifier", obj.Properties[0].Type)
	assert.False(t, obj.Properties[0].Predict)

	assert.Equal(t, "tipo", obj.Properties[1].Name)
	assert.Len(t, obj.Properties[1].Maps, 2)
	assert.Equal(t, "01", obj.Properties[1].Maps[0].From)
	assert.Equal(t, "aperta", obj.Properties[1].Maps[0].To)

	assert.True(t, obj.Properties[2].Predict, "predict flag should be true")

	assert.Equal(t, "seasonality", obj.Factors[0].Name)
	assert.Equal(t, "time", obj.Factors[0].Type)

	assert.Equal(t, "importo", obj.Filters[0].Field)
	assert.Equal(t, "gt", obj.Filters[0].Op)
	assert.Equal(t, "10000", obj.Filters[0].Value)

	assert.Equal(t, "sum", obj.Aggregates[0].Function)
	assert.Equal(t, "importo", obj.Aggregates[0].Field)
	assert.Equal(t, "totale", obj.Aggregates[0].Alias)
}

// TestAST_ConstructEdgeCases verifies AST types handle optional/zero-value
// fields gracefully: Property without From, empty maps, nil slices.
func TestAST_ConstructEdgeCases(t *testing.T) {
	// Object with minimal required fields only
	obj := &ObjectDefinition{
		Name:       "Minimal",
		FromSource: "src",
		ID:         "id",
		Properties: []*Property{
			{
				Name: "name_only",
				Type: "text",
				From: "", // explicitly empty
			},
		},
	}

	assert.Equal(t, "Minimal", obj.Name)
	assert.Equal(t, "", obj.Properties[0].From, "From should be empty when not set")
	assert.Nil(t, obj.Factors, "Factors should be nil for minimal object")
	assert.Nil(t, obj.Filters, "Filters should be nil for minimal object")
	assert.Nil(t, obj.Aggregates, "Aggregates should be nil for minimal object")

	// ToolDefinition with empty inputs/outputs
	toolDef := &ToolDefinition{
		Name:        "bare_minimum",
		Description: "",
		Inputs:      nil,
		Outputs:     nil,
		Handler: &ToolHandler{
			Language:   "go",
			EntryPoint: "Handle",
		},
		Deps: nil,
	}
	assert.Equal(t, "bare_minimum", toolDef.Name)
	assert.Empty(t, toolDef.Inputs)
	assert.Empty(t, toolDef.Outputs)
	assert.NotNil(t, toolDef.Handler)
	assert.Equal(t, "go", toolDef.Handler.Language)

	// DatasetDefinition with version as string
	ds := &DatasetDefinition{
		Name:    "raw_data",
		Version: "auto",
		From:    "importer",
	}
	assert.Equal(t, "raw_data", ds.Name)
	assert.Equal(t, "auto", ds.Version)
	assert.Equal(t, "importer", ds.From)
}

// TestAST_ConstructOrphanRelation verifies that a RelationDefinition
// referencing non-existent target objects can be detected as invalid
// (no matching ObjectDefinition in the program).
func TestAST_ConstructOrphanRelation(t *testing.T) {
	prog := &Program{
		Statements: []*Statement{
			{
				Object: &ObjectDefinition{
					Name:       "Appalto",
					FromSource: "bandi",
					ID:         "cig",
					Properties: []*Property{
						{Name: "cig", Type: "text"},
					},
				},
			},
			{
		Relation: &RelationDefinition{
			Name:    "Fornitore",
			From:    "Appalto",
			To:      "Azienda",
			LeftOn:  "cig",
			RightOn: "piva",
		},
			},
		},
	}

	rel := prog.Statements[1].Relation
	assert.NotNil(t, rel)
	assert.Equal(t, "Fornitore", rel.Name)
	assert.Equal(t, "Azienda", rel.To)

	objectNames := map[string]bool{}
	for _, stmt := range prog.Statements {
		if stmt.Object != nil {
			objectNames[stmt.Object.Name] = true
		}
	}

	assert.False(t, objectNames["Azienda"], "Azienda should not exist in the program")
	assert.True(t, objectNames["Appalto"], "Appalto should exist in the program")
}
