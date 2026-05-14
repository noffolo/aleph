package handler

import (
	"testing"
	"time"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/stretchr/testify/assert"
)

func TestStrPtr(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", *strPtr(s))

	empty := ""
	assert.Equal(t, "", *strPtr(empty))
	assert.Equal(t, "world", *strPtr("world"))
}

func TestDerefStr(t *testing.T) {
	assert.Equal(t, "", derefStr(nil))
	s := "value"
	assert.Equal(t, "value", derefStr(&s))
}

func TestFloatPtr(t *testing.T) {
	f32 := float32Ptr(3.14)
	assert.InDelta(t, float32(3.14), *f32, 0.001)

	f64 := float64Ptr(2.718)
	assert.InDelta(t, 2.718, *f64, 0.001)
}

func TestDerefFloat(t *testing.T) {
	assert.Equal(t, float64(0), derefFloat64(nil))
	assert.Equal(t, float64(0), derefFloat32(nil))

	f64 := 3.14
	assert.Equal(t, 3.14, derefFloat64(&f64))

	f32 := float32(2.718)
	assert.InDelta(t, 2.718, derefFloat32(&f32), 0.001)
}

func TestProtoFromMeta_BasicFields(t *testing.T) {
	now := time.Now().UTC()
	meta := registry.ComponentMetadata{
		ID:                "comp-1",
		Name:              "test-component",
		Description:       "A test component",
		Version:           "1.0.0",
		Type:              "tool",
		Category:          "finance",
		Source:            "internal",
		Status:            "active",
		ApprovalStatus:    "approved",
		CreationTimestamp: now,
		LastUpdatedTimestamp: now,
	}

	result := protoFromMeta(meta)

	assert.Equal(t, "comp-1", result.Id)
	assert.Equal(t, "test-component", result.Name)
	assert.Equal(t, "A test component", result.Description)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "tool", result.Type)
	assert.Equal(t, "finance", result.Category)
	assert.Equal(t, "internal", result.Source)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, "approved", result.ApprovalStatus)
}

func TestProtoFromMeta_NilFields(t *testing.T) {
	meta := registry.ComponentMetadata{
		ID:          "comp-min",
		Name:        "minimal",
		Description: "",
	}
	result := protoFromMeta(meta)

	// strPtr maps empty string to pointer-to-empty-string, not nil
	assert.NotNil(t, result.ConfigSchemaJson)
	assert.Equal(t, "", *result.ConfigSchemaJson)

	assert.NotNil(t, result.ExecutionCommand)
	assert.Equal(t, "", *result.ExecutionCommand)

	assert.NotNil(t, result.InputSchemaJson)
	assert.Equal(t, "", *result.InputSchemaJson)

	assert.NotNil(t, result.OutputSchemaJson)
	assert.Equal(t, "", *result.OutputSchemaJson)

	assert.NotNil(t, result.PromptTemplate)
	assert.Equal(t, "", *result.PromptTemplate)

	assert.NotNil(t, result.CreatedByAgentId)
	assert.Equal(t, "", *result.CreatedByAgentId)

	// float64Ptr maps 0 to pointer-to-0, not nil
	assert.NotNil(t, result.AvgCpuUsage)
	assert.Equal(t, float64(0), *result.AvgCpuUsage)

	assert.NotNil(t, result.AvgMemoryMb)
	assert.Equal(t, float64(0), *result.AvgMemoryMb)
}

func TestProtoFromMeta_WithOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	meta := registry.ComponentMetadata{
		ID:               "comp-full",
		Name:             "full",
		ConfigSchemaJSON: `{"type":"object"}`,
		InputSchemaJSON:  `{"properties":{}}`,
		OutputSchemaJSON: `{"result":"string"}`,
		DependenciesJSON: `["dep1"]`,
		PromptTemplate:   "You are {{name}}",
		AvgCpuUsage:      0.75,
		AvgMemoryMb:      128.0,
		AvgExecTimeMs:    42.0,
		AvgBrierScore:    0.15,
		AvgLatencyMs:     200.0,
		TrustScore:       0.95,
		CreatedByAgentId: "agent-42",
		CreationTimestamp: now,
		LastUpdatedTimestamp: now,
	}

	result := protoFromMeta(meta)

	assert.Equal(t, `{"type":"object"}`, *result.ConfigSchemaJson)
	assert.Equal(t, `{"properties":{}}`, *result.InputSchemaJson)
	assert.Equal(t, `{"result":"string"}`, *result.OutputSchemaJson)
	assert.Equal(t, `["dep1"]`, *result.DependenciesJson)
	assert.Equal(t, "You are {{name}}", *result.PromptTemplate)
	assert.InDelta(t, 0.75, *result.AvgCpuUsage, 0.001)
	assert.InDelta(t, 128.0, *result.AvgMemoryMb, 0.001)
	assert.InDelta(t, 42.0, *result.AvgExecTimeMs, 0.001)
	assert.InDelta(t, 0.15, *result.AvgBrierScore, 0.001)
	assert.InDelta(t, 200.0, *result.AvgLatencyMs, 0.001)
	assert.InDelta(t, float64(0.95), float64(*result.TrustScore), 0.01)
	assert.Equal(t, "agent-42", *result.CreatedByAgentId)
}

func TestNewRegistryServiceHandler(t *testing.T) {
	h := NewRegistryServiceHandler(nil, nil)
	assert.NotNil(t, h)
	assert.Nil(t, h.registryMgr)
}

func TestRegisterComponentRequest_NilMetadata(t *testing.T) {
	req := &v1.RegisterComponentRequest{}
	assert.Nil(t, req.Metadata)
}

func TestListComponentsRequest_EmptyFilter(t *testing.T) {
	req := &v1.ListComponentsRequest{}
	assert.Empty(t, req.Filter)
}

func TestUpdateComponentStatusRequest(t *testing.T) {
	req := &v1.UpdateComponentStatusRequest{
		Id:     "comp-1",
		Status: "deprecated",
	}
	assert.Equal(t, "comp-1", req.Id)
	assert.Equal(t, "deprecated", req.Status)
}
