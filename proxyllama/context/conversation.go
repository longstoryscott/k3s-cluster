package context

import (
	"context"
	"encoding/json"
	"fmt"
	"proxyllama/models"
	"proxyllama/storage"
	"proxyllama/util"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

// ConversationContext manages context for a conversation
type ConversationContext struct {
	UserID            string
	ConversationID    int
	Title             string
	MasterSummary     *models.Summary
	Summaries         []models.Summary
	Messages          []models.Message
	RetrievedMemories []models.Memory  // Memories retrieved using semantic or keyword search
	SearchResults     []models.Message // Search results from web search
}

// GetOrCreateConversation retrieves or creates a conversation context
func GetOrCreateConversation(ctx context.Context, userID string, conversationID *int) (*ConversationContext, error) {
	// Check if we have a conversation ID
	if conversationID != nil {
		// Try to get from cache first
		if cache := GetCache(); cache != nil {
			if cachedContext, found := cache.Get(userID, *conversationID); found {
				util.LogInfo("Retrieved conversation from cache", logrus.Fields{
					"conversationId": *conversationID,
					"userId":         userID,
				})
				return cachedContext, nil
			}
		}
	}

	// Ensure user exists and load user-specific configuration
	if err := storage.EnsureUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to ensure user exists: %w", err)
	}

	var convContext ConversationContext
	convContext.UserID = userID
	// convContext.Model = model

	// If conversationID is provided, load that conversation
	if conversationID != nil {
		// Verify the conversation exists and belongs to the user
		conv, err := storage.GetConversation(ctx, *conversationID)
		if err != nil {
			if err == pgx.ErrNoRows {
				cid, err := storage.CreateConversation(ctx, userID, "")
				if err != nil {
					return nil, fmt.Errorf("failed to create conversation: %w", err)
				}
				return GetOrCreateConversation(ctx, userID, &cid)
			}
			return nil, fmt.Errorf("failed to get conversation: %w", err)
		}

		if conv.UserID != userID {
			return nil, fmt.Errorf("conversation does not belong to user")
		}

		convContext.ConversationID = conv.ID
		convContext.Title = conv.Title

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
			util.LogInfo("Cached conversation", logrus.Fields{
				"conversationId": convContext.ConversationID,
				"userId":         userID,
			})
		}
	} else {
		// Create a new conversation
		id, err := storage.CreateConversation(ctx, userID, "New conversation")
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
func (cc *ConversationContext) AddUserMessage(ctx context.Context, content string) ([][]float32, error) {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, util.HandleError(fmt.Errorf("failed to get user config: %w", err))
	}
	// Add to database
	msgID, embedding, err := storage.AddMessage(ctx, cc.ConversationID, "user", content, usrCfg)
	if err != nil {
		return nil, util.HandleError(fmt.Errorf("failed to add user message: %w", err))
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
			util.HandleError(fmt.Errorf("failed to update conversation title: %v", err))
		}
	}

	// Update cache with modified conversation context
	if cache := GetCache(); cache != nil {
		cache.Set(cc)
	}

	return embedding, nil
}

// AddAssistantMessage adds an assistant message to the conversation
func (cc *ConversationContext) AddAssistantMessage(ctx context.Context, content string) ([][]float32, error) {
	util.LogInfo("Storing assistant response", logrus.Fields{"length": len(content)})

	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}
	// Add to database
	msgID, embedding, err := storage.AddMessage(ctx, cc.ConversationID, "assistant", content, usrCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Add to context
	cc.Messages = append(cc.Messages, models.Message{
		Role:    "assistant",
		Content: content,
		ID:      msgID,
	})

	// Check if we need to summarize messages
	if cc.shouldSummarize() {
		util.LogInfo("Summarizing messages for conversation")
		go func() {
			bgrndCtx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
			defer cancel()
			if _, err := cc.SummarizeMessages(bgrndCtx); err != nil {
				util.HandleError(fmt.Errorf("failed to summarize messages: %w", err))
				// Continue even if summarization fails
			}
		}()
	}

	go func(cc *ConversationContext) {
		// Update cache with modified conversation context
		if cache := GetCache(); cache != nil {
			cache.Set(cc)
		}
	}(cc)

	return embedding, nil
}

// ToJSON converts the conversation context to Ollama-compatible format
func (cc *ConversationContext) ToJSON() ([]byte, error) {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, util.HandleError(fmt.Errorf("failed to get user config: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	defaultModel, err := storage.GetModelProfile(ctx, usrCfg.ModelProfiles.PrimaryProfileID)
	if err != nil {
		return nil, util.HandleError(fmt.Errorf("failed to get default model profile: %w", err))
	}

	req := models.OllamaChatReq{
		Model:   defaultModel.ModelName,
		Stream:  true,
		Options: defaultModel.Parameters.ToMap(),
	}

	// Add retrieved memories first if available
	if len(cc.RetrievedMemories) > 0 {
		req.Messages = append(req.Messages, CreateSystemMessage(
			"Here is some potentially relevant context from our past discussions:"))

		for _, mem := range cc.RetrievedMemories {
			// For system role messages, keep them as system
			role := "system"
			if strings.Contains(mem.Content, "From conversation") {
				// Already formatted as system message
				content := mem.Content
				req.Messages = append(req.Messages, models.OllamaChatMessage{
					Role:    role,
					Content: content,
				})
			} else {
				// Format the memory with role information
				content := fmt.Sprintf("Previously (%s): %s", mem.Role, mem.Content)
				req.Messages = append(req.Messages, models.OllamaChatMessage{
					Role:    role,
					Content: content,
				})
			}
		}

		util.LogInfo("Added retrieved memories to request context", logrus.Fields{
			"count": len(cc.RetrievedMemories),
		})
	}

	// Add search results if available
	if len(cc.SearchResults) > 0 {
		req.Messages = append(req.Messages, CreateSystemMessage(
			"Here are some relevant search results that might help:"))

		for _, result := range cc.SearchResults {
			req.Messages = append(req.Messages, models.OllamaChatMessage{
				Role:    "system",
				Content: result.Content,
			})
		}
	}

	// Add master summary if available
	if cc.MasterSummary != nil {
		// Add system message to introduce the conversation with master summary
		req.Messages = append(req.Messages, CreateSystemMessage(
			"This is a continued conversation. Here is a comprehensive summary of the conversation history:"))

		// Add the master summary
		req.Messages = append(req.Messages, CreateSystemMessage(cc.MasterSummary.Content))

		util.LogInfo("Using master summary for conversation context", logrus.Fields{
			"summaryId": cc.MasterSummary.ID,
		})
	}

	// Add level summaries (one from each level)
	if err := cc.addLevelSummariesToReq(&req); err != nil {
		return nil, err
	}

	// Add recent messages
	if err := cc.addRecentMessagesToReq(&req); err != nil {
		return nil, err
	}

	// Apply RAG enhancement if enabled (adds relevant memories from previous conversations)
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		util.LogWarning("Could not load user configuration, using system defaults", logrus.Fields{"error": err})
		return nil, err
	}

	if userConfig.Summarization.EnableRAG {
		defer cancel()

		if err := cc.EnhanceRequestWithRAG(ctx, &req); err != nil {
			util.LogWarning("Failed to enhance request with RAG", logrus.Fields{"error": err})
		}
	}

	// debug log the content of each message in the request
	for i, msg := range req.Messages {
		util.LogDebug(truncateForLog(msg.Content), logrus.Fields{
			"index": i,
			"role":  msg.Role,
		})
	}

	util.LogDebug("Message chain order", logrus.Fields{
		"chain": describeMessageChain(req.Messages),
	})

	return json.Marshal(req)
}

// addLevelSummariesToReq adds one summary from each level to the request
func (cc *ConversationContext) addLevelSummariesToReq(req *models.OllamaChatReq) error {
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		util.LogWarning("Could not load user configuration, using system defaults", logrus.Fields{"error": err})
		return err
	}

	highestLevel := findMaxLevel(cc.Summaries)
	summariesByLevel := groupSummariesByLevel(cc.Summaries)
	summaryCount := 0
	maxLevel := userConfig.Summarization.MaxSummaryLevels
	for level := highestLevel; level >= 1 && level <= maxLevel; level-- {
		levelSummaries := summariesByLevel[level]
		if len(levelSummaries) == 0 {
			continue
		}
		mostRecentSummary := levelSummaries[len(levelSummaries)-1]
		req.Messages = append(req.Messages, CreateSystemMessage(
			fmt.Sprintf("Previous conversation summary (level %d): %s", level, mostRecentSummary.Content)))
		summaryCount++
	}
	if summaryCount > 0 {
		util.LogInfo("Using summaries (one per level) for conversation context", logrus.Fields{
			"count": summaryCount,
		})
	}
	return nil
}

// addRecentMessagesToReq adds the most recent messages to the request
func (cc *ConversationContext) addRecentMessagesToReq(req *models.OllamaChatReq) error {
	// Get user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		util.LogWarning("Could not load user configuration, using system defaults", logrus.Fields{"error": err})
		return err
	}

	// Include only N most recent messages (user or assistant)
	messagesToInclude := min(len(cc.Messages), userConfig.Summarization.MessagesBeforeSummary)

	// Calculate starting index for messages
	startIndex := max(len(cc.Messages)-messagesToInclude, 0)

	util.LogInfo("Including most recent messages in request to Ollama", logrus.Fields{
		"count": messagesToInclude,
	})

	// Add regular messages (most recent based on configuration)
	// Ensure messages are in chronological order (oldest first, newest last)
	for i := startIndex; i < len(cc.Messages); i++ {
		req.Messages = append(req.Messages, models.OllamaChatMessage{
			Role:    cc.Messages[i].Role,
			Content: cc.Messages[i].Content,
		})
	}

	return nil
}

// CreateSystemMessage is a helper to create a system message
func CreateSystemMessage(content string) models.OllamaChatMessage {
	return models.OllamaChatMessage{
		Role:    "system",
		Content: content,
	}
}

// describeMessageChain returns a string describing the message chain order
func describeMessageChain(messages []models.OllamaChatMessage) string {
	result := ""
	for _, msg := range messages {
		// Append first character of role (S for system, U for user, A for assistant)
		result += string(msg.Role[0])
	}
	return result
}

// generateTitle creates a title from the first user message
func generateTitle(content string) string {
	// Simple implementation: use first 30 chars of content or less
	maxLen := 30
	title := content
	if len(title) > maxLen {
		title = title[:maxLen] + "..."
	}
	return title
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(content string) string {
	maxLen := 100
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// shouldSummarize determines if the conversation needs to be summarized
func (cc *ConversationContext) shouldSummarize() bool {
	// Load user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		util.LogWarning("Could not load user configuration, using system defaults", logrus.Fields{"error": err})
		return false
	}

	// Get message threshold from configuration
	messageThreshold := userConfig.Summarization.MessagesBeforeSummary

	// Count messages since last summary
	messagesSinceLastSummary := len(cc.Messages)

	// If we have summaries, adjust the count to only include messages since the last summary
	if len(cc.Summaries) > 0 || cc.MasterSummary != nil {
		// Find the most recent summary ID
		var maxSummaryID int
		if cc.MasterSummary != nil {
			maxSummaryID = cc.MasterSummary.ID
		}

		for _, summary := range cc.Summaries {
			if summary.ID > maxSummaryID {
				maxSummaryID = summary.ID
			}
		}

		// Count only messages that came after the most recent summary
		messagesSinceLastSummary = 0
		for _, msg := range cc.Messages {
			if msg.ID > maxSummaryID {
				messagesSinceLastSummary++
			}
		}
	}

	// Determine if we need to summarize
	return messagesSinceLastSummary >= messageThreshold
}
