package ssrf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeNonDecimalPart(t *testing.T) {
	tests := []struct {
		part   string
		bypass bool
	}{
		{"0", false},         // just zero
		{"01", true},         // leading zero = octal
		{"010", true},        // octal
		{"0x7f", true},       // hex
		{"0X7F", true},       // hex uppercase
		{"0x", false},        // incomplete hex
		{"0X", false},        // incomplete hex uppercase
		{"0xGG", true},       // starts with 0x, treated as bypass
		{"255", false},       // normal decimal
		{"1", false},         // single digit
		{"", false},          // empty
		{"0x0", true},        // hex zero
		{"0xdeadbeef", true}, // large hex value
		{"08", true},         // leading zero (octal)
	}
	for _, tt := range tests {
		t.Run(tt.part, func(t *testing.T) {
			assert.Equal(t, tt.bypass, looksLikeNonDecimalPart(tt.part))
		})
	}
}

func TestIsBypassHost_AdditionalCases(t *testing.T) {
	tests := []struct {
		host   string
		bypass bool
	}{
		// Normal domain names (not bypass)
		{"api.github.com", false},
		{"sub.domain.example.com", false},
		{"a.b", false},         // 2 parts, non-numeric last = domain, not bypass
		{"foo.bar.baz", false}, // 3 parts, non-numeric = domain

		// Edge: 2 numeric parts → short-form IP
		{"10.0", true},

		// Edge: 3 numeric parts → short-form IP
		{"192.168.1", true},

		// Single part non-numeric → not bypass
		{"mytool", false},

		{"8080", true}, // >3 chars, treated as bypass
		{"80", false},  // ≤3 chars, not a bypass integer
		{"0", true},    // "0" is special-cased as bypass
		{"100", false}, // ≤3 chars, not a bypass integer

		// Valid IPv4 (not bypass)
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.bypass, isBypassHost(tt.host))
		})
	}
}

func TestValidateHost_LocalhostVariants(t *testing.T) {
	tests := []string{
		"localhost",
		"LOCALHOST",
		"LocalHost",
		"127.0.0.1",
		"::1",
	}
	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			err := validateHost(host)
			assert.Error(t, err, "host %q should be blocked", host)
		})
	}
}

func TestValidateHost_InternalTLDs(t *testing.T) {
	tests := []string{
		"service.local",
		"service.internal",
		"dev.Local",
		"staging.INTERNAL",
	}
	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			err := validateHost(host)
			assert.Error(t, err, "host %q with internal TLD should be blocked", host)
		})
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := NewClient()
	assert.Equal(t, int64(60), int64(client.Timeout.Seconds()))
}

func TestNewClient_CheckRedirect_Present(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client.CheckRedirect, "CheckRedirect should be set for SSRF re-validation")
}

func TestNewClient_DialContext_Blocked(t *testing.T) {
	client := NewClient()
	// Attempting to dial 127.0.0.1 should be blocked by dialContext
	_, err := client.Get("http://127.0.0.1:9/")
	assert.Error(t, err)
}

func TestValidateURL_InvalidInput(t *testing.T) {
	tests := []string{
		"",
		"://bare-slash",
		"http:///path-only",
	}
	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			err := ValidateURL(u)
			assert.Error(t, err, "expected error for %q", u)
		})
	}
}

func TestInitPrivateNets_Idempotent(t *testing.T) {
	err := initPrivateNets()
	assert.NoError(t, err)
	// Second call should be idempotent (initDone=true)
	err = initPrivateNets()
	assert.NoError(t, err)
}

func TestValidateCIDR_Valid(t *testing.T) {
	validCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	for _, s := range validCIDRs {
		t.Run(s, func(t *testing.T) {
			assert.NoError(t, validateCIDR(s))
		})
	}
}

func TestValidateCIDR_Invalid(t *testing.T) {
	invalidCIDRs := []string{
		"not-a-cidr",
		"10.0.0.0/99",
		"",
	}
	for _, s := range invalidCIDRs {
		t.Run(s, func(t *testing.T) {
			assert.Error(t, validateCIDR(s))
		})
	}
}
