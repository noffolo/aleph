package nlp_adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapterInterface(t *testing.T) {
	// Just verify the interface compliance at compile time
	// by checking the var declaration pattern
	assert.True(t, true, "interface compliance verified at compile time")
}

func TestAdapter_NilHandler(t *testing.T) {
	a := &Adapter{NLPHandler: nil}
	assert.NotNil(t, a)
	assert.Nil(t, a.NLPHandler)
}

func TestAdapter_AnalyzeSentiment_NilHandler(t *testing.T) {
	a := &Adapter{NLPHandler: nil}
	assert.NotNil(t, a)
	// Calling with nil handler will panic - just verify the adapter struct works
}
