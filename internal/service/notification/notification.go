package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

type WebhookJob struct {
	URL     string
	Payload interface{}
}

type NotificationService struct {
	client  *http.Client
	jobs    chan WebhookJob
	stop    chan struct{}
	wg      sync.WaitGroup
	closeMu sync.Once
}

func NewNotificationService() *NotificationService {
	svc := &NotificationService{
		client: ssrf.NewClient(),
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
	s.closeMu.Do(func() {
		close(s.stop)
	})
	s.wg.Wait()
}

func (s *NotificationService) SendWebhook(rawURL string, payload interface{}) error {
	if err := ssrf.ValidateURL(rawURL); err != nil {
		return fmt.Errorf("webhook URL invalid: %w", err)
	}
	select {
	case s.jobs <- WebhookJob{URL: rawURL, Payload: payload}:
		return nil
	default:
		log.Printf("[Notification] Webhook queue full, dropping message for %s", rawURL)
		return nil
	}
}

