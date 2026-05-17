package genesis

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type SuggesterInput struct {
	ProjectID     string
	AgentID       string
	ChatHistory   []ChatMessage
	ToolUsage     []ToolUsageStat
	ExistingTools []string
}

type ChatMessage struct {
	Role    string
	Content string
}

type ToolUsageStat struct {
	ToolName string
	Count    int
}

type Suggester struct{}

func NewSuggester() *Suggester {
	return &Suggester{}
}

var (
	whatAboutRe = regexp.MustCompile(`(?i)\bwhat\s+about\s+(\w+)`)
	canYouRe    = regexp.MustCompile(`(?i)\bcan\s+you\s+(\w+)`)
	howAboutRe  = regexp.MustCompile(`(?i)\bhow\s+about\s+(\w+)`)
	isThereRe   = regexp.MustCompile(`(?i)\bis\s+there\s+(?:a|an)\s+(\w+)`)
)

// commonEnglishWords lists short/common capitalized words to exclude from ontology detection.
var commonEnglishWords = map[string]bool{
	"i": true, "II": true, "III": true, "IV": true,
	"Hello": true, "Hi": true, "Hey": true,
	"Thanks": true, "Thank": true, "Please": true,
	"Yes": true, "No": true, "Yeah": true,
	"Sure": true, "Ok": true, "Okay": true,
	"Let": true, "Would": true, "Could": true, "Should": true,
	"Also": true, "Then": true, "Now": true, "Here": true,
	"Just": true, "Like": true,
}

func (s *Suggester) Analyze(ctx context.Context, input SuggesterInput) ([]Suggestion, error) {
	slog.Info("genesis: analyzing patterns",
		"project", input.ProjectID,
		"agent", input.AgentID,
		"chat_messages", len(input.ChatHistory),
		"tool_usage_stats", len(input.ToolUsage),
		"existing_tools", len(input.ExistingTools),
	)

	existingSet := make(map[string]bool, len(input.ExistingTools))
	for _, t := range input.ExistingTools {
		existingSet[strings.ToLower(t)] = true
	}

	var results []Suggestion

	// PASS 1 — Chat Pattern Analysis (ontology suggestions)
	results = append(results, s.analyzeChatPatterns(input.ChatHistory, existingSet)...)

	// PASS 2 — Tool Usage Analysis
	results = append(results, s.analyzeToolUsage(input.ToolUsage, input.ChatHistory, existingSet)...)

	// PASS 3 — Query Pattern Analysis (with existing tool filtering)
	for _, qp := range s.analyzeQueryPatterns(input.ChatHistory) {
		if existingSet[strings.ToLower(qp.Name)] {
			slog.Debug("genesis: skipping existing tool in query patterns", "name", qp.Name)
			continue
		}
		results = append(results, qp)
	}

	// Deduplicate by Description
	seen := make(map[string]bool, len(results))
	deduped := make([]Suggestion, 0, len(results))
	for _, sug := range results {
		key := strings.ToLower(strings.TrimSpace(sug.Description))
		if seen[key] {
			slog.Debug("genesis: deduplicating suggestion", "description", sug.Description)
			continue
		}
		seen[key] = true
		deduped = append(deduped, sug)
	}

	slog.Info("genesis: analysis complete",
		"raw_suggestions", len(results),
		"unique_suggestions", len(deduped),
	)

	if deduped == nil {
		return []Suggestion{}, nil
	}
	return deduped, nil
}

// analyzeChatPatterns detects ontology suggestions from chat history.
// Heuristic: user asks "what about X?" or mentions a capitalized noun 3+ times
// that isn't already an existing tool.
func (s *Suggester) analyzeChatPatterns(history []ChatMessage, existingSet map[string]bool) []Suggestion {
	if len(history) == 0 {
		return nil
	}

	nounCounts := make(map[string]int)
	whatAboutTopics := make(map[string]int)

	for _, msg := range history {
		if msg.Role != "user" {
			continue
		}
		content := msg.Content

		for _, re := range []*regexp.Regexp{whatAboutRe, howAboutRe, isThereRe} {
			matches := re.FindAllStringSubmatch(content, -1)
			for _, m := range matches {
				if len(m) > 1 {
					topic := strings.TrimSpace(m[1])
					if topic != "" {
						whatAboutTopics[topic]++
					}
				}
			}
		}

		// Find capitalized nouns (words starting with uppercase)
		words := strings.Fields(content)
		for _, word := range words {
			clean := strings.Trim(word, ".,!?;:\"'()[]{}")
			if clean == "" {
				continue
			}
			runes := []rune(clean)
			if !unicode.IsUpper(runes[0]) {
				continue
			}
			if commonEnglishWords[clean] {
				continue
			}
			if existingSet[strings.ToLower(clean)] {
				continue
			}
			if len(clean) < 3 {
				continue
			}
			nounCounts[clean]++
		}
	}

	var suggestions []Suggestion

	// Generate ontology suggestions from repeated nouns
	for noun, count := range nounCounts {
		if count < 3 {
			continue
		}
		confidence := 0.3
		if count >= 5 {
			confidence = 0.7
		} else if count >= 4 {
			confidence = 0.5
		}
		priority := 3
		if confidence >= 0.7 {
			priority = 1
		} else if confidence >= 0.5 {
			priority = 2
		}

		slog.Info("genesis: ontology suggestion from chat pattern",
			"noun", noun, "mentions", count, "confidence", confidence,
		)

		suggestions = append(suggestions, Suggestion{
			ID:          fmt.Sprintf("onto-%s-%d", strings.ToLower(noun), time.Now().Unix()),
			Name:        noun,
			Type:        "ontology",
			Description: fmt.Sprintf("New ontology object detected: %q mentioned %d times in chat — consider adding to knowledge graph", noun, count),
			Priority:    priority,
			Confidence:  confidence,
			Status:      "pending",
			CreatedAt:   time.Now(),
		})
	}

	// Generate ontology suggestions from "what about X?" patterns
	for topic, count := range whatAboutTopics {
		if count < 1 {
			continue
		}
		alreadyCovered := false
		for _, sug := range suggestions {
			if strings.EqualFold(sug.Name, topic) {
				alreadyCovered = true
				break
			}
		}
		if alreadyCovered {
			continue
		}
		if existingSet[strings.ToLower(topic)] {
			continue
		}

		confidence := 0.3 + float64(count)*0.15
		if confidence > 0.8 {
			confidence = 0.8
		}
		priority := 2
		if count >= 3 {
			priority = 1
		}

		slog.Info("genesis: ontology suggestion from inquiry pattern",
			"topic", topic, "count", count, "confidence", confidence,
		)

		suggestions = append(suggestions, Suggestion{
			ID:          fmt.Sprintf("onto-inquiry-%s-%d", strings.ToLower(topic), time.Now().Unix()),
			Name:        topic,
			Type:        "ontology",
			Description: fmt.Sprintf("User inquired about %q %d time(s) — consider adding as a knowledge graph object", topic, count),
			Priority:    priority,
			Confidence:  confidence,
			Status:      "pending",
			CreatedAt:   time.Now(),
		})
	}

	return suggestions
}

// analyzeToolUsage detects missing or underused tools from usage stats and chat history.
// Heuristic: tools with Count < 2 are underused; user phrases like "can you X?"
// that don't match existing tools suggest missing functionality.
func (s *Suggester) analyzeToolUsage(usage []ToolUsageStat, history []ChatMessage, existingSet map[string]bool) []Suggestion {
	var suggestions []Suggestion

	// Find underused tools (Count < 2)
	for _, stat := range usage {
		if stat.Count >= 2 {
			continue
		}

		slog.Info("genesis: underused tool detected",
			"tool", stat.ToolName, "count", stat.Count,
		)

		suggestions = append(suggestions, Suggestion{
			ID:          fmt.Sprintf("tool-underused-%s-%d", stat.ToolName, time.Now().Unix()),
			Name:        stat.ToolName,
			Type:        "tool",
			Description: fmt.Sprintf("Tool %q is underused (%d invocation(s)) — consider merging or deprecating", stat.ToolName, stat.Count),
			Priority:    3,
			Confidence:  0.5,
			Status:      "pending",
			CreatedAt:   time.Now(),
		})
	}

	// Detect missing tools from "can you X?" patterns in chat history
	requestedActions := make(map[string]int)
	for _, msg := range history {
		if msg.Role != "user" {
			continue
		}
		matches := canYouRe.FindAllStringSubmatch(msg.Content, -1)
		for _, m := range matches {
			if len(m) > 1 {
				action := strings.ToLower(strings.TrimSpace(m[1]))
				if action != "" && len(action) > 2 {
					requestedActions[action]++
				}
			}
		}
	}

	for action, count := range requestedActions {
		// Check if any existing tool name contains this action (fuzzy check)
		matched := false
		for existing := range existingSet {
			if strings.Contains(existing, action) || strings.Contains(action, existing) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		// Skip very generic actions
		if action == "help" || action == "do" || action == "make" || action == "tell" {
			continue
		}

		confidence := 0.3 + float64(count)*0.15
		if confidence > 0.8 {
			confidence = 0.8
		}

		slog.Info("genesis: missing tool detected from chat",
			"action", action, "requests", count, "confidence", confidence,
		)

		suggestions = append(suggestions, Suggestion{
			ID:          fmt.Sprintf("tool-missing-%s-%d", action, time.Now().Unix()),
			Name:        action,
			Type:        "tool",
			Description: fmt.Sprintf("Users requested %q %d time(s) with no matching tool — consider implementing", action, count),
			Priority:    2,
			Confidence:  confidence,
			Status:      "pending",
			CreatedAt:   time.Now(),
		})
	}

	return suggestions
}

// analyzeQueryPatterns detects repeated question patterns from chat history.
// Heuristic: identify same keywords appearing 3+ times in user messages.
func (s *Suggester) analyzeQueryPatterns(history []ChatMessage) []Suggestion {
	if len(history) == 0 {
		return nil
	}

	var userMessages []string
	for _, msg := range history {
		if msg.Role == "user" {
			userMessages = append(userMessages, msg.Content)
		}
	}

	if len(userMessages) == 0 {
		return nil
	}

	// Extract meaningful words (nouns, verbs, length > 3)
	stopWords := map[string]bool{
		"the": true, "this": true, "that": true, "what": true, "with": true,
		"from": true, "have": true, "been": true, "were": true, "they": true,
		"tell": true, "will": true, "there": true, "when": true, "where": true,
		"which": true, "their": true, "your": true, "about": true, "would": true,
		"could": true, "should": true, "does": true, "just": true, "like": true,
		"also": true, "then": true, "than": true, "more": true, "some": true,
		"into": true, "over": true, "such": true, "only": true, "other": true,
		"these": true, "those": true, "here": true,
		"want": true, "need": true, "know": true, "think": true,
	}

	wordCounts := make(map[string]int)
	for _, content := range userMessages {
		words := strings.Fields(content)
		for _, word := range words {
			clean := strings.Trim(strings.ToLower(word), ".,!?;:\"'()[]{}")
			if len(clean) < 4 {
				continue
			}
			if stopWords[clean] {
				continue
			}
			wordCounts[clean]++
		}
	}

	var suggestions []Suggestion
	for word, count := range wordCounts {
		if count < 3 {
			continue
		}

		confidence := 0.4 + float64(count-2)*0.1
		if confidence > 0.9 {
			confidence = 0.9
		}
		priority := 2
		if count >= 5 {
			priority = 1
		}

		slog.Info("genesis: query pattern detected",
			"keyword", word, "occurrences", count, "confidence", confidence,
		)

		suggestions = append(suggestions, Suggestion{
			ID:          fmt.Sprintf("query-%s-%d", word, time.Now().Unix()),
			Name:        word,
			Type:        "query_pattern",
			Description: fmt.Sprintf("Frequent query word %q appears %d time(s) — consider pre-computed view or cached response", word, count),
			Priority:    priority,
			Confidence:  confidence,
			Status:      "pending",
			CreatedAt:   time.Now(),
		})
	}

	return suggestions
}
