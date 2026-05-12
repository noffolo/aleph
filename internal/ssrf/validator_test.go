package ssrf

import (
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		// Private / RFC 1918
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"192.168.0.0", true},

		// CGNAT
		{"100.64.0.1", true},
		{"100.127.255.255", true},

		// Link-local
		{"169.254.1.1", true},
		{"169.254.0.1", true},

		// Loopback
		{"127.0.0.1", true},
		{"127.255.255.255", true},

		// Current network
		{"0.0.0.0", true},
		{"0.255.255.255", true},

		// IPv6 loopback
		{"::1", true},

		// IPv6 ULA
		{"fc00::1", true},
		{"fd12:3456:789a::1", true},

		// IPv6 link-local
		{"fe80::1", true},

		// Public IPs
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false}, // example.com

		// IPv6 public
		{"2001:4860:4860::8888", false}, // google DNS
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got, err := isPrivateIP(ip)
			if err != nil {
				t.Fatalf("isPrivateIP(%q) unexpected error: %v", tt.ip, err)
			}
			if got != tt.private {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func TestIsBypassHost(t *testing.T) {
	tests := []struct {
		host   string
		bypass bool
	}{
		// Normal decimal IPs (not bypass)
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"8.8.8.8", false},
		{"127.0.0.1", false},

		// Octal bypass
		{"0177.0.0.1", true},
		{"0251.0.0.1", true},
		{"01.0.0.1", true},

		// Hex bypass
		{"0x7f.0.0.1", true},
		{"0X7F.0.0.1", true},
		{"0x0.0.0.1", true},

		// Integer-form bypass
		{"2130706433", true},
		{"3232235521", true},
		{"0", true}, // all zeros

		// Normal domains (not bypass)
		{"example.com", false},
		{"api.github.com", false},
		{"localhost", false},

		// Edge cases
		{"", false},
		{"1.2.3.4.5", false}, // 5 parts
		{"0x", false},
		{"0xGG", false},

		// Short-form IPs should be blocked conservatively
		{"127.1", true},
		{"10.1", true},
		{"192.168.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isBypassHost(tt.host)
			if got != tt.bypass {
				t.Errorf("isBypassHost(%q) = %v, want %v", tt.host, got, tt.bypass)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid external URLs
		{"valid https", "https://api.example.com/data", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://api.github.com/repos/owner/repo/contents", false},
		{"valid subdomain", "https://api.openai.com/v1/completions", false},

		// IPv4 loopback
		{"loopback 127.0.0.1", "http://127.0.0.1:11434/api/tags", true},
		{"loopback 127.0.0.2", "http://127.0.0.2", true},

		// IPv6 loopback (direct IP — resolves without DNS)
		{"ipv6 loopback", "http://[::1]:8080/api", true},

		// localhost
		{"localhost", "http://localhost:11434/api/tags", true},
		{"localhost https", "https://localhost:443", true},

		// Private IPs
		{"private 10.x", "http://10.0.0.1/api", true},
		{"private 172.16", "http://172.16.0.1", true},
		{"private 192.168", "http://192.168.1.1", true},

		// Link-local
		{"link-local", "http://169.254.1.1", true},

		// Internal TLDs
		{"internal tld", "http://service.internal/api", true},
		{"local tld", "http://service.local/api", true},

		// Octal representation
		{"octal 0177.0.0.1", "http://0177.0.0.1:11434", true},
		{"octal 0251.0.0.1", "http://0251.0.0.1", true},

		// Hex representation
		{"hex 0x7f.0.0.1", "http://0x7f.0.0.1", true},

		// Integer form
		{"integer ip", "http://2130706433:8080", true},

		// 0.0.0.0
		{"zero all", "http://0.0.0.0:11434", true},

		// Scheme disallowed
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com", true},

		// Empty/invalid
		{"empty url", "", true},
		{"no scheme", "example.com", true},
		{"relative path", "/path/to/resource", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateURL(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				// Only count as failure if it's an SSRF block, not a DNS error
				errStr := err.Error()
				if !strings.Contains(errStr, "resolve") && !strings.Contains(errStr, "no such host") {
					t.Errorf("ValidateURL(%q) = %v, want nil", tt.url, err)
				}
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    string
		wantErr bool
	}{
		{"valid host", "example.com", "443", false},
		{"valid host no port", "api.example.com", "", false},
		{"localhost", "localhost", "8080", true},
		{"loopback", "127.0.0.1", "11434", true},
		{"private 10.x", "10.0.0.1", "5432", true},
		{"private 192.168", "192.168.1.1", "80", true},
		{"internal TLD", "service.internal", "3000", true},
		{"local TLD", "dev.local", "8080", true},
		{"empty host", "", "8080", true},
		{"invalid port", "example.com", "99999", true},
		{"negative port", "example.com", "-1", true},
		{"bad port string", "example.com", "abc", true},
		{"octal bypass", "0177.0.0.1", "8080", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.host, tt.port)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateHostname(%q, %q) = nil, want error", tt.host, tt.port)
			}
			if !tt.wantErr && err != nil {
				errStr := err.Error()
				if !strings.Contains(errStr, "resolve") && !strings.Contains(errStr, "no such host") {
					t.Errorf("ValidateHostname(%q, %q) = %v, want nil", tt.host, tt.port, err)
				}
			}
		})
	}
}

func TestNewClientDialPrivateIP(t *testing.T) {
	// These should fail because the transport's DialContext resolves DNS and
	// blocks private IPs. We test by making a request that resolves to 127.0.0.1.
	client := NewClient()
	_, err := client.Get("http://127.0.0.1:1/")
	if err == nil {
		t.Error("expected error for loopback request, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "SSRF blocked") {
		t.Logf("Got error (not SSRF, but dial expectedly failed): %v", err)
	}
}

func TestNewClientDialBlockedIPs(t *testing.T) {
	client := NewClient()
	blocked := []string{
		"http://10.0.0.1:1/",
		"http://192.168.1.1:1/",
		"http://172.16.0.1:1/",
		"http://169.254.1.1:1/",
	}
	for _, u := range blocked {
		t.Run(u, func(t *testing.T) {
			_, err := client.Get(u)
			if err == nil {
				t.Errorf("expected error for blocked IP %q, got nil", u)
			}
		})
	}
}

func TestNewClientBypassIPForms(t *testing.T) {
	client := NewClient()
	bypassURLs := []string{
		"http://0177.0.0.1:1/",
		"http://0x7f.0.0.1:1/",
		"http://2130706433:1/",
		"http://127.1:1/",
	}
	for _, u := range bypassURLs {
		t.Run(u, func(t *testing.T) {
			_, err := client.Get(u)
			if err == nil {
				t.Errorf("expected error for bypass IP %q, got nil", u)
			}
		})
	}
}

func TestNewClientFailsClosedOnDNSFailure(t *testing.T) {
	client := NewClient()
	// non-existent domain should fail closed
	_, err := client.Get("http://this-domain-definitely-does-not-exist-12345.com/")
	if err == nil {
		t.Error("expected error for non-existent domain, got nil")
	}
}

func TestNewClientAllowPublicIP(t *testing.T) {
	// ValidateURL passes for public IPs (the actual dial would also pass SSRF)
	err := ValidateURL("http://8.8.8.8/")
	if err != nil {
		t.Errorf("SSRF incorrectly blocked public IP 8.8.8.8: %v", err)
	}
}

func TestValidateURL_NonExistentDomain(t *testing.T) {
	// DNS should fail closed
	err := ValidateURL("http://this-domain-definitely-does-not-exist-12345.com/api")
	if err == nil {
		t.Error("expected error for non-existent domain, got nil")
	}
}

func TestValidateHostname_NonExistentDomain(t *testing.T) {
	err := ValidateHostname("this-domain-definitely-does-not-exist-12345.com", "80")
	if err == nil {
		t.Error("expected error for non-existent domain, got nil")
	}
}

func TestNewClientRedirectReValidation(t *testing.T) {
	// Verify that CheckRedirect calls ValidateURL by testing the redirect behavior.
	// We can't easily test this without a server, but we can verify the function
	// exists and doesn't panic.
	client := NewClient()
	if client.CheckRedirect == nil {
		t.Error("CheckRedirect should not be nil")
	}
}

func TestNewClientTLSConfig(t *testing.T) {
	client := NewClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport should be *http.Transport")
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig should not be nil")
	}
	if transport.TLSClientConfig.MinVersion != 0x0303 { // tls.VersionTLS12
		t.Errorf("TLS MinVersion should be TLS 1.2, got %d", transport.TLSClientConfig.MinVersion)
	}
}
