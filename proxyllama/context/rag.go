package context

import (
	"context"
	"fmt"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"proxyllama/util"
	"time"
)

// RetrievedMemory represents a message that was retrieved based on vector similarity
type RetrievedMemory struct {
	Message    storage.Message
	Similarity float32
}

// GetRelevantMemories retrieves semantically similar messages based on a query
func (cc *ConversationContext) GetRelevantMemories(ctx context.Context, query string) ([]models.OllamaChatMessage, error) {
	usrCfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	// Skip if RAG is not enabled
	if !usrCfg.Summarization.EnableRAG {
		return nil, nil
	}

	profile, err := storage.GetModelProfile(ctx, usrCfg.ModelProfiles.EmbeddingProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model profile: %w", err)
	}

	util.LogInfo("Generating embedding for query to find relevant memories")

	queryEmbedding, err := proxy.GetOllamaEmbedding(ctx, query, profile.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get similar messages from database
	similarMessages, err := storage.GetSimilarMessages(ctx, queryEmbedding, float32(usrCfg.Retrieval.SimilarityThreshold), usrCfg.Retrieval.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve similar messages: %w", err)
	}

	if len(similarMessages) == 0 {
		util.LogInfo("No relevant memories found for query")
		return nil, nil
	}

	util.LogInfo("Found relevant memories for query", map[string]interface{}{"count": len(similarMessages)})

	// Convert to OllamaMessages for context
	var ollamaMessages []models.OllamaChatMessage
	for _, msg := range similarMessages {
		ollamaMessages = append(ollamaMessages, models.OllamaChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return ollamaMessages, nil
}

// EnhanceRequestWithRAG adds relevant memories to the request based on the latest user query
func (cc *ConversationContext) EnhanceRequestWithRAG(ctx context.Context, req *models.OllamaChatReq) error {
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
	relevantMemories, err := cc.GetRelevantMemories(timeoutCtx, latestUserMessage)
	if err != nil {
		util.LogWarning(fmt.Sprintf("failed to get relevant memories: %v", err))
		return nil // Continue without RAG enhancement
	}

	if len(relevantMemories) == 0 {
		return nil // No relevant memories found, continue with original request
	}

	// Create a new request with memories inserted at the right position
	var enhancedMessages []models.OllamaChatMessage

	// First add any system messages (these should come first)
	systemMessagesCount := 0
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			enhancedMessages = append(enhancedMessages, msg)
			systemMessagesCount++
		}
	}

	// Add a system message to explain the memories
	enhancedMessages = append(enhancedMessages, models.OllamaChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("Here are %d relevant memories from previous conversations that may help answer the current query:", len(relevantMemories)),
	})

	// Add the retrieved memories
	enhancedMessages = append(enhancedMessages, relevantMemories...)

	// Add another system message to separate memories from the current conversation
	enhancedMessages = append(enhancedMessages, models.OllamaChatMessage{
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

	util.LogInfo("Enhanced request with relevant memories", map[string]interface{}{"count": len(relevantMemories)})
	return nil
}
