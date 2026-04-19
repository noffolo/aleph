package dsl

import (
	"testing"
	"github.com/stretchr/testify/assert"
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