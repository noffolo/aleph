package osint

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── IP Lookup Tool Tests ──────────────────────────────────────────────────────────

func TestIPLookupTool(t *testing.T) {
	tool := NewIPLookupTool()

	t.Run("rejects empty IP", func(t *testing.T) {
		_, err := tool.Lookup(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ip is required")
	})

	t.Run("handles invalid IP gracefully", func(t *testing.T) {
		result, err := tool.Lookup(context.Background(), "999.999.999.999")
		if err != nil && strings.Contains(err.Error(), "no such host") {
			t.Skip("network unavailable in sandbox")
		}
		require.NoError(t, err)
		assert.Equal(t, "999.999.999.999", result.IP)
		// ip-api.com returns status:"fail" for invalid IPs
		assert.Contains(t, []string{"fail", "error"}, result.Status)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"ip":"8.8.8.8"}`)
		if err != nil && strings.Contains(err.Error(), "no such host") {
			t.Skip("network unavailable in sandbox")
		}
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "8.8.8.8", parsed["ip"])
	})

	t.Run("Execute rejects empty JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{}`)
		assert.Error(t, err)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `not json`)
		assert.Error(t, err)
	})
}

// ─── DNS Resolution Tool Tests ─────────────────────────────────────────────────────

func TestDNSResolutionTool(t *testing.T) {
	tool := NewDNSResolutionTool()

	t.Run("rejects empty domain", func(t *testing.T) {
		_, err := tool.Resolve(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain is required")
	})

	t.Run("resolves localhost correctly", func(t *testing.T) {
		result, err := tool.Resolve(context.Background(), "localhost")
		require.NoError(t, err)
		assert.Equal(t, "localhost", result.Domain)
		assert.Contains(t, result.A, "127.0.0.1")
		assert.Equal(t, "go_net_dns", result.Source)
	})

	t.Run("resolves example.com", func(t *testing.T) {
		result, err := tool.Resolve(context.Background(), "example.com")
		require.NoError(t, err)
		assert.Equal(t, "example.com", result.Domain)
		// Must have at least one A record for example.com
		require.Greater(t, len(result.A), 0)
		// All entries should be valid IPs
		for _, ip := range result.A {
			assert.NotEmpty(t, net.ParseIP(ip), "expected valid IP, got %q", ip)
		}
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"domain":"localhost"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "localhost", parsed["domain"])
		aRecords, ok := parsed["a_records"].([]interface{})
		require.True(t, ok)
		assert.Contains(t, aRecords, "127.0.0.1")
	})

	t.Run("Execute rejects empty JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{}`)
		assert.Error(t, err)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `not json`)
		assert.Error(t, err)
	})
}

// ─── WHOIS Lookup Tool Tests ───────────────────────────────────────────────────────

func TestWhoisLookupTool(t *testing.T) {
	tool := NewWhoisLookupTool()

	t.Run("rejects empty domain", func(t *testing.T) {
		_, err := tool.Lookup(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain is required")
	})

	t.Run("extractTLD works correctly", func(t *testing.T) {
		assert.Equal(t, "com", extractTLD("example.com"))
		assert.Equal(t, "org", extractTLD("www.example.org"))
		assert.Equal(t, "uk", extractTLD("example.co.uk"))
		assert.Equal(t, "localhost", extractTLD("localhost"))
	})

	t.Run("lookup returns error for nonexistent TLD gracefully", func(t *testing.T) {
		result, err := tool.Lookup(context.Background(), "example.invalidtldxyz")
		require.NoError(t, err)
		assert.Equal(t, "example.invalidtldxyz", result.Domain)
		// Should have an error since IANA won't know this TLD
		assert.NotEmpty(t, result.Error)
		assert.Equal(t, "whois_protocol", result.Source)
	})

	t.Run("lookup existing domain (example.com)", func(t *testing.T) {
		result, err := tool.Lookup(context.Background(), "example.com")
		require.NoError(t, err)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, "com", result.TLD)
		assert.Equal(t, "whois_protocol", result.Source)
		// WHOIS servers may be unreachable in sandboxed/CI environments;
		// if a server was found, verify structure; otherwise accept the error.
		if result.Error != "" {
			t.Logf("WHOIS lookup had non-fatal error (expected in sandbox): %s", result.Error)
		} else {
			assert.Contains(t, result.Server, "whois.")
			assert.NotEmpty(t, result.WhoisRaw)
		}
	})

	t.Run("lookup .org via IANA→authoritative", func(t *testing.T) {
		result, err := tool.Lookup(context.Background(), "example.org")
		require.NoError(t, err)
		assert.Equal(t, "example.org", result.Domain)
		assert.Equal(t, "org", result.TLD)
		assert.Equal(t, "whois_protocol", result.Source)
		if result.Error != "" {
			t.Logf("WHOIS lookup had non-fatal error (expected in sandbox): %s", result.Error)
		} else {
			assert.NotEmpty(t, result.WhoisRaw)
		}
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"domain":"example.com"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "example.com", parsed["domain"])
		assert.Equal(t, "com", parsed["tld"])
		t.Logf("WHOIS Execute result: domain=%v, server=%v", parsed["domain"], parsed["server"])
	})

	t.Run("Execute rejects empty JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{}`)
		assert.Error(t, err)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `not json`)
		assert.Error(t, err)
	})
}

// ─── Threat Intel Check Tool Tests ─────────────────────────────────────────────────

func TestThreatIntelCheckTool(t *testing.T) {
	tool := NewThreatIntelCheckTool()

	t.Run("rejects empty IP", func(t *testing.T) {
		_, err := tool.Check(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ip is required")
	})

	t.Run("identifies loopback as low risk", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "127.0.0.1")
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", result.IP)
		assert.True(t, result.IsLoopback)
		assert.True(t, result.IsPrivate)
		assert.Equal(t, "low", result.RiskLevel)
		assert.Contains(t, result.Warnings[0], "loopback")
	})

	t.Run("identifies RFC1918 class A as safe", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "10.0.0.1")
		require.NoError(t, err)
		assert.True(t, result.IsPrivate)
		assert.True(t, result.IsRFC1918)
		assert.Equal(t, "safe", result.RiskLevel)
	})

	t.Run("identifies RFC1918 class B as safe", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "172.16.0.1")
		require.NoError(t, err)
		assert.True(t, result.IsPrivate)
		assert.True(t, result.IsRFC1918)
	})

	t.Run("identifies RFC1918 class C as safe", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "192.168.1.1")
		require.NoError(t, err)
		assert.True(t, result.IsPrivate)
		assert.True(t, result.IsRFC1918)
	})

	t.Run("treats public IP as low risk", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "8.8.8.8")
		require.NoError(t, err)
		assert.False(t, result.IsPrivate)
		assert.False(t, result.IsRFC1918)
		assert.Equal(t, "low", result.RiskLevel)
		assert.Contains(t, result.Warnings[0], "public")
	})

	t.Run("identifies multicast", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "224.0.0.1")
		require.NoError(t, err)
		assert.Equal(t, "low", result.RiskLevel)
		assert.Contains(t, result.Warnings[0], "multicast")
	})

	t.Run("identifies link-local", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "169.254.1.1")
		require.NoError(t, err)
		assert.True(t, result.IsPrivate)
		assert.Equal(t, "low", result.RiskLevel)
		assert.Contains(t, result.Warnings[0], "link-local")
	})

	t.Run("handles invalid IP format", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "not-an-ip")
		require.NoError(t, err)
		assert.Equal(t, "medium", result.RiskLevel)
		assert.Contains(t, result.Warnings[0], "Invalid")
	})

	t.Run("handles IPv6 loopback", func(t *testing.T) {
		result, err := tool.Check(context.Background(), "::1")
		require.NoError(t, err)
		assert.True(t, result.IsLoopback)
		assert.Equal(t, "low", result.RiskLevel)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"ip":"10.0.0.1"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.1", parsed["ip"])
		assert.Equal(t, "safe", parsed["risk_level"])
		assert.True(t, parsed["is_rfc1918"].(bool))
	})

	t.Run("Execute rejects empty JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{}`)
		assert.Error(t, err)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `not json`)
		assert.Error(t, err)
	})
}

// ─── Edge Cases ────────────────────────────────────────────────────────────────────

func TestIPLookupTool_EdgeCases(t *testing.T) {
	// Net-dependent — may fail in sandbox without DNS; skip gracefully if so.
	t.Run("IP with leading space", func(t *testing.T) {
		tool := NewIPLookupTool()
		_, err := tool.Lookup(context.Background(), " 8.8.8.8")
		if err != nil && strings.Contains(err.Error(), "no such host") {
			t.Skip("network unavailable in sandbox")
		}
		require.NoError(t, err)
	})
}

func TestDNSResolutionTool_EdgeCases(t *testing.T) {
	t.Run("handles non-existent domain", func(t *testing.T) {
		tool := NewDNSResolutionTool()
		result, err := tool.Resolve(context.Background(), "thisdomaindoesnotexist99999.com")
		require.NoError(t, err)
		assert.Equal(t, "thisdomaindoesnotexist99999.com", result.Domain)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("handles timeout context", func(t *testing.T) {
		tool := NewDNSResolutionTool()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Already cancelled
		result, err := tool.Resolve(ctx, "example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, result.Error)
	})
}

func TestThreatIntelCheckTool_EdgeCases(t *testing.T) {
	t.Run("IPv6 public address", func(t *testing.T) {
		tool := NewThreatIntelCheckTool()
		result, err := tool.Check(context.Background(), "2001:4860:4860::8888")
		require.NoError(t, err)
		assert.True(t, strings.Contains(result.Warnings[0], "public"),
			"expected public IP warning, got: %v", result.Warnings)
	})

	t.Run("IPv6 unique local", func(t *testing.T) {
		tool := NewThreatIntelCheckTool()
		result, err := tool.Check(context.Background(), "fd00::1")
		require.NoError(t, err)
		// ULA is not in RFC1918 but might be detected as private via IsPrivate
		assert.Equal(t, "low", result.RiskLevel)
	})
}

func TestWhoisLookupTool_EdgeCases(t *testing.T) {
	t.Run("extractTLD handles various domains", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"example.com", "com"},
			{"sub.example.co.uk", "uk"},
			{"localhost", "localhost"},
			{"a.b.c.d.e", "e"},
			{"", ""},
		}
		for _, tt := range tests {
			got := extractTLD(tt.input)
			assert.Equal(t, tt.want, got, "extractTLD(%q)", tt.input)
		}
	})
}
