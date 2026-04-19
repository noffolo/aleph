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

	p := &v1.Project{Id: id, Name: req.Msg.Name, CreatedAt: 0} // Simplify for now
	return connect.NewResponse(&v1.CreateProjectResponse{Project: p}), nil
}

func (h *ProjectHandler) GetOntology(
	ctx context.Context,
	req *connect.Request[v1.GetOntologyRequest],
) (*connect.Response[v1.GetOntologyResponse], error) {
	projectID := req.Msg.ProjectId
	ontPath := filepath.Join(h.projectsRoot, projectID, "ontologies", "core.aleph")
	
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
	ontPath := filepath.Join(h.projectsRoot, projectID, "ontologies", "core.aleph")
	
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
	projectID := req.Msg.ProjectId
	rawPath := filepath.Join(h.projectsRoot, projectID, "raw")

	entries, err := os.ReadDir(rawPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var alephDef strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			datasetName := entry.Name()
			parquetPattern := filepath.Join(rawPath, datasetName, "latest", "*.parquet")
			
			query := fmt.Sprintf("DESCRIBE SELECT * FROM read_parquet('%s')", parquetPattern)
			
			queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			rows, err := h.db.QueryContext(queryCtx, query)
			if err != nil {
				cancel()
				continue
			}

			alephDef.WriteString(fmt.Sprintf("object %s\n", datasetName))
			alephDef.WriteString(fmt.Sprintf("from dataset %s\n", datasetName))
			alephDef.WriteString("id id\n")

			for rows.Next() {
				var colName, colType, colNull, colKey, colDefault, colExtra interface{}
				rows.Scan(&colName, &colType, &colNull, &colKey, &colDefault, &colExtra)
				mappedType := "text"
				dt := fmt.Sprintf("%v", colType)
				if strings.Contains(dt, "INT") || strings.Contains(dt, "DOUBLE") {
					mappedType = "number"
				} else if strings.Contains(dt, "TIMESTAMP") || strings.Contains(dt, "DATE") {
					mappedType = "datetime"
				}
				alephDef.WriteString(fmt.Sprintf("property %v type %s from %v\n", colName, mappedType, colName))
			}
			
			rows.Close()
			cancel()
			alephDef.WriteString("\n")
			rows.Close()
		}
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

	path := filepath.Join(h.projectsRoot, id)
	if err := os.RemoveAll(path); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DeleteProjectResponse{Success: true}), nil
}
