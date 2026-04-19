package notification

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
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

func (s *NotificationService) SendWebhook(url string, payload interface{}) error {
	select {
	case s.jobs <- WebhookJob{URL: url, Payload: payload}:
		return nil
	default:
		log.Printf("[Notification] Webhook queue full, dropping message for %s", url)
		return nil
	}
}
