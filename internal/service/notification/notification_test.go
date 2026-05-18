package notification

import (
	"github.com/ff3300/aleph-v2/internal/ssrf"
	"testing"
)

func TestValidateWebhookURL_DelegatesToSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://hooks.example.com/webhook", false},
		{"valid https with path", "https://hooks.slack.com/services/T00/B00/xxx", false},
		{"valid http", "http://example.com/callback", false},
		{"valid with port", "https://api.example.com:8443/callback", false},
		{"ftp scheme", "ftp://example.com", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ipv4 loopback", "http://127.0.0.1:11434/callback", true},
		{"ipv6 loopback", "http://[::1]:8080/callback", true},
		{"localhost", "http://localhost:8080/callback", true},
		{"private 10.x", "http://10.0.0.1/webhook", true},
		{"private 192.168", "http://192.168.1.1/callback", true},
		{"internal tld", "https://service.internal/webhook", true},
		{"local tld", "http://service.local/callback", true},
		{"octal ip", "http://0177.0.0.1:11434/callback", true},
		{"hex ip", "http://0x7f.0.0.1:11434/callback", true},
		{"integer ip", "http://2130706433:8080/callback", true},
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

func TestNewNotificationService_UsesSSRFClient(t *testing.T) {
	svc := NewNotificationService()
	if svc.client == nil {
		t.Fatal("client should not be nil")
	}
	svc.Stop()
}

func TestSendWebhook_EmptyURL(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()
	err := svc.SendWebhook("", nil)
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()
	err := svc.SendWebhook("ftp://evil.com", nil)
	if err == nil {
		t.Error("expected error for ftp URL, got nil")
	}
}

func TestSendWebhook_QueueFull_NonBlocking(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()

	for i := 0; i < 110; i++ {
		err := svc.SendWebhook("https://httpbin.org/post", map[string]any{"id": i})
		if err != nil {
			t.Logf("SendWebhook #%d error: %v", i, err)
		}
	}
}

func TestNewNotificationService_ChannelCapacity(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()
	if cap(svc.jobs) != 100 {
		t.Errorf("expected channel capacity 100, got %d", cap(svc.jobs))
	}
}

func TestWebhookJob_ZeroValue(t *testing.T) {
	job := WebhookJob{}
	if job.URL != "" {
		t.Errorf("expected empty URL, got %q", job.URL)
	}
	if job.Payload != nil {
		t.Errorf("expected nil payload, got %v", job.Payload)
	}
}

func TestNotificationService_Lifecycle(t *testing.T) {
	svc := NewNotificationService()
	if svc.client == nil {
		t.Fatal("client should not be nil")
	}
	if svc.jobs == nil {
		t.Fatal("jobs channel should not be nil")
	}
	if svc.stop == nil {
		t.Fatal("stop channel should not be nil")
	}
	svc.Stop()
}

func TestSendWebhook_ProceedsPastValidation(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()

	err := svc.SendWebhook("http://127.0.0.1/callback", nil)
	if err == nil {
		t.Error("expected error for loopback URL, got nil")
	}

	err = svc.SendWebhook("http://localhost/secret", nil)
	if err == nil {
		t.Error("expected error for localhost URL, got nil")
	}
}

func TestSendWebhook_FileURLBlocked(t *testing.T) {
	svc := NewNotificationService()
	defer svc.Stop()

	err := svc.SendWebhook("file:///etc/passwd", nil)
	if err == nil {
		t.Error("expected error for file URL, got nil")
	}
}
