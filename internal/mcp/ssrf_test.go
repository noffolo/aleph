package mcp

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateSSRF
// ---------------------------------------------------------------------------

func TestValidateSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// --- Blocked: IPv4 loopback ---
		{"loopback explicit 127.0.0.1", "http://127.0.0.1:11434/api/tags", true},
		{"loopback 127.0.0.2", "http://127.0.0.2", true},

		// --- Blocked: IPv6 loopback ---
		{"ipv6 loopback", "http://[::1]:8080/api", true},

		// --- Blocked: localhost ---
		{"localhost http", "http://localhost:11434/api/tags", true},
		{"localhost https", "https://localhost:443", true},

		// --- Blocked: private IP ranges ---
		{"private 10.x", "http://10.0.0.1/api", true},
		{"private 172.16.x", "http://172.16.0.1", true},
		{"private 192.168.x", "http://192.168.1.1", true},

		// --- Blocked: link-local ---
		{"link-local", "http://169.254.1.1", true},

		// --- Blocked: zero network ---
		{"zero all", "http://0.0.0.0:11434", true},

		// --- Blocked: internal TLDs ---
		{"internal tld", "http://service.internal/api", true},
		{"local tld", "http://service.local/api", true},

		// --- Blocked: disallowed schemes ---
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com", true},

		// --- Blocked: bypass IP forms ---
		{"octal bypass 0177.0.0.1", "http://0177.0.0.1:11434", true},
		{"hex bypass 0x7f.0.0.1", "http://0x7f.0.0.1", true},
		{"integer bypass", "http://2130706433:8080", true},

		// --- Edge cases: empty/invalid ---
		{"empty url", "", true},
		{"no scheme", "example.com", true},
		{"relative path", "/path/to/resource", true},

		// --- Allowed: public IPs ---
		{"public ip 8.8.8.8", "http://8.8.8.8/", false},
		{"public ip 1.1.1.1", "http://1.1.1.1/", false},

		// --- Allowed: valid external URLs (DNS may fail in test env) ---
		{"valid https", "https://api.example.com/data", false},
		{"valid http", "http://example.com", false},
		{"valid github", "https://api.github.com/repos/owner/repo/contents", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSRF(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateSSRF(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				errStr := err.Error()
				// Allow transient DNS failures for external URLs in test environments
				if !strings.Contains(errStr, "resolve") && !strings.Contains(errStr, "no such host") {
					t.Errorf("ValidateSSRF(%q) = %v, want nil", tt.url, err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateSSRF helper assertions
// ---------------------------------------------------------------------------

// assertSSRFBlocks is a test helper that asserts a URL is blocked by SSRF.
func assertSSRFBlocks(t *testing.T, url string, msg ...string) {
	t.Helper()
	err := ValidateSSRF(url)
	if err == nil {
		detail := "expected SSRF block"
		if len(msg) > 0 {
			detail += ": " + msg[0]
		}
		t.Errorf("%s: ValidateSSRF(%q) = nil, want error", detail, url)
	}
}

// assertSSRFAllows is a test helper that asserts a URL passes SSRF validation.
// DNS failures in test environments are tolerated.
func assertSSRFAllows(t *testing.T, url string, msg ...string) {
	t.Helper()
	err := ValidateSSRF(url)
	if err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "resolve") && !strings.Contains(errStr, "no such host") {
			detail := "expected SSRF pass"
			if len(msg) > 0 {
				detail += ": " + msg[0]
			}
			t.Errorf("%s: ValidateSSRF(%q) = %v, want nil", detail, url, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Comprehensive blocklist test using helper
// ---------------------------------------------------------------------------

func TestValidateSSRF_BlocksAllPrivateRanges(t *testing.T) {
	privateURLs := []string{
		"http://10.0.0.1/admin",
		"http://10.255.255.255/api",
		"http://172.16.0.1/secret",
		"http://172.31.255.255/",
		"http://192.168.1.1/",
		"http://192.168.0.0/",
		"http://100.64.0.1/",
		"http://100.127.255.255/",
		"http://169.254.1.1/",
		"http://169.254.0.1/",
	}
	for _, u := range privateURLs {
		assertSSRFBlocks(t, u, "private range should be blocked")
	}
}

func TestValidateSSRF_AllowsPublicIPs(t *testing.T) {
	publicURLs := []string{
		"http://8.8.8.8/",
		"http://1.1.1.1/",
		"http://93.184.216.34/", // example.com
	}
	for _, u := range publicURLs {
		assertSSRFAllows(t, u, "public IP should be allowed")
	}
}

func TestValidateSSRF_BlocksBypassForms(t *testing.T) {
	bypassURLs := []string{
		"http://0177.0.0.1/",
		"http://0251.0.0.1/",
		"http://0x7f.0.0.1/",
		"http://0X7F.0.0.1/",
		"http://2130706433/",
		"http://3232235521/",
	}
	for _, u := range bypassURLs {
		assertSSRFBlocks(t, u, "bypass form should be blocked")
	}
}

func TestValidateSSRF_RejectsDisallowedSchemes(t *testing.T) {
	disallowed := []string{
		"file:///etc/passwd",
		"file://localhost/etc/shadow",
		"ftp://example.com/file",
		"ftps://example.com/file",
	}
	for _, u := range disallowed {
		err := ValidateSSRF(u)
		if err == nil {
			t.Errorf("ValidateSSRF(%q) = nil, want error for disallowed scheme", u)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases for ValidateSSRF
// ---------------------------------------------------------------------------

func TestValidateSSRF_InvalidInputsEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"no scheme colon", "example.com:8080"},
		{"relative path only", "/path/to/resource"},
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/plain,hello"},
		{"gopher scheme", "gopher://localhost"},
		{"missing host", "http:///path"},
		{"unclosed bracket ipv6", "http://[::1/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSRF(tt.url)
			if err == nil {
				t.Errorf("ValidateSSRF(%q) = nil, want error for edge case", tt.url)
			}
		})
	}
}

func TestValidateSSRF_BlocksLocalhostViaDNSResolution(t *testing.T) {
	ips, err := net.LookupIP("localhost")
	if err != nil {
		t.Skipf("cannot resolve localhost in this environment: %v", err)
	}
	if len(ips) == 0 {
		t.Skip("localhost resolved to no IPs")
	}

	hasPrivate := false
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() {
			hasPrivate = true
			break
		}
	}
	if !hasPrivate {
		t.Skip("localhost did not resolve to a private IP in this environment")
	}

	assertSSRFBlocks(t, "http://localhost:8080/api",
		"localhost resolves to private IP and must be blocked")
	assertSSRFBlocks(t, "https://localhost/",
		"https localhost should also be blocked")
}

func TestValidateSSRF_BlocksHTTPLocalhostServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assertSSRFBlocks(t, server.URL, "httptest server URL (127.0.0.1) must be blocked")
}

func TestValidateSSRF_BlocksShortFormLoopbackAndPrivate(t *testing.T) {
	tests := []string{
		"http://127.1/",
		"http://127.1:8080/",
		"http://10.1/",
		"http://192.168.1/",
	}
	for _, u := range tests {
		assertSSRFBlocks(t, u, "short-form IP should be blocked")
	}
}

func TestValidateSSRF_BlocksIPv6LoopbackAndLinkLocal(t *testing.T) {
	tests := []string{
		"http://[::1]/",
		"http://[fe80::1]/",
		"http://[fc00::1]/",
	}
	for _, u := range tests {
		assertSSRFBlocks(t, u, "IPv6 loopback/link-local should be blocked")
	}
}

// ---------------------------------------------------------------------------
// ParseMCPURI
// ---------------------------------------------------------------------------

func TestParseMCPURI_Valid(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		scheme string
		host   string
		port   string
		path   string
	}{
		{
			name:   "minimal",
			uri:    "mcp://localhost:8080/tools",
			scheme: "mcp",
			host:   "localhost",
			port:   "8080",
			path:   "/tools",
		},
		{
			name:   "default port when missing",
			uri:    "mcp://example.com",
			scheme: "mcp",
			host:   "example.com",
			port:   "8080",
			path:   "/",
		},
		{
			name:   "root path",
			uri:    "mcp://server:9090/",
			scheme: "mcp",
			host:   "server",
			port:   "9090",
			path:   "/",
		},
		{
			name:   "ipv4 host",
			uri:    "mcp://192.168.1.1:3000/plugin",
			scheme: "mcp",
			host:   "192.168.1.1",
			port:   "3000",
			path:   "/plugin",
		},
		{
			name:   "deep path",
			uri:    "mcp://hub.example.com:8080/tools/call/discover",
			scheme: "mcp",
			host:   "hub.example.com",
			port:   "8080",
			path:   "/tools/call/discover",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, host, port, path, err := ParseMCPURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseMCPURI(%q) unexpected error: %v", tt.uri, err)
			}
			if scheme != tt.scheme {
				t.Errorf("scheme = %q, want %q", scheme, tt.scheme)
			}
			if host != tt.host {
				t.Errorf("host = %q, want %q", host, tt.host)
			}
			if port != tt.port {
				t.Errorf("port = %q, want %q", port, tt.port)
			}
			if path != tt.path {
				t.Errorf("path = %q, want %q", path, tt.path)
			}
		})
	}
}

func TestParseMCPURI_Invalid(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"empty", ""},
		{"no scheme at all", "localhost:8080"},
		{"http scheme instead of mcp", "http://localhost:8080/tools"},
		{"https scheme", "https://example.com:8080/tools"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := ParseMCPURI(tt.uri)
			if err == nil {
				t.Errorf("ParseMCPURI(%q) = nil, want error", tt.uri)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidatePrivateRanges (backward compat — always returns nil)
// ---------------------------------------------------------------------------

func TestValidatePrivateRanges_AlwaysNil(t *testing.T) {
	err := ValidatePrivateRanges()
	if err != nil {
		t.Errorf("ValidatePrivateRanges() = %v, want nil (backward compat)", err)
	}
}

// ---------------------------------------------------------------------------
// DNS fail-closed: non-existent domains must be rejected
// ---------------------------------------------------------------------------

func TestValidateSSRF_NonExistentDomainFailsClosed(t *testing.T) {
	err := ValidateSSRF("http://this-domain-definitely-does-not-exist-12345.com/api")
	if err == nil {
		t.Error("ValidateSSRF for non-existent domain should fail, got nil")
	}
}

func TestValidateSSRF_MultipleNonExistentDomains(t *testing.T) {
	domains := []string{
		"http://surely-nonexistent-92837465.example.org/",
		"http://this-will-never-resolve-11111.test/",
	}
	for _, u := range domains {
		t.Run(u, func(t *testing.T) {
			err := ValidateSSRF(u)
			if err == nil {
				t.Errorf("ValidateSSRF(%q) should fail closed for non-existent domain", u)
			}
		})
	}
}
