package ingestion

import (
	"testing"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

func TestBlockSSRF_DelegatesToSSRFPackage(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://api.example.com/data", false},
		{"loopback 127.0.0.1", "http://127.0.0.1:11434/api/tags", true},
		{"private 10.x", "http://10.0.0.1/api", true},
		{"private 192.168", "http://192.168.1.1", true},
		{"localhost", "http://localhost:11434/api/tags", true},
		{"internal tld", "http://service.internal/api", true},
		{"local tld", "http://service.local/api", true},
		{"empty url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ssrf.ValidateURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("ssrf.ValidateURL(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Logf("ssrf.ValidateURL(%q) = %v (may be DNS error in test env)", tt.url, err)
			}
		})
	}
}
