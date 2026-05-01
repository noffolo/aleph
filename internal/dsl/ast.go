package dsl

type Program struct {
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Object   *ObjectDefinition   `parser:"  @@"`
	Relation *RelationDefinition `parser:"| @@"`
	Dataset  *DatasetDefinition  `parser:"| @@"`
	Action   *ActionDefinition   `parser:"| @@"`
	Tool     *ParsedTool         `parser:"| @@"`
}

type ParsedTool struct {
	Name        string          `parser:"('tool' @Ident)"`
	Description string          `parser:"('{' 'name' @String)"`
	Inputs      []*ToolParamDef `parser:"'inputs' '{' @@* '}'"`
	Outputs     []*ToolParamDef `parser:"'outputs' '{' @@* '}'"`
	Handler     *HandlerDef     `parser:"'handler' '{' @@ '}'"`
	Close       string          `parser:"'}'"`
}

type ToolParamDef struct {
	Name        string `parser:"@Ident"`
	Type        string `parser:"'type' @Ident"`
	Required    bool   `parser:"@'required'?"`
	Description string `parser:"@String"`
}

type HandlerDef struct {
	Language   string `parser:"'language' @Ident"`
	EntryPoint string `parser:"'entry' @String"`
}

type ObjectDefinition struct {
	Name       string                 `parser:"'object' @Ident"`
	FromSource string                 `parser:"'from' 'dataset' @Ident"`
	ID         string                 `parser:"'id' @Ident"`
	Properties []*Property            `parser:"@@*"`
	Factors    []*Factor              `parser:"@@*"`
	Filters    []*FilterDefinition    `parser:"@@*"`
	Aggregates []*AggregateDefinition `parser:"@@*"`
}

type FilterDefinition struct {
	Field string `parser:"'filter' @Ident"`
	Op    string `parser:"@('eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'like')"`
	Value string `parser:"@(String | Ident | Int | Float)"`
}

type AggregateDefinition struct {
	Function string `parser:"'aggregate' @('count' | 'sum' | 'avg' | 'min' | 'max')"`
	Field    string `parser:"'(' @Ident ')'"`
	Alias    string `parser:"'as' @Ident"`
}

type Factor struct {
	Name string `parser:"'factor' @Ident"`
	Type string `parser:"'type' @Ident"`
	From string `parser:"'from' @Ident"`
}

type Property struct {
	Name    string `parser:"'property' @Ident"`
	Type    string `parser:"'type' @Ident"`
	From    string `parser:"('from' @Ident)?"`
	Predict bool   `parser:"@'predict'?"`
	Maps    []*Map `parser:"@@*"`
}

type Map struct {
	From string `parser:"'map' @String"`
	To   string `parser:"'to' @String"`
}

type ActionDefinition struct {
	Name       string      `parser:"'action' @Ident"`
	OnObject   string      `parser:"'on' @Ident"`
	Parameters []*Property `parser:"@@*"`
}

type RelationDefinition struct {
	Name    string `parser:"'relation' @Ident"`
	From    string `parser:"'from' @Ident"`
	To      string `parser:"'to' @Ident"`
	LeftOn  string `parser:"'on' @Ident"`
	RightOn string `parser:"'equals' @Ident"`
}

type DatasetDefinition struct {
	Name    string `parser:"'dataset' @Ident"`
	Version string `parser:"'version' (@Int | 'auto')"`
	From    string `parser:"'from' @Ident"`
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
