package dsl

type Program struct {
	Statements []*Statement `@@*`
}

type Statement struct {
	Object   *ObjectDefinition   `  @@`
	Relation *RelationDefinition `| @@`
	Dataset  *DatasetDefinition  `| @@`
	Action   *ActionDefinition   `| @@`
	Tool     *ParsedTool         `| @@`
}

type ParsedTool struct {
	Name        string          `("tool" @Ident)`
	Description string          `("{" "name" @String)`
	Inputs      []*ToolParamDef `"inputs" "{" @@* "}"`
	Outputs     []*ToolParamDef `"outputs" "{" @@* "}"`
	Handler     *HandlerDef     `"handler" "{" @@ "}"`
	Close       string          `"}"`
}

type ToolParamDef struct {
	Name        string `@Ident`
	Type        string `"type" @Ident`
	Required    bool   `@"required"?`
	Description string `@String`
}

type HandlerDef struct {
	Language   string `"language" @Ident`
	EntryPoint string `"entry" @String`
}

type ObjectDefinition struct {
	Name       string                `"object" @Ident`
	FromSource string                `"from" "dataset" @Ident`
	ID         string                `"id" @Ident`
	Properties []*Property            `@@*`
	Factors    []*Factor             `@@*`
	Filters    []*FilterDefinition   `@@*`
	Aggregates []*AggregateDefinition `@@*`
}

type FilterDefinition struct {
	Field string `"filter" @Ident`
	Op    string `@("eq" | "neq" | "gt" | "gte" | "lt" | "lte" | "like")`
	Value string `@(String | Ident | Int | Float)`
}

type AggregateDefinition struct {
	Function string `"aggregate" @("count" | "sum" | "avg" | "min" | "max")`
	Field    string `"(" @Ident ")"`
	Alias    string `"as" @Ident`
}

type Factor struct {
	Name string `"factor" @Ident`
	Type string `"type" @Ident`
	From string `"from" @Ident`
}

type Property struct {
	Name    string `"property" @Ident`
	Type    string `"type" @Ident`
	From    string `("from" @Ident)?`
	Predict bool   `@"predict"?`
	Maps    []*Map `@@*`
}

type Map struct {
	From string `"map" @String`
	To   string `"to" @String`
}

type ActionDefinition struct {
	Name       string      `"action" @Ident`
	OnObject   string      `"on" @Ident`
	Parameters []*Property `@@*`
}

type RelationDefinition struct {
	Name    string `"relation" @Ident`
	From    string `"from" @Ident`
	To      string `"to" @Ident`
	LeftOn  string `"on" @Ident`
	RightOn string `"equals" @Ident`
}

type DatasetDefinition struct {
	Name    string `"dataset" @Ident`
	Version string `"version" (@Int | "auto")`
	From    string `"from" @Ident`
}

// DSL Compiler Types (used by compiler_tool.go for code generation)
type ToolParam struct {
	Name     string
	Type     string
	Required bool
}

type ToolHandler struct {
	Language   string
	EntryPoint string
}

type ToolDep struct {
	Name string
	Type string
}

type ToolDefinition struct {
	Name        string
	Description string
	Inputs      []*ToolParam
	Outputs     []*ToolParam
	Handler     *ToolHandler
	Deps        []*ToolDep
}
