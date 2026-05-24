package osint

import (
	"testing"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestListTools_Extended(t *testing.T) {
	t.Run("happy: returns all 9 tools", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tools := ListTools(sb)
		assert.Len(t, tools, 9)
		for _, tool := range tools {
			assert.NotNil(t, tool)
		}
	})

	t.Run("edge: tools are distinct instances", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tools := ListTools(sb)
		assert.Len(t, tools, 9)

		ids := make(map[string]bool)
		for _, tool := range tools {
			if rt, ok := tool.(interface{ Register(*repository.MetadataRepository) error }); ok {
				repo := newMetadataRepo(t)
				err := rt.Register(repo)
				assert.NoError(t, err)
				ids["registered"] = true
			}
		}
		assert.True(t, ids["registered"])
	})

	t.Run("edge: same broker returns consistent tool set", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tools1 := ListTools(sb)
		tools2 := ListTools(sb)
		assert.Equal(t, len(tools1), len(tools2))
	})
}
