package sources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestISTATLookup(t *testing.T) {
	codes := NewISTATLookup()
	// Roma
	name, found := codes.Lookup("058091")
	assert.True(t, found)
	assert.Contains(t, name, "Roma")
	// Nonexistent
	_, found = codes.Lookup("999999")
	assert.False(t, found)
}

func TestPartyMappingExactMatch(t *testing.T) {
	mapper := NewPartyMapper()
	mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
	mapper.AddAlias("FRATELLI D'ITALIA - GIORGIA MELONI", "fratelli-italia")
	mapper.AddAlias("PARTITO DEMOCRATICO", "partito-democratico")
	mapper.AddAlias("PD", "partito-democratico")

	// Exact match
	canonical, found := mapper.Lookup("PARTITO DEMOCRATICO")
	require.True(t, found)
	assert.Equal(t, "partito-democratico", canonical)

	// Alias match
	canonical, found = mapper.Lookup("PD")
	require.True(t, found)
	assert.Equal(t, "partito-democratico", canonical)

	// No match
	_, found = mapper.Lookup("LISTA INESISTENTE XYZ")
	assert.False(t, found)
}

func TestPartyMappingManualOverride(t *testing.T) {
	mapper := NewPartyMapper()
	mapper.AddAlias("FRATELLI D'ITALIA", "fratelli-italia")
	mapper.SetOverride("FRATELLI D'ITALIA - ROMA", "fratelli-italia-roma")

	// Manual override takes priority over alias match
	canonical, found := mapper.Lookup("FRATELLI D'ITALIA - ROMA")
	require.True(t, found)
	assert.Equal(t, "fratelli-italia-roma", canonical)

	// Without override, uses alias table
	canonical, found = mapper.Lookup("FRATELLI D'ITALIA")
	require.True(t, found)
	assert.Equal(t, "fratelli-italia", canonical)
}

func TestElectionConfigValidation(t *testing.T) {
	valid := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 2022}
	assert.NoError(t, valid.Validate())

	invalidType := ElectionConfig{ElectionType: "fantasia", Level: "comune", Year: 2022}
	assert.ErrorContains(t, invalidType.Validate(), "invalid election_type")

	invalidLevel := ElectionConfig{ElectionType: "politiche", Level: "quartiere", Year: 2022}
	assert.ErrorContains(t, invalidLevel.Validate(), "invalid level")

	invalidYear := ElectionConfig{ElectionType: "politiche", Level: "comune", Year: 1990}
	assert.ErrorContains(t, invalidYear.Validate(), "year before 2000")
}
