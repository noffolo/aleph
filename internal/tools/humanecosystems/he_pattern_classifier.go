package humanecosystems

import (
	"context"
	"fmt"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic mock data, not security-sensitive
	"strings"
	"time"
)

// PatternClassifier identifies and classifies behavioral and structural
// patterns in human ecosystem interaction data.
type PatternClassifier struct {
	db *DuckDBLayer
}

// Name returns the tool name.
func (p *PatternClassifier) Name() string {
	return "he_pattern_classifier"
}

// Description returns the tool description.
func (p *PatternClassifier) Description() string {
	return "Classifies and identifies patterns in human ecosystem interactions (beta) | is_synthetic=true | privacy-preserving"
}

// NewPatternClassifier creates a PatternClassifier backed by the given DuckDB layer.
func NewPatternClassifier(db *DuckDBLayer) *PatternClassifier {
	return &PatternClassifier{db: db}
}

// Execute runs pattern classification. Expects args with optional "data" string.
// Returns classified patterns with confidence scores — no PII.
func (p *PatternClassifier) Execute(ctx context.Context, args map[string]any) (any, error) {
	data, _ := args["data"].(string)
	if data == "" {
		data = "default_pattern_data"
	}

	if p.db.IsAvailable() {
		return p.queryPatterns(ctx, data)
	}
	return p.syntheticPatterns(data), nil
}

func (p *PatternClassifier) queryPatterns(ctx context.Context, data string) (any, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, name, category FROM system_tools WHERE source_type = 'package' LIMIT 5`)
	if err != nil {
		return p.syntheticPatterns(data), nil
	}
	defer rows.Close()

	var patterns []map[string]any
	for rows.Next() {
		var id, name, category string
		if err := rows.Scan(&id, &name, &category); err != nil {
			continue
		}
		for _, pat := range builtinPatterns {
			if strings.Contains(strings.ToLower(name), pat.Keyword) {
				patterns = append(patterns, map[string]any{
					"pattern_id":   sha256Hash(fmt.Sprintf("pat:%s:%s", data, id)),
					"pattern_type": pat.Name,
					"confidence":   0.8,
					"matched_on":   name,
					"is_synthetic": false,
				})
				break
			}
		}
	}

	if patterns == nil {
		patterns = []map[string]any{}
	}

	return map[string]any{
		"patterns":     patterns,
		"is_synthetic": false,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (p *PatternClassifier) syntheticPatterns(data string) map[string]any {
	seed := int64(hashString(data))
	rng := rand.New(rand.NewSource(seed))
	count := 2 + rng.Intn(4)

	patterns := make([]map[string]any, count)
	for i := 0; i < count; i++ {
		pat := builtinPatterns[rng.Intn(len(builtinPatterns))]
		patterns[i] = map[string]any{
			"pattern_id":   sha256Hash(fmt.Sprintf("pat:%s:%d", data, i)),
			"pattern_type": pat.Name,
			"confidence":   roundFloat(0.5+rng.Float64()*0.5, 2),
			"matched_on":   pat.Keyword,
			"is_synthetic": true,
		}
	}

	return map[string]any{
		"patterns":     patterns,
		"is_synthetic": true,
		"data":         data,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

type patternDef struct {
	Name    string
	Keyword string
}

var builtinPatterns = []patternDef{
	{Name: "collaborative_network", Keyword: "collaborat"},
	{Name: "hierarchical_structure", Keyword: "hierarchy"},
	{Name: "peer_to_peer", Keyword: "peer"},
	{Name: "centralized_hub", Keyword: "central"},
	{Name: "distributed_mesh", Keyword: "distributed"},
	{Name: "cyclic_dependency", Keyword: "cycle"},
	{Name: "linear_pipeline", Keyword: "pipeline"},
	{Name: "star_topology", Keyword: "star"},
}
