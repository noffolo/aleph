package humanecosystems

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// UsagePattern represents a single usage pattern entry for a tool.
type UsagePattern struct {
	ToolID    string    `json:"tool_id"`
	UserID    string    `json:"user_id"`
	Frequency int       `json:"frequency"`
	TimeOfDay string    `json:"time_of_day"` // "morning", "afternoon", "evening", "night"
	Context   string    `json:"context"`
	LastUsed  time.Time `json:"last_used"`
}

// Relation describes a relational connection between tools.
type Relation struct {
	ToolID      string `json:"tool_id"`
	RelatedID   string `json:"related_id"`
	RelationType string `json:"relation_type"` // "co_used", "sequence", "depends_on"
	Strength    int    `json:"strength"`       // 0-100
}

type usageEntry struct {
	UserID    string
	ToolID    string
	Context   string
	Timestamp time.Time
}

// ToolUsageTracker tracks tool usage patterns and relational context.
type ToolUsageTracker struct {
	mu       sync.RWMutex
	entries  []usageEntry
	patterns map[string]*UsagePattern // key: "userID:toolID"
	maxEntries int
}

// NewToolUsageTracker creates a new ToolUsageTracker.
func NewToolUsageTracker() *ToolUsageTracker {
	return &ToolUsageTracker{
		entries:    make([]usageEntry, 0, 1000),
		patterns:   make(map[string]*UsagePattern),
		maxEntries: 10000,
	}
}

func timeOfDay(t time.Time) string {
	h := t.Hour()
	switch {
	case h >= 5 && h < 12:
		return "morning"
	case h >= 12 && h < 17:
		return "afternoon"
	case h >= 17 && h < 22:
		return "evening"
	default:
		return "night"
	}
}

// RecordUsage records a tool usage event.
func (tut *ToolUsageTracker) RecordUsage(ctx context.Context, userID, toolID, context string) error {
	if userID == "" || toolID == "" {
		return fmt.Errorf("userID and toolID cannot be empty")
	}

	entry := usageEntry{
		UserID:    userID,
		ToolID:    toolID,
		Context:   context,
		Timestamp: time.Now(),
	}

	tut.mu.Lock()
	defer tut.mu.Unlock()

	tut.entries = append(tut.entries, entry)
	if len(tut.entries) > tut.maxEntries {
		tut.entries = tut.entries[len(tut.entries)-tut.maxEntries:]
	}

	key := userID + ":" + toolID
	existing, ok := tut.patterns[key]
	if !ok {
		tut.patterns[key] = &UsagePattern{
			ToolID:    toolID,
			UserID:    userID,
			Frequency: 1,
			TimeOfDay: timeOfDay(entry.Timestamp),
			Context:   context,
			LastUsed:  entry.Timestamp,
		}
	} else {
		existing.Frequency++
		existing.TimeOfDay = timeOfDay(entry.Timestamp)
		existing.Context = context
		existing.LastUsed = entry.Timestamp
	}

	return nil
}

// GetUsagePatterns returns usage patterns for a given tool across all users.
func (tut *ToolUsageTracker) GetUsagePatterns(ctx context.Context, toolID string) ([]UsagePattern, error) {
	if toolID == "" {
		return nil, fmt.Errorf("toolID cannot be empty")
	}

	tut.mu.RLock()
	defer tut.mu.RUnlock()

	var result []UsagePattern
	for _, p := range tut.patterns {
		if p.ToolID == toolID {
			result = append(result, *p)
		}
	}

	if result == nil {
		return []UsagePattern{}, nil
	}
	return result, nil
}

// GetRelationalContext returns relational connections for the given tool IDs.
// Relations are derived from co-usage patterns (tools used by the same user in sequence).
func (tut *ToolUsageTracker) GetRelationalContext(ctx context.Context, toolIDs []string) (map[string][]Relation, error) {
	if len(toolIDs) == 0 {
		return nil, fmt.Errorf("toolIDs cannot be empty")
	}

	tut.mu.RLock()
	defer tut.mu.RUnlock()

	result := make(map[string][]Relation)
	toolSet := make(map[string]bool)
	for _, id := range toolIDs {
		toolSet[id] = true
		result[id] = []Relation{}
	}

	// Build user -> []toolID mappings with timestamps
	userTools := make(map[string][]usageEntry)
	for _, e := range tut.entries {
		if toolSet[e.ToolID] {
			userTools[e.UserID] = append(userTools[e.UserID], e)
		}
	}

	// Find co-used tools: for each user, the other tools they used
	for _, entries := range userTools {
		used := make(map[string]int)
		for _, e := range entries {
			// Look for other entries by same user within 1 hour window
			for _, other := range tut.entries {
				if other.UserID == e.UserID && other.ToolID != e.ToolID {
					diff := other.Timestamp.Sub(e.Timestamp)
					if diff < 0 {
						diff = -diff
					}
					if diff <= time.Hour {
						used[other.ToolID]++
					}
				}
			}
		}

		for _, tid := range toolIDs {
			for otherID, count := range used {
				if otherID != tid {
					strength := count * 20
					if strength > 100 {
						strength = 100
					}
					result[tid] = append(result[tid], Relation{
						ToolID:       tid,
						RelatedID:    otherID,
						RelationType: "co_used",
						Strength:     strength,
					})
				}
			}
		}
	}

	return result, nil
}

// GetTopUsers returns the top N users by usage frequency for a given tool.
func (tut *ToolUsageTracker) GetTopUsers(ctx context.Context, toolID string, limit int) ([]string, error) {
	tut.mu.RLock()
	defer tut.mu.RUnlock()

	type userFreq struct {
		userID string
		freq   int
	}

	var users []userFreq
	for _, p := range tut.patterns {
		if p.ToolID == toolID {
			users = append(users, userFreq{userID: p.UserID, freq: p.Frequency})
		}
	}

	// Simple bubble sort for top N (small n expected)
	for i := 0; i < len(users); i++ {
		for j := i + 1; j < len(users); j++ {
			if users[j].freq > users[i].freq {
				users[i], users[j] = users[j], users[i]
			}
		}
	}

	if limit > len(users) {
		limit = len(users)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = users[i].userID
	}

	return result, nil
}

// GetToolFrequency returns the total usage count for a given tool.
func (tut *ToolUsageTracker) GetToolFrequency(ctx context.Context, toolID string) (int, error) {
	tut.mu.RLock()
	defer tut.mu.RUnlock()

	var total int
	for _, p := range tut.patterns {
		if p.ToolID == toolID {
			total += p.Frequency
		}
	}
	return total, nil
}
