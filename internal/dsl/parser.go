package dsl

import (
	"github.com/alecthomas/participle/v2"
)

var parser = participle.MustBuild[Program](
	participle.Unquote("String"),
)

func Parse(input string) (*Program, error) {
	return parser.ParseString("", input)
}