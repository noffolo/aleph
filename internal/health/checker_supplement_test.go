package health

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPToolHealth_Defaults(t *testing.T) {
	m := MCPToolHealth{Name: "server-x"}
	assert.Equal(t, "server-x", m.Name)
	assert.Equal(t, "", m.Status)
	assert.Equal(t, 0, m.RestartCount)
}

func TestHealthChecker_checkMCPHealth_UpStatus(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-up", Status: "up", LastPing: "2ms"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	history := hc.GetHistory("tool-up")
	assert.Len(t, history, 1)
	assert.Equal(t, StatusHealthy, history[0].Status)
	assert.Equal(t, "2ms", history[0].ResponseTime)
}

func TestHealthChecker_checkMCPHealth_DownStatus(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-down", Status: "down", LastError: "process crashed", RestartCount: 2},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	history := hc.GetHistory("tool-down")
	assert.Len(t, history, 1)
	assert.Equal(t, StatusDown, history[0].Status)
	assert.Equal(t, "process crashed", history[0].Error)
}

func TestHealthChecker_checkMCPHealth_UnknownStatus(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-unk", Status: "unknown"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	history := hc.GetHistory("tool-unk")
	assert.Len(t, history, 1)
	assert.Equal(t, StatusUnknown, history[0].Status)
}

func TestHealthChecker_checkMCPHealth_UnexpectedStatusValue(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-weird", Status: "flapping"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	history := hc.GetHistory("tool-weird")
	assert.Len(t, history, 1)
	assert.Equal(t, StatusUnknown, history[0].Status)
}

func TestHealthChecker_checkMCPHealth_EmptyNameSkipped(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "", Status: "up"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	all := hc.GetAllHistory()
	assert.Empty(t, all)
}

func TestHealthChecker_checkMCPHealth_MultipleTools(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-a", Status: "up"},
		{Name: "tool-b", Status: "down", LastError: "crashed"},
		{Name: "tool-c", Status: "unknown"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	all := hc.GetAllHistory()
	assert.Len(t, all, 3)
	assert.Equal(t, StatusHealthy, hc.GetLatestStatus("tool-a"))
	assert.Equal(t, StatusDown, hc.GetLatestStatus("tool-b"))
	assert.Equal(t, StatusUnknown, hc.GetLatestStatus("tool-c"))
}

func TestHealthChecker_GetHistory_Empty(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	assert.Empty(t, hc.GetHistory("nonexistent"))
}

func TestHealthChecker_GetLatestStatus_MixedHistory(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool-x", HealthRecord{Status: StatusHealthy})
	hc.history.Add("tool-x", HealthRecord{Status: StatusDown})
	hc.history.Add("tool-x", HealthRecord{Status: StatusHealthy})
	assert.Equal(t, StatusHealthy, hc.GetLatestStatus("tool-x"))
}

func TestHealthChecker_ConsecutiveFailures_Mixed(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool-y", HealthRecord{Status: StatusHealthy})
	hc.history.Add("tool-y", HealthRecord{Status: StatusDown})
	hc.history.Add("tool-y", HealthRecord{Status: StatusDown})
	assert.Equal(t, 2, hc.ConsecutiveFailures("tool-y"))
}

func TestHealthChecker_ConsecutiveFailures_NoFailures(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool-z", HealthRecord{Status: StatusHealthy})
	hc.history.Add("tool-z", HealthRecord{Status: StatusHealthy})
	assert.Equal(t, 0, hc.ConsecutiveFailures("tool-z"))
}

func TestHealthChecker_ConsecutiveFailures_NonexistentTool(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	assert.Equal(t, 0, hc.ConsecutiveFailures("nonexistent"))
}

func TestHealthRecord_ZeroValue(t *testing.T) {
	r := HealthRecord{}
	assert.Equal(t, "", r.ToolID)
	assert.Equal(t, "", r.Status)
	assert.True(t, r.CheckedAt.IsZero())
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", StatusHealthy)
	assert.Equal(t, "degraded", StatusDegraded)
	assert.Equal(t, "down", StatusDown)
	assert.Equal(t, "unknown", StatusUnknown)
}

func TestMCPToolHealth_AllFields(t *testing.T) {
	m := MCPToolHealth{
		Name:         "test-tool",
		Status:       "up",
		LastPing:     "100ms",
		LastError:    "",
		RestartCount: 0,
	}
	assert.Equal(t, "test-tool", m.Name)
	assert.Equal(t, "up", m.Status)
	assert.Equal(t, "100ms", m.LastPing)
	assert.Equal(t, "", m.LastError)
	assert.Equal(t, 0, m.RestartCount)
}
