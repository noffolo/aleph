package notification

import (
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://hooks.example.com/webhook", false},
		{"valid https with path", "https://hooks.slack.com/services/T00/B00/xxx", false},
		{"valid http", "http://example.com/callback", false},
		{"valid with port", "https://api.example.com:8443/callback", false},
		{"valid subdomain", "https://mycompany.webhook.office.com", false},

		// Scheme
		{"ftp scheme", "ftp://example.com", true},
		{"file scheme", "file:///etc/passwd", true},

		// Loopback
		{"ipv4 loopback", "http://127.0.0.1:11434/callback", true},
		{"ipv6 loopback", "http://[::1]:8080/callback", true},
		{"localhost", "http://localhost:8080/callback", true},

		// 0.0.0.0
		{"zero all", "http://0.0.0.0:11434/callback", true},

		// Private
		{"private 10.x", "http://10.0.0.1/webhook", true},
		{"private 192.168", "http://192.168.1.1/callback", true},

		// Internal TLDs
		{"internal tld", "https://service.internal/webhook", true},
		{"local tld", "http://service.local/callback", true},

		// Octal/hex bypass
		{"octal ip", "http://0177.0.0.1:11434/callback", true},
		{"hex ip", "http://0x7f.0.0.1:11434/callback", true},
		{"integer ip", "http://2130706433:8080/callback", true},

		// Invalid
		{"empty url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("validateWebhookURL(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateWebhookURL(%q) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestLooksLikeBypassIP(t *testing.T) {
	tests := []struct {
		host     string
		isBypass bool
	}{
		{"8.8.8.8", false},
		{"example.com", false},
		{"0177.0.0.1", true},
		{"0x7f.0.0.1", true},
		{"2130706433", true},
		{"0.0.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := looksLikeBypassIP(tt.host)
			if got != tt.isBypass {
				t.Errorf("looksLikeBypassIP(%q) = %v, want %v", tt.host, got, tt.isBypass)
			}
		})
	}
}
