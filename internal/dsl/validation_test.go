package dsl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDSLInput_HappyPath(t *testing.T) {
	input := "object Appalto from dataset bandi id cig property cig type text"
	err := ValidateDSLInput(input)
	assert.NoError(t, err)
}

func TestValidateDSLInput_ValidProgram(t *testing.T) {
	input := `
		dataset raw_data version 1 from importer
		object Appalto
		from dataset bandi
		id cig
		property cig type text from CIG
		property importo type float predict
		filter importo gt 0
		aggregate sum(importo) as totale
		factor seasonality type time from ts
		object Azienda
		from dataset imprese
		id piva
		property piva type text
		relation Fornitore from Appalto to Azienda on cig equals piva
		action export on Appalto
			property format type text
	`
	err := ValidateDSLInput(input)
	assert.NoError(t, err)
}

func TestValidateDSLInput_EdgeCases(t *testing.T) {
	err := ValidateDSLInput("")
	assert.NoError(t, err, "empty input should be valid (parsing handles emptiness)")

	err = ValidateDSLInput("   \n  \t  \n  ")
	assert.NoError(t, err, "whitespace-only input should pass validation")

	atLimit := strings.Repeat("x", maxDSLInputSize)
	err = ValidateDSLInput(atLimit)
	assert.NoError(t, err, "input exactly at size limit should pass")
}

func TestValidateDSLInput_ErrorCases(t *testing.T) {
	tooLarge := strings.Repeat("x", maxDSLInputSize+1)
	err := ValidateDSLInput(tooLarge)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")

	err = ValidateDSLInput("valid start\x00invalid end")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "null bytes")

	err = ValidateDSLInput("DROP TABLE users; -- malicious")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden SQL pattern")

	err = ValidateDSLInput("drop schema public cascade")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden SQL pattern")
}
