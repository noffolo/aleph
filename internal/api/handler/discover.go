package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	_ "github.com/marcboeker/go-duckdb"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/errors"
	"github.com/ff3300/aleph-v2/internal/ingestion/manifest"
	"github.com/ff3300/aleph-v2/internal/storage"
)

var _ v1connect.DiscoveryServiceHandler = (*DiscoveryHandler)(nil)

type DiscoveryHandler struct {
	db           storage.DBExecutor
	projectsRoot string
	engine       *manifest.ManifestEngine
}

func NewDiscoveryHandler(db storage.DBExecutor, projectsRoot string) *DiscoveryHandler {
	cfg := manifest.DefaultDomainConfig()
	return &DiscoveryHandler{
		db:           db,
		projectsRoot: projectsRoot,
		engine: manifest.NewManifestEngine(
			manifest.NewScanner(cfg),
			manifest.NewClassifier(cfg),
			manifest.NewEntityInferrer(cfg),
			manifest.NewRelationDiscoverer(cfg),
			manifest.NewMetricSuggester(cfg),
			manifest.NewGraphManifestBuilder(),
		),
	}
}

func (h *DiscoveryHandler) DiscoverDatabase(
	ctx context.Context,
	req *connect.Request[v1.DiscoverDatabaseRequest],
) (*connect.Response[v1.DiscoverDatabaseResponse], error) {
	projectID := req.Msg.ProjectId
	if projectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id is required"))
	}

	manifestID := newID()
	targetDB := h.db

	if dbPath := req.Msg.DbPath; dbPath != "" {
		opened, err := sql.Open("duckdb", dbPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("cannot open discovery DB at %s: %w", dbPath, err))
		}
		defer opened.Close()
		targetDB = opened
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("discover goroutine panic", "projectID", projectID, "manifestID", manifestID, "recover", r)
			}
		}()

		discoverCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		result, err := h.engine.Discover(discoverCtx, targetDB)
		if err != nil {
			slog.Error("discover pipeline failed", "projectID", projectID, "manifestID", manifestID, "error", err)
			return
		}

		manifestPath, pathErr := sanitizePath(h.projectsRoot, projectID, "manifests", manifestID+".json")
		if pathErr != nil {
			slog.Error("discover: invalid manifest path", "projectID", projectID, "error", pathErr)
			return
		}

		pbManifest := buildProtoManifest(manifestID, projectID, result)
		if saveErr := saveManifestToDisk(manifestPath, pbManifest); saveErr != nil {
			slog.Error("discover: failed to save manifest", "projectID", projectID, "manifestID", manifestID, "error", saveErr)
		}
	}()

	return connect.NewResponse(&v1.DiscoverDatabaseResponse{
		TaskId: manifestID,
		Status: "started",
	}), nil
}

func (h *DiscoveryHandler) GetManifest(
	_ context.Context,
	req *connect.Request[v1.GetManifestRequest],
) (*connect.Response[v1.GetManifestResponse], error) {
	projectID := req.Msg.ProjectId
	manifestID := req.Msg.ManifestId
	if projectID == "" || manifestID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and manifest_id are required"))
	}

	manifestPath, err := sanitizePath(h.projectsRoot, projectID, "manifests", manifestID+".json")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound,
			errors.NewAPIErrorWithMeta(errors.ErrNotFound, "manifest not found", err, "discovery", "read", false, 0))
	}

	var pbManifest v1.Manifest
	if err := json.Unmarshal(data, &pbManifest); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse manifest: %w", err))
	}

	return connect.NewResponse(&v1.GetManifestResponse{Manifest: &pbManifest}), nil
}

func (h *DiscoveryHandler) SaveManifest(
	_ context.Context,
	req *connect.Request[v1.SaveManifestRequest],
) (*connect.Response[v1.SaveManifestResponse], error) {
	projectID := req.Msg.ProjectId
	pbManifest := req.Msg.Manifest
	if projectID == "" || pbManifest == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and manifest are required"))
	}

	manifestID := pbManifest.Id
	if manifestID == "" {
		manifestID = newID()
		pbManifest.Id = manifestID
	}
	pbManifest.ProjectId = projectID

	manifestPath, err := sanitizePath(h.projectsRoot, projectID, "manifests", manifestID+".json")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := saveManifestToDisk(manifestPath, pbManifest); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save manifest: %w", err))
	}

	return connect.NewResponse(&v1.SaveManifestResponse{Manifest: pbManifest}), nil
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func buildProtoManifest(id, projectID string, result *manifest.DiscoverResult) *v1.Manifest {
	pb := &v1.Manifest{
		Id:        id,
		ProjectId: projectID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, e := range result.Entities {
		pb.Entities = append(pb.Entities, entityToProto(e))
	}
	for _, r := range result.Relations {
		pb.Relations = append(pb.Relations, relationToProto(r))
	}
	for _, m := range result.Metrics {
		pb.Metrics = append(pb.Metrics, metricToProto(m))
	}
	pb.Graph = graphToProto(result.Graph)

	return pb
}

func entityToProto(e manifest.Entity) *v1.Entity {
	pb := &v1.Entity{
		Name:        e.Name,
		Table:       e.Table,
		KeyColumn:   e.KeyColumn,
		LabelColumn: e.LabelColumn,
	}
	for _, p := range e.Properties {
		pb.Properties = append(pb.Properties, &v1.Property{
			Name:        p.Name,
			Type:        p.Type,
			ColumnClass: colClassToString(p.Class),
		})
	}
	return pb
}

func relationToProto(r manifest.Relation) *v1.Relation {
	return &v1.Relation{
		Source:     r.Source,
		Target:     r.Target,
		ViaColumn:  r.ViaColumn,
		ViaTable:   r.ViaTable,
		Type:       r.Type,
		Confidence: r.Confidence,
	}
}

func metricToProto(m manifest.MetricSuggestion) *v1.Metric {
	return &v1.Metric{
		Name:        m.Name,
		SourceTable: m.SourceTable,
		Dimensions:  m.Dimensions,
		Measure:     m.Measure,
		TemporalKey: m.TemporalKey,
		Aggregation: aggTypeToString(m.Aggregation),
	}
}

func graphToProto(g manifest.GraphConfig) *v1.GraphConfig {
	pb := &v1.GraphConfig{Name: g.Name}
	for _, e := range g.Entities {
		pb.Entities = append(pb.Entities, &v1.EntityRef{
			Name:        e.Name,
			KeyColumn:   e.KeyColumn,
			LabelColumn: e.LabelColumn,
		})
	}
	for _, e := range g.Relations {
		pb.Edges = append(pb.Edges, &v1.Edge{
			Source:       e.Source,
			Target:       e.Target,
			Type:         e.Type,
			WeightColumn: e.WeightColumn,
		})
	}
	return pb
}

func colClassToString(c manifest.ColumnClass) string {
	switch c {
	case manifest.PrimaryKey:
		return "PRIMARY_KEY"
	case manifest.ForeignKey:
		return "FOREIGN_KEY"
	case manifest.Label:
		return "LABEL"
	case manifest.Category:
		return "CATEGORY"
	case manifest.Measure:
		return "MEASURE"
	case manifest.Temporal:
		return "TEMPORAL"
	case manifest.Boolean:
		return "BOOLEAN"
	case manifest.Coordinate:
		return "COORDINATE"
	case manifest.Ignored:
		return "IGNORED"
	default:
		return "UNKNOWN"
	}
}

func aggTypeToString(a manifest.AggType) string {
	switch a {
	case manifest.Sum:
		return "SUM"
	case manifest.Avg:
		return "AVG"
	case manifest.Count:
		return "COUNT"
	case manifest.Min:
		return "MIN"
	case manifest.Max:
		return "MAX"
	default:
		return "AVG"
	}
}

func saveManifestToDisk(path string, pb *v1.Manifest) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}

	data, err := json.MarshalIndent(pb, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}
