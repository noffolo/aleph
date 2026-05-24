package manifest

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ff3300/aleph-v2/internal/storage"
)

// relationDiscoverer discovers entity relationships via FK constraints,
// junction-table analysis, and cross-table column containment probes.
type relationDiscoverer struct {
	cfg DomainConfig
}

// NewRelationDiscoverer creates a RelationDiscoverer configured with the given DomainConfig.
func NewRelationDiscoverer(cfg DomainConfig) RelationDiscoverer {
	return &relationDiscoverer{cfg: cfg}
}

// Discover finds relationships among entities using three strategies:
//  1. FK-based — direct foreign-key columns, confidence 1.0
//  2. Junction tables — tables with 2+ FK and 0 PK, produces M:N pairs, confidence 1.0
//  3. Containment ratio — INTERSECT probes on overlapping VARCHAR columns, confidence 0.8
//
// Results are sorted by confidence descending, then deterministically by source and target name.
func (r *relationDiscoverer) Discover(entities []Entity, tables []TableSchema, db storage.DBExecutor) ([]Relation, error) {
	// Build lookups.
	entityByTable := make(map[string]Entity, len(entities))
	for _, e := range entities {
		entityByTable[e.Table] = e
	}

	tableByName := make(map[string]TableSchema, len(tables))
	for _, t := range tables {
		tableByName[t.Name] = t
	}

	// Track which entity pairs are already linked (by source-target key)
	// so we avoid redundant containment probes.
	linkedPairs := make(map[string]bool)

	var relations []Relation

	// ── Step 1: FK-based relations (confidence 1.0) ──────────────────────
	for _, table := range tables {
		for _, col := range table.Columns {
			if col.FKTarget == "" {
				continue
			}
			sourceEntity, sourceOK := entityByTable[table.Name]
			targetEntity, targetOK := entityByTable[col.FKTarget]
			if !sourceOK || !targetOK {
				continue
			}
			// Skip self-references.
			if sourceEntity.Name == targetEntity.Name {
				continue
			}
			pairKey := relationPairKey(sourceEntity.Name, targetEntity.Name)
			if linkedPairs[pairKey] {
				continue
			}
			linkedPairs[pairKey] = true

			relations = append(relations, Relation{
				Source:     sourceEntity.Name,
				Target:     targetEntity.Name,
				ViaColumn:  col.Name,
				Type:       inferRelationType(col.Name),
				Confidence: 1.0,
			})
		}
	}

	// ── Step 2: Junction tables → M:N relations (confidence 1.0) ─────────
	for _, table := range tables {
		if !isJunctionForRelations(table.Columns) {
			continue
		}
		fkTargets := resolveFKTargets(table.Columns, entityByTable)
		if len(fkTargets) < 2 {
			continue
		}
		// Create all pairwise M:N relations between the FK-referenced entities.
		for i := 0; i < len(fkTargets); i++ {
			for j := i + 1; j < len(fkTargets); j++ {
				relations = append(relations, Relation{
					Source:     fkTargets[i],
					Target:     fkTargets[j],
					ViaTable:   table.Name,
					Type:       "membership",
					Confidence: 1.0,
				})
				relations = append(relations, Relation{
					Source:     fkTargets[j],
					Target:     fkTargets[i],
					ViaTable:   table.Name,
					Type:       "membership",
					Confidence: 1.0,
				})
			}
		}
	}

	// ── Step 3: Cross-table containment (confidence 0.8) ─────────────────
	ctx := context.Background()
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			eA, eB := entities[i], entities[j]
			pairKey := relationPairKey(eA.Name, eB.Name)
			if linkedPairs[pairKey] {
				continue
			}

			tA, okA := tableByName[eA.Table]
			tB, okB := tableByName[eB.Table]
			if !okA || !okB {
				continue
			}

			found := false
			for _, colA := range tA.Columns {
				if found {
					break
				}
				for _, colB := range tB.Columns {
					if found {
						break
					}
					if !eligibleForContainment(colA, colB) {
						continue
					}

					shared, minCard, err := r.runContainmentProbe(ctx, db,
						eA.Table, colA.Name,
						eB.Table, colB.Name,
						colA.DistinctCount, colB.DistinctCount,
					)
					if err != nil || minCard < 3 || shared < 3 {
						continue
					}

					containment := float64(shared) / float64(minCard)
					if containment < 0.7 {
						continue
					}

					relations = append(relations, Relation{
						Source:     eA.Name,
						Target:     eB.Name,
						ViaColumn:  colA.Name,
						Type:       "inferred",
						Confidence: 0.8,
					})
					linkedPairs[pairKey] = true
					found = true
				}
			}
		}
	}

	// ── Sort: confidence descending → source ascending → target ascending ──
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].Confidence != relations[j].Confidence {
			return relations[i].Confidence > relations[j].Confidence
		}
		if relations[i].Source != relations[j].Source {
			return relations[i].Source < relations[j].Source
		}
		return relations[i].Target < relations[j].Target
	})

	return relations, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// relationPairKey produces a deterministic key for an unordered entity pair.
func relationPairKey(a, b string) string {
	if a < b {
		return a + "||" + b
	}
	return b + "||" + a
}

// inferRelationType maps a column name to a relation type.
// Columns suggesting numeric flow (amount, cost, value) produce "financial_flow";
// everything else defaults to "membership".
func inferRelationType(colName string) string {
	lower := strings.ToLower(colName)
	for _, kw := range []string{"importo", "amount", "costo", "cost", "valore", "value", "prezzo", "price", "revenue", "spesa"} {
		if strings.Contains(lower, kw) {
			return "financial_flow"
		}
	}
	return "membership"
}

// isJunctionForRelations detects junction tables: 2+ FK columns, 0 PK, ≤3 total columns.
// Uses FKTarget (set by the classifier) rather than the IsFK struct field.
func isJunctionForRelations(cols []ColumnSchema) bool {
	if len(cols) > 3 {
		return false
	}
	fkCount := 0
	hasPK := false
	for _, col := range cols {
		if col.FKTarget != "" {
			fkCount++
		}
		if col.IsPK || col.Class == PrimaryKey {
			hasPK = true
		}
	}
	return fkCount >= 2 && !hasPK
}

// resolveFKTargets extracts entity names from a junction table's FK columns.
// Each FKTarget references a table name; we look up its corresponding entity.
// Returns deduplicated entity names.
func resolveFKTargets(cols []ColumnSchema, entityByTable map[string]Entity) []string {
	seen := make(map[string]bool)
	var targets []string
	for _, col := range cols {
		if col.FKTarget == "" {
			continue
		}
		ent, ok := entityByTable[col.FKTarget]
		if !ok {
			continue
		}
		if !seen[ent.Name] {
			seen[ent.Name] = true
			targets = append(targets, ent.Name)
		}
	}
	sort.Strings(targets)
	return targets
}

// eligibleForContainment returns true when two columns are candidates for a
// containment-ratio probe:
//   - Same data type (case-insensitive)
//   - Both are VARCHAR / TEXT / CHAR / STRING
//   - Neither is an FK column
//   - Names overlap (case-insensitive substring containment)
func eligibleForContainment(colA, colB ColumnSchema) bool {
	if !strings.EqualFold(colA.Type, colB.Type) {
		return false
	}
	upper := strings.ToUpper(colA.Type)
	if !isVarcharish(upper) {
		return false
	}
	if colA.FKTarget != "" || colB.FKTarget != "" {
		return false
	}
	nameA := strings.ToLower(colA.Name)
	nameB := strings.ToLower(colB.Name)
	return strings.Contains(nameA, nameB) || strings.Contains(nameB, nameA)
}

// isVarcharish returns true for string-like SQL types.
func isVarcharish(t string) bool {
	return strings.Contains(t, "VARCHAR") ||
		strings.Contains(t, "TEXT") ||
		strings.Contains(t, "CHAR") ||
		strings.Contains(t, "STRING")
}

// runContainmentProbe executes an INTERSECT query to measure value overlap
// between two columns from different tables.
//
// Returns:
//
//	shared  — row count of the intersection
//	minCard — the smaller of the two column's distinct counts
//	err     — any query or scan error
func (r *relationDiscoverer) runContainmentProbe(
	ctx context.Context,
	db storage.DBExecutor,
	tableA, colA string,
	tableB, colB string,
	distA, distB int64,
) (shared int64, minCard int64, err error) {
	// Guard against missing or zero distinct counts.
	if distA <= 0 || distB <= 0 {
		return 0, 0, nil
	}
	if distA < distB {
		minCard = distA
	} else {
		minCard = distB
	}

	query := fmt.Sprintf(
		`SELECT COUNT(*) FROM (SELECT DISTINCT a."%s" FROM "%s" a INTERSECT SELECT DISTINCT b."%s" FROM "%s" b) shared`,
		colA, tableA, colB, tableB,
	)

	if err := db.QueryRowContext(ctx, query).Scan(&shared); err != nil {
		return 0, minCard, fmt.Errorf("containment probe (%s.%s ↔ %s.%s): %w",
			tableA, colA, tableB, colB, err)
	}

	return shared, minCard, nil
}
