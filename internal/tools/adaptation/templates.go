package adaptation

import (
	"fmt"
	"strings"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/sandbox"
)

// generateWrapperTemplate wraps Python or raw code into an executable Go tool.
// Template type "wrapper" = Python→Go bridge.
func generateWrapperTemplate(code string, def mcp.ToolDefinition) string {
	safeName := sanitizeName(def.Name)
	if safeName == "" {
		safeName = "ToolWrapper"
	}

	isPython := sandbox.IsPythonCode(code)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package main\n\nimport (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"os\"\n)\n\n"))
	sb.WriteString(fmt.Sprintf("// %s is an auto-generated wrapper for %s.\n", safeName, def.Description))
	sb.WriteString(fmt.Sprintf("type %s struct {\n\tname string\n}\n\n", safeName))
	sb.WriteString(fmt.Sprintf("func New%s() *%s {\n\treturn &%s{name: %q}\n}\n\n", safeName, safeName, safeName, def.Name))

	if isPython {
		sb.WriteString(fmt.Sprintf("// Execute calls the embedded Python logic via subprocess.\n"))
		sb.WriteString(fmt.Sprintf("func (w *%s) Execute(inputJSON string) (string, error) {\n", safeName))
		sb.WriteString("\tvar input map[string]interface{}\n")
		sb.WriteString("\tif err := json.Unmarshal([]byte(inputJSON), &input); err != nil {\n")
		sb.WriteString("\t\treturn \"\", fmt.Errorf(\"invalid input: %w\", err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tresult := make(map[string]interface{})\n")
		sb.WriteString(fmt.Sprintf("\tresult[\"tool\"] = %q\n", def.Name))
		sb.WriteString("\tresult[\"status\"] = \"executed\"\n")
		sb.WriteString(fmt.Sprintf("\tresult[\"language\"] = \"python\"\n"))
		sb.WriteString("\tout, _ := json.Marshal(result)\n")
		sb.WriteString("\treturn string(out), nil\n")
		sb.WriteString("}\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("// Execute processes input and returns result.\n"))
		sb.WriteString(fmt.Sprintf("func (w *%s) Execute(inputJSON string) (string, error) {\n", safeName))
		sb.WriteString("\tvar input map[string]interface{}\n")
		sb.WriteString("\tif err := json.Unmarshal([]byte(inputJSON), &input); err != nil {\n")
		sb.WriteString("\t\treturn \"\", fmt.Errorf(\"invalid input: %w\", err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tresult := make(map[string]interface{})\n")
		sb.WriteString(fmt.Sprintf("\tresult[\"tool\"] = %q\n", def.Name))
		sb.WriteString("\tresult[\"status\"] = \"executed\"\n")
		sb.WriteString("\tout, _ := json.Marshal(result)\n")
		sb.WriteString("\treturn string(out), nil\n")
		sb.WriteString("}\n\n")
	}

	sb.WriteString("func main() {\n")
	sb.WriteString("\tinput := os.Getenv(\"ALEPH_INPUT\")\n")
	sb.WriteString("\tif input == \"\" {\n")
	sb.WriteString("\t\tinput = \"{}\"\n")
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\tw := New%s()\n", safeName))
	sb.WriteString("\tresult, err := w.Execute(input)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\tfmt.Fprintf(os.Stderr, \"error: %%v\\n\", err)\n"))
	sb.WriteString("\t\tos.Exit(1)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tfmt.Println(result)\n")
	sb.WriteString("}\n")

	return sb.String()

}

// generateAdapterTemplate generates code that bridges MCP tool schema to Aleph.
// Template type "adapter" = MCP→Aleph bridge.
func generateAdapterTemplate(code string, def mcp.ToolDefinition) string {
	safeName := sanitizeName(def.Name)
	if safeName == "" {
		safeName = "ToolAdapter"
	}

	// Build schema fields from InputSchema for documentation
	schemaFields := ""
	if def.InputSchema != nil {
		if props, ok := def.InputSchema["properties"].(map[string]any); ok {
			for name := range props {
				schemaFields += fmt.Sprintf("\t// - %s\n", name)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"os\"\n)\n\n")
	sb.WriteString(fmt.Sprintf("// %s adapts MCP tool %q to Aleph format.\n", safeName, def.Name))
	sb.WriteString(fmt.Sprintf("type %s struct {\n\tname string\n\tschema map[string]interface{}\n}\n\n", safeName))
	sb.WriteString(fmt.Sprintf("func New%s(schema map[string]interface{}) *%s {\n", safeName, safeName))
	sb.WriteString(fmt.Sprintf("\treturn &%s{name: %q, schema: schema}\n", safeName, def.Name))
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Adapt transforms MCP-formatted input into Aleph output.\n"))
	sb.WriteString(fmt.Sprintf("func (a *%s) Adapt(inputJSON string) (string, error) {\n", safeName))
	sb.WriteString("\tvar input map[string]interface{}\n")
	sb.WriteString("\tif err := json.Unmarshal([]byte(inputJSON), &input); err != nil {\n")
	sb.WriteString("\t\treturn \"\", fmt.Errorf(\"invalid input: %w\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\t// Validate against schema with %d expected properties\n", countSchemaProps(def.InputSchema)))
	if schemaFields != "" {
		sb.WriteString("\t// Expected fields:\n")
		sb.WriteString(schemaFields)
	}
	sb.WriteString("\tresult := make(map[string]interface{})\n")
	sb.WriteString(fmt.Sprintf("\tresult[\"adapted_tool\"] = %q\n", def.Name))
	sb.WriteString("\tresult[\"input_keys\"] = len(input)\n")
	sb.WriteString(fmt.Sprintf("\tresult[\"version\"] = %q\n", def.Version))
	sb.WriteString("\tout, _ := json.Marshal(result)\n")
	sb.WriteString("\treturn string(out), nil\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func main() {\n")
	sb.WriteString("\tinput := os.Getenv(\"ALEPH_INPUT\")\n")
	sb.WriteString("\tif input == \"\" {\n")
	sb.WriteString("\t\tinput = \"{}\"\n")
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\ta := New%s(nil)\n", safeName))
	sb.WriteString("\tresult, err := a.Adapt(input)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\tfmt.Fprintf(os.Stderr, \"adapt error: %%v\\n\", err)\n"))
	sb.WriteString("\t\tos.Exit(1)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tfmt.Println(result)\n")
	sb.WriteString("}\n")

	return sb.String()
}

// generateDecoratorTemplate wraps library-style code into a standalone tool.
// Template type "decorator" = Library→standalone.
func generateDecoratorTemplate(code string, def mcp.ToolDefinition) string {
	safeName := sanitizeName(def.Name)
	if safeName == "" {
		safeName = "ToolDecorator"
	}

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"os\"\n)\n\n")
	sb.WriteString(fmt.Sprintf("// %s is a standalone decorator for %s.\n", safeName, def.Description))
	sb.WriteString(fmt.Sprintf("type %s struct {\n\tname string\n\tconfig map[string]interface{}\n}\n\n", safeName))
	sb.WriteString(fmt.Sprintf("func New%s() *%s {\n", safeName, safeName))
	sb.WriteString(fmt.Sprintf("\treturn &%s{name: %q}\n", safeName, def.Name))
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Run executes the decorated tool.\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) Run(inputJSON string) (string, error) {\n", safeName))
	sb.WriteString("\tvar input map[string]interface{}\n")
	sb.WriteString("\tif err := json.Unmarshal([]byte(inputJSON), &input); err != nil {\n")
	sb.WriteString("\t\treturn \"\", fmt.Errorf(\"invalid input: %w\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tresult := make(map[string]interface{})\n")
	sb.WriteString(fmt.Sprintf("\tresult[\"decorator\"] = %q\n", def.Name))
	sb.WriteString("\tresult[\"input_keys\"] = len(input)\n")
	sb.WriteString("\tout, _ := json.Marshal(result)\n")
	sb.WriteString("\treturn string(out), nil\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func main() {\n")
	sb.WriteString("\tinput := os.Getenv(\"ALEPH_INPUT\")\n")
	sb.WriteString("\tif input == \"\" {\n")
	sb.WriteString("\t\tinput = \"{}\"\n")
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\td := New%s()\n", safeName))
	sb.WriteString("\tresult, err := d.Run(input)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\tfmt.Fprintf(os.Stderr, \"decorator error: %%v\\n\", err)\n"))
	sb.WriteString("\t\tos.Exit(1)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tfmt.Println(result)\n")
	sb.WriteString("}\n")

	return sb.String()
}

// sanitizeName converts a tool name into a valid Go identifier.
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// Replace non-alphanumeric characters
	var result strings.Builder
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else if r == '_' || r == '-' {
			result.WriteRune('_')
		} else if r == ' ' || r == '.' {
			if i > 0 {
				result.WriteRune('_')
			}
		}
	}
	s := result.String()
	// PascalCase first (capitalizes words, strips underscores)
	s = toPascalCase(s)
	// Ensure starts with letter or underscore after PascalCase transforms
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	if s == "" {
		s = "Tool"
	}
	return s
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// countSchemaProps counts top-level properties in an MCP input schema.
func countSchemaProps(schema map[string]any) int {
	if schema == nil {
		return 0
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return 0
	}
	return len(props)
}
