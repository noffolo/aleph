package dsl

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2"
)

var parser = participle.MustBuild[Program](
	participle.Unquote("String"),
)

func Parse(input string) (*Program, error) {
	prog, err := parser.ParseString("", input)
	if err != nil {
		var perr participle.Error
		if errors.As(err, &perr) {
			pos := perr.Position()
			return nil, fmt.Errorf("parse error at line %d, column %d: %s", pos.Line, pos.Column, err.Error())
		}
		return nil, fmt.Errorf("parse error: %s", err.Error())
	}
	return prog, nil
}