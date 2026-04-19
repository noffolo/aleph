package handler

import (
	"context"
	"fmt"
	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/registry"
    // Importo il pacchetto generato corretto
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

type RegistryServiceHandler struct {
	registryMgr *registry.DuckDBRegistry
}

func NewRegistryServiceHandler(regMgr *registry.DuckDBRegistry, logger any) *RegistryServiceHandler {
	return &RegistryServiceHandler{registryMgr: regMgr}
}

func (h *RegistryServiceHandler) RegisterComponent(ctx context.Context, req *connect.Request[v1.RegisterComponentRequest]) (*connect.Response[v1.RegisterComponentResponse], error) {
	meta := registry.ComponentMetadata{
		Name: req.Msg.Metadata.Name,
		Type: req.Msg.Metadata.Type,
	}
	id, err := h.registryMgr.RegisterComponent(meta)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&v1.RegisterComponentResponse{ComponentId: id}), nil
}

func (h *RegistryServiceHandler) ListComponents(ctx context.Context, req *connect.Request[v1.ListComponentsRequest]) (*connect.Response[v1.ListComponentsResponse], error) {
	comps, _ := h.registryMgr.ListComponents(req.Msg.Filter)
	var protoComps []*v1.ComponentMetadata
	for _, c := range comps {
		protoComps = append(protoComps, &v1.ComponentMetadata{Id: c.ID, Name: c.Name, Type: c.Type})
	}
	return connect.NewResponse(&v1.ListComponentsResponse{Components: protoComps}), nil
}

func (h *RegistryServiceHandler) GetComponent(ctx context.Context, req *connect.Request[v1.GetComponentRequest]) (*connect.Response[v1.GetComponentResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func (h *RegistryServiceHandler) UpdateComponentStatus(ctx context.Context, req *connect.Request[v1.UpdateComponentStatusRequest]) (*connect.Response[v1.UpdateComponentStatusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}
