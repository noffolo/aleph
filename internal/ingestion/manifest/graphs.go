package manifest

import "sort"

// graphBuilder implements GraphManifestBuilder.
type graphBuilder struct{}

// NewGraphManifestBuilder creates a new GraphManifestBuilder.
func NewGraphManifestBuilder() GraphManifestBuilder {
	return &graphBuilder{}
}

// Build transforms entities, relations, and metrics into a GraphConfig.
func (g *graphBuilder) Build(entities []Entity, relations []Relation, metrics []MetricSuggestion) GraphConfig {
	// Build EntityRef slice from entities.
	refs := make([]EntityRef, 0, len(entities))
	for _, e := range entities {
		refs = append(refs, EntityRef{
			Name:        e.Name,
			KeyColumn:   e.KeyColumn,
			LabelColumn: e.LabelColumn,
		})
	}

	// Deterministic sort by Name.
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Name < refs[j].Name
	})

	// Build EdgeConfig slice from relations.
	edges := make([]EdgeConfig, 0, len(relations))
	for _, r := range relations {
		edges = append(edges, EdgeConfig{
			Source: r.Source,
			Target: r.Target,
			Type:   r.Type,
		})
	}

	// Deterministic sort by (Source, Target).
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	return GraphConfig{
		Name:      "auto_manifest_graph",
		Entities:  refs,
		Relations: edges,
	}
}
