package context

import (
	"context"
	"fmt"
	"path/filepath"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// RetrievedMemory represents a message that was retrieved based on vector similarity
type RetrievedMemory struct {
	Message    storage.Message
	Similarity float32
}

// GetRelevantMemories retrieves semantically similar messages based on a query
func (cc *ConversationContext) GetRelevantMemories(ctx context.Context, query string, limit int) ([]models.OllamaMessage, error) {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	// Skip if RAG is not enabled
	if !usrCfg.Summarization.EnableRAG {
		return nil, nil
	}

	// Set default limit if not specified
	if limit <= 0 {
		limit = 5 // Default to 5 most relevant memories
	}

	profile, err := storage.GetModelProfile(ctx, usrCfg.ModelProfiles.EmbeddingProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model profile: %w", err)
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Generating embedding for query to find relevant memories")

	queryEmbedding, err := proxy.GetEmbedding(ctx, query, profile.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Define similarity threshold (adjust as needed)
	similarityThreshold := float32(0.7) // Messages with 70%+ similarity

	// Get similar messages from database
	similarMessages, err := storage.GetSimilarMessages(ctx, queryEmbedding, similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve similar messages: %w", err)
	}

	if len(similarMessages) == 0 {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("No relevant memories found for query")
		return nil, nil
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": len(similarMessages),
	}).Info("Found relevant memories for query")

	// Convert to OllamaMessages for context
	var ollamaMessages []models.OllamaMessage
	for _, msg := range similarMessages {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return ollamaMessages, nil
}

// EnhanceRequestWithRAG adds relevant memories to the request based on the latest user query
func (cc *ConversationContext) EnhanceRequestWithRAG(ctx context.Context, req *models.OllamaReq) error {
	// Find the latest user message to use as query
	var latestUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			latestUserMessage = req.Messages[i].Content
			break
		}
	}

	if latestUserMessage == "" {
		return fmt.Errorf("no user message found in request")
	}

	// Set context timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get relevant memories
	relevantMemories, err := cc.GetRelevantMemories(timeoutCtx, latestUserMessage, 5)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to get relevant memories")
		return nil // Continue without RAG enhancement
	}

	if len(relevantMemories) == 0 {
		return nil // No relevant memories found, continue with original request
	}

	// Create a new request with memories inserted at the right position
	var enhancedMessages []models.OllamaMessage

	// First add any system messages (these should come first)
	systemMessagesCount := 0
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			enhancedMessages = append(enhancedMessages, msg)
			systemMessagesCount++
		}
	}

	// Add a system message to explain the memories
	enhancedMessages = append(enhancedMessages, models.OllamaMessage{
		Role:    "system",
		Content: fmt.Sprintf("Here are %d relevant memories from previous conversations that may help answer the current query:", len(relevantMemories)),
	})

	// Add the retrieved memories
	enhancedMessages = append(enhancedMessages, relevantMemories...)

	// Add another system message to separate memories from the current conversation
	enhancedMessages = append(enhancedMessages, models.OllamaMessage{
		Role:    "system",
		Content: "Now continuing with the current conversation:",
	})

	// Add the non-system messages from the original request
	for i, msg := range req.Messages {
		if i >= systemMessagesCount {
			enhancedMessages = append(enhancedMessages, msg)
		}
	}

	// Replace messages in the request
	req.Messages = enhancedMessages

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": len(relevantMemories),
	}).Info("Enhanced request with relevant memories")
	return nil
}
