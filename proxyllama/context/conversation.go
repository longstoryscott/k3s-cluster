package context

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"runtime"
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
	RetrievedMemories []models.Message // Memories retrieved using semantic or keyword search
}

// GetOrCreateConversation retrieves or creates a conversation context
func GetOrCreateConversation(ctx context.Context, userID string, conversationID *int) (*ConversationContext, error) {
	// Check if we have a conversation ID
	if conversationID != nil {
		// Try to get from cache first
		if cache := GetCache(); cache != nil {
			if cachedContext, found := cache.Get(userID, *conversationID); found {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":           line,
					"conversationId": *conversationID,
					"userId":         userID,
				}).Info("Retrieved conversation from cache")
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
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":           line,
				"conversationId": convContext.ConversationID,
				"userId":         userID,
			}).Info("Cached conversation")
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
func (cc *ConversationContext) AddUserMessage(ctx context.Context, content string) error {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user config: %w", err)
	}
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "user", content, usrCfg)
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
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":           line,
				"error":          err,
				"conversationId": cc.ConversationID,
			}).Error("Failed to update conversation title")
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
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user config: %w", err)
	}
	// Add to database
	msgID, err := storage.AddMessage(ctx, cc.ConversationID, "assistant", content, usrCfg)
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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":           line,
			"conversationId": cc.ConversationID,
		}).Info("Summarizing messages for conversation")
		go func() {
			bgrndCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if _, err := cc.SummarizeMessages(bgrndCtx); err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": err,
				}).Error("Failed to summarize messages")
				// Continue even if summarization fails
			}
		}()
	}

	// Update cache with modified conversation context
	if cache := GetCache(); cache != nil {
		cache.Set(cc)
	}

	return nil
}

// ToJSON converts the conversation context to Ollama-compatible format
func (cc *ConversationContext) ToJSON() ([]byte, error) {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	defaultModel, err := storage.GetModelProfile(context.Background(), usrCfg.ModelProfiles.PrimaryProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default model profile: %w", err)
	}

	req := models.OllamaReq{
		Model:  defaultModel.ModelName,
		Stream: true,
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
				req.Messages = append(req.Messages, models.OllamaMessage{
					Role:    role,
					Content: content,
				})
			} else {
				// Format the memory with role information
				content := fmt.Sprintf("Previously (%s): %s", mem.Role, mem.Content)
				req.Messages = append(req.Messages, models.OllamaMessage{
					Role:    role,
					Content: content,
				})
			}
		}

		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"count": len(cc.RetrievedMemories),
		}).Info("Added retrieved memories to request context")
	}

	// Add master summary if available
	if cc.MasterSummary != nil {
		// Add system message to introduce the conversation with master summary
		req.Messages = append(req.Messages, CreateSystemMessage(
			"This is a continued conversation. Here is a comprehensive summary of the conversation history:"))

		// Add the master summary
		req.Messages = append(req.Messages, CreateSystemMessage(cc.MasterSummary.Content))

		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":      line,
			"summaryId": cc.MasterSummary.ID,
		}).Info("Using master summary for conversation context")
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
	ctx := context.Background()
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load user configuration, using system defaults")
		return nil, err
	}

	if userConfig.Summarization.EnableRAG {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := cc.EnhanceRequestWithRAG(timeoutCtx, &req); err != nil {
			// Log the error but continue without RAG
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Warn("Failed to enhance request with RAG")
		}
	}

	// debug log the content of each message in the request
	for i, msg := range req.Messages {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"index": i,
			"role":  msg.Role,
		}).Debug(truncateForLog(msg.Content))
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"chain": describeMessageChain(req.Messages),
	}).Debug("Message chain order")

	return json.Marshal(req)
}

// addLevelSummariesToReq adds one summary from each level to the request
func (cc *ConversationContext) addLevelSummariesToReq(req *models.OllamaReq) error {
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load user configuration, using system defaults")
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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"count": summaryCount,
		}).Info("Using summaries (one per level) for conversation context")
	}
	return nil
}

// addRecentMessagesToReq adds the most recent messages to the request
func (cc *ConversationContext) addRecentMessagesToReq(req *models.OllamaReq) error {
	// Get user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load user configuration, using system defaults")
		return err
	}

	// Include only N most recent messages (user or assistant)
	messagesToInclude := userConfig.Summarization.MessagesBeforeSummary
	if len(cc.Messages) < messagesToInclude {
		messagesToInclude = len(cc.Messages)
	}

	// Calculate starting index for messages
	startIndex := len(cc.Messages) - messagesToInclude
	if startIndex < 0 {
		startIndex = 0
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": messagesToInclude,
	}).Info("Including most recent messages in request to Ollama")

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

// CreateSystemMessage is a helper to create a system message
func CreateSystemMessage(content string) models.OllamaMessage {
	return models.OllamaMessage{
		Role:    "system",
		Content: content,
	}
}

// describeMessageChain returns a string describing the message chain order
func describeMessageChain(messages []models.OllamaMessage) string {
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

// RetrieveAndInjectMemories retrieves relevant memories based on the current user query
func (cc *ConversationContext) RetrieveAndInjectMemories(ctx context.Context, currentUserQuery string) error {
	// Clear any previous memories
	cc.RetrievedMemories = nil

	// Get user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load user configuration, using system defaults")
		return err
	}

	if !userConfig.Retrieval.Enabled {
		return nil // Retrieval disabled in user config
	}

	// Check if query suggests memory retrieval would be helpful
	shouldRetrieve := cc.shouldRetrieveMemories(currentUserQuery)
	if !shouldRetrieve && !userConfig.Retrieval.AlwaysRetrieve {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"query": truncateForLog(currentUserQuery),
		}).Info("Query doesn't appear to need memory retrieval, skipping")
		return nil
	}

	// Set a search limit from user config
	limit := userConfig.Retrieval.Limit
	if limit <= 0 {
		limit = 5
	}

	profile, err := storage.GetModelProfile(ctx, userConfig.ModelProfiles.EmbeddingProfileID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load embedding model profile, using system defaults")
		return nil
	}

	// First try vector similarity search if RAG is enabled
	if userConfig.Summarization.EnableRAG {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Performing semantic search for memories")
		queryEmbedding, err := proxy.GetEmbedding(ctx, currentUserQuery, profile.ModelName)
		if err == nil && len(queryEmbedding) > 0 {
			// Search in the current conversation first
			similarMessages, err := storage.SearchMessagesBySimilarity(ctx, cc.ConversationID, queryEmbedding, limit)
			if err == nil && len(similarMessages) > 0 {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"count": len(similarMessages),
				}).Info("Found semantically similar messages in current conversation")
				// Convert and add to retrieved memories
				for _, msg := range similarMessages {
					cc.RetrievedMemories = append(cc.RetrievedMemories, models.Message{
						Role:    msg.Role,
						Content: msg.Content,
						ID:      msg.ID,
					})
				}
				return nil
			}

			// If cross-conversation memory retrieval is enabled, try that
			if userConfig.Retrieval.EnableCrossConversation {
				// Search across all conversations with similarity threshold
				threshold := userConfig.Retrieval.SimilarityThreshold
				if threshold <= 0 {
					threshold = 0.7 // Default threshold
				}

				similarMessages, err = storage.SearchAllMessagesBySimilarity(ctx, queryEmbedding, threshold, limit)
				if err == nil && len(similarMessages) > 0 {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":  line,
						"count": len(similarMessages),
					}).Info("Found semantically similar messages across conversations")
					// Convert and add to retrieved memories
					for _, msg := range similarMessages {
						formattedContent := fmt.Sprintf(
							"From conversation #%d (%s): %s",
							msg.ConversationID,
							msg.Role,
							msg.Content,
						)
						cc.RetrievedMemories = append(cc.RetrievedMemories, models.Message{
							Role:    "system", // Treat cross-conversation memories as system context
							Content: formattedContent,
							ID:      msg.ID,
						})
					}
					return nil
				}
			}
		}
	}

	// Fall back to keyword search if vector search didn't yield results
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Falling back to keyword search")
	keywordMessages, err := storage.SearchMessagesByKeyword(ctx, cc.ConversationID, currentUserQuery, limit)
	if err == nil && len(keywordMessages) > 0 {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"count": len(keywordMessages),
		}).Info("Found keyword-matched messages")
		// Convert and add to retrieved memories
		for _, msg := range keywordMessages {
			cc.RetrievedMemories = append(cc.RetrievedMemories, models.Message{
				Role:    msg.Role,
				Content: msg.Content,
				ID:      msg.ID,
			})
		}
		return nil
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"query": truncateForLog(currentUserQuery),
	}).Info("No relevant memories found for query")
	return nil
}

// shouldRetrieveMemories determines if a query likely needs memory retrieval
func (cc *ConversationContext) shouldRetrieveMemories(query string) bool {
	// Convert query to lowercase for case-insensitive matching
	lowercaseQuery := strings.ToLower(query)

	// List of keywords and phrases that suggest the user is asking about past information
	memoryTriggers := []string{
		"remember", "recall", "previous", "earlier", "before", "last time",
		"you said", "mentioned", "told me", "yesterday", "last week",
		"forgot", "remind me", "i asked", "we discussed", "we talked about",
		"what did i", "what did you", "did i tell", "did you tell",
	}

	// Check if any memory trigger is in the query
	for _, trigger := range memoryTriggers {
		if strings.Contains(lowercaseQuery, trigger) {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":    line,
				"trigger": trigger,
			}).Info("Memory retrieval triggered by keyword")
			return true
		}
	}

	// Question patterns that often benefit from memory retrieval
	questionPatterns := []string{
		"what was", "who was", "where was", "when was", "how was",
		"what were", "who were", "where were", "when were", "how were",
		"what did", "who did", "where did", "when did", "how did",
	}

	// Check for question patterns
	for _, pattern := range questionPatterns {
		if strings.Contains(lowercaseQuery, pattern) {
			// Check if the question appears to be about past interactions
			if strings.Contains(lowercaseQuery, "you") || strings.Contains(lowercaseQuery, "we") || strings.Contains(lowercaseQuery, "i") {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":    line,
					"pattern": pattern,
				}).Info("Memory retrieval triggered by question pattern")
				return true
			}
		}
	}

	return false
}

// shouldSummarize determines if the conversation needs to be summarized
func (cc *ConversationContext) shouldSummarize() bool {
	// Load user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Could not load user configuration, using system defaults")
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
