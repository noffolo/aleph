package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHistoryStore(t *testing.T) {
	s := NewHistoryStore(10)
	assert.NotNil(t, s)
	assert.Equal(t, 10, s.maxLen)
}

func TestNewHistoryStore_DefaultMaxLen(t *testing.T) {
	s := NewHistoryStore(0)
	assert.Equal(t, 10, s.maxLen)
}

func TestNewHistoryStore_NegativeMaxLen(t *testing.T) {
	s := NewHistoryStore(-5)
	assert.Equal(t, 10, s.maxLen)
}

func TestHistoryStore_Add(t *testing.T) {
	s := NewHistoryStore(3)
	now := time.Now()

	s.Add("tool_a", HealthRecord{Status: StatusHealthy, CheckedAt: now, ResponseTime: "10ms"})
	records := s.GetHistory("tool_a")
	assert.Len(t, records, 1)
	assert.Equal(t, StatusHealthy, records[0].Status)
	assert.Equal(t, "tool_a", records[0].ToolID)
}

func TestHistoryStore_AddMultiple(t *testing.T) {
	s := NewHistoryStore(3)

	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	s.Add("tool_a", HealthRecord{Status: StatusDegraded})
	s.Add("tool_a", HealthRecord{Status: StatusDown})

	records := s.GetHistory("tool_a")
	assert.Len(t, records, 3)
	assert.Equal(t, StatusHealthy, records[0].Status)
	assert.Equal(t, StatusDegraded, records[1].Status)
	assert.Equal(t, StatusDown, records[2].Status)
}

func TestHistoryStore_RingBuffer(t *testing.T) {
	s := NewHistoryStore(2)

	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	s.Add("tool_a", HealthRecord{Status: StatusDegraded})
	s.Add("tool_a", HealthRecord{Status: StatusDown})

	records := s.GetHistory("tool_a")
	assert.Len(t, records, 2)
	assert.Equal(t, StatusDegraded, records[0].Status)
	assert.Equal(t, StatusDown, records[1].Status)
}

func TestHistoryStore_GetHistoryNoTool(t *testing.T) {
	s := NewHistoryStore(10)
	records := s.GetHistory("nonexistent")
	assert.Empty(t, records)
}

func TestHistoryStore_GetHistoryCopy(t *testing.T) {
	s := NewHistoryStore(10)
	s.Add("tool_a", HealthRecord{Status: StatusHealthy})

	records := s.GetHistory("tool_a")
	assert.Len(t, records, 1)

	// Modifying the returned slice should not affect the store
	records[0].Status = StatusDown
	stored := s.GetHistory("tool_a")
	assert.Equal(t, StatusHealthy, stored[0].Status)
}

func TestConsecutiveFailures(t *testing.T) {
	s := NewHistoryStore(10)

	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	assert.Equal(t, 0, s.ConsecutiveFailures("tool_a"))

	s.Add("tool_a", HealthRecord{Status: StatusDown})
	assert.Equal(t, 1, s.ConsecutiveFailures("tool_a"))

	s.Add("tool_a", HealthRecord{Status: StatusDown})
	assert.Equal(t, 2, s.ConsecutiveFailures("tool_a"))

	s.Add("tool_a", HealthRecord{Status: StatusDegraded})
	assert.Equal(t, 3, s.ConsecutiveFailures("tool_a"))

	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	assert.Equal(t, 0, s.ConsecutiveFailures("tool_a"))
}

func TestConsecutiveFailures_NoRecords(t *testing.T) {
	s := NewHistoryStore(10)
	assert.Equal(t, 0, s.ConsecutiveFailures("nonexistent"))
}

func TestUptimePercentage(t *testing.T) {
	s := NewHistoryStore(10)

	// No records = 0
	assert.Equal(t, float64(0), s.UptimePercentage("tool_a"))

	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	s.Add("tool_a", HealthRecord{Status: StatusDown})
	s.Add("tool_a", HealthRecord{Status: StatusHealthy})

	assert.InDelta(t, 75.0, s.UptimePercentage("tool_a"), 0.001)
}

func TestUptimePercentage_AllDown(t *testing.T) {
	s := NewHistoryStore(10)
	s.Add("tool_a", HealthRecord{Status: StatusDown})
	s.Add("tool_a", HealthRecord{Status: StatusDegraded})
	assert.Equal(t, float64(0), s.UptimePercentage("tool_a"))
}

func TestGetToolIDs(t *testing.T) {
	s := NewHistoryStore(10)
	s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	s.Add("tool_b", HealthRecord{Status: StatusDegraded})
	s.Add("tool_c", HealthRecord{Status: StatusDown})

	ids := s.GetToolIDs()
	assert.ElementsMatch(t, []string{"tool_a", "tool_b", "tool_c"}, ids)
}

func TestGetToolIDs_Empty(t *testing.T) {
	s := NewHistoryStore(10)
	assert.Empty(t, s.GetToolIDs())
}

func TestHealthRecord(t *testing.T) {
	now := time.Now()
	rec := HealthRecord{
		ToolID:       "tool_1",
		Status:       StatusHealthy,
		CheckedAt:    now,
		ResponseTime: "50ms",
	}

	assert.Equal(t, "tool_1", rec.ToolID)
	assert.Equal(t, StatusHealthy, rec.Status)
	assert.Equal(t, "50ms", rec.ResponseTime)
	assert.Equal(t, now, rec.CheckedAt)
	assert.Empty(t, rec.Error)
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", StatusHealthy)
	assert.Equal(t, "degraded", StatusDegraded)
	assert.Equal(t, "down", StatusDown)
	assert.Equal(t, "unknown", StatusUnknown)
}

func TestHistoryStore_ConcurrentAccess(t *testing.T) {
	s := NewHistoryStore(100)

	for i := 0; i < 50; i++ {
		s.Add("tool_a", HealthRecord{Status: StatusHealthy})
	}

	done := make(chan bool, 2)
	go func() {
		s.GetHistory("tool_a")
		done <- true
	}()
	go func() {
		s.ConsecutiveFailures("tool_a")
		done <- true
	}()

	for i := 0; i < 2; i++ {
		<-done
	}
}
