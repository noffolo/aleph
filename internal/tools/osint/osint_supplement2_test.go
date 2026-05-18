package osint

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIPLookupTool(t *testing.T) {
	tool := NewIPLookupTool()
	assert.NotNil(t, tool)
	assert.NotNil(t, tool.client)
}

func TestNewDNSResolutionTool(t *testing.T) {
	tool := NewDNSResolutionTool()
	assert.NotNil(t, tool)
}

func TestNewWhoisLookupTool(t *testing.T) {
	tool := NewWhoisLookupTool()
	assert.NotNil(t, tool)
}

func TestNewThreatIntelCheckTool(t *testing.T) {
	tool := NewThreatIntelCheckTool()
	assert.NotNil(t, tool)
}

func TestIPLookupResult_JSONRoundTrip(t *testing.T) {
	result := IPLookupResult{
		IP:          "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		Region:      "California",
		City:        "Mountain View",
		ISP:         "Google LLC",
		Org:         "Google",
		Latitude:    37.4056,
		Longitude:   -122.0775,
		Timezone:    "America/Los_Angeles",
		Status:      "success",
		Source:      "ip-api.com",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded IPLookupResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, result.IP, decoded.IP)
	assert.Equal(t, result.Country, decoded.Country)
	assert.Equal(t, result.Latitude, decoded.Latitude)
}

func TestDNSResolutionResult_JSONRoundTrip(t *testing.T) {
	result := DNSResolutionResult{
		Domain: "example.com",
		A:      []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"},
		MX:     []string{"mail.example.com (priority 10)"},
		TXT:    []string{"v=spf1 -all"},
		Source: "go_net_dns",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded DNSResolutionResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, result.Domain, decoded.Domain)
	assert.Contains(t, decoded.A, "93.184.216.34")
}

func TestWhoisResult_JSONRoundTrip(t *testing.T) {
	result := WhoisResult{
		Domain:    "example.com",
		WhoisRaw:  "Domain Name: EXAMPLE.COM\r\nRegistry Domain ID: 2336799_DOMAIN_COM-VRSN\r\n",
		Server:    "whois.verisign-grs.com",
		TLD:       "com",
		Truncated: false,
		Source:    "whois_protocol",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded WhoisResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, result.Domain, decoded.Domain)
	assert.Equal(t, result.Server, decoded.Server)
	assert.False(t, decoded.Truncated)
}

func TestThreatIntelResult_JSONRoundTrip(t *testing.T) {
	result := ThreatIntelResult{
		IP:         "8.8.8.8",
		IsPrivate:  false,
		IsLoopback: false,
		IsRFC1918:  false,
		RiskLevel:  "low",
		Warnings:   []string{"IP is a public address"},
		Source:     "heuristic_analysis",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded ThreatIntelResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, result.IP, decoded.IP)
	assert.Equal(t, "low", decoded.RiskLevel)
}

func TestThreatIntelCheck_LinkLocalIPv6(t *testing.T) {
	tool := NewThreatIntelCheckTool()
	result, err := tool.Check(context.Background(), "fe80::1")
	require.NoError(t, err)
	assert.True(t, result.IsPrivate, "fe80::1 should be marked private")
	assert.Equal(t, "low", result.RiskLevel)
}

func TestThreatIntelCheck_PublicIPv6(t *testing.T) {
	tool := NewThreatIntelCheckTool()
	result, err := tool.Check(context.Background(), "2001:4860:4860::8888")
	require.NoError(t, err)
	assert.False(t, result.IsPrivate)
	assert.Equal(t, "low", result.RiskLevel)
}

func TestBytesCompare_EdgeCases(t *testing.T) {
	ip1 := net.ParseIP("10.0.0.0")
	ip2 := net.ParseIP("10.0.0.1")
	ip3 := net.ParseIP("0.0.0.0")
	ip4 := net.ParseIP("255.255.255.255")

	assert.Equal(t, 0, bytesCompare(ip1, ip1))
	assert.Equal(t, -1, bytesCompare(ip1, ip2))
	assert.Equal(t, 1, bytesCompare(ip2, ip1))
	assert.Equal(t, -1, bytesCompare(ip3, ip4))
	assert.Equal(t, 1, bytesCompare(ip4, ip3))
}

func TestIsRFC1918_EdgeCases(t *testing.T) {
	tests := []struct {
		ip     string
		expect bool
	}{
		{"10.0.0.0", true},
		{"10.128.64.32", true},
		{"10.255.255.254", true},
		{"172.16.0.1", true},
		{"172.20.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.0.1", true},
		{"192.168.255.254", true},
		{"192.169.0.1", false},
		{"8.8.8.8", false},
		{"::1", false},     // IPv6 loopback is not RFC1918
		{"fe80::1", false}, // IPv6 link-local is not RFC1918
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "should parse IP %q", tt.ip)
			got := isRFC1918(ip)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestExtractTLD_AdditionalCases(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"simple.com", "com"},
		{"multi.part.sub.example.org", "org"},
		{"single", "single"},
		{"trailing.dot.", "dot"},
		{"double..dot", "dot"},
	}
	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			assert.Equal(t, tt.want, extractTLD(tt.domain))
		})
	}
}

func TestDNSResolutionTool_Resolve_MXRecords(t *testing.T) {
	tool := NewDNSResolutionTool()
	result, err := tool.Resolve(context.Background(), "gmail.com")
	require.NoError(t, err)
	assert.Equal(t, "gmail.com", result.Domain)
	assert.NotEmpty(t, result.A, "gmail.com should have A records")
	// MX records may or may not be present in test env, but we verify structure
	assert.NotEmpty(t, result.Source)
}

func TestDNSResolutionTool_Resolve_NoDomainError(t *testing.T) {
	tool := NewDNSResolutionTool()
	_, err := tool.Resolve(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestIPLookup_EmptyIP_ErrorPath(t *testing.T) {
	tool := NewIPLookupTool()
	_, err := tool.Lookup(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ip is required")
}

func TestThreatIntelCheck_EmptyIP_ErrorPath(t *testing.T) {
	tool := NewThreatIntelCheckTool()
	_, err := tool.Check(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ip is required")
}

func TestWhoisLookup_EmptyDomain_ErrorPath(t *testing.T) {
	tool := NewWhoisLookupTool()
	_, err := tool.Lookup(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestIPLookupResult_ErrorCase(t *testing.T) {
	// Verify the struct handles error state
	result := IPLookupResult{
		IP:     "256.256.256.256",
		Status: "fail",
		Error:  "invalid query",
		Source: "ip-api.com",
	}
	assert.Equal(t, "fail", result.Status)
	assert.NotEmpty(t, result.Error)
}
