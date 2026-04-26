package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolDefinition_Basic(t *testing.T) {
	input := `
		tool data_aggregator {
			name "Data Aggregator"
			inputs {
				source type string required "Data source name"
				limit type int "Max records to process"
			}
			outputs {
				summary type string "Aggregation summary"
				count type int "Number of records"
			}
			handler {
				language go
				entry "AggregateData"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	tool := program.Statements[0].Tool
	require.NotNil(t, tool)
	assert.Equal(t, "data_aggregator", tool.Name)
	assert.Equal(t, "Data Aggregator", tool.Description)

	require.Len(t, tool.Inputs, 2)
	assert.Equal(t, "source", tool.Inputs[0].Name)
	assert.Equal(t, "string", tool.Inputs[0].Type)
	assert.True(t, tool.Inputs[0].Required)
	assert.Equal(t, "Data source name", tool.Inputs[0].Description)

	assert.Equal(t, "limit", tool.Inputs[1].Name)
	assert.Equal(t, "int", tool.Inputs[1].Type)
	assert.False(t, tool.Inputs[1].Required)
	assert.Equal(t, "Max records to process", tool.Inputs[1].Description)

	require.Len(t, tool.Outputs, 2)
	assert.Equal(t, "summary", tool.Outputs[0].Name)
	assert.Equal(t, "string", tool.Outputs[0].Type)
	assert.Equal(t, "count", tool.Outputs[1].Name)
	assert.Equal(t, "int", tool.Outputs[1].Type)

	require.NotNil(t, tool.Handler)
	assert.Equal(t, "go", tool.Handler.Language)
	assert.Equal(t, "AggregateData", tool.Handler.EntryPoint)
}

func TestParseToolDefinition_AllTypes(t *testing.T) {
	input := `
		tool type_demo {
			name "Type Demo"
			inputs {
				name type string required "Name field"
				price type float "Price field"
				count type int required "Count field"
				active type bool "Active flag"
			}
			outputs {
				result type string "Result"
			}
			handler {
				language python
				entry "main"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	tool := program.Statements[0].Tool
	require.NotNil(t, tool)

	require.Len(t, tool.Inputs, 4)
	assert.Equal(t, "string", tool.Inputs[0].Type)
	assert.Equal(t, "float", tool.Inputs[1].Type)
	assert.Equal(t, "int", tool.Inputs[2].Type)
	assert.Equal(t, "bool", tool.Inputs[3].Type)
	assert.True(t, tool.Inputs[0].Required)
	assert.False(t, tool.Inputs[1].Required)
	assert.True(t, tool.Inputs[2].Required)
	assert.False(t, tool.Inputs[3].Required)

	assert.Equal(t, "python", tool.Handler.Language)
	assert.Equal(t, "main", tool.Handler.EntryPoint)
}

func TestParseToolDefinition_EmptyInputs(t *testing.T) {
	input := `
		tool empty_tool {
			name "Empty Tool"
			inputs {
			}
			outputs {
				result type string "Output result"
			}
			handler {
				language go
				entry "Handle"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	tool := program.Statements[0].Tool
	require.NotNil(t, tool)
	assert.Empty(t, tool.Inputs)
	assert.Len(t, tool.Outputs, 1)
}

func TestParseToolDefinition_InvalidName(t *testing.T) {
	input := `
		tool 123invalid {
			name "Bad Name"
			inputs {
			}
			outputs {
				result type string "Result"
			}
			handler {
				language go
				entry "Handle"
			}
		}
	`
	_, err := Parse(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse error")
}

func TestParseToolDefinition_InvalidType(t *testing.T) {
	input := `
		tool bad_type {
			name "Bad Type"
			inputs {
				field type unknown_type "Invalid type"
			}
			outputs {
				result type string "Result"
			}
			handler {
				language go
				entry "Handle"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, program)
	// The parser accepts any @Ident for type; invalid type validation
	// is handled by the compiler, not the parser.
	require.Len(t, program.Statements, 1)
	tool := program.Statements[0].Tool
	require.NotNil(t, tool)
	assert.Len(t, tool.Inputs, 1)
	assert.Equal(t, "field", tool.Inputs[0].Name)
	assert.Equal(t, "unknown_type", tool.Inputs[0].Type)
}

func TestParseToolDefinition_MissingHandler(t *testing.T) {
	input := `
		tool no_handler {
			name "No Handler"
			inputs {
			}
			outputs {
				result type string "Result"
			}
		}
	`
	_, err := Parse(input)
	assert.Error(t, err)
}

func TestParseToolDefinition_MultipleTools(t *testing.T) {
	input := `
		object Appalto
		from dataset bandi
		id cig
		property cig type text

		tool analyze_appalti {
			name "Analyze Appalti"
			inputs {
				year type int required "Year to analyze"
			}
			outputs {
				report type string "Analysis report"
			}
			handler {
				language go
				entry "Analyze"
			}
		}

		tool export_results {
			name "Export Results"
			inputs {
				format type string "Export format"
			}
			outputs {
				url type string "Download URL"
			}
			handler {
				language python
				entry "export"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 3)

	assert.NotNil(t, program.Statements[0].Object)
	assert.Equal(t, "Appalto", program.Statements[0].Object.Name)

	tool1 := program.Statements[1].Tool
	require.NotNil(t, tool1)
	assert.Equal(t, "analyze_appalti", tool1.Name)
	assert.Len(t, tool1.Inputs, 1)

	tool2 := program.Statements[2].Tool
	require.NotNil(t, tool2)
	assert.Equal(t, "export_results", tool2.Name)
	assert.Equal(t, "python", tool2.Handler.Language)
}

func TestParseToolDefinition_InlineCode(t *testing.T) {
	input := `
		tool api_fetcher {
			name "API Fetcher"
			inputs {
				url type string required "URL to fetch"
			}
			outputs {
				data type string "Response data"
			}
			handler {
				language go
				entry "FetchURL"
			}
		}
	`
	program, err := Parse(input)
	require.NoError(t, err)
	tool := program.Statements[0].Tool
	require.NotNil(t, tool)
	assert.Equal(t, "api_fetcher", tool.Name)
	assert.Equal(t, "go", tool.Handler.Language)
}
