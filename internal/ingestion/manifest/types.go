package manifest

import (
	"context"
	"fmt"

	"github.com/ff3300/aleph-v2/internal/storage"
)

// ── Column classification ──────────────────────────────────────────

// ColumnClass represents the semantic role of a database column.
type ColumnClass int

const (
	PrimaryKey ColumnClass = iota
	ForeignKey
	Label
	Category
	Measure
	Temporal
	Boolean
	Coordinate
	Ignored
)

// ── Schema types ────────────────────────────────────────────────────

// ColumnSchema holds metadata and classification for a single column.
type ColumnSchema struct {
	Name          string      // column name
	Type          string      // from PRAGMA table_info
	Nullable      bool
	IsPK          bool        // from PRAGMA pk column
	IsFK          bool
	FKTarget      string      // referenced table, if FK detected
	DistinctCount int64       // approx_unique from SUMMARIZE
	RowCount      int64       // COUNT(*) for rate computation
	SampleValues  []any       // SELECT DISTINCT col LIMIT 5
	Class         ColumnClass // filled by classifier
}

// TableSchema holds metadata and column definitions for a single table.
type TableSchema struct {
	Name     string
	Columns  []ColumnSchema
	RowCount int64
}

// ── Entity + Relation types ─────────────────────────────────────────

// Entity represents an inferred business entity from a database table.
type Entity struct {
	Name        string         // inferred from table name
	Table       string         // source table
	KeyColumn   string         // PK column name
	LabelColumn string         // display column (first label found)
	Properties  []ColumnSchema // properties (category, temporal, etc.)
}

// Relation represents a discovered relationship between two entities.
type Relation struct {
	Source     string  // source entity name
	Target     string  // target entity name
	ViaColumn  string  // FK column or join column
	ViaTable   string  // junction table (for M:N)
	Type       string  // membership, financial_flow, inferred, etc.
	Confidence float64 // 0.0-1.0
}

// ── Metric suggestion types ─────────────────────────────────────────

// AggType represents an aggregation function for metric computation.
type AggType int

const (
	Sum AggType = iota
	Avg
	Count
	Min
	Max
)

// MetricSuggestion represents a suggested aggregatable metric.
type MetricSuggestion struct {
	Name        string   // "trend_consenso", "volume_donazioni"
	SourceTable string
	Dimensions  []string // category columns for grouping
	Measure     string   // numeric column
	TemporalKey string   // DATE column, if any
	Aggregation AggType
}

// ── Graph manifest types ────────────────────────────────────────────

// GraphConfig represents a complete graph manifest with entities and relations.
type GraphConfig struct {
	Name      string
	Entities  []EntityRef
	Relations []EdgeConfig
}

// EntityRef is a lightweight reference to an entity in a graph manifest.
type EntityRef struct {
	Name        string
	KeyColumn   string
	LabelColumn string
}

// EdgeConfig represents a typed relationship edge in a graph manifest.
type EdgeConfig struct {
	Source       string
	Target       string
	Type         string
	WeightColumn string // optional: column for edge weight
}

// ── Domain Config ───────────────────────────────────────────────────

// DomainConfig holds domain-specific configuration for the manifest engine.
// It controls keyword matching, classification thresholds, and table filtering.
type DomainConfig struct {
	MeasureKeywords    []string            // e.g. ["importo","amount","value"]
	MetricMappings     map[string]AggType  // e.g. {"importo":Sum, "percentuale":Avg}
	LabelPatterns      []string            // e.g. ["nome","name","label","title"]
	IgnoreTables       []string            // tables to skip entirely
	MinDistinctRatio   float64             // threshold for label vs category (default 0.2)
	MaxCategoryDistinct int                // category threshold (default 20)
}

// DefaultDomainConfig returns a DomainConfig pre-configured for Italian political data.
// Supports both Italian and English keywords for cross-domain generalization.
func DefaultDomainConfig() DomainConfig {
	return DomainConfig{
		MeasureKeywords: []string{"importo", "amount", "value", "costo", "price", "voti", "quantity", "revenue", "spesa"},
		MetricMappings: map[string]AggType{
			"importo":      Avg,
			"amount":       Avg,
			"value":        Avg,
			"costo":        Avg,
			"price":        Avg,
			"voti":         Sum,
			"quantity":     Sum,
			"revenue":      Sum,
			"spesa":        Sum,
			"percentuale":  Avg,
			"perc":         Avg,
			"rate":         Avg,
			"tasso":        Avg,
		},
		LabelPatterns:      []string{"nome", "name", "label", "title", "descrizione"},
		MinDistinctRatio:   0.2,
		MaxCategoryDistinct: 20,
	}
}

// ── Interfaces ──────────────────────────────────────────────────────

// SchemaScanner introspects a database schema and produces table metadata.
type SchemaScanner interface {
	Scan(ctx context.Context, db storage.DBExecutor) ([]TableSchema, error)
}

// ColumnClassifier assigns semantic classes to columns based on heuristics.
type ColumnClassifier interface {
	Classify(tables []TableSchema) ([]TableSchema, error)
}

// EntityInferrer groups classified columns into entity definitions.
type EntityInferrer interface {
	Infer(tables []TableSchema) ([]Entity, error)
}

// RelationDiscoverer discovers relationships between entities via FK constraints
// and cross-table column containment analysis.
type RelationDiscoverer interface {
	Discover(entities []Entity, tables []TableSchema, db storage.DBExecutor) ([]Relation, error)
}

// MetricSuggester suggests aggregatable metrics from measure columns.
type MetricSuggester interface {
	Suggest(entities []Entity, tables []TableSchema) ([]MetricSuggestion, error)
}

// GraphManifestBuilder combines entities, relations, and metrics into a
// complete graph manifest structure.
type GraphManifestBuilder interface {
	Build(entities []Entity, relations []Relation, metrics []MetricSuggestion) GraphConfig
}

// ManifestEngine orchestrates the full manifest pipeline:
// scan → classify → infer → discover → suggest → build.
type ManifestEngine struct {
	scanner    SchemaScanner
	classifier ColumnClassifier
	inferrer   EntityInferrer
	discoverer RelationDiscoverer
	suggester  MetricSuggester
	builder    GraphManifestBuilder
}

// DiscoverResult holds the output of a full manifest discovery pipeline run.
type DiscoverResult struct {
	Entities  []Entity
	Relations []Relation
	Metrics   []MetricSuggestion
	Graph     GraphConfig
}

// Discover runs the full manifest pipeline: scan → classify → infer → discover → suggest → build.
func (e *ManifestEngine) Discover(ctx context.Context, db storage.DBExecutor) (*DiscoverResult, error) {
	tables, err := e.scanner.Scan(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("discover: scan: %w", err)
	}

	classified, err := e.classifier.Classify(tables)
	if err != nil {
		return nil, fmt.Errorf("discover: classify: %w", err)
	}

	entities, err := e.inferrer.Infer(classified)
	if err != nil {
		return nil, fmt.Errorf("discover: infer: %w", err)
	}

	relations, err := e.discoverer.Discover(entities, classified, db)
	if err != nil {
		return nil, fmt.Errorf("discover: discover relations: %w", err)
	}

	metrics, err := e.suggester.Suggest(entities, classified)
	if err != nil {
		return nil, fmt.Errorf("discover: suggest metrics: %w", err)
	}

	graph := e.builder.Build(entities, relations, metrics)

	return &DiscoverResult{
		Entities:  entities,
		Relations: relations,
		Metrics:   metrics,
		Graph:     graph,
	}, nil
}

// NewManifestEngine creates a new ManifestEngine with the provided components.
// Components are wired in by the caller to allow injection of any implementation.
func NewManifestEngine(scanner SchemaScanner, classifier ColumnClassifier, inferrer EntityInferrer, discoverer RelationDiscoverer, suggester MetricSuggester, builder GraphManifestBuilder) *ManifestEngine {
	return &ManifestEngine{
		scanner:    scanner,
		classifier: classifier,
		inferrer:   inferrer,
		discoverer: discoverer,
		suggester:  suggester,
		builder:    builder,
	}
}
