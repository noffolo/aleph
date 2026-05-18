package suggestion

import (
	"math"
	"sort"
)

type ToolEmbed struct {
	Name        string
	Category    string
	Description string
	Embedding   []float32
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	sim := dot / math.Sqrt(normA*normB)
	if sim < 0 {
		return 0
	}
	return sim
}

func normalizeScores(suggestions []Suggestion) {
	if len(suggestions) == 0 {
		return
	}
	minScore := suggestions[0].Score
	maxScore := suggestions[0].Score
	for _, s := range suggestions {
		if s.Score < minScore {
			minScore = s.Score
		}
		if s.Score > maxScore {
			maxScore = s.Score
		}
	}
	scoreRange := maxScore - minScore
	for i := range suggestions {
		if scoreRange == 0 {
			suggestions[i].Score = 1.0
		} else {
			suggestions[i].Score = (suggestions[i].Score - minScore) / scoreRange
		}
	}
}

func maxUsage(stats map[string]int) int {
	high := 0
	for _, v := range stats {
		if v > high {
			high = v
		}
	}
	return high
}

func Rank(toolDefs []ToolEmbed, userEmbedding []float32, usageStats map[string]int, toolKey func(cat, name string) string) []Suggestion {
	if len(toolDefs) == 0 {
		return nil
	}

	const similarityWeight = 0.7
	const usageWeight = 0.3

	maxUse := maxUsage(usageStats)
	suggestions := make([]Suggestion, 0, len(toolDefs))

	for _, td := range toolDefs {
		sim := cosineSimilarity(userEmbedding, td.Embedding)
		var usageScore float64
		if maxUse > 0 {
			usageScore = float64(usageStats[toolKey(td.Category, td.Name)]) / float64(maxUse)
		}
		composite := similarityWeight*sim + usageWeight*usageScore

		suggestions = append(suggestions, Suggestion{
			ToolName:    td.Name,
			Category:    td.Category,
			Description: td.Description,
			Score:       composite,
			Similarity:  sim,
			UsageScore:  usageScore,
			Reason:      buildReason(sim, usageScore),
		})
	}

	normalizeScores(suggestions)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})
	return suggestions
}

func buildReason(similarity, usageScore float64) string {
	if usageScore > 0.5 && similarity > 0.5 {
		return "Similar to tools you've used before"
	}
	if similarity > 0.7 {
		return "Strong semantic match with your request"
	}
	if usageScore > 0.5 {
		return "Frequently used tool that may help"
	}
	if similarity > 0.3 {
		return "Partial semantic match with your request"
	}
	return "Available tool that may be relevant"
}
