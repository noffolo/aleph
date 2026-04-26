package adaptation

import (
	"context"
	"fmt"
	"time"

	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// Suggester provides tool adaptation suggestions.
type Suggester struct {
	metaRepo  *repository.MetadataRepository
	discovery *mcp.DiscoveryEngine
}

// NewSuggester creates a new suggestion engine.
func NewSuggester(metaRepo *repository.MetadataRepository, discovery *mcp.DiscoveryEngine) *Suggester {
	return &Suggester{
		metaRepo:  metaRepo,
		discovery: discovery,
	}
}

// matchQuality checks how well a query matches a tool's name and description.
// Returns a confidence score and a human-readable reason.
func matchQuality(query, name, description string) (float64, string) {
	if query == "" {
		return 0.0, "empty query"
	}

	if name == query {
		return 0.95, "exact tool name match"
	}

	if description == query {
		return 0.70, "exact tool description match"
	}

	nameMatch := false
	descMatch := false

	if len(query) <= len(name) {
		lowerName := name
		lowerQuery := query
		for i := 0; i <= len(lowerName)-len(lowerQuery); i++ {
			if lowerName[i:i+len(lowerQuery)] == lowerQuery {
				nameMatch = true
				break
			}
		}
	}

	if len(query) <= len(description) {
		lowerDesc := description
		lowerQuery := query
		for i := 0; i <= len(lowerDesc)-len(lowerQuery); i++ {
			if lowerDesc[i:i+len(lowerQuery)] == lowerQuery {
				descMatch = true
				break
			}
		}
	}

	if nameMatch {
		return 0.70, "partial tool name match"
	}
	if descMatch {
		return 0.50, "partial tool description match"
	}

	return 0.0, "no match"
}

// Suggest returns adaptation suggestions for a query.
// It queries the metadata repository for tool definitions matching the query
// text and returns real Suggestion objects populated with actual tool data.
func (s *Suggester) Suggest(ctx context.Context, query string) ([]Suggestion, error) {
	tools, err := s.metaRepo.ListTools()
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	var suggestions []Suggestion
	for _, tool := range tools {
		confidence, reason := matchQuality(query, tool.Name, tool.Description)
		if confidence <= 0.0 {
			continue
		}

		suggestions = append(suggestions, Suggestion{
			Query: query,
			ToolDef: mcp.ToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Category:    tool.Category,
				Version:     tool.Version,
			},
			Confidence:   confidence,
			Reason:       reason,
			TemplateType: TemplateWrapper,
		})
	}

	if suggestions == nil {
		suggestions = []Suggestion{}
	}

	return suggestions, nil
}

// VersionSnapshot represents a point-in-time snapshot of a tool definition.
type VersionSnapshot struct {
	Version   string
	Tool      mcp.ToolDefinition
	Timestamp time.Time
	Reason    string
}

// VersioningRollback handles tool version management and rollback.
type VersioningRollback struct {
	metaRepo *repository.MetadataRepository
	versions []VersionSnapshot
}

// NewVersioningRollback creates a versioning rollback handler.
func NewVersioningRollback(metaRepo *repository.MetadataRepository) *VersioningRollback {
	return &VersioningRollback{
		metaRepo: metaRepo,
		versions: make([]VersionSnapshot, 0),
	}
}

// Snapshot saves the current tool definition state as a versioned snapshot.
// Versions are generated sequentially (v1, v2, v3, ...).
func (v *VersioningRollback) Snapshot(tool mcp.ToolDefinition, reason string) {
	ver := fmt.Sprintf("v%d", len(v.versions)+1)
	v.versions = append(v.versions, VersionSnapshot{
		Version:   ver,
		Tool:      tool,
		Timestamp: time.Now(),
		Reason:    reason,
	})
}

// ListVersions returns all available version snapshots.
func (v *VersioningRollback) ListVersions() ([]VersionSnapshot, error) {
	if len(v.versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}
	result := make([]VersionSnapshot, len(v.versions))
	copy(result, v.versions)
	return result, nil
}

// Rollback reverts a tool to a previous version.
// It finds the snapshot by version string and returns the snapshot data.
// Actual DB restore is deferred to a future implementation.
func (v *VersioningRollback) Rollback(version string) error {
	if version == "" {
		return fmt.Errorf("version required for rollback")
	}
	for _, snap := range v.versions {
		if snap.Version == version {
			return nil
		}
	}
	return fmt.Errorf("version %q not found", version)
}
