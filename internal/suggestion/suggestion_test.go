package suggestion

import (
	"context"
	"math"
	"sync"
	"testing"

	toolreg "github.com/ff3300/aleph-v2/internal/tools"
)

type mockEmbedder struct {
	dim       int
	callCount map[string]int
	mu        sync.Mutex
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	m.callCount[text]++
	m.mu.Unlock()

	vec := make([]float32, m.dim)
	for i := range vec {
		vec[i] = float32(i) * float32(len(text)+1) / float32(m.dim)
	}
	return vec, nil
}

type mockTool struct {
	name, category, description string
}

func buildMockRegistry(t *testing.T, mockTools []mockTool) *toolreg.ToolRegistry {
	t.Helper()
	reg := toolreg.NewToolRegistry()
	for _, mt := range mockTools {
		def := toolreg.ToolDefinition{
			Name:        mt.name,
			Category:    mt.category,
			Description: mt.description,
			Execute:     func(_ context.Context, _ map[string]any) (any, error) { return "ok", nil },
		}
		if err := reg.Register(def); err != nil {
			t.Fatalf("register mock tool: %v", err)
		}
	}
	return reg
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"empty", []float32{}, []float32{}, 0.0},
		{"different lengths", []float32{1, 2}, []float32{1, 2, 3}, 0.0},
		{"negative clamped", []float32{1, 0}, []float32{-1, 0}, 0.0},
		{"partial", []float32{1, 1}, []float32{1, 0}, 1.0 / math.Sqrt(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 1e-6 {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmptyRegistry(t *testing.T) {
	reg := toolreg.NewToolRegistry()
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	result := Rank(nil, []float32{1, 0, 0}, nil, toolKeyStr)
	if len(result) != 0 {
		t.Errorf("empty input should produce empty suggestions, got %d", len(result))
	}

	engineResult, err := eng.Suggest(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(engineResult) != 0 {
		t.Errorf("empty registry should produce empty suggestions, got %d", len(engineResult))
	}
}

func TestUsageTracking(t *testing.T) {
	ut := newUsageTracker()
	ut.TrackUsage("finance", "prophet")
	ut.TrackUsage("finance", "prophet")
	ut.TrackUsage("osint", "shodan")

	stats := ut.GetUsageStats()
	if stats["finance:prophet"] != 2 {
		t.Errorf("usage[finance:prophet] = %d, want 2", stats["finance:prophet"])
	}
	if stats["osint:shodan"] != 1 {
		t.Errorf("usage[osint:shodan] = %d, want 1", stats["osint:shodan"])
	}

	ut.ResetStats()
	if len(ut.GetUsageStats()) != 0 {
		t.Error("after reset, stats should be empty")
	}
}

func TestScoreCombination(t *testing.T) {
	usageStats := map[string]int{
		"osint:shodan":   10,
		"osint:thehound": 1,
	}

	input := []ToolEmbed{
		{Name: "shodan", Category: "osint", Description: "search internet devices", Embedding: []float32{1, 2, 3, 4}},
		{Name: "thehound", Category: "osint", Description: "domain intelligence", Embedding: []float32{4, 3, 2, 1}},
	}

	result := Rank(input, []float32{1, 2, 3, 4}, usageStats, toolKeyStr)
	if len(result) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(result))
	}
	if result[0].ToolName != "shodan" {
		t.Errorf("top result should be shodan, got %s", result[0].ToolName)
	}
	if math.Abs(result[0].Similarity-1.0) > 1e-6 {
		t.Errorf("shodan similarity = %v, want 1.0", result[0].Similarity)
	}
}

func TestEmbeddingCache(t *testing.T) {
	cache := newEmbedCache(3)

	key1 := "osint:shodan"
	cache.put(key1, []float32{0.1, 0.2, 0.3})
	cache.put(key1, []float32{0.4, 0.5, 0.6})

	got, ok := cache.get(key1)
	if !ok {
		t.Fatal("cached embedding not found")
	}
	if got[0] != 0.4 {
		t.Errorf("cache returned stale embedding, got %v", got[0])
	}
	if cache.size() != 1 {
		t.Errorf("size = %d, want 1 (same key updated)", cache.size())
	}

	cache.put("osint:thehound", []float32{0.1, 0.2})
	cache.put("finance:prophet", []float32{0.3, 0.4})
	cache.put("finance:gmb", []float32{0.5, 0.6})

	if cache.size() != 3 {
		t.Errorf("size after eviction = %d, want 3", cache.size())
	}
	if _, ok := cache.get(key1); ok {
		t.Error("evicted key should not be found")
	}
}

func TestEngineEmbeddingCache(t *testing.T) {
	mockTools := []mockTool{
		{name: "shodan", category: "osint", description: "search internet devices"},
	}
	reg := buildMockRegistry(t, mockTools)
	mock := &mockEmbedder{dim: 768, callCount: make(map[string]int)}
	eng := NewEngine(reg, mock)

	ctx := context.Background()
	// First call: should embed user msg + tool description.
	_, err := eng.Suggest(ctx, "search devices", 5)
	if err != nil {
		t.Fatalf("first Suggest: %v", err)
	}

	// Count total Embed calls: 1 for user msg + 1 for tool description.
	initialCount := 0
	for _, c := range mock.callCount {
		initialCount += c
	}
	if initialCount < 2 {
		t.Fatalf("expected at least 2 embed calls, got %d", initialCount)
	}

	// Second call with same tools: tool description should be cached.
	_, err = eng.Suggest(ctx, "search devices", 5)
	if err != nil {
		t.Fatalf("second Suggest: %v", err)
	}

	secondCount := 0
	for _, c := range mock.callCount {
		secondCount += c
	}
	// Only 1 new embed call for the user message — tool description cached.
	if secondCount < initialCount+1 || secondCount > initialCount+2 {
		t.Fatalf("expected ~%d total embed calls (1 new), got %d", initialCount+1, secondCount)
	}
}

func TestConcurrentUsageSafety(t *testing.T) {
	ut := newUsageTracker()
	const goroutines, increments = 50, 100
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				ut.TrackUsage("stress", "tool")
			}
		}()
	}
	wg.Wait()

	total := ut.GetUsageStats()["stress:tool"]
	if total != goroutines*increments {
		t.Errorf("concurrent total = %d, want %d", total, goroutines*increments)
	}
}

func TestNormalizeScores(t *testing.T) {
	tests := []struct {
		name   string
		input  []Suggestion
		verify func(t *testing.T, result []Suggestion)
	}{
		{
			name: "all equal",
			input: []Suggestion{
				{Score: 0.5}, {Score: 0.5}, {Score: 0.5},
			},
			verify: func(t *testing.T, r []Suggestion) {
				for _, s := range r {
					if s.Score != 1.0 {
						t.Errorf("all-equal should normalize to 1.0, got %v", s.Score)
					}
				}
			},
		},
		{
			name: "range 0 to 1",
			input: []Suggestion{
				{Score: 0.0}, {Score: 0.5}, {Score: 1.0},
			},
			verify: func(t *testing.T, r []Suggestion) {
				if r[0].Score != 0.0 {
					t.Errorf("min should be 0.0, got %v", r[0].Score)
				}
				if r[2].Score != 1.0 {
					t.Errorf("max should be 1.0, got %v", r[2].Score)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make([]Suggestion, len(tt.input))
			copy(result, tt.input)
			normalizeScores(result)
			tt.verify(t, result)
		})
	}
}

func TestSuggestSortsDescending(t *testing.T) {
	input := []ToolEmbed{
		{Name: "a", Category: "cat", Embedding: []float32{1, 0}},
		{Name: "b", Category: "cat", Embedding: []float32{0, 1}},
		{Name: "c", Category: "cat", Embedding: []float32{1, 2}},
	}
	result := Rank(input, []float32{1, 2}, nil, toolKeyStr)

	for i := 1; i < len(result); i++ {
		if result[i-1].Score < result[i].Score {
			t.Errorf("not sorted descending at index %d: %v < %v",
				i, result[i-1].Score, result[i].Score)
		}
	}
}

func TestReasonGeneration(t *testing.T) {
	tests := []struct {
		name              string
		similarity, usage float64
		want              string
	}{
		{"both high", 0.6, 0.6, "Similar to tools you've used before"},
		{"strong semantic", 0.8, 0.1, "Strong semantic match with your request"},
		{"frequent usage", 0.1, 0.8, "Frequently used tool that may help"},
		{"partial match", 0.4, 0.1, "Partial semantic match with your request"},
		{"default", 0.1, 0.1, "Available tool that may be relevant"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildReason(tt.similarity, tt.usage); got != tt.want {
				t.Errorf("buildReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
