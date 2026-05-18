package handler

import (
	"connectrpc.com/connect"
	"context"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RegistryServiceHandler struct {
	registryMgr *registry.DuckDBRegistry
}

func NewRegistryServiceHandler(regMgr *registry.DuckDBRegistry, logger any) *RegistryServiceHandler {
	return &RegistryServiceHandler{registryMgr: regMgr}
}

func strPtr(s string) *string       { return &s }
func float32Ptr(f float64) *float32 { v := float32(f); return &v }
func float64Ptr(f float64) *float64 { return &f }
func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
func derefFloat64(f *float64) float64 {
	if f != nil {
		return *f
	}
	return 0
}
func derefFloat32(f *float32) float64 {
	if f != nil {
		return float64(*f)
	}
	return 0
}

func protoFromMeta(c registry.ComponentMetadata) *v1.ComponentMetadata {
	return &v1.ComponentMetadata{
		Id:                   c.ID,
		Name:                 c.Name,
		Description:          c.Description,
		Version:              c.Version,
		Type:                 c.Type,
		Category:             c.Category,
		Source:               c.Source,
		Status:               c.Status,
		ApprovalStatus:       c.ApprovalStatus,
		ConfigSchemaJson:     strPtr(c.ConfigSchemaJSON),
		ExecutionCommand:     strPtr(c.ExecutionCommand),
		DependenciesJson:     strPtr(c.DependenciesJSON),
		InputSchemaJson:      strPtr(c.InputSchemaJSON),
		OutputSchemaJson:     strPtr(c.OutputSchemaJSON),
		PromptTemplate:       strPtr(c.PromptTemplate),
		ToolIdsJson:          strPtr(c.ToolIdsJSON),
		AvgCpuUsage:          float64Ptr(c.AvgCpuUsage),
		AvgMemoryMb:          float64Ptr(c.AvgMemoryMb),
		AvgExecTimeMs:        float64Ptr(c.AvgExecTimeMs),
		AvgBrierScore:        float64Ptr(c.AvgBrierScore),
		AvgLatencyMs:         float64Ptr(c.AvgLatencyMs),
		TrustScore:           float32Ptr(c.TrustScore),
		CreatedByAgentId:     strPtr(c.CreatedByAgentId),
		CreationTimestamp:    timestamppb.New(c.CreationTimestamp),
		LastUpdatedTimestamp: timestamppb.New(c.LastUpdatedTimestamp),
	}
}

func (h *RegistryServiceHandler) RegisterComponent(ctx context.Context, req *connect.Request[v1.RegisterComponentRequest]) (*connect.Response[v1.RegisterComponentResponse], error) {
	m := req.Msg.Metadata
	meta := registry.ComponentMetadata{
		Name:             m.Name,
		Description:      m.Description,
		Version:          m.Version,
		Type:             m.Type,
		Category:         m.Category,
		Source:           m.Source,
		Status:           m.Status,
		ApprovalStatus:   m.ApprovalStatus,
		ConfigSchemaJSON: derefStr(m.ConfigSchemaJson),
		ExecutionCommand: derefStr(m.ExecutionCommand),
		DependenciesJSON: derefStr(m.DependenciesJson),
		InputSchemaJSON:  derefStr(m.InputSchemaJson),
		OutputSchemaJSON: derefStr(m.OutputSchemaJson),
		PromptTemplate:   derefStr(m.PromptTemplate),
		ToolIdsJSON:      derefStr(m.ToolIdsJson),
		AvgCpuUsage:      derefFloat64(m.AvgCpuUsage),
		AvgMemoryMb:      derefFloat64(m.AvgMemoryMb),
		AvgExecTimeMs:    derefFloat64(m.AvgExecTimeMs),
		AvgBrierScore:    derefFloat64(m.AvgBrierScore),
		AvgLatencyMs:     derefFloat64(m.AvgLatencyMs),
		TrustScore:       derefFloat32(m.TrustScore),
		CreatedByAgentId: derefStr(m.CreatedByAgentId),
	}
	id, err := h.registryMgr.RegisterComponent(meta)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RegisterComponentResponse{ComponentId: id}), nil
}

func (h *RegistryServiceHandler) ListComponents(ctx context.Context, req *connect.Request[v1.ListComponentsRequest]) (*connect.Response[v1.ListComponentsResponse], error) {
	comps, err := h.registryMgr.ListComponents(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var protoComps []*v1.ComponentMetadata
	for _, c := range comps {
		protoComps = append(protoComps, protoFromMeta(c))
	}
	return connect.NewResponse(&v1.ListComponentsResponse{Components: protoComps}), nil
}

func (h *RegistryServiceHandler) GetComponent(ctx context.Context, req *connect.Request[v1.GetComponentRequest]) (*connect.Response[v1.GetComponentResponse], error) {
	meta, err := h.registryMgr.GetComponentByID(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.GetComponentResponse{Metadata: protoFromMeta(*meta)}), nil
}

func (h *RegistryServiceHandler) UpdateComponentStatus(ctx context.Context, req *connect.Request[v1.UpdateComponentStatusRequest]) (*connect.Response[v1.UpdateComponentStatusResponse], error) {
	err := h.registryMgr.UpdateComponentStatus(req.Msg.Id, req.Msg.Status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateComponentStatusResponse{}), nil
}
