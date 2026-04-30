package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

type ProjectHandler struct {
	projectsRoot string
	db           *storage.DuckDB
	llm          llm.Provider
	ontoRepo     *repository.OntologyRepository
}

func NewProjectHandler(projectsRoot string, db *storage.DuckDB) *ProjectHandler {
	return &ProjectHandler{projectsRoot: projectsRoot, db: db}
}

// SetLLMProvider sets the LLM provider for ontology emergence.
func (h *ProjectHandler) SetLLMProvider(p llm.Provider) {
	h.llm = p
}

// SetOntologyRepository sets the ontology repository for version management.
func (h *ProjectHandler) SetOntologyRepository(r *repository.OntologyRepository) {
	h.ontoRepo = r
}

// columnInfo describes a single column from information_schema.
type columnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// tableSchema holds schema info for a single table.
type tableSchema struct {
	Name    string       `json:"name"`
	Columns []columnInfo `json:"columns"`
}

// detectedRelationship holds an inferred FK or naming-based relationship.
type detectedRelationship struct {
	Name       string `json:"name"`
	FromObject string `json:"from_object"`
	FromColumn string `json:"from_column"`
	ToObject   string `json:"to_object"`
	ToColumn   string `json:"to_column"`
	Type       string `json:"type"` // "fk", "name_match"
	Confidence string `json:"confidence"` // "high", "medium", "low"
}

// mapDuckDBType maps DuckDB column types to aleph DSL property types.
func mapDuckDBType(rawType string) string {
	upper := strings.ToUpper(rawType)
	switch {
	case strings.Contains(upper, "INT"), strings.Contains(upper, "DOUBLE"), strings.Contains(upper, "FLOAT"), strings.Contains(upper, "DECIMAL"):
		return "number"
	case strings.Contains(upper, "TIMESTAMP"), strings.Contains(upper, "DATE"), strings.Contains(upper, "TIME"):
		return "datetime"
	case strings.Contains(upper, "BOOLEAN"), strings.Contains(upper, "BOOL"):
		return "boolean"
	default:
		return "text"
	}
}

func (h *ProjectHandler) ListProjects(
	ctx context.Context,
	req *connect.Request[v1.ListProjectsRequest],
) (*connect.Response[v1.ListProjectsResponse], error) {
	entries, err := os.ReadDir(h.projectsRoot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var projects []*v1.Project
	for _, entry := range entries {
		if entry.IsDir() {
			info, _ := entry.Info()
			projects = append(projects, &v1.Project{
				Id:        entry.Name(),
				Name:      entry.Name(),
				CreatedAt: info.ModTime().Unix(),
			})
		}
	}

	return connect.NewResponse(&v1.ListProjectsResponse{Projects: projects}), nil
}

func (h *ProjectHandler) CreateProject(
	ctx context.Context,
	req *connect.Request[v1.CreateProjectRequest],
) (*connect.Response[v1.CreateProjectResponse], error) {
	id := req.Msg.Id
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project id is required"))
	}

	path := filepath.Join(h.projectsRoot, id)
	dirs := []string{
		filepath.Join(path, "raw"),
		filepath.Join(path, "ontologies"),
		filepath.Join(path, "agents"),
		filepath.Join(path, "skills"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// Create a placeholder core.aleph if it doesn't exist
	ontFile := filepath.Join(path, "ontologies", "core.aleph")
	if _, err := os.Stat(ontFile); os.IsNotExist(err) {
		os.WriteFile(ontFile, []byte("// Define your ontology here\n"), 0644)
	}

	p := &v1.Project{Id: id, Name: req.Msg.Name, CreatedAt: time.Now().Unix()}
	return connect.NewResponse(&v1.CreateProjectResponse{Project: p}), nil
}

func (h *ProjectHandler) GetOntology(
	ctx context.Context,
	req *connect.Request[v1.GetOntologyRequest],
) (*connect.Response[v1.GetOntologyResponse], error) {
	projectID := req.Msg.ProjectId
	ontPath, err := sanitizePath(h.projectsRoot, projectID, "ontologies", "core.aleph")
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	
	content, err := os.ReadFile(ontPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Parse to get object names
	prog, err := dsl.Parse(string(content))
	var names []string
	if err == nil {
		for _, s := range prog.Statements {
			if s.Object != nil {
				names = append(names, s.Object.Name)
			}
		}
	}

	if len(names) == 0 {
		rows, err := h.db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' AND table_name NOT LIKE 'system_%' ORDER BY table_name")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var tn string
				if rows.Scan(&tn) == nil {
					names = append(names, tn)
				}
			}
		}
	}

	return connect.NewResponse(&v1.GetOntologyResponse{
		AlephDefinition: string(content),
		ObjectNames:     names,
	}), nil
}

func (h *ProjectHandler) SaveOntology(
	ctx context.Context,
	req *connect.Request[v1.SaveOntologyRequest],
) (*connect.Response[v1.SaveOntologyResponse], error) {
	projectID := req.Msg.ProjectId
	ontPath, err := sanitizePath(h.projectsRoot, projectID, "ontologies", "core.aleph")
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	
	// Atomic Backup
	if _, err := os.Stat(ontPath); err == nil {
		bakPath := ontPath + "." + fmt.Sprintf("%d", time.Now().Unix()) + ".bak"
		content, _ := os.ReadFile(ontPath)
		// Atomic write for backup too
		tmpBak := bakPath + ".tmp"
		os.WriteFile(tmpBak, content, 0644)
		os.Rename(tmpBak, bakPath)
	}

	// Atomic Save
	tmpFile := ontPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(req.Msg.AlephDefinition), 0644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := os.Rename(tmpFile, ontPath); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.SaveOntologyResponse{Success: true}), nil
}

func (h *ProjectHandler) collectTableSchemas(ctx context.Context) ([]tableSchema, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' AND table_name NOT LIKE 'system_%' ORDER BY table_name")
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	// Drain the first cursor fully before opening new queries to avoid
	// deadlock on :memory: DuckDB (SetMaxOpenConns=1)
	var tableNames []string
	for rows.Next() {
		var tn string
		if rows.Scan(&tn) == nil {
			tableNames = append(tableNames, tn)
		}
	}
	rows.Close()

	var schemas []tableSchema
	for _, tn := range tableNames {
		colRows, err := h.db.QueryContext(ctx,
			"SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position", tn)
		if err != nil {
			continue
		}
		s := tableSchema{Name: tn}
		for colRows.Next() {
			var cn, ct string
			if colRows.Scan(&cn, &ct) == nil {
				s.Columns = append(s.Columns, columnInfo{Name: cn, Type: ct})
			}
		}
		colRows.Close()
		schemas = append(schemas, s)
	}
	return schemas, nil
}

// detectFKRelationships infers FK relationships from column naming patterns.
// Looks for columns like "user_id" → "users.id", "customer_id" → "customers.id", etc.
func detectFKRelationships(schemas []tableSchema) []detectedRelationship {
	// Build column index: object name → set of column names
	objCols := make(map[string]map[string]bool)
	objLower := make(map[string]string) // lowercase name → canonical name
	for _, s := range schemas {
		cols := make(map[string]bool)
		for _, c := range s.Columns {
			cols[strings.ToLower(c.Name)] = true
		}
		objCols[strings.ToLower(s.Name)] = cols
		objLower[strings.ToLower(s.Name)] = s.Name
	}

	var rels []detectedRelationship
	for _, s := range schemas {
		for _, c := range s.Columns {
			lowCol := strings.ToLower(c.Name)
			// Pattern: singular_id → look for plural table
			// e.g. "customer_id" → "customers" or "customer"
			if strings.HasSuffix(lowCol, "_id") && len(lowCol) > 3 {
				refBase := strings.TrimSuffix(lowCol, "_id")
				// Try exact, plural, and singular matches
				candidates := []string{refBase, refBase + "s", strings.TrimSuffix(refBase, "s")}
				for _, cand := range candidates {
					if target, ok := objLower[cand]; ok && target != strings.ToLower(s.Name) {
						rels = append(rels, detectedRelationship{
							Name:       fmt.Sprintf("%s_has_%s", s.Name, target),
							FromObject: s.Name,
							FromColumn: c.Name,
							ToObject:   target,
							ToColumn:   "id",
							Type:       "fk",
							Confidence: "high",
						})
						break
					}
				}
			}
			// Pattern: column name matches another object's name (e.g. "category" column → "categories" table)
			if !strings.HasSuffix(lowCol, "_id") {
				for objNameLower, objCanon := range objLower {
					if objNameLower != strings.ToLower(s.Name) && lowCol == objNameLower {
						rels = append(rels, detectedRelationship{
							Name:       fmt.Sprintf("%s_references_%s", s.Name, objCanon),
							FromObject: s.Name,
							FromColumn: c.Name,
							ToObject:   objCanon,
							ToColumn:   "id",
							Type:       "name_match",
							Confidence: "medium",
						})
					}
				}
			}
		}
	}
	return rels
}

// buildAlephDefinition generates an aleph DSL definition from schemas and relationships.
func buildAlephDefinition(schemas []tableSchema, rels []detectedRelationship) string {
	var buf strings.Builder
	for _, s := range schemas {
		buf.WriteString(fmt.Sprintf("object %s\n", s.Name))
		buf.WriteString(fmt.Sprintf("  from dataset %s\n", s.Name))
		hasID := false
		for _, c := range s.Columns {
			if !hasID && strings.EqualFold(c.Name, "id") {
				buf.WriteString("  id id\n")
				hasID = true
				continue
			}
			buf.WriteString(fmt.Sprintf("  property %s type %s from %s\n", c.Name, mapDuckDBType(c.Type), c.Name))
		}
		if !hasID {
			buf.WriteString("  id id\n")
		}
		buf.WriteString("\n")
	}
	for _, r := range rels {
		buf.WriteString(fmt.Sprintf("relation %s from %s.%s to %s.%s equals %s.%s\n",
			r.Name, r.FromObject, r.FromColumn, r.ToObject, r.ToColumn, r.FromObject, r.FromColumn))
		buf.WriteString(fmt.Sprintf("  type %s\n", r.Type))
		buf.WriteString(fmt.Sprintf("  // confidence: %s\n", r.Confidence))
	}
	return buf.String()
}

// emergePrompt generates the LLM prompt for ontology emergence.
func (h *ProjectHandler) emergePrompt(schemas []tableSchema, rels []detectedRelationship) string {
	var buf strings.Builder
	buf.WriteString("Sei un ontologo esperto. Analizzi sorgenti dati e produci un'ontologia strutturata nel formato DSL aleph.\n\n")
	buf.WriteString("Regole:\n")
	buf.WriteString("- Ogni \"source\" (tabella) diventa un Object\n")
	buf.WriteString("- Colonne con nomi simili tra sorgenti diverse potrebbero essere relazioni\n")
	buf.WriteString("- Type inference: string → text, int/float → number, date/time → datetime, bool → boolean\n")
	buf.WriteString("- Relazioni: FK naming (user_id → users.id), name overlap (customer_name → customer.name)\n")
	buf.WriteString("- Output in formato DSL aleph (objects + relations)\n\n")
	buf.WriteString("Database tables and columns:\n\n")
	for _, s := range schemas {
		buf.WriteString(fmt.Sprintf("Table: %s\n", s.Name))
		for _, c := range s.Columns {
			buf.WriteString(fmt.Sprintf("  - %s (%s)\n", c.Name, c.Type))
		}
		buf.WriteString("\n")
	}
	if len(rels) > 0 {
		buf.WriteString("Automatically detected relationships (for reference):\n")
		for _, r := range rels {
			buf.WriteString(fmt.Sprintf("  - %s.%s → %s.%s (confidence: %s)\n", r.FromObject, r.FromColumn, r.ToObject, r.ToColumn, r.Confidence))
		}
		buf.WriteString("\n")
	}
	buf.WriteString("Generate the full aleph DSL ontology definition for these tables. Include all objects, properties with correct types, and all detected relationships.\n")
	return buf.String()
}

func (h *ProjectHandler) EmergeOntology(
	ctx context.Context,
	req *connect.Request[v1.EmergeOntologyRequest],
) (*connect.Response[v1.EmergeOntologyResponse], error) {
	schemas, err := h.collectTableSchemas(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("collect schemas: %w", err))
	}

	rels := detectFKRelationships(schemas)

	alephDef := buildAlephDefinition(schemas, rels)

	if h.llm != nil {
		resp, llmErr := h.llm.Complete(ctx, llm.CompletionRequest{
			Model: "llama3.2",
			Messages: []map[string]interface{}{
				{"role": "user", "content": h.emergePrompt(schemas, rels)},
			},
		})
		if llmErr == nil && resp.Content != "" {
			alephDef = resp.Content
		}
	}

	return connect.NewResponse(&v1.EmergeOntologyResponse{
		AlephDefinition: alephDef,
	}), nil
}

func (h *ProjectHandler) DeleteProject(
	ctx context.Context,
	req *connect.Request[v1.DeleteProjectRequest],
) (*connect.Response[v1.DeleteProjectResponse], error) {
	id := req.Msg.Id
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project id is required"))
	}

	path, err := sanitizePath(h.projectsRoot, id)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }

	if err := os.RemoveAll(path); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteProjectResponse{Success: true}), nil
}

// ── Ontology Negotiation HTTP Handlers (W2C-01) ────────────────────────────

// NegotiatePropose handles POST /api/v1/ontology/propose
// Accepts a JSON body with {project_id, parent_version_id, diff, source_description, rationale, confidence}
// and creates a new ontology version proposal.
func (h *ProjectHandler) NegotiatePropose(w http.ResponseWriter, r *http.Request) {
	if h.ontoRepo == nil {
		http.Error(w, `{"error":"ontology repository not available"}`, http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	var req struct {
		ProjectID        string `json:"project_id"`
		ParentVersionID  string `json:"parent_version_id"`
		AlephDefinition  string `json:"aleph_definition"`
		DiffJSON         string `json:"diff_json"`
		SourceDescription string `json:"source_description"`
		Rationale        string `json:"rationale"`
		Confidence       float64 `json:"confidence"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	if req.ProjectID == "" || req.AlephDefinition == "" {
		http.Error(w, `{"error":"project_id and aleph_definition are required"}`, http.StatusBadRequest)
		return
	}
	versionID, err := h.ontoRepo.ProposeOntologyDiff(r.Context(),
		req.ProjectID, req.ParentVersionID, req.DiffJSON, req.AlephDefinition,
		req.SourceDescription, req.Rationale, req.Confidence)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version_id": versionID,
		"preview":    req.AlephDefinition,
		"warnings":   []string{},
	})
}

// NegotiateAccept handles POST /api/v1/ontology/accept
func (h *ProjectHandler) NegotiateAccept(w http.ResponseWriter, r *http.Request) {
	if h.ontoRepo == nil {
		http.Error(w, `{"error":"ontology repository not available"}`, http.StatusServiceUnavailable)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		VersionID string `json:"version_id"`
	}
	json.Unmarshal(body, &req)
	if req.VersionID == "" {
		http.Error(w, `{"error":"version_id is required"}`, http.StatusBadRequest)
		return
	}
	if err := h.ontoRepo.AcceptDiff(r.Context(), req.VersionID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version_id": req.VersionID, "status": "accepted"})
}

// NegotiateReject handles POST /api/v1/ontology/reject
func (h *ProjectHandler) NegotiateReject(w http.ResponseWriter, r *http.Request) {
	if h.ontoRepo == nil {
		http.Error(w, `{"error":"ontology repository not available"}`, http.StatusServiceUnavailable)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		VersionID string `json:"version_id"`
		Reason    string `json:"reason"`
	}
	json.Unmarshal(body, &req)
	if req.VersionID == "" {
		http.Error(w, `{"error":"version_id is required"}`, http.StatusBadRequest)
		return
	}
	if err := h.ontoRepo.RejectDiff(r.Context(), req.VersionID, req.Reason); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version_id": req.VersionID, "status": "rejected"})
}

// NegotiateList handles GET /api/v1/ontology/versions?project_id=X&limit=N
func (h *ProjectHandler) NegotiateList(w http.ResponseWriter, r *http.Request) {
	if h.ontoRepo == nil {
		http.Error(w, `{"error":"ontology repository not available"}`, http.StatusServiceUnavailable)
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, `{"error":"project_id is required"}`, http.StatusBadRequest)
		return
	}
	versions, err := h.ontoRepo.ListVersions(r.Context(), projectID, 20)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	type versionEntry struct {
		VersionID        string `json:"version_id"`
		ParentVersionID  string `json:"parent_version_id,omitempty"`
		CreatedAt        string `json:"created_at"`
		Status           string `json:"status"`
		SourceDescription string `json:"source_description,omitempty"`
		Rationale        string `json:"rationale,omitempty"`
		Confidence       float64 `json:"confidence,omitempty"`
	}
	var entries []versionEntry
	for _, v := range versions {
		e := versionEntry{
			VersionID:        v.VersionID,
			CreatedAt:        v.CreatedAt.Format(time.RFC3339),
			Status:           v.Status,
			SourceDescription: v.SourceDescription.String,
			Rationale:        v.Rationale.String,
			Confidence:       v.Confidence.Float64,
		}
		if v.ParentVersionID.Valid {
			e.ParentVersionID = v.ParentVersionID.String
		}
		entries = append(entries, e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"versions": entries})
}
