package humanecosystems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic profile data, not security-sensitive
	"time"
)

// ResearchProfiles generates privacy-preserving research profile metadata.
// No PII is ever stored or returned; all identifiers are SHA-256 hashed.
type ResearchProfiles struct {
	db *DuckDBLayer
}

// Name returns the tool name.
func (r *ResearchProfiles) Name() string {
	return "he_research_profiles"
}

// Description returns the tool description.
func (r *ResearchProfiles) Description() string {
	return "Analyzes research profiles and academic contributions in human ecosystems (beta) | is_synthetic=true | privacy-preserving"
}

// NewResearchProfiles creates a ResearchProfiles tool backed by the given DuckDB layer.
func NewResearchProfiles(db *DuckDBLayer) *ResearchProfiles {
	return &ResearchProfiles{db: db}
}

// Execute runs research profile analysis. Expects args with optional "query" string.
// Returns structured research profile data with hashed identifiers — no PII.
func (r *ResearchProfiles) Execute(ctx context.Context, args map[string]any) (any, error) {
	query, _ := args["query"].(string)

	if query == "" {
		query = "default ecosystem analysis"
	}

	if r.db.IsAvailable() {
		return r.queryProfiles(ctx, query)
	}
	return r.syntheticProfiles(query), nil
}

func (r *ResearchProfiles) queryProfiles(ctx context.Context, query string) (any, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT COUNT(*) as total FROM system_tools WHERE category LIKE 'human-ecosystems'`)
	if err != nil {
		return r.syntheticProfiles(query), nil
	}
	defer rows.Close()

	var total int
	if rows.Next() {
		rows.Scan(&total)
	}

	profiles := []map[string]any{
		{
			"profile_id":    sha256Hash("ecosystem:" + query),
			"research_area": query,
			"tool_count":    total,
			"is_synthetic":  false,
		},
	}

	return map[string]any{
		"profiles":     profiles,
		"is_synthetic": false,
		"query":        query,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (r *ResearchProfiles) syntheticProfiles(query string) map[string]any {
	seed := int64(hashString(query))
	rng := rand.New(rand.NewSource(seed))
	count := 3 + rng.Intn(5)

	profiles := make([]map[string]any, count)
	for i := 0; i < count; i++ {
		profiles[i] = map[string]any{
			"profile_id":    sha256Hash(fmt.Sprintf("profile:%s:%d", query, i)),
			"research_area": fmt.Sprintf("Area %d: %s", i+1, query),
			"tool_count":    rng.Intn(50),
			"is_synthetic":  true,
		}
	}

	return map[string]any{
		"profiles":     profiles,
		"is_synthetic": true,
		"query":        query,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// sha256Hash returns a hex-encoded SHA-256 hash for privacy-preserving
// pseudonymisation. The same input always produces the same hash.
func sha256Hash(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func hashString(s string) uint32 {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	return uint32(sum[0])<<24 | uint32(sum[1])<<16 | uint32(sum[2])<<8 | uint32(sum[3])
}

// marshalJSON is a helper that writes pretty-printed JSON for tool output.
func marshalJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}
