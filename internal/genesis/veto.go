package genesis

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type VetoRegistry struct {
	mu          sync.RWMutex
	suggestions map[string]Suggestion
	ttl         time.Duration
	cancel      context.CancelFunc
}

func NewVetoRegistry(ctx context.Context, ttl time.Duration) *VetoRegistry {
	derivedCtx, cancel := context.WithCancel(ctx)
	v := &VetoRegistry{
		suggestions: make(map[string]Suggestion),
		ttl:         ttl,
		cancel:      cancel,
	}
	go v.cleanupLoop(derivedCtx)
	return v
}

func (v *VetoRegistry) cleanupLoop(ctx context.Context) {
	cleanupInterval := v.ttl / 2
	if cleanupInterval < time.Millisecond {
		cleanupInterval = time.Millisecond
	}
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			v.mu.Lock()
			for id, s := range v.suggestions {
				if time.Now().After(s.ExpiresAt) {
					delete(v.suggestions, id)
				}
			}
			v.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (v *VetoRegistry) Shutdown() {
	if v.cancel != nil {
		v.cancel()
	}
}

func (v *VetoRegistry) Register(s Suggestion) {
	v.mu.Lock()
	defer v.mu.Unlock()
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(v.ttl)
	if s.Status == "" {
		s.Status = "pending"
	}
	v.suggestions[s.ID] = s
}

func (v *VetoRegistry) Approve(ctx context.Context, id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	s, ok := v.suggestions[id]
	if !ok {
		return fmt.Errorf("suggestion %s not found", id)
	}
	s.Status = "approved"
	v.suggestions[id] = s
	return nil
}

func (v *VetoRegistry) Reject(ctx context.Context, id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	s, ok := v.suggestions[id]
	if !ok {
		return fmt.Errorf("suggestion %s not found", id)
	}
	s.Status = "rejected"
	v.suggestions[id] = s
	return nil
}

func (v *VetoRegistry) ListPending(ctx context.Context) ([]Suggestion, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	var result []Suggestion
	for _, s := range v.suggestions {
		if s.Status == "pending" && time.Now().Before(s.ExpiresAt) {
			result = append(result, s)
		}
	}
	return result, nil
}
