package dsl

// OntologySuggestion represents a proposed change to an ontology.
// It wraps a diff with metadata about the source (e.g. LLM suggestion)
// and the current negotiation state.
type OntologySuggestion struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	Diff        *OntologyDiff     `json:"diff"`
	Status      NegotiationStatus `json:"status"`
	SuggestedBy string            `json:"suggested_by,omitempty"` // "llm" | "user" | "system"
	Rationale   string            `json:"rationale,omitempty"`    // LLM rationale
	Confidence  float64           `json:"confidence,omitempty"`   // LLM confidence 0.0-1.0
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// NegotiationStatus represents the state of an ontology suggestion.
type NegotiationStatus int

const (
	NegotiationPending    NegotiationStatus = 0
	NegotiationAccepted   NegotiationStatus = 1
	NegotiationRejected   NegotiationStatus = 2
	NegotiationModified   NegotiationStatus = 3
	NegotiationSuperseded NegotiationStatus = 4
)

// String returns the human-readable form of NegotiationStatus.
func (s NegotiationStatus) String() string {
	switch s {
	case NegotiationPending:
		return "pending"
	case NegotiationAccepted:
		return "accepted"
	case NegotiationRejected:
		return "rejected"
	case NegotiationModified:
		return "modified"
	case NegotiationSuperseded:
		return "superseded"
	default:
		return "unknown"
	}
}

// OntologyDiff describes an atomic set of changes to an ontology.
type OntologyDiff struct {
	ID                string            `json:"id"`
	ProjectID         string            `json:"project_id"`
	ParentVersionID   string            `json:"parent_version_id,omitempty"`
	ObjectsAdd        []ObjectAdd       `json:"objects_add,omitempty"`
	ObjectsModify     []ObjectModify    `json:"objects_modify,omitempty"`
	ObjectsRemove     []ObjectRemove    `json:"objects_remove,omitempty"`
	RelationsAdd      []RelationAdd     `json:"relations_add,omitempty"`
	RelationsModify   []RelationModify  `json:"relations_modify,omitempty"`
	RelationsRemove   []RelationRemove  `json:"relations_remove,omitempty"`
	SourceDescription string            `json:"source_description,omitempty"`
	Rationale         string            `json:"rationale,omitempty"`
	Confidence        float64           `json:"confidence,omitempty"`
	Status            NegotiationStatus `json:"status"`
	CoreAlephPreview  string            `json:"core_aleph_preview,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
}

// Relationship describes a parsed relation between two ontology objects.
// This is the runtime representation of RelationDefinition enriched
// with a typed relation type for the decision engine.
type Relationship struct {
	Name         string `json:"name"`
	FromObject   string `json:"from_object"`
	ToObject     string `json:"to_object"`
	OnProperty   string `json:"on_property"`
	RelationType string `json:"relation_type"` // "fk" | "contains" | "references" | "derives_from"
}

// ObjectAdd describes a new object to add to the ontology.
type ObjectAdd struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Properties  []string          `json:"properties,omitempty"`
	TypeHints   map[string]string `json:"type_hints,omitempty"` // property_name → "text|number|datetime|boolean"
	FromSource  string            `json:"from_source,omitempty"`
}

// ObjectModify describes modifications to an existing ontology object.
type ObjectModify struct {
	Name              string            `json:"name"`
	PropertiesAdd     []string          `json:"properties_add,omitempty"`
	PropertiesRemove  []string          `json:"properties_remove,omitempty"`
	TypeHintsUpdate   map[string]string `json:"type_hints_update,omitempty"`
	DescriptionUpdate string            `json:"description_update,omitempty"`
}

// ObjectRemove describes an object to remove from the ontology.
type ObjectRemove struct {
	Name string `json:"name"`
}

// RelationAdd describes a new relation to add.
type RelationAdd struct {
	Name         string `json:"name"`
	FromObject   string `json:"from_object"`
	ToObject     string `json:"to_object"`
	OnProperty   string `json:"on_property"`
	RelationType string `json:"relation_type"` // "fk" | "contains" | "references" | "derives_from"
}

// RelationModify describes modifications to an existing relation.
type RelationModify struct {
	Name               string `json:"name"`
	OnPropertyUpdate   string `json:"on_property_update,omitempty"`
	RelationTypeUpdate string `json:"relation_type_update,omitempty"`
}

// RelationRemove describes a relation to remove.
type RelationRemove struct {
	Name string `json:"name"`
}

// VersionEntry represents a single entry in the ontology version history.
type VersionEntry struct {
	VersionID         string `json:"version_id"`
	ParentVersionID   string `json:"parent_version_id,omitempty"`
	CreatedAt         string `json:"created_at"`
	Status            string `json:"status"` // "accepted" | "rejected" | "superseded"
	SourceDescription string `json:"source_description,omitempty"`
}
