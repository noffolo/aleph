package repair

import (
	"log/slog"
	"sync"
	"time"
)

// RepairStatus represents the outcome of a repair attempt.
type RepairStatus string

const (
	StatusSuccess RepairStatus = "success"
	StatusFailed  RepairStatus = "failed"
)

// RepairRecord stores a single repair attempt.
type RepairRecord struct {
	ID         int           `json:"id"`
	ToolID     string        `json:"tool_id"`
	PlanID     string        `json:"plan_id"`
	ActionID   string        `json:"action_id"`
	ActionType string        `json:"action_type"`
	Status     RepairStatus  `json:"status"`
	ErrorMsg   string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
}

// RepairHistory tracks repair attempts in memory.
type RepairHistory struct {
	mu       sync.RWMutex
	records  []RepairRecord
	nextID   int
	logger   *slog.Logger
}

// NewRepairHistory creates a RepairHistory.
func NewRepairHistory(logger *slog.Logger) *RepairHistory {
	return &RepairHistory{
		records: make([]RepairRecord, 0),
		nextID:  1,
		logger:  logger,
	}
}

// Record adds a repair record to the history.
func (h *RepairHistory) Record(r RepairRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r.ID = h.nextID
	h.nextID++
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	h.records = append(h.records, r)

	h.logger.Info("repair recorded",
		"record_id", r.ID,
		"tool_id", r.ToolID,
		"action_type", r.ActionType,
		"status", r.Status,
		"duration", r.Duration,
	)
}

// GetHistory returns all repair records for a specific tool.
func (h *RepairHistory) GetHistory(toolID string) []RepairRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []RepairRecord
	for _, r := range h.records {
		if r.ToolID == toolID {
			result = append(result, r)
		}
	}
	return result
}

// GetAll returns all repair records.
func (h *RepairHistory) GetAll() []RepairRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]RepairRecord, len(h.records))
	copy(result, h.records)
	return result
}

// SuccessRate calculates the success rate for a tool's repair attempts.
// Returns 0.0 if no attempts exist, or the ratio of successful to total attempts.
func (h *RepairHistory) SuccessRate(toolID string) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var total, success int
	for _, r := range h.records {
		if r.ToolID == toolID {
			total++
			if r.Status == StatusSuccess {
				success++
			}
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(success) / float64(total)
}

// OverallSuccessRate calculates the success rate across all tools.
func (h *RepairHistory) OverallSuccessRate() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.records) == 0 {
		return 0.0
	}
	var success int
	for _, r := range h.records {
		if r.Status == StatusSuccess {
			success++
		}
	}
	return float64(success) / float64(len(h.records))
}
