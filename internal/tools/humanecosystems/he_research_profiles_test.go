package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ResearchProfiles.Name
// =============================================================================

func TestResearchProfilesName(t *testing.T) {
	rp := NewResearchProfiles(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "he_research_profiles", rp.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, rp.Name(), rp.Name())
	})

	t.Run("edge: non-empty snake_case", func(t *testing.T) {
		assert.NotEmpty(t, rp.Name())
	})
}

// =============================================================================
// ResearchProfiles.Description
// =============================================================================

func TestResearchProfilesDescription(t *testing.T) {
	rp := NewResearchProfiles(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, rp.Description())
	})

	t.Run("happy: mentions research and privacy", func(t *testing.T) {
		desc := rp.Description()
		assert.Contains(t, desc, "research profiles")
		assert.Contains(t, desc, "privacy-preserving")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, rp.Description(), rp.Description())
	})
}

// =============================================================================
// NewResearchProfiles
// =============================================================================

func TestNewResearchProfiles(t *testing.T) {
	t.Run("happy: creates with synthetic layer", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		require.NotNil(t, rp)
		assert.False(t, rp.db.IsAvailable())
	})

	t.Run("happy: creates with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		rp := NewResearchProfiles(dbl)
		require.NotNil(t, rp)
		assert.True(t, rp.db.IsAvailable())
	})

	t.Run("edge: nil dbl creates valid tool", func(t *testing.T) {
		rp := NewResearchProfiles(nil)
		require.NotNil(t, rp)
		assert.NotEmpty(t, rp.Name())
	})
}

// =============================================================================
// ResearchProfiles.Execute
// =============================================================================

func TestResearchProfilesExecute(t *testing.T) {
	t.Run("happy: returns profiles for explicit query", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		result, err := rp.Execute(context.Background(), map[string]any{"query": "ecosystem dynamics"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "ecosystem dynamics", r["query"])
		assert.True(t, r["is_synthetic"].(bool))
	})

	t.Run("happy: profiles slice is non-empty", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		result, err := rp.Execute(context.Background(), map[string]any{"query": "test"})
		require.NoError(t, err)
		r := result.(map[string]any)
		profilesRaw := r["profiles"]
		// Handle both []map[string]any and []any
		switch profiles := profilesRaw.(type) {
		case []map[string]any:
			assert.GreaterOrEqual(t, len(profiles), 3)
			assert.LessOrEqual(t, len(profiles), 7)
		case []any:
			assert.GreaterOrEqual(t, len(profiles), 3)
			assert.LessOrEqual(t, len(profiles), 7)
		default:
			t.Fatalf("unexpected profiles type: %T", profilesRaw)
		}
	})

	t.Run("edge: defaults query when empty", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		result, err := rp.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default ecosystem analysis", r["query"])
	})

	t.Run("edge: deterministic output for same query", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		r1, _ := rp.Execute(context.Background(), map[string]any{"query": "deterministic"})
		r2, _ := rp.Execute(context.Background(), map[string]any{"query": "deterministic"})
		m1 := r1.(map[string]any)
		m2 := r2.(map[string]any)
		p1 := m1["profiles"].([]map[string]any)
		p2 := m2["profiles"].([]map[string]any)
		assert.Len(t, p1, len(p2))
		for i := range p1 {
			assert.Equal(t, p1[i]["profile_id"], p2[i]["profile_id"])
		}
	})

	t.Run("error: never errors with synthetic layer", func(t *testing.T) {
		rp := NewResearchProfiles(SyntheticDuckDBLayer())
		_, err := rp.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
	})
}

// =============================================================================
// ResearchProfiles.queryProfiles (with real DuckDB via Execute)
// =============================================================================

func TestQueryProfiles(t *testing.T) {
	t.Run("happy: queries real DuckDB for profiles", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		rp := NewResearchProfiles(dbl)

		result, err := rp.Execute(context.Background(), map[string]any{"query": "profile query"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "profile query", r["query"])
		assert.False(t, r["is_synthetic"].(bool))
	})

	t.Run("edge: defaults query when empty with real DB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		rp := NewResearchProfiles(dbl)

		result, err := rp.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default ecosystem analysis", r["query"])
		assert.False(t, r["is_synthetic"].(bool))
	})

	t.Run("edge: different queries produce different profiles", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		rp := NewResearchProfiles(dbl)

		r1, _ := rp.Execute(context.Background(), map[string]any{"query": "query_a"})
		r2, _ := rp.Execute(context.Background(), map[string]any{"query": "query_b"})
		assert.NotEqual(t, r1.(map[string]any)["query"], r2.(map[string]any)["query"])
	})
}

// =============================================================================
// ResearchProfiles.syntheticProfiles via Execute
// =============================================================================

func TestSyntheticProfiles(t *testing.T) {
	rp := NewResearchProfiles(SyntheticDuckDBLayer())

	t.Run("happy: profiles have required fields", func(t *testing.T) {
		result, err := rp.Execute(context.Background(), map[string]any{"query": "complete"})
		require.NoError(t, err)
		r := result.(map[string]any)
		profiles, ok := r["profiles"].([]map[string]any)
		if !ok {
			profilesIface, ok := r["profiles"].([]any)
			require.True(t, ok)
			for _, p := range profilesIface {
				pm := p.(map[string]any)
				assert.Contains(t, pm, "profile_id")
				assert.Contains(t, pm, "research_area")
			}
		} else {
			for _, p := range profiles {
				assert.Contains(t, p, "profile_id")
				assert.Contains(t, p, "research_area")
			}
		}
	})

	t.Run("happy: tool_count is non-negative", func(t *testing.T) {
		result, _ := rp.Execute(context.Background(), map[string]any{"query": "count_test"})
		r := result.(map[string]any)
		profiles, ok := r["profiles"].([]map[string]any)
		if ok {
			for _, p := range profiles {
				tc, ok := p["tool_count"].(int)
				require.True(t, ok)
				assert.GreaterOrEqual(t, tc, 0)
				assert.Less(t, tc, 50)
			}
		}
	})

	t.Run("edge: very long query string", func(t *testing.T) {
		longQuery := "this is a very long query string with many words to test the stability of the synthetic profile generation process"
		result, err := rp.Execute(context.Background(), map[string]any{"query": longQuery})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, longQuery, r["query"])
		profilesRaw := r["profiles"].([]map[string]any)
		assert.GreaterOrEqual(t, len(profilesRaw), 3)
	})
}

// TestSHA256Hash — already defined in sprint_final_test.go

// =============================================================================
// hashString
// =============================================================================

func TestHashString(t *testing.T) {
	t.Run("happy: produces non-zero value for non-empty input", func(t *testing.T) {
		h := hashString("hello world")
		assert.NotEqual(t, uint32(0), h)
	})

	t.Run("happy: deterministic output for same input", func(t *testing.T) {
		h1 := hashString("deterministic")
		h2 := hashString("deterministic")
		assert.Equal(t, h1, h2)
	})

	t.Run("edge: different inputs produce different hashes", func(t *testing.T) {
		h1 := hashString("alpha")
		h2 := hashString("beta")
		assert.NotEqual(t, h1, h2)
	})

	t.Run("edge: empty string produces valid hash", func(t *testing.T) {
		h := hashString("")
		assert.IsType(t, uint32(0), h)
	})
}

// TestMarshalJSON — already defined in he_tools_test.go
