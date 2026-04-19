package dsl

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCompileObject(t *testing.T) {
	input := `
		object AppaltoPubblico
		from dataset anac_bandi_cig
		id cig
		property cig type identifier from CIG
		property tipo_procedura type enum from TIPO_SCELTA_CONTRAENTE
			map "01" to "aperta"
			map "02" to "ristretta"
	`
	dataRoot := "/tmp/aleph-data"
	program, err := Parse(input)
	assert.NoError(t, err)
	
	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("AppaltoPubblico")
	
	assert.NoError(t, err)
	assert.Contains(t, sql, "SELECT")
	assert.Contains(t, sql, `"AppaltoPubblico"."CIG" AS "cig"`)
	assert.Contains(t, sql, `CASE "AppaltoPubblico"."TIPO_SCELTA_CONTRAENTE" WHEN '01' THEN 'aperta' WHEN '02' THEN 'ristretta' END AS "tipo_procedura"`)
	assert.Contains(t, sql, "read_parquet('/tmp/aleph-data/anac_bandi_cig/latest/*.parquet')")
}

func TestCompileWithRelation(t *testing.T) {
	input := `
		object Appalto
		from dataset d1
		id id1
		property id1 type text
		
		object Azienda
		from dataset d2
		id id2
		property id2 type text
		
		relation Fornitore from Appalto to Azienda on p_id equals id2
	`
	dataRoot := "/data"
	program, _ := Parse(input)
	compiler := NewCompiler(program, dataRoot)
	sql, err := compiler.CompileObject("Appalto")
	
	assert.NoError(t, err)
	assert.Contains(t, sql, `LEFT JOIN read_parquet('/data/d2/latest/*.parquet') AS "Azienda"`)
	assert.Contains(t, sql, `ON "Appalto"."p_id" = "Azienda"."id2"`)
}
