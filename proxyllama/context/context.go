package context

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"proxyllama/config"
	"proxyllama/storage"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ConversationContext holds data needed to maintain context across requests
type ConversationContext struct {
	ConversationID int       `json:"conversation_id"`
	Title          string    `json:"title"`
	UserID         string    `json:"user_id"`
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Summaries      []Summary `json:"summaries"` // Holds summaries of older messages
}

// Message represents a single exchange in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	ID      int    `json:"-"` // Internal use only, not sent to LLM
}

// Summary represents a consolidated summary of messages or other summaries
type Summary struct {
	Content string `json:"content"`
	Level   int    `json:"-"` // Internal use only, not sent to LLM
	ID      int    `json:"-"` // Internal use only, not sent to LLM
}

// GetOrCreateConversation retrieves or creates a conversation context
func GetOrCreateConversation(ctx context.Context, userID, model string, conversationID *int) (*ConversationContext, error) {
	// Ensure user exists
	if err := storage.EnsureUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to ensure user exists: %w", err)
	}

	var convContext ConversationContext
	convContext.UserID = userID
	convContext.Model = model

	// If conversationID is provided, load that conversation
	if conversationID != nil {
		// Verify the conversation exists and belongs to the user
		conv, err := storage.GetConversation(ctx, *conversationID)
		if err != nil {
			if err == pgx.ErrNoRows {
				cid, err := storage.CreateConversation(ctx, userID, model, "")
				if err != nil {
					return nil, fmt.Errorf("failed to create conversation: %w", err)
				}
				return GetOrCreateConversation(ctx, userID, model, &cid)
			}
			return nil, fmt.Errorf("failed to get conversation: %w", err)
		}

		if conv.UserID != userID {
			return nil, fmt.Errorf("conversation does not belong to user")
		}

		convContext.ConversationID = conv.ID
		convContext.Title = conv.Title

		// Load messages
		messages, err := storage.GetConversationHistory(ctx, *conversationID)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("failed to load conversation history: %w", err)
		}

		// Convert storage.Message to context.Message
		for _, msg := range messages {
			convContext.Messages = append(convContext.Messages, Message{
				Role:    msg.Role,
				Content: msg.Content,
				ID:      msg.ID,
			})
		}

		// Load summaries
		summaries, err := storage.GetSummariesForConversation(ctx, *conversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to load summaries: %w", err)
		}

		// Convert storage.Summary to context.Summary
		for _, summary := range summaries {
			convContext.Summaries = append(convContext.Summaries, Summary{
				Content: summary.Content,
				Level:   summary.Level,
				ID:      summary.ID,
			})
		}

	} else {
		// Create a new conversation
		id, err := storage.CreateConversation(ctx, userID, model, "New conversation")
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
		convContext.ConversationID = id
	}

	return &convContext, nil
}

// AddUserMessage adds a user message to the conversation
func (cc *ConversationContext) AddUserMessage(ctx context.Context, content string) error {
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "user", content)
	if err != nil {
		return err
	}

	// Add to context
	cc.Messages = append(cc.Messages, Message{
		Role:    "user",
		Content: content,
		ID:      msgID,
	})

	// Update title if this is the first message
	if len(cc.Messages) == 1 {
		title := generateTitle(content)
		if err := storage.UpdateConversationTitle(ctx, cc.ConversationID, title); err != nil {
			log.Printf("Failed to update conversation title: %v", err)
		}
	}

	return nil
}

// AddAssistantMessage adds an assistant message to the conversation
func (cc *ConversationContext) AddAssistantMessage(ctx context.Context, content string) error {
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "assistant", content)
	if err != nil {
		return err
	}

	// Add to context
	cc.Messages = append(cc.Messages, Message{
		Role:    "assistant",
		Content: content,
		ID:      msgID,
	})

	// Check if we need to summarize messages
	if cc.shouldSummarize() {
		if err := cc.summarizeMessages(ctx); err != nil {
			log.Printf("Failed to summarize messages: %v", err)
			// Continue even if summarization fails
		}
	}

	return nil
}

// shouldSummarize checks if we have enough messages to create a summary
func (cc *ConversationContext) shouldSummarize() bool {
	// We need at least N messages (where N is configurable) to create a summary
	return len(cc.Messages) >= config.GetConfig().Summarization.MessagesBeforeSummary
}

// summarizeMessages creates a summary of the oldest messages
func (cc *ConversationContext) summarizeMessages(ctx context.Context) error {
	// Calculate how many messages to summarize
	// We always keep the most recent messages (the last N messages)
	messagesToKeep := config.GetConfig().Summarization.MessagesBeforeSummary / 2
	if messagesToKeep < 2 {
		messagesToKeep = 2 // Always keep at least the last user and assistant message
	}

	messagesToSummarize := len(cc.Messages) - messagesToKeep
	if messagesToSummarize <= 0 {
		return nil // Not enough messages to summarize
	}

	// Extract the messages to summarize
	var messagesToSummarizeContent []Message
	var messageIDs []int

	for i := 0; i < messagesToSummarize; i++ {
		messagesToSummarizeContent = append(messagesToSummarizeContent, cc.Messages[i])
		messageIDs = append(messageIDs, cc.Messages[i].ID)
	}

	// Generate the summary using Ollama
	summaryContent, err := cc.generateSummary(ctx, messagesToSummarizeContent)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create a new summary record in the database
	summaryID, err := storage.CreateSummary(ctx, cc.ConversationID, summaryContent, 1, messageIDs)
	if err != nil {
		return fmt.Errorf("failed to store summary: %w", err)
	}

	// Add the summary to our context
	summary := Summary{
		Content: summaryContent,
		Level:   1,
		ID:      summaryID,
	}
	cc.Summaries = append(cc.Summaries, summary)

	// Remove the summarized messages from our in-memory context
	cc.Messages = cc.Messages[messagesToSummarize:]

	// Check if we need to consolidate summaries
	if cc.shouldConsolidateSummaries(ctx) {
		if err := cc.consolidateSummaries(ctx); err != nil {
			log.Printf("Failed to consolidate summaries: %v", err)
		}
	}

	return nil
}

// shouldConsolidateSummaries checks if we have enough summaries to consolidate them
func (cc *ConversationContext) shouldConsolidateSummaries(ctx context.Context) bool {
	// Count the number of summaries at each level
	levelCounts := make(map[int]int)

	for _, summary := range cc.Summaries {
		levelCounts[summary.Level]++
	}

	// For each level, check if we have enough summaries to consolidate
	for _, count := range levelCounts {
		if count >= config.GetConfig().Summarization.SummariesBeforeConsolidation {
			return true
		}
	}

	return false
}

// consolidateSummaries creates a summary of summaries for each level
func (cc *ConversationContext) consolidateSummaries(ctx context.Context) error {
	// Get all summaries for this conversation
	allSummaries, err := storage.GetSummariesForConversation(ctx, cc.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation summaries: %w", err)
	}

	// Group summaries by level
	summariesByLevel := make(map[int][]storage.Summary)
	for _, summary := range allSummaries {
		summariesByLevel[summary.Level] = append(summariesByLevel[summary.Level], summary)
	}

	// For each level that has enough summaries, consolidate them
	for level, summaries := range summariesByLevel {
		if len(summaries) < config.GetConfig().Summarization.SummariesBeforeConsolidation {
			continue
		}

		// Prepare messages for summarization
		var messagesToSummarize []Message
		var summaryIDs []int

		for _, summary := range summaries {
			messagesToSummarize = append(messagesToSummarize, Message{
				Role:    "system",
				Content: summary.Content,
				ID:      summary.ID,
			})
			summaryIDs = append(summaryIDs, summary.ID)
		}

		// Generate a new higher-level summary
		summaryContent, err := cc.generateSummary(ctx, messagesToSummarize)
		if err != nil {
			return fmt.Errorf("failed to generate higher-level summary: %w", err)
		}

		// Store the new summary with an incremented level
		nextLevel := level + 1
		summaryID, err := storage.CreateSummary(ctx, cc.ConversationID, summaryContent, nextLevel, summaryIDs)
		if err != nil {
			return fmt.Errorf("failed to store higher-level summary: %w", err)
		}

		// Add the summary to our context
		cc.Summaries = append(cc.Summaries, Summary{
			Content: summaryContent,
			Level:   nextLevel,
			ID:      summaryID,
		})

		// Remove the consolidated summaries from our in-memory context
		var updatedSummaries []Summary
		for _, s := range cc.Summaries {
			shouldKeep := true
			for _, id := range summaryIDs {
				if s.ID == id {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				updatedSummaries = append(updatedSummaries, s)
			}
		}

		cc.Summaries = updatedSummaries
	}

	return nil
}

// generateSummary uses Ollama to generate a summary of the provided messages
func (cc *ConversationContext) generateSummary(ctx context.Context, messages []Message) (string, error) {
	// Determine which model to use for summarization
	summaryModel := cc.Model
	conf := config.GetConfig()
	if conf.Summarization.SummaryModel != "" {
		summaryModel = conf.Summarization.SummaryModel
	}

	// Format the messages into a conversation for Ollama
	var ollamaMessages []map[string]string

	// Add system prompt
	ollamaMessages = append(ollamaMessages, map[string]string{
		"role":    "system",
		"content": conf.Summarization.SystemPrompt,
	})

	// Add conversation messages
	for _, message := range messages {
		ollamaMessages = append(ollamaMessages, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}

	// Create request payload
	requestBody := map[string]interface{}{
		"model":    summaryModel,
		"messages": ollamaMessages,
		"stream":   false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling summary request: %w", err)
	}

	// Send request to Ollama
	ollamaURL := fmt.Sprintf("%s/api/chat", strings.TrimSuffix(conf.Ollama.BaseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating summary request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use a client with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending summary request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error response from Ollama: %s, status: %d", string(body), resp.StatusCode)
	}

	// Parse the response
	var ollamaResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResponse); err != nil {
		return "", fmt.Errorf("error decoding summary response: %w", err)
	}

	// Extract the summary from the response
	messageObj, ok := ollamaResponse["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response structure: %v", ollamaResponse)
	}

	content, ok := messageObj["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing content in response: %v", messageObj)
	}

	return content, nil
}

// ToJSON converts the conversation context to Ollama-compatible format
func (cc *ConversationContext) ToJSON() ([]byte, error) {
	// Ollama expects specific format for chat endpoints
	type OllamaMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type OllamaRequest struct {
		Model    string          `json:"model"`
		Messages []OllamaMessage `json:"messages"`
		Stream   bool            `json:"stream"`
	}

	req := OllamaRequest{
		Model:  cc.Model,
		Stream: true,
	}

	conf := config.GetConfig()
	messagesBeforeSummary := conf.Summarization.MessagesBeforeSummary
	summariesBeforeConsolidation := conf.Summarization.SummariesBeforeConsolidation

	// Only add system messages if we actually have summaries to include
	if len(cc.Summaries) > 0 {
		// Start with adding a system message to introduce the conversation
		req.Messages = append(req.Messages, OllamaMessage{
			Role:    "system",
			Content: "This is a continued conversation. The historical context is provided in the following messages.",
		})

		// Find highest summary level
		highestLevel := 0
		for _, summary := range cc.Summaries {
			if summary.Level > highestLevel {
				highestLevel = summary.Level
			}
		}

		// Group summaries by level for more efficient access
		summariesByLevel := make(map[int][]Summary)
		for _, summary := range cc.Summaries {
			summariesByLevel[summary.Level] = append(summariesByLevel[summary.Level], summary)
		}

		// Add summaries from highest level to lowest level 1
		summaryCount := 0
		for level := highestLevel; level >= 1; level-- {
			levelSummaries := summariesByLevel[level]
			if len(levelSummaries) == 0 {
				continue
			}

			// For higher levels, include all summaries as they're already consolidated
			// For level 1, limit to the most recent ones based on configuration
			maxToInclude := len(levelSummaries)
			if level == 1 && maxToInclude > summariesBeforeConsolidation {
				maxToInclude = summariesBeforeConsolidation
			}

			// Add newest summaries first (they're more relevant)
			for i := len(levelSummaries) - 1; i >= len(levelSummaries)-maxToInclude && i >= 0; i-- {
				req.Messages = append(req.Messages, OllamaMessage{
					Role:    "system",
					Content: fmt.Sprintf("Previous conversation summary (level %d): %s", level, levelSummaries[i].Content),
				})
				summaryCount++
			}

			// If we've added enough summaries, stop
			if summaryCount >= summariesBeforeConsolidation && level > 1 {
				break
			}
		}
	}

	// Calculate how many recent messages to include (more efficiently)
	messagesToInclude := messagesBeforeSummary
	if len(cc.Messages) < messagesToInclude {
		messagesToInclude = len(cc.Messages)
	}

	// Add the most recent messages
	startIndex := len(cc.Messages) - messagesToInclude
	if startIndex < 0 {
		startIndex = 0
	}

	// Add regular messages (most recent based on configuration)
	for i := startIndex; i < len(cc.Messages); i++ {
		req.Messages = append(req.Messages, OllamaMessage{
			Role:    cc.Messages[i].Role,
			Content: cc.Messages[i].Content,
		})
	}

	return json.Marshal(req)
}

// Simple function to generate a title from the first message content
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
