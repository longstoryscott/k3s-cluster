package context

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"proxyllama/storage"
	"strings"
)

// ConversationContext holds data needed to maintain context across requests
type ConversationContext struct {
	ConversationID int       `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
}

// Message represents a single exchange in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
			return nil, fmt.Errorf("failed to get conversation: %w", err)
		}

		if conv.UserID != userID {
			return nil, fmt.Errorf("conversation does not belong to user")
		}

		convContext.ConversationID = *conversationID

		// Load messages
		messages, err := storage.GetConversationHistory(ctx, *conversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation history: %w", err)
		}

		// Convert storage.Message to context.Message
		for _, msg := range messages {
			convContext.Messages = append(convContext.Messages, Message{
				Role:    msg.Role,
				Content: msg.Content,
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
	_, err := storage.AddMessage(ctx, cc.ConversationID, "user", content)
	if err != nil {
		return err
	}

	// Add to context
	cc.Messages = append(cc.Messages, Message{
		Role:    "user",
		Content: content,
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
	_, err := storage.AddMessage(ctx, cc.ConversationID, "assistant", content)
	if err != nil {
		return err
	}

	// Add to context
	cc.Messages = append(cc.Messages, Message{
		Role:    "assistant",
		Content: content,
	})

	return nil
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
	}

	req := OllamaRequest{
		Model: cc.Model,
	}

	// Convert our messages to Ollama format
	for _, msg := range cc.Messages {
		req.Messages = append(req.Messages, OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
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
