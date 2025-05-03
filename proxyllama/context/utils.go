package context

import (
	"proxyllama/models"
	"strings"
)

// generateTitle creates a title from the first message content
func generateTitle(content string) string {
	// Get first 30 chars or less if the string is shorter
	maxLen := 30
	if len(content) < maxLen {
		maxLen = len(content)
	}
	title := content[:maxLen]

	// Remove newlines
	title = strings.ReplaceAll(title, "\n", " ")

	// Add ellipsis if we truncated
	if len(content) > 30 {
		title += "..."
	}

	return title
}

// describeMessageChain formats a message chain for logging
func describeMessageChain(messages []models.OllamaMessage) string {
	var roles []string
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			roles = append(roles, "SYS")
		case "user":
			roles = append(roles, "USER")
		case "assistant":
			roles = append(roles, "ASST")
		default:
			roles = append(roles, msg.Role)
		}
	}
	return strings.Join(roles, " → ")
}

// CreateSystemMessage creates an OllamaMessage with system role
func CreateSystemMessage(content string) models.OllamaMessage {
	return models.OllamaMessage{
		Role:    "system",
		Content: content,
	}
}

// FindMaxLevel returns the highest summary level in a slice of summaries
func FindMaxLevel(summaries []models.Summary) int {
	maxLevel := 0
	for _, summary := range summaries {
		if summary.Level > maxLevel {
			maxLevel = summary.Level
		}
	}
	return maxLevel
}

// GroupSummariesByLevel organizes summaries into a map keyed by level
func GroupSummariesByLevel(summaries []models.Summary) map[int][]models.Summary {
	summariesByLevel := make(map[int][]models.Summary)
	for _, summary := range summaries {
		summariesByLevel[summary.Level] = append(summariesByLevel[summary.Level], summary)
	}
	return summariesByLevel
}

// CountSummariesAtLevel counts how many summaries exist at a specific level
func CountSummariesAtLevel(summaries []models.Summary, level int) int {
	count := 0
	for _, summary := range summaries {
		if summary.Level == level {
			count++
		}
	}
	return count
}
