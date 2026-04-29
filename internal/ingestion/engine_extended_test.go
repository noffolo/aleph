package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsColon(t *testing.T) {
	tests := []struct {
		s       string
		hasColon bool
	}{
		{"host:993", true},
		{"imap.gmail.com", false},
		{":", true},
		{"", false},
		{"host:port:extra", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.hasColon, containsColon(tt.s), "containsColon(%q)", tt.s)
		})
	}
}

func TestNewEngine(t *testing.T) {
	type args struct {
		projectsRoot string
	}
	tests := []struct {
		name string
		args args
	}{
		{"empty root", args{""}},
		{"with path", args{"/tmp/test-projects"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(tt.args.projectsRoot, nil, nil, nil)
			assert.NotNil(t, eng)
			assert.Equal(t, tt.args.projectsRoot, eng.projectsRoot)
			assert.Nil(t, eng.metaRepo)
			assert.Nil(t, eng.db)
			assert.Nil(t, eng.nlpHandler)
			assert.NotNil(t, eng.tasks)
			assert.Empty(t, eng.tasks)
		})
	}
}

func TestNewEngine_WithDependencies(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NotNil(t, eng.tasks)
	assert.Empty(t, eng.tasks)

	// Map should be ready to use
	eng.tasks["test"] = nil
	assert.Contains(t, eng.tasks, "test")
}

func TestEngine_CloseExtended(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	err := eng.Close()
	assert.NoError(t, err)
}

func TestEngine_CloseMultiple(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NoError(t, eng.Close())
	assert.NoError(t, eng.Close()) // Closing twice should be safe
}

func TestVerifyChecksum_EmptyExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), ""))
}

func TestVerifyChecksum_ShortExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), "short"))
}
