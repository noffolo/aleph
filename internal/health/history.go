package health

import (
	"sync"
	"time"
)

// HealthStatus represents the health state of a tool.
const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusDown     = "down"
	StatusUnknown  = "unknown"
)

// HealthRecord represents a single health check result for a tool.
type HealthRecord struct {
	ToolID       string    `json:"tool_id"`
	Status       string    `json:"status"`
	CheckedAt    time.Time `json:"checked_at"`
	ResponseTime string    `json:"response_time"` // e.g., "150ms"
	Error         string    `json:"error,omitempty"`
}

// HistoryStore keeps the last N health records per tool.
type HistoryStore struct {
	mu      sync.RWMutex
	records map[string][]HealthRecord // toolID -> records (ring buffer)
	maxLen  int
}

// NewHistoryStore creates a history store keeping the last maxLen records per tool.
func NewHistoryStore(maxLen int) *HistoryStore {
	if maxLen <= 0 {
		maxLen = 10
	}
	return &HistoryStore{
		records: make(map[string][]HealthRecord),
		maxLen:  maxLen,
	}
}

// Add adds a health record for a tool, maintaining the ring buffer.
func (s *HistoryStore) Add(toolID string, record HealthRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record.ToolID = toolID
	records := s.records[toolID]
	records = append(records, record)
	if len(records) > s.maxLen {
		records = records[len(records)-s.maxLen:]
	}
	s.records[toolID] = records
}

// GetHistory returns the last N health records for a tool (most recent last).
func (s *HistoryStore) GetHistory(toolID string) []HealthRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, ok := s.records[toolID]
	if !ok {
		return []HealthRecord{}
	}
	result := make([]HealthRecord, len(records))
	copy(result, records)
	return result
}

// ConsecutiveFailures returns the number of consecutive down/degraded records
// at the end of the history for a tool.
func (s *HistoryStore) ConsecutiveFailures(toolID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.records[toolID]
	count := 0
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Status == StatusDown || records[i].Status == StatusDegraded {
			count++
		} else {
			break
		}
	}
	return count
}

// UptimePercentage returns the percentage of healthy records in the history.
func (s *HistoryStore) UptimePercentage(toolID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.records[toolID]
	if len(records) == 0 {
		return 0
	}
	healthy := 0
	for _, r := range records {
		if r.Status == StatusHealthy {
			healthy++
		}
	}
	return float64(healthy) / float64(len(records)) * 100
}

// GetToolIDs returns all tool IDs that have health history records.
func (s *HistoryStore) GetToolIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	return ids
}