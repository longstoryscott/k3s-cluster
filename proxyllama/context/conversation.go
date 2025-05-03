package context

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"proxyllama/config"
	"proxyllama/models"
	"proxyllama/storage"

	"github.com/jackc/pgx/v5"
)

// ConversationContext holds data needed to maintain context across requests
type ConversationContext struct {
	ConversationID int              `json:"conversation_id"`
	Title          string           `json:"title"`
	UserID         string           `json:"user_id"`
	Model          string           `json:"model"`
	Messages       []models.Message `json:"messages"`
	Summaries      []models.Summary `json:"summaries"`                // Holds summaries of older messages
	MasterSummary  *models.Summary  `json:"master_summary,omitempty"` // Special summary of all summaries
}

// GetOrCreateConversation retrieves or creates a conversation context
func GetOrCreateConversation(ctx context.Context, userID, model string, conversationID *int) (*ConversationContext, error) {
	// Check if we have a conversation ID
	if conversationID != nil {
		// Try to get from cache first
		if cache := GetCache(); cache != nil {
			if cachedContext, found := cache.Get(userID, *conversationID); found {
				log.Printf("Retrieved conversation %d for user %s from cache", *conversationID, userID)
				return cachedContext, nil
			}
		}
	}

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

		if model == "" {
			convContext.Model = conv.Model
		}

		if convContext.Model == "" {
			return nil, fmt.Errorf("model not specified and conversation has no model")
		}

		// Load messages
		if err := loadConversationMessages(ctx, &convContext); err != nil {
			return nil, err
		}

		// Load summaries
		if err := loadConversationSummaries(ctx, &convContext); err != nil {
			return nil, err
		}

		// Store in cache
		if cache := GetCache(); cache != nil {
			cache.Set(&convContext)
			log.Printf("Cached conversation %d for user %s", convContext.ConversationID, userID)
		}
	} else {
		// Create a new conversation
		id, err := storage.CreateConversation(ctx, userID, model, "New conversation")
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
		convContext.ConversationID = id

		// Store new conversation in cache
		if cache := GetCache(); cache != nil {
			cache.Set(&convContext)
		}
	}

	return &convContext, nil
}

// loadConversationMessages loads messages for a conversation
func loadConversationMessages(ctx context.Context, cc *ConversationContext) error {
	messages, err := storage.GetConversationHistory(ctx, cc.ConversationID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to load conversation history: %w", err)
	}

	// Convert storage.Message to models.Message
	for _, msg := range messages {
		cc.Messages = append(cc.Messages, models.Message{
			Role:    msg.Role,
			Content: msg.Content,
			ID:      msg.ID,
		})
	}

	return nil
}

// loadConversationSummaries loads summaries for a conversation
func loadConversationSummaries(ctx context.Context, cc *ConversationContext) error {
	summaries, err := storage.GetSummariesForConversation(ctx, cc.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to load summaries: %w", err)
	}

	// Convert storage.Summary to models.Summary
	for _, summary := range summaries {
		// Check if this is a master summary (level 0)
		if summary.Level == 0 {
			// Find the most recent master summary
			if cc.MasterSummary == nil || summary.ID > cc.MasterSummary.ID {
				cc.MasterSummary = &models.Summary{
					Content: summary.Content,
					Level:   summary.Level,
					ID:      summary.ID,
				}
			}
		} else {
			cc.Summaries = append(cc.Summaries, models.Summary{
				Content: summary.Content,
				Level:   summary.Level,
				ID:      summary.ID,
			})
		}
	}

	return nil
}

// AddUserMessage adds a user message to the conversation
func (cc *ConversationContext) AddUserMessage(ctx context.Context, content string) error {
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "user", content)
	if err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Add to context
	cc.Messages = append(cc.Messages, models.Message{
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

	// Update cache with modified conversation context
	if cache := GetCache(); cache != nil {
		cache.Set(cc)
	}

	return nil
}

// AddAssistantMessage adds an assistant message to the conversation
func (cc *ConversationContext) AddAssistantMessage(ctx context.Context, content string) error {
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "assistant", content)
	if err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Add to context
	cc.Messages = append(cc.Messages, models.Message{
		Role:    "assistant",
		Content: content,
		ID:      msgID,
	})

	// Check if we need to summarize messages
	if cc.shouldSummarize() {
		log.Printf("Summarizing messages for conversation %d", cc.ConversationID)
		if _, err := cc.SummarizeMessages(ctx); err != nil {
			log.Printf("Failed to summarize messages: %v", err)
			// Continue even if summarization fails
		}
	}

	// Update cache with modified conversation context
	if cache := GetCache(); cache != nil {
		cache.Set(cc)
	}

	return nil
}

// ToJSON converts the conversation context to Ollama-compatible format
func (cc *ConversationContext) ToJSON() ([]byte, error) {
	req := models.OllamaReq{
		Model:  cc.Model,
		Stream: true,
	}

	// Add master summary if available
	if cc.MasterSummary != nil {
		// Add system message to introduce the conversation with master summary
		req.Messages = append(req.Messages, CreateSystemMessage(
			"This is a continued conversation. Here is a comprehensive summary of the conversation history:"))

		// Add the master summary
		req.Messages = append(req.Messages, CreateSystemMessage(cc.MasterSummary.Content))

		log.Printf("Using master summary (ID: %d) for conversation context", cc.MasterSummary.ID)
	}

	// Add level summaries (one from each level)
	if err := cc.addLevelSummariesToReq(&req); err != nil {
		return nil, err
	}

	// Add recent messages
	if err := cc.addRecentMessagesToReq(&req); err != nil {
		return nil, err
	}

	// debug log the content of each message in the request
	for _, msg := range req.Messages {
		log.Printf("Message: %s", msg.Content)
	}

	log.Printf("Message chain order: %s", describeMessageChain(req.Messages))

	return json.Marshal(req)
}

// addLevelSummariesToReq adds one summary from each level to the request
func (cc *ConversationContext) addLevelSummariesToReq(req *models.OllamaReq) error {
	conf := config.GetConfig()

	// Find highest summary level
	highestLevel := FindMaxLevel(cc.Summaries)

	// Group summaries by level for more efficient access
	summariesByLevel := GroupSummariesByLevel(cc.Summaries)

	// Track how many summaries we've added
	summaryCount := 0

	// Add exactly one summary from each level, starting with highest
	maxLevel := conf.Summarization.MaxSummaryLevels
	for level := highestLevel; level >= 1 && level <= maxLevel; level-- {
		levelSummaries := summariesByLevel[level]
		if len(levelSummaries) == 0 {
			continue
		}

		// Take only the most recent summary from this level
		mostRecentSummary := levelSummaries[len(levelSummaries)-1]

		req.Messages = append(req.Messages, CreateSystemMessage(
			fmt.Sprintf("Previous conversation summary (level %d): %s", level, mostRecentSummary.Content)))
		summaryCount++
	}

	if summaryCount > 0 {
		log.Printf("Using %d summaries (one per level) for conversation context", summaryCount)
	}

	return nil
}

// addRecentMessagesToReq adds the most recent messages to the request
func (cc *ConversationContext) addRecentMessagesToReq(req *models.OllamaReq) error {
	conf := config.GetConfig()

	// Include only N most recent messages (user or assistant)
	messagesToInclude := conf.Summarization.MessagesBeforeSummary
	if len(cc.Messages) < messagesToInclude {
		messagesToInclude = len(cc.Messages)
	}

	// Calculate starting index for messages
	startIndex := len(cc.Messages) - messagesToInclude
	if startIndex < 0 {
		startIndex = 0
	}

	log.Printf("Including %d most recent messages in request to Ollama", messagesToInclude)

	// Add regular messages (most recent based on configuration)
	// Ensure messages are in chronological order (oldest first, newest last)
	for i := startIndex; i < len(cc.Messages); i++ {
		req.Messages = append(req.Messages, models.OllamaMessage{
			Role:    cc.Messages[i].Role,
			Content: cc.Messages[i].Content,
		})
	}

	return nil
}
