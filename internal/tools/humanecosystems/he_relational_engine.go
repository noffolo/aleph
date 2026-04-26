package humanecosystems

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RelationalEngine maps and analyzes relational connections between
// ecosystem participants using DuckDB queries when available.
type RelationalEngine struct {
	db *DuckDBLayer
}

// Name returns the tool name.
func (r *RelationalEngine) Name() string {
	return "he_relational_engine"
}

// Description returns the tool description.
func (r *RelationalEngine) Description() string {
	return "Maps and analyzes relational connections between ecosystem participants (beta) | is_synthetic=true | privacy-preserving"
}

// NewRelationalEngine creates a RelationalEngine backed by the given DuckDB layer.
func NewRelationalEngine(db *DuckDBLayer) *RelationalEngine {
	return &RelationalEngine{db: db}
}

// Execute runs relational analysis. Expects args with optional "entity" string.
// Returns structured relation data with hashed identifiers — no PII.
func (r *RelationalEngine) Execute(ctx context.Context, args map[string]any) (any, error) {
	entity, _ := args["entity"].(string)
	if entity == "" {
		entity = "default"
	}

	if r.db.IsAvailable() {
		return r.queryRelational(ctx, entity)
	}
	return r.syntheticRelational(entity), nil
}

func (r *RelationalEngine) queryRelational(ctx context.Context, entity string) (any, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, category FROM system_tools WHERE source_type = 'package' LIMIT 10`)
	if err != nil {
		return r.syntheticRelational(entity), nil
	}
	defer rows.Close()

	var relations []map[string]interface{}
	for rows.Next() {
		var id, name, category string
		if err := rows.Scan(&id, &name, &category); err != nil {
			continue
		}
		relations = append(relations, map[string]interface{}{
			"relation_id":    sha256Hash(fmt.Sprintf("rel:%s:%s", entity, id)),
			"related_entity": id,
			"relation_type":  "dependency",
			"strength":       50,
			"is_synthetic":   false,
		})
	}

	if relations == nil {
		relations = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"entity":      entity,
		"relations":   relations,
		"is_synthetic": false,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (r *RelationalEngine) syntheticRelational(entity string) map[string]interface{} {
	seed := int64(hashString(entity))
	rng := rand.New(rand.NewSource(seed))
	count := 3 + rng.Intn(5)

	relations := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		relations[i] = map[string]interface{}{
			"relation_id":    sha256Hash(fmt.Sprintf("rel:%s:%d", entity, i)),
			"related_entity": fmt.Sprintf("entity_%s_%d", entity, i),
			"relation_type":  relationTypes[rng.Intn(len(relationTypes))],
			"strength":       rng.Intn(100),
			"is_synthetic":   true,
		}
	}

	return map[string]interface{}{
		"entity":      entity,
		"relations":   relations,
		"is_synthetic": true,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

var relationTypes = []string{
	"dependency", "collaboration", "hierarchy", "peer", "influence",
}
