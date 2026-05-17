package osint

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTools(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: "http://localhost"})
	tools := ListTools(sb)
	assert.Len(t, tools, 9)
}

func TestIPLookupTool_Execute_MarshalError(t *testing.T) {
	tool := NewIPLookupTool()
	_, err := tool.Execute(context.Background(), `{"ip":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ip is required")
}

func TestDNSResolutionTool_Execute_MarshalError(t *testing.T) {
	tool := NewDNSResolutionTool()
	_, err := tool.Execute(context.Background(), `{"domain":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestWhoisLookupTool_Execute_MarshalError(t *testing.T) {
	tool := NewWhoisLookupTool()
	_, err := tool.Execute(context.Background(), `{"domain":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestThreatIntelCheckTool_Execute_MarshalError(t *testing.T) {
	tool := NewThreatIntelCheckTool()
	_, err := tool.Execute(context.Background(), `{"ip":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ip is required")
}

func TestVesselTrackingTool_Execute_MarshalError(t *testing.T) {
	tool := NewVesselTrackingTool(nil)
	_, err := tool.Execute(context.Background(), `{"mmsi":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mmsi is required")
}

func TestFlightTrackingTool_Execute_MarshalError(t *testing.T) {
	tool := NewFlightTrackingTool(nil)
	_, err := tool.Execute(context.Background(), `{"flight_number":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flight_number is required")
}

func TestRegionDossierTool_Execute_MarshalError(t *testing.T) {
	tool := NewRegionDossierTool(nil)
	_, err := tool.Execute(context.Background(), `{"region_id":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "region_id is required")
}

func TestThreatLevelTool_Execute_MarshalError(t *testing.T) {
	tool := NewThreatLevelTool(nil)
	_, err := tool.Execute(context.Background(), `{"target":""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target is required")
}

func TestCorrelationAlertsTool_Execute_MarshalError(t *testing.T) {
	tool := NewCorrelationAlertsTool(nil)
	_, err := tool.Execute(context.Background(), `{"signals":null}`)
	assert.Error(t, err)
}

func TestIPLookup_ExecuteEmptyArgs(t *testing.T) {
	tool := NewIPLookupTool()
	result, err := tool.Execute(context.Background(), `{}`)
	assert.Error(t, err)
	_ = result
}

func TestDeriveRegionName(t *testing.T) {
	assert.Equal(t, "Eastern Harbor Region", deriveRegionName("en_harbor"))
	assert.Equal(t, "Northern Rise Territory", deriveRegionName("northern_rise"))
	assert.Equal(t, "Region unknown_place", deriveRegionName("unknown_place"))
}

func TestBytesCompare(t *testing.T) {
	assert.Equal(t, 0, bytesCompare(net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.1")))
	assert.Equal(t, -1, bytesCompare(net.ParseIP("10.0.0.0"), net.ParseIP("10.0.0.255")))
	assert.Equal(t, 1, bytesCompare(net.ParseIP("10.0.0.255"), net.ParseIP("10.0.0.0")))
}

func TestToolSecurityProfile_JSON(t *testing.T) {
	profile := ToolSecurityProfile{
		ToolName:  "test-tool",
		RiskScore: 42.0,
		RiskLevel: "medium",
		Warnings:  []string{"warning1"},
		Recommendations: []string{"fix it"},
		Sources:   []string{"test"},
	}
	data, err := json.Marshal(profile)
	require.NoError(t, err)
	var decoded ToolSecurityProfile
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, profile.ToolName, decoded.ToolName)
	assert.Equal(t, profile.RiskScore, decoded.RiskScore)
}

func TestDiscoverToolSecurity_NoBaseURL(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
	ctx := context.Background()
	profile, err := sb.DiscoverToolSecurity(ctx, "some_tool")
	require.NoError(t, err)
	assert.Equal(t, "some_tool", profile.ToolName)
	assert.Equal(t, "low", profile.RiskLevel)
	assert.Equal(t, float64(0), profile.RiskScore)
	assert.Contains(t, profile.Sources, "shadowbroker_intel")
}

func TestDiscoverToolSecurity_EmptyToolName(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: "http://localhost"})
	_, err := sb.DiscoverToolSecurity(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "toolName cannot be empty")
}

func TestDiscoverToolSecurity_CachedResult(t *testing.T) {
	sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
	ctx := context.Background()

	profile1, err := sb.DiscoverToolSecurity(ctx, "cached_tool")
	require.NoError(t, err)

	profile2, err := sb.DiscoverToolSecurity(ctx, "cached_tool")
	require.NoError(t, err)

	assert.Equal(t, profile1.RiskLevel, profile2.RiskLevel)
}
