package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type WebhookJob struct {
	URL     string
	Payload interface{}
}

type NotificationService struct {
	client *http.Client
	jobs   chan WebhookJob
	stop   chan struct{}
	wg     sync.WaitGroup
}

func NewNotificationService() *NotificationService {
	svc := &NotificationService{
		client: &http.Client{Timeout: 10 * time.Second},
		jobs:   make(chan WebhookJob, 100),
		stop:   make(chan struct{}),
	}

	for i := 0; i < 3; i++ {
		svc.wg.Add(1)
		go svc.worker()
	}

	return svc
}

func (s *NotificationService) worker() {
	defer s.wg.Done()
	for {
		select {
		case job, ok := <-s.jobs:
			if !ok {
				return
			}
			s.sendWebhook(job)
		case <-s.stop:
			return
		}
	}
}

func (s *NotificationService) sendWebhook(job WebhookJob) {
	data, err := json.Marshal(job.Payload)
	if err != nil {
		log.Printf("[Notification] Webhook payload marshal failed: %v", err)
		return
	}

	resp, err := s.client.Post(job.URL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[Notification] Webhook failed to %s: %v", job.URL, err)
		return
	}
	resp.Body.Close()

	log.Printf("[Notification] Webhook sent to %s. Status: %s", job.URL, resp.Status)
}

func (s *NotificationService) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *NotificationService) SendWebhook(rawURL string, payload interface{}) error {
	if err := validateWebhookURL(rawURL); err != nil {
		return fmt.Errorf("URL webhook non valido: %w", err)
	}
	select {
	case s.jobs <- WebhookJob{URL: rawURL, Payload: payload}:
		return nil
	default:
		log.Printf("[Notification] Webhook queue full, dropping message for %s", rawURL)
		return nil
	}
}

func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("validateWebhookURL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("schema non permesso: %s", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("host vuoto")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("indirizzo IP interno non permesso: %s", host)
		}
	}
	if looksLikeBypassIP(host) {
		return fmt.Errorf("possibile IP bypass rilevato: %s", host)
	}
	if strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".local") || host == "localhost" {
		return fmt.Errorf("host interno non permesso: %s", host)
	}
	return nil
}

// looksLikeBypassIP checks if a host string looks like an IP address in
// octal, hex, or integer form — often used to bypass URL validation.
func looksLikeBypassIP(host string) bool {
	// Check octal (leading zero octets): 0177.0.0.1 (but 0.0.0.0 is regular)
	if strings.HasPrefix(host, "0.") {
		return false
	}
	if strings.HasPrefix(host, "0") && len(host) > 1 && host[0] == '0' && host[1] != 'x' && host[1] != 'X' && strings.Contains(host, ".") {
		return true
	}
	// Check hex IP: 0x7f.0.0.1
	if strings.HasPrefix(host, "0x") || strings.HasPrefix(host, "0X") {
		return true
	}
	// Check integer IP: 2130706433
	if !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		for _, c := range host {
			if c < '0' || c > '9' {
				return false
			}
		}
		if len(host) > 3 { // longer than "255" suggests integer-form IP
			return true
		}
	}
	return false
}
