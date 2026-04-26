package ingestion

import (
	"testing"
)

func TestBlockSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Allowed URLs
		{"valid https", "https://api.example.com/data", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://api.github.com/repos/owner/repo/contents", false},
		{"valid subdomain", "https://api.openai.com/v1/completions", false},

		// IPv4 loopback
		{"loopback 127.0.0.1", "http://127.0.0.1:11434/api/tags", true},
		{"loopback 127.0.0.2", "http://127.0.0.2", true},
		{"loopback 127.1.2.3", "http://127.1.2.3", true},

		// IPv6 loopback
		{"ipv6 loopback", "http://[::1]:8080/api", true},
		{"ipv6 full loopback", "http://[0:0:0:0:0:0:0:1]", true},

		// 0.0.0.0
		{"zero zero zero zero", "http://0.0.0.0:11434", true},

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
		{"integer ip 2130706433", "http://2130706433:8080", true},

		// Empty/invalid
		{"empty url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := blockSSRF(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("blockSSRF(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("blockSSRF(%q) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestLooksLikeNonDecimalIP(t *testing.T) {
	tests := []struct {
		host     string
		isBypass bool
	}{
		// Normal IPs (not bypass)
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"8.8.8.8", false},
		{"127.0.0.1", false},

		// Octal bypass
		{"0177.0.0.1", true},
		{"0251.0.0.1", true},
		{"0.0.0.0", false}, // "0" alone is valid

		// Hex bypass
		{"0x7f.0.0.1", true},
		{"0X7F.0.0.1", true},

		// Integer bypass
		{"2130706433", true},
		{"3232235521", true},

		// Normal domains (not bypass)
		{"example.com", false},
		{"api.github.com", false},
		{"localhost", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := looksLikeNonDecimalIP(tt.host)
			if got != tt.isBypass {
				t.Errorf("looksLikeNonDecimalIP(%q) = %v, want %v", tt.host, got, tt.isBypass)
			}
		})
	}
}

func TestBlockSSRF_LocalhostVariants(t *testing.T) {
	variants := []struct {
		host string
	}{
		{"http://127.0.0.1"},
		{"http://127.1"},
		{"http://0"},
		{"http://2130706433"},
	}
	for _, v := range variants {
		t.Run(v.host, func(t *testing.T) {
			err := blockSSRF(v.host)
			if err == nil {
				t.Errorf("blockSSRF(%q) = nil, expected block", v.host)
			}
		})
	}
}
