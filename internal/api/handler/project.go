package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/storage"
)

type ProjectHandler struct {
	projectsRoot string
	db           *storage.DuckDB
}

func NewProjectHandler(projectsRoot string, db *storage.DuckDB) *ProjectHandler {
	return &ProjectHandler{projectsRoot: projectsRoot, db: db}
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

func (h *ProjectHandler) EmergeOntology(
	ctx context.Context,
	req *connect.Request[v1.EmergeOntologyRequest],
) (*connect.Response[v1.EmergeOntologyResponse], error) {
	rows, err := h.db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' AND table_name NOT LIKE 'system_%' ORDER BY table_name")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tn string
		if rows.Scan(&tn) == nil {
			tableNames = append(tableNames, tn)
		}
	}

	var alephDef strings.Builder
	for _, tableName := range tableNames {
		colRows, err := h.db.QueryContext(ctx,
			"SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position",
			tableName)
		if err != nil {
			continue
		}

		alephDef.WriteString(fmt.Sprintf("object %s\n", tableName))
		alephDef.WriteString(fmt.Sprintf("  from dataset %s\n", tableName))

		hasID := false
		for colRows.Next() {
			var colName, colType string
			colRows.Scan(&colName, &colType)
			mappedType := "text"
			if strings.Contains(strings.ToUpper(colType), "INT") || strings.Contains(strings.ToUpper(colType), "DOUBLE") || strings.Contains(strings.ToUpper(colType), "FLOAT") || strings.Contains(strings.ToUpper(colType), "DECIMAL") {
				mappedType = "number"
			} else if strings.Contains(strings.ToUpper(colType), "TIMESTAMP") || strings.Contains(strings.ToUpper(colType), "DATE") || strings.Contains(strings.ToUpper(colType), "TIME") {
				mappedType = "datetime"
			} else if strings.Contains(strings.ToUpper(colType), "BOOLEAN") || strings.Contains(strings.ToUpper(colType), "BOOL") {
				mappedType = "boolean"
			}
			if strings.EqualFold(colName, "id") && !hasID {
				alephDef.WriteString("  id id\n")
				hasID = true
				continue
			}
			alephDef.WriteString(fmt.Sprintf("  property %s type %s from %s\n", colName, mappedType, colName))
		}
		colRows.Close()
		if !hasID {
			alephDef.WriteString("  id id\n")
		}
		alephDef.WriteString("\n")
	}

	return connect.NewResponse(&v1.EmergeOntologyResponse{
		AlephDefinition: alephDef.String(),
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
