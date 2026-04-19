package dsl

type Program struct {
	Statements []*Statement `@@*`
}

type Statement struct {
	Object   *ObjectDefinition   `  @@`
	Relation *RelationDefinition `| @@`
	Dataset  *DatasetDefinition  `| @@`
	Action   *ActionDefinition   `| @@`
}

type ObjectDefinition struct {
	Name       string      `"object" @Ident`
	FromSource string      `"from" "dataset" @Ident`
	ID         string      `"id" @Ident`
	Properties []*Property `@@*`
	Factors    []*Factor   `@@*`
}

type Factor struct {
	Name string `"factor" @Ident`
	Type string `"type" @Ident` // volatility, trend, sentiment
	From string `"from" @Ident`
}

type Property struct {
	Name    string `"property" @Ident`
	Type    string `"type" @Ident` // string, float, timestamp, probability
	From    string `("from" @Ident)?`
	Predict bool   `@"predict"?`
	Maps    []*Map `@@*`
}

type ActionDefinition struct {
	Name       string      `"action" @Ident`
	OnObject   string      `"on" @Ident`
	Parameters []*Property `@@*`
}

type Map struct {
	From string `"map" @String`
	To   string `"to" @String`
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