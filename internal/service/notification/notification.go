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
	"time"
)

type WebhookJob struct {
	URL     string
	Payload interface{}
}

type NotificationService struct {
	client *http.Client
	jobs   chan WebhookJob
}

func NewNotificationService() *NotificationService {
	svc := &NotificationService{
		client: &http.Client{Timeout: 10 * time.Second},
		jobs:   make(chan WebhookJob, 100),
	}

	for i := 0; i < 3; i++ {
		go svc.worker()
	}

	return svc
}

func (s *NotificationService) worker() {
	for job := range s.jobs {
		data, err := json.Marshal(job.Payload)
		if err != nil {
			log.Printf("[Notification] Webhook payload marshal failed: %v", err)
			continue
		}

		resp, err := s.client.Post(job.URL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("[Notification] Webhook failed to %s: %v", job.URL, err)
			continue
		}
		resp.Body.Close()

		log.Printf("[Notification] Webhook sent to %s. Status: %s", job.URL, resp.Status)
	}
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
		return err
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
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("indirizzo IP interno non permesso: %s", host)
		}
	}
	if strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".local") || host == "localhost" {
		return fmt.Errorf("host interno non permesso: %s", host)
	}
	return nil
}
